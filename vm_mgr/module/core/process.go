/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
)

type processState int

// all process state synchronization
// p(ready->idle): p==ready, invoke ChangeProcessState, process manager->idle, process->idle
// p(idle->busy): p==busy, invoke ChangeProcessState, process manager->busy, process->busy
// pm(idle->del): invoke ChangeSandbox, p==idle ? kill, ->changing, newctx->created, pm del old , add new->busy : revert
// p(before started->killed): restart ? nothing : return exit resp
// p(close signal->killed)
const (
	created  processState = iota // creating process (busy in process manager)
	ready                        // ready to recv tx (busy in process manager)
	busy                         // running tx (busy in process manager)
	idle                         // idling (idle in process manager)
	changing                     // changing sandbox (busy in process manager)
	closing                      // closing sandbox (idle in process manager)
	timeout                      // busy timeout (busy in process manager)
)

const (
	_forcedKillWaitTime = 200 * time.Millisecond
	_removeTxTime       = 9000 * time.Millisecond
)

const (
	initContract    = "init_contract"
	upgradeContract = "upgrade"
)

// exitErr is the sandbox exit err
type exitErr struct {
	err  error
	desc string
}

// Process manage the sandbox process life cycle
type Process struct {
	processName string

	chainID         string
	contractName    string
	contractVersion string

	cGroupPath string
	user       interfaces.User
	cmd        *exec.Cmd

	processState  processState
	isOrigProcess bool

	cmdReadyCh chan bool
	exitCh     chan *exitErr
	updateCh   chan struct{}
	txCh       chan *messages.TxPayload
	respCh     chan *protogo.DockerVMMessage
	timer      *time.Timer

	Tx *protogo.DockerVMMessage

	logger *zap.SugaredLogger

	stream protogo.DockerVMRpc_DockerVMCommunicateServer

	processManager   interfaces.ProcessManager
	requestGroup     interfaces.RequestGroup
	requestScheduler interfaces.RequestScheduler

	lock sync.RWMutex

	timerPopped bool
}

// check interface implement
var _ interfaces.Process = (*Process)(nil)

// NewProcess new process, process working on main contract which is not called cross contract
func NewProcess(user interfaces.User, chainID, contractName, contractVersion, processName string,
	manager interfaces.ProcessManager, scheduler interfaces.RequestScheduler, isOrigProcess bool) *Process {

	process := &Process{
		processName: processName,

		chainID:         chainID,
		contractName:    contractName,
		contractVersion: contractVersion,

		cGroupPath: filepath.Join(security.CGroupRoot, security.ProcsFile),
		user:       user,

		processState:  created,
		isOrigProcess: isOrigProcess,

		cmdReadyCh: make(chan bool, 1),
		exitCh:     make(chan *exitErr),
		updateCh:   make(chan struct{}, 1),
		respCh:     make(chan *protogo.DockerVMMessage, 1),
		timer:      time.NewTimer(math.MaxInt32 * time.Second), //initial tx timer, never triggered

		logger: logger.NewDockerLogger(logger.GenerateProcessLoggerName(processName)),

		processManager:   manager,
		requestScheduler: scheduler,

		lock: sync.RWMutex{},
	}

	// GetRequestGroup is safe here, process has added to process manager, not nil
	process.requestGroup, _ = scheduler.GetRequestGroup(chainID, contractName, contractVersion)

	process.txCh = process.requestGroup.GetTxCh(isOrigProcess)

	return process
}

// PutMsg put invoking requests to chan, waiting for process to handle request
//  @param req types include DockerVMType_TX_REQUEST, ChangeSandboxReqMsg and CloseSandboxReqMsg
func (p *Process) PutMsg(msg *protogo.DockerVMMessage) {

	p.respCh <- msg
}

// Start process, listen channels and exec cmd
func (p *Process) Start() {

	p.logger.Debugf("start process")

	go p.listenProcess()
	p.startProcess()
}

// startProcess starts the process cmd
func (p *Process) startProcess() {
	err := p.launchProcess()
	p.exitCh <- err
}

// launchProcess launch a new process
func (p *Process) launchProcess() *exitErr {

	p.logger.Debugf("[%s] start launch process", p.getTxId())

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	tcpPort := config.DockerVMConfig.RPC.SandboxRPCPort
	if config.DockerVMConfig.RPC.ChainRPCProtocol == config.UDS {
		tcpPort = 0
	}

	path := p.requestGroup.GetContractPath()
	cfv := p.requestGroup.GetContractFileVersion()

	cmd := exec.Cmd{
		Path: path,
		Args: []string{
			p.user.GetSockPath(),
			p.processName,
			p.contractName,
			p.contractVersion,
			config.DockerVMConfig.Log.SandboxLog.Level,
			strconv.Itoa(tcpPort),
			config.DockerVMConfig.RPC.ChainHost,
			strconv.FormatBool(config.DockerVMConfig.Slow.Disable),
			strconv.Itoa(config.DockerVMConfig.RPC.MaxSendMsgSize),
			strconv.Itoa(config.DockerVMConfig.RPC.MaxRecvMsgSize),
		},
		Stderr: &stderr,
	}

	contractOut, err := cmd.StdoutPipe()
	if err != nil {
		return &exitErr{
			err:  err,
			desc: stderr.String(),
		}
	}
	// these settings just working on linux,
	// but it doesn't affect running, because it will put into docker to run
	// setting pid namespace and allocate special uid for process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(p.user.GetUid()),
		},
		Cloneflags: syscall.CLONE_NEWPID,
	}
	p.cmd = &cmd

	if err = cmd.Start(); err != nil {
		p.logger.Errorf("failed to start process: %v, %v", err, stderr.String())
		if os.IsNotExist(err) {
			return &exitErr{
				err:  utils.ContractNotExistError,
				desc: strconv.FormatInt(cfv, 10),
			}
		}
		return &exitErr{
			err:  utils.ContractExecError,
			desc: strconv.FormatInt(cfv, 10),
		}
	}
	p.cmdReadyCh <- true

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, strconv.Itoa(cmd.Process.Pid)); err != nil {
		p.logger.Errorf("failed to add cgroup: %s", err)
		return &exitErr{
			err:  err,
			desc: "",
		}
	}
	p.logger.Debugf("[%s] add process to cgroup", p.getTxId())
	p.logger.Debugf("[%s] process started", p.getTxId())

	p.printContractLog(contractOut)

	if err = cmd.Wait(); err != nil {
		var txId string
		if p.Tx != nil {
			txId = p.Tx.TxId
		}
		p.logger.Debugf("process stopped for tx [%s], %v, %v", txId, err, stderr.String())
		return &exitErr{
			err:  err,
			desc: stderr.String(),
		}
	}

	return nil
}

// printContractLog print the sandbox cmd log
func (p *Process) printContractLog(contractPipe io.ReadCloser) {
	contractLogger := logger.NewDockerLogger(logger.MODULE_CONTRACT)
	zapLogger := contractLogger.Desugar()
	logLevel := logger.GetLogLevel()
	rd := bufio.NewReader(contractPipe)
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			if err.Error() != "EOF" {
				contractLogger.Info(err)
			}
			return
		}
		str = strings.TrimSuffix(str, "\n")
		zapLogger.Check(logLevel, str).Write()
	}
}

// listenProcess listen to channels
func (p *Process) listenProcess() {

	for {
		if p.processState == ready {
			select {

			case tx := <-p.txCh:
				// condition: during cmd.wait
				if err := p.handleTxRequest(tx); err != nil {
					p.logger.Errorf("failed to handle tx [%s] request, %v", tx.Tx.TxId, err)
					if err := p.returnTxErrorResp(tx.Tx.TxId, err.Error()); err != nil {
						p.logger.Errorf("failed to return tx error response, %v", err)
					}
				}

			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle ready timeout timer, %v", err)
				}

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
			}
		} else if p.processState == busy {
			select {

			case resp := <-p.respCh:
				if err := p.handleTxResp(resp); err != nil {
					p.logger.Warnf("failed to handle tx response, %v", err)
				}
				utils.ReturnToPool(resp)

			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle busy timeout timer, %v, resp chan length: %d", err, len(p.respCh))
				}

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
			}
		} else if p.processState == idle {
			select {

			case tx := <-p.txCh:
				// condition: during cmd.wait
				if err := p.handleIdleTxRequest(tx); err != nil {
					p.logger.Errorf("failed to handle tx [%s] request when idle, %v", tx.Tx.TxId, err)
				}

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
			}
		} else {
			select {

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
			case _ = <-p.updateCh:
			}
		}
	}
}

// IsReadyOrBusy returns processState == ready || processState == busy
func (p *Process) IsReadyOrBusy() bool {

	return p.processState == ready || p.processState == busy
}

// GetProcessName returns process name
func (p *Process) GetProcessName() string {

	return p.processName
}

// GetChainID returns chain id
func (p *Process) GetChainID() string {

	return p.chainID
}

// GetContractName returns contract name
func (p *Process) GetContractName() string {

	return p.contractName
}

// GetContractVersion returns contract version
func (p *Process) GetContractVersion() string {

	return p.contractVersion
}

// GetUser returns user
func (p *Process) GetUser() interfaces.User {
	return p.user
}

// GetTx returns tx
func (p *Process) GetTx() *protogo.DockerVMMessage {
	return p.Tx
}

// SetStream sets grpc stream
func (p *Process) SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer) {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.updateProcessState(ready)
	p.stream = stream
}

// ChangeSandbox changes sandbox of process
func (p *Process) ChangeSandbox(chainID, contractName, contractVersion, processName string) error {

	p.logger.Debugf("[%s] process [%s] is changing to [%s]...", p.getTxId(), p.processName, processName)

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	if err := p.resetContext(chainID, contractName, contractVersion, processName); err != nil {
		return fmt.Errorf("failed to reset context, %v", err)
	}

	// TODO: kill process by send signal, reset exitCh, new process will never blocked
	if err := p.killProcess(syscall.SIGTERM); err != nil {
		return err
	}

	// if sandbox exited here, process while be holding util deleted from process manager
	p.updateProcessState(changing)

	return nil
}

// CloseSandbox close sandbox
func (p *Process) CloseSandbox() error {

	p.logger.Debugf("[%s] start to close sandbox", p.getTxId())

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	if err := p.killProcess(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to kill process, %v", err)
	}

	p.updateProcessState(closing)

	return nil
}

// handleTxRequest handle tx request from request group chan
func (p *Process) handleTxRequest(tx *messages.TxPayload) error {

	p.lock.Lock()
	defer p.lock.Unlock()

	elapsedTime := time.Since(tx.StartTime)
	if elapsedTime > _removeTxTime {
		p.logger.Warnf("tx [%s] expired for %v, elapsed time: %v", tx.Tx.TxId, _removeTxTime, elapsedTime)
		return nil
	} else if elapsedTime > config.DockerVMConfig.Process.ExecTxTimeout {
		return fmt.Errorf("tx [%s] expired for %v, elapsed time: %v", tx.Tx.TxId,
			config.DockerVMConfig.Process.ExecTxTimeout, elapsedTime)
	}

	logger.DebugDynamic(p.logger, func() string {
		return fmt.Sprintf("[%s] start handle tx req [%s]", p.getTxId(), tx.Tx.TxId)
	})

	utils.ReturnToPool(p.Tx)
	p.Tx = tx.Tx

	p.updateProcessState(busy)

	msg := utils.DockerVMMessageFromPool()
	msg.ChainId = p.chainID
	msg.TxId = p.Tx.TxId
	msg.CrossContext = p.Tx.CrossContext
	msg.Request = p.Tx.Request
	msg.StepDurations = tx.Tx.StepDurations

	utils.EnterNextStep(tx.Tx, protogo.StepType_ENGINE_PROCESS_SEND_TX_REQUEST, func() string {
		return strings.Join([]string{"waitingLen", strconv.Itoa(len(p.txCh))}, ":")
	})

	// send message to sandbox
	if err := p.sendMsg(msg); err != nil {
		return err
	}

	return nil
}

// handleIdleTxRequest handle tx request from request group chan when process state is idle
func (p *Process) handleIdleTxRequest(tx *messages.TxPayload) error {

	defer func() {
		// return tx
		if err := p.requestScheduler.PutMsg(tx.Tx); err != nil {
			p.logger.Errorf("failed to put msg [%s] to request group, %v", tx.Tx.TxId, err)
		}
	}()

	// change state from idle to busy
	if err := p.processManager.ChangeProcessState(p.processName, true); err != nil {
		// failed to change state to busy, return
		p.logger.Warnf("[%s] failed to change state to ready, %v", p.getTxId(), err)
		return nil
	}
	// succeed to change state to busy, update state for itself
	logger.DebugDynamic(p.logger, func() string {
		return fmt.Sprintf("[%s] change state from idle to ready", p.getTxId())
	})

	p.lock.Lock()
	defer p.lock.Unlock()
	p.updateProcessState(ready)
	return nil
}

// handleTxResp handle tx response
func (p *Process) handleTxResp(msg *protogo.DockerVMMessage) error {

	p.lock.Lock()
	defer p.lock.Unlock()

	logger.DebugDynamic(p.logger, func() string {
		return fmt.Sprintf("[%s] start handle tx resp [%s]", p.getTxId(), msg.TxId)
	})

	utils.EnterNextStep(msg, protogo.StepType_ENGINE_PROCESS_RECEIVE_TX_RESPONSE, func() string {
		return ""
	})
	if str, ok := utils.PrintTxStepsWithTime(msg); ok {
		p.logger.Warnf("[%s] slow tx execution, %s", msg.TxId, str)
	}

	if msg.TxId != p.Tx.TxId {
		p.logger.Warnf("abandon tx response due to different tx id, response tx id [%s], "+
			"current tx id [%s]", msg.TxId, p.Tx.TxId)
	}

	// after timeout, abandon tx response
	if p.processState != busy {
		p.logger.Warnf("abandon tx response due to busy timeout, tx id [%s]", msg.TxId)
	}

	// change state from busy to ready
	p.updateProcessState(ready)

	// failed for init / upgrade, contract err, exit process & remove contract
	if msg.Type == protogo.DockerVMType_ERROR &&
		(p.Tx.Request.Method == initContract || p.Tx.Request.Method == upgradeContract) {
		if err := p.killProcess(syscall.SIGTERM); err != nil {
			return fmt.Errorf("failed to kill process, %v", err)
		}
	}

	return nil
}

// handleTimeout handle busy timeout (sandbox timeout) and ready timeout (tx chan empty)
func (p *Process) handleTimeout() error {

	p.timerPopped = true

	switch p.processState {

	// busy timeout, restart, process state: busy -> timeout -> created -> ready, process manager keep busy
	case busy:
		p.lock.Lock()
		defer p.lock.Unlock()
		p.logger.Warnf("tx [%s] busy timeout", p.Tx.TxId)
		p.updateProcessState(timeout)
		p.logger.Errorf("timeout stack: %s", debug.Stack())
		if err := p.killProcess(syscall.SIGINT); err != nil {
			p.logger.Warnf("failed to kill timeout process, %v", err)
		}

	// ready timeout, process state: ready -> idle, process manager: busy -> idle
	case ready:
		p.lock.Lock()
		defer p.lock.Unlock()
		logger.DebugDynamic(p.logger, func() string {
			return fmt.Sprintf("[%s] ready timeout, go to idle", p.getTxId())
		})
		if err := p.processManager.ChangeProcessState(p.processName, false); err != nil {
			return fmt.Errorf("change process state error, %v", err)
		}
		p.updateProcessState(idle)

	default:
		p.logger.Warnf("[%s] process state should be busy / ready / idle, current state is %v", p.getTxId(), p.processState)
	}
	return nil
}

// TODO: 存在changing状态更改延迟的问题，可能需要加锁
// handleProcessExit handle process exit
func (p *Process) handleProcessExit(exitError *exitErr) bool {

	defer func() {
		p.popTimer()
		p.timerPopped = true
	}()

	var restartSandbox, returnErrResp, exitSandbox, restartAll, returnBadContractResp bool

	if exitError == nil {
		exitError = &exitErr{
			err:  utils.SandboxExitDefaultError,
			desc: "",
		}
	} else if exitError.err == nil {
		exitError.err = utils.SandboxExitDefaultError
	}

	errRet := exitError.err.Error()

	defer func() {
		if restartSandbox {
			go p.startProcess()
		}
		var txId string
		if p.Tx != nil {
			txId = p.Tx.TxId
		}
		if returnErrResp {
			if err := p.returnTxErrorResp(txId, errRet); err != nil {
				p.logger.Errorf("failed to return tx error response, %v", err)
			}
		}
		if exitSandbox {
			if err := p.returnSandboxExitResp(exitError.err); err != nil {
				p.logger.Errorf("failed to return sandbox exit resp, %v", err)
			}
		}
		if restartAll {
			p.Start()
		}
		if returnBadContractResp {
			contractFileVersion, err := strconv.ParseInt(exitError.desc, 10, 64)
			if err != nil {
				contractFileVersion = p.requestGroup.GetContractFileVersion()
			}
			if err = p.returnBadContractResp(contractFileVersion); err != nil {
				p.logger.Errorf("failed to return bad contract resp, %v", err)
			}
		}
	}()

	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError:
	//  a. return tx error
	//  b. return bad contract resp
	//  c. send exit sandbox msg
	//  d. exit process
	if exitError.err == utils.ContractExecError {
		p.logger.Errorf("process exited when launch process, start to release process, %s",
			exitError.err.Error())
		exitSandbox = true
		// contract panic when process start, pop oldest tx, retry get bytecode
		select {
		case tx := <-p.txCh:
			p.Tx = tx.Tx
			logger.DebugDynamic(p.logger, func() string {
				return fmt.Sprintf("[%s] contract exec start failed, remove tx %s", p.getTxId(), p.Tx.TxId)
			})
			returnErrResp = true
			returnBadContractResp = true
			break
		default:
			p.logger.Warn("contract exec start failed, no available tx")
			break
		}
		return true
	}

	// 2. created fail, ContractNotExistError:
	//  a. return bad contract resp
	//  b. send exit sandbox msg
	//  c. exit process
	if exitError.err == utils.ContractNotExistError {
		p.logger.Errorf("process exited for contract not found, %s, try to retrieve from blockchain",
			exitError.err.Error())
		exitSandbox = true
		// contract panic when process start, pop oldest tx, retry get bytecode
		select {
		case tx := <-p.txCh:
			p.Tx = tx.Tx
			// return tx
			if err := p.requestScheduler.PutMsg(tx.Tx); err != nil {
				p.logger.Errorf("failed to put msg [%s] to request group, %v", tx.Tx.TxId, err)
			}
			returnBadContractResp = true
			break
		default:
			p.logger.Debug("contract exec start failed, no available tx")
			break
		}
		return true
	}

	// 3. created fail, err from cmd.StdoutPipe() or writeToFile fail
	//  a. restart sandbox
	if p.processState == created {
		p.logger.Errorf("process exited when created, %s, %s",
			exitError.err.Error(), exitError.desc)
		restartSandbox = true
		return false
	}

	// =========  condition: cmd.wait
	// 4. sandbox exited when process state is ready:
	//  a. tx != nil && method == init or upgrade: return bad contract resp
	//  b. send exit sandbox msg
	//  c. exit process
	if p.processState == ready {
		p.logger.Errorf("process exited when ready, %s, %s",
			exitError.err.Error(), exitError.desc)
		exitSandbox = true
		if p.Tx == nil {
			p.logger.Warnf("exit with none tx")
			return true
		}

		if p.Tx.Request == nil {
			p.logger.Warnf("exit with invalid tx")
			return true
		}

		// error after exec init or upgrade
		if p.Tx.Request.Method == initContract || p.Tx.Request.Method == upgradeContract {
			returnBadContractResp = true
		}
		return true
	}

	// 5. sandbox exited when process state is idle:
	//  a. send exit sandbox msg
	//  b. exit process
	if p.processState == idle {
		p.logger.Errorf("process exited when idle, %s, %s",
			exitError.err.Error(), exitError.desc)
		exitSandbox = true
		return true
	}

	// 6. sandbox exited when process state is busy:
	// 	a. return tx error
	//  b. method == init or upgrade: return bad contract resp
	//  c. send exit sandbox msg
	//  d. exit process
	if p.processState == busy {
		p.logger.Errorf("[%s] process exited when busy, %s, %s", p.getTxId(),
			exitError.err.Error(), exitError.desc)
		p.updateProcessState(created)
		exitSandbox = true
		returnErrResp = true
		errRet = utils.RuntimePanicError.Error()
		// panic while exec init or upgrade
		if p.Tx.Request.Method == initContract || p.Tx.Request.Method == upgradeContract {
			returnBadContractResp = true
		}
		return true
	}

	// 7. sandbox exited when process state is changing (change sandbox to another contract):
	// 	a. restart sandbox (already replaced with new sandbox context)
	//  b. restart process
	if p.processState == changing {
		p.logger.Debugf("changing process to [%s]", p.processName)
		p.updateProcessState(created)
		// restart process
		restartAll = true
		return true
	}

	// 8. sandbox exited when process state is changing (process release):
	//  a. exit process
	if p.processState == closing {
		p.logger.Info("process killed for periodic process cleaning")
		return true
	}

	// 9. sandbox exited when process state is timeout (timeout for sandbox execution):
	//  a. send exit sandbox msg
	//  b. return tx error
	//  c. exit process
	if p.processState == timeout {
		p.logger.Errorf("[%s] process killed for timeout, %s, %s",
			p.getTxId(), exitError.err.Error(), exitError.desc)
		p.updateProcessState(created)
		exitSandbox = true
		returnErrResp = true
		errRet = utils.TxTimeoutPanicError.Error()
		return true
	}

	// 10. sandbox exited for other reasons
	p.logger.Errorf("[%s] process killed for other reasons, %s, %s",
		p.getTxId(), exitError.err.Error(), exitError.desc)
	return false
}

// resetContext reset sandbox context to new request group
func (p *Process) resetContext(chainID, contractName, contractVersion, processName string) error {

	// reset request group
	var ok bool
	// GetRequestGroup is safe because waiting group exists -> request group exists
	if p.requestGroup, ok = p.requestScheduler.GetRequestGroup(chainID, contractName, contractVersion); !ok {
		return fmt.Errorf("failed to get request group")
	}

	// reset process info
	p.chainID = chainID
	p.processName = processName
	p.contractName = contractName
	p.contractVersion = contractVersion
	p.logger = logger.NewDockerLogger(logger.GenerateProcessLoggerName(processName))

	// reset tx chan
	p.txCh = p.requestGroup.GetTxCh(p.isOrigProcess)

	return nil
}

// killProcess kills main process when process encounter error
func (p *Process) killProcess(sig os.Signal) error {
	<-p.cmdReadyCh
	p.logger.Debugf("[%s] start to kill process", p.getTxId())
	if p.cmd == nil {
		return errors.New("process cmd is nil")
	}

	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("failed to kill process, %v", err)
	}

	return nil
}

func (p *Process) forcedKill() {
	time.Sleep(_forcedKillWaitTime)
	err := p.cmd.Process.Signal(syscall.SIGKILL)
	if err != nil {
		p.logger.Warnf("failed to kill process with SIGKILL, err: %s", err.Error())
	}
}

// updateProcessState updates process state
func (p *Process) updateProcessState(state processState) {

	//p.logger.Debugf("[%s] update process state from [%+v] to [%+v]", p.getTxId(), p.processState, state)

	oldState := p.processState
	p.processState = state

	if oldState != ready && oldState != busy && oldState != idle && (state == ready || state == busy || state == idle) {
		p.updateCh <- struct{}{}
	}

	// jump out from the state that need to be timed
	//if oldState == ready || oldState == busy {
	//	p.stopTimer()
	//}

	// jump in the state that need to be timed
	//var txId string
	//if p.Tx != nil {
	//	txId = p.Tx.TxId
	//}
	if state == ready {
		//p.logger.Debugf("[%s] start ready tx timer for tx [%s]", p.getTxId(), txId)
		p.startReadyTimer()
	} else if state == busy {
		//p.logger.Debugf("[%s] start busy tx timer for tx [%s]", p.getTxId(), txId)
		p.startBusyTimer()
	}
}

// returnTxErrorResp return error to request scheduler
func (p *Process) returnTxErrorResp(txId string, errMsg string) error {
	errResp := &protogo.DockerVMMessage{
		Type:    protogo.DockerVMType_ERROR,
		ChainId: p.chainID,
		TxId:    txId,
		Response: &protogo.TxResponse{
			Code:    protogo.DockerVMCode_FAIL,
			Result:  nil,
			Message: errMsg,
		},
	}
	p.logger.Errorf("return error result of tx [%s]", txId)
	if err := p.requestScheduler.PutMsg(errResp); err != nil {
		return fmt.Errorf("failed to invoke request scheduler PutMsg, %v", err)
	}
	return nil
}

// returnTxErrorResp return error to request scheduler
func (p *Process) returnSandboxExitResp(err error) error {
	errResp := &messages.SandboxExitMsg{
		ChainID:         p.chainID,
		ContractName:    p.contractName,
		ContractVersion: p.contractVersion,
		ProcessName:     p.processName,
		Err:             err,
	}
	if putMsgErr := p.processManager.PutMsg(errResp); putMsgErr != nil {
		return fmt.Errorf("failed to invoke process manager PutMsg, %v", err)
	}
	return nil
}

// returnBadContractResp return bad contract resp to contract manager
func (p *Process) returnBadContractResp(cfv int64) error {
	resp := &messages.BadContractResp{
		Tx: &protogo.DockerVMMessage{
			ChainId: p.chainID,
			TxId:    p.Tx.TxId,
			Request: &protogo.TxRequest{
				ContractName:    p.contractName,
				ContractVersion: p.contractVersion,
				ChainId:         p.chainID,
			},
		},
		IsOrig:              p.isOrigProcess,
		ContractFileVersion: cfv,
	}

	if err := p.requestGroup.PutMsg(resp); err != nil {
		return fmt.Errorf("failed to invoke request group PutMsg, %v", err)
	}

	return nil
}

// sendMsg sends messages to sandbox
func (p *Process) sendMsg(msg *protogo.DockerVMMessage) error {
	//p.logger.Debugf("[%s] send msg to sandbox, tx_id: %s", msg.TxId, p.getTxId())
	if err := p.stream.Send(msg); err != nil {
		return fmt.Errorf("failed to send msg to stream")
	}
	return nil
}

// startBusyTimer start timer at busy state
// start when new tx come
func (p *Process) startBusyTimer() {
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.ExecTxTimeout)
	p.timerPopped = false
}

// startReadyTimer start timer at ready state
// start when process ready, resp come
func (p *Process) startReadyTimer() {
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.WaitingTxTime)
	p.timerPopped = false
}

func (p *Process) popTimer() {
	if !p.timer.Stop() && !p.timerPopped {
		<-p.timer.C
	}
}

func (p *Process) getTxId() string {
	if p.Tx != nil {
		return p.Tx.TxId
	}
	return ""
}

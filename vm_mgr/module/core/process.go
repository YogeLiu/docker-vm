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
	_readyToIdleTimeoutRatio = 5
	_forcedKillWaitTime      = 200 * time.Millisecond
	_removeTxTime            = 9000 * time.Millisecond
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

	p.logger.Debugf("start launch process")

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	tcpPort := config.DockerVMConfig.RPC.SandboxRPCPort
	if config.DockerVMConfig.RPC.ChainRPCProtocol == config.UDS {
		tcpPort = 0
	}
	cmd := exec.Cmd{
		Path: p.requestGroup.GetContractPath(),
		Args: []string{
			p.user.GetSockPath(),
			p.processName,
			p.contractName,
			p.contractVersion,
			config.DockerVMConfig.Log.SandboxLog.Level,
			strconv.Itoa(tcpPort),
			config.DockerVMConfig.RPC.ChainHost,
		},
		Stderr: &stderr,
	}

	contractOut, err := cmd.StdoutPipe()
	if err != nil {
		return &exitErr{
			err:  err,
			desc: "",
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
		return &exitErr{
			err:  utils.ContractExecError,
			desc: "",
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
	p.logger.Debugf("add process to cgroup")
	p.logger.Debugf("process started")

	p.printContractLog(contractOut)

	if err = cmd.Wait(); err != nil {
		var txId string
		if p.Tx != nil {
			txId = p.Tx.TxId
		}
		p.logger.Warnf("process stopped for tx [%s], %v, %v", txId, err, stderr.String())
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
			contractLogger.Info(err)
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
				break

			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle ready timeout timer, %v", err)
				}
				break

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
				break
			}
		} else if p.processState == busy {
			select {

			case resp := <-p.respCh:
				if err := p.handleTxResp(resp); err != nil {
					p.logger.Warnf("failed to handle tx response, %v", err)
				}
				break

			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle busy timeout timer, %v", err)
				}
				break

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
				break
			}
		} else if p.processState == idle {
			select {

			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle idle timeout timer, %v", err)
				}
				break

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
				break
			}
		} else {
			select {

			case err := <-p.exitCh:
				processReleased := p.handleProcessExit(err)
				if processReleased {
					return
				}
				break
			case _ = <-p.updateCh:
				break
			}
		}
	}
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

	p.logger.Debugf("process [%s] is changing to [%s]...", p.processName, processName)

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

	p.logger.Debugf("start to close sandbox")

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

	p.logger.Debugf("start handle tx req [%s]", tx.Tx.TxId)

	p.Tx = tx.Tx

	p.updateProcessState(busy)

	msg := &protogo.DockerVMMessage{
		ChainId:      p.chainID,
		TxId:         p.Tx.TxId,
		CrossContext: p.Tx.CrossContext,
		Request:      p.Tx.Request,
	}

	// send message to sandbox
	if err := p.sendMsg(msg); err != nil {
		return err
	}

	return nil
}

// handleTxResp handle tx response
func (p *Process) handleTxResp(msg *protogo.DockerVMMessage) error {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.logger.Debugf("start handle tx resp [%s]", p.Tx.TxId)

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

	switch p.processState {

	// busy timeout, restart, process state: busy -> timeout -> created -> ready, process manager keep busy
	case busy:
		p.lock.Lock()
		defer p.lock.Unlock()
		p.logger.Warnf("tx [%s] busy timeout", p.Tx.TxId)
		p.updateProcessState(timeout)
		if err := p.killProcess(syscall.SIGINT); err != nil {
			p.logger.Warnf("failed to kill timeout process, %v", err)
		}

	// ready timeout, process state: ready -> idle, process manager: busy -> idle
	case ready:
		p.lock.Lock()
		defer p.lock.Unlock()
		p.logger.Debugf("ready timeout, go to idle")
		if err := p.processManager.ChangeProcessState(p.processName, false); err != nil {
			return fmt.Errorf("change process state error, %v", err)
		}
		p.updateProcessState(idle)

	case idle:
		if len(p.txCh) > 0 {
			p.logger.Debugf("idle timeout, txCh len > 0, go to ready")
			// change state from idle to busy
			if err := p.processManager.ChangeProcessState(p.processName, true); err != nil {
				p.logger.Debugf("failed to change state, %v", err)
				return nil
			}
			p.lock.Lock()
			defer p.lock.Unlock()
			p.updateProcessState(ready)
		} else {
			p.startIdleTimer()
		}

	default:
		p.logger.Debugf("process state should be busy / ready / idle, current state is %v", p.processState)
	}
	return nil
}

// handleProcessExit handle process exit
func (p *Process) handleProcessExit(exitError *exitErr) bool {

	defer p.popTimer()

	var restartSandbox, returnErrResp, exitSandbox, restartAll, returnBadContractResp bool

	if exitError == nil {
		exitError = &exitErr{
			err:  utils.SandboxExitDefaultError,
			desc: "",
		}
	}

	if exitError.err == nil {
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
			if err := p.returnBadContractResp(); err != nil {
				p.logger.Errorf("failed to return bad contract resp, %v", err)
			}
		}
	}()

	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if exitError.err == utils.ContractExecError {
		p.logger.Warnf("process exited when launch process, start to release process")
		exitSandbox = true
		// contract panic when process start, pop oldest tx, retry get bytecode
		select {
		case tx := <-p.txCh:
			p.Tx = tx.Tx
			p.logger.Debugf("contract exec start failed, remove tx %s", p.Tx.TxId)
			returnErrResp = true
			break
		default:
			p.logger.Warn("contract exec start failed, no available tx")
			break
		}
		returnBadContractResp = true
		return true
	}

	// 2. created fail, err from cmd.StdoutPipe() -> relaunch
	// 3. created fail, writeToFile fail -> relaunch
	if p.processState == created {
		p.logger.Warnf("process exited when created: %s", exitError.err)
		restartSandbox = true
		return false
	}

	if p.processState == ready {
		p.logger.Warnf("process exited when ready: %s", exitError.err)
		exitSandbox = true
		// error after exec init or upgrade
		if p.Tx.Request.Method == initContract || p.Tx.Request.Method == upgradeContract {
			returnBadContractResp = true
		}
		return true
	}

	if p.processState == idle {
		p.logger.Warnf("process exited when idle: %s", exitError.err)
		exitSandbox = true
		return true
	}

	// 7. process panic, return error response
	if p.processState == busy {
		p.logger.Warnf("process exited when busy: %s", exitError.err)
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

	//  ========= condition: after cmd.wait
	// 4. process change context, restart process
	if p.processState == changing {
		p.logger.Warnf("changing process to [%s]", p.processName)
		p.updateProcessState(created)
		// restart process
		restartAll = true
		return true
	}

	//  ========= condition: after cmd.wait
	// 5. process killed because resource release
	if p.processState == closing {
		p.logger.Warnf("process killed for periodic process cleaning")
		return true
	}

	// 6. process killed because of timeout, return error response and relaunch
	if p.processState == timeout {
		p.logger.Warnf("process killed for timeout")
		p.updateProcessState(created)
		exitSandbox = true
		returnErrResp = true
		errRet = utils.TxTimeoutPanicError.Error()
		return true
	}

	p.logger.Warnf("process killed for other reasons")
	return false
}

// resetContext reset sandbox context to new request group
func (p *Process) resetContext(chainID, contractName, contractVersion, processName string) error {

	// reset request group
	var ok bool
	// GetRequestGroup is safe because waiting group exists -> request group exists
	if p.requestGroup, ok = p.requestScheduler.GetRequestGroup(p.chainID, contractName, contractVersion); !ok {
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
	p.logger.Debugf("start to kill process")
	if p.cmd == nil {
		return errors.New("process cmd is nil")
	}

	if err := p.cmd.Process.Signal(sig); err != nil {
		return fmt.Errorf("failed to kill process, %v", err)
	}

	go p.forcedKill()

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

	p.logger.Debugf("update process state from [%+v] to [%+v]", p.processState, state)

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
	var txId string
	if p.Tx != nil {
		txId = p.Tx.TxId
	}
	if state == ready {
		p.logger.Debugf("start ready tx timer for tx [%s]", txId)
		p.startReadyTimer()
	} else if state == busy {
		p.logger.Debugf("start busy tx timer for tx [%s]", txId)
		p.startBusyTimer()
	} else if state == idle {
		p.logger.Debugf("start idle tx timer for tx [%s]", txId)
		p.startIdleTimer()
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
func (p *Process) returnBadContractResp() error {
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
		IsOrig: p.isOrigProcess,
	}
	if err := p.requestScheduler.GetContractManager().PutMsg(resp); err != nil {
		return fmt.Errorf("failed to invoke contract manager PutMsg, %v", err)
	}

	if err := p.requestGroup.PutMsg(resp); err != nil {
		return fmt.Errorf("failed to invoke request group PutMsg, %v", err)
	}

	return nil
}

// sendMsg sends messages to sandbox
func (p *Process) sendMsg(msg *protogo.DockerVMMessage) error {
	p.logger.Debugf("send msg to sandbox, tx_id: %s", msg.TxId)
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
}

// startReadyTimer start timer at ready state
// start when process ready, resp come
func (p *Process) startReadyTimer() {
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.WaitingTxTime)
}

// startIdleTimer start timer at idle state
// start when process idle, len(txCh) > 0
func (p *Process) startIdleTimer() {
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.WaitingTxTime/_readyToIdleTimeoutRatio + 1)
}

func (p *Process) popTimer() {
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

// stopTimer stop timer
func (p *Process) stopTimer() {
	var txId string
	if p.Tx != nil {
		txId = p.Tx.TxId
	}
	p.logger.Debugf("stop tx timer for tx [%s]", txId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

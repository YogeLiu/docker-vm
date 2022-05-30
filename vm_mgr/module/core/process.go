/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"math"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
)

const (
	initContract    = "init_contract"
	invokeContract  = "invoke_contract"
	upgradeContract = "upgrade"
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
	readyToIdleTimeoutRatio = 5
)

// exitErr is the sandbox exit err
type exitErr struct {
	err  error
	desc string
}

// Process manage the sandbox process life cycle
type Process struct {
	processName string

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
	txCh       chan *protogo.DockerVMMessage
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

// NewProcess new process, process working on main contract which is not called cross contract
func NewProcess(user interfaces.User, contractName, contractVersion, processName string,
	manager interfaces.ProcessManager, scheduler interfaces.RequestScheduler, isOrigProcess bool) *Process {

	process := &Process{
		processName: processName,

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
	process.requestGroup, _ = scheduler.GetRequestGroup(contractName, contractVersion)

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
		p.logger.Errorf("failed to start process: %v", err)
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

	go p.printContractLog(contractOut)

	p.logger.Debugf("process started")

	if err = cmd.Wait(); err != nil {
		var txId string
		if p.Tx != nil {
			txId = p.Tx.TxId
		}
		p.logger.Warnf("process stopped for tx [%s], err is %v", txId, err)
		return &exitErr{
			err:  err,
			desc: stderr.String(),
		}
	}

	return nil
}

// listenProcess listen to channels
func (p *Process) listenProcess() {

	for {
		if p.processState == ready {
			select {

			case tx := <-p.txCh:
				// condition: during cmd.wait
				if err := p.handleTxRequest(tx); err != nil {
					p.logger.Errorf("failed to handle tx [%s] request, %v", tx.TxId, err)
					p.returnTxErrorResp(tx.TxId, err.Error())
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

// SetStream sets grpc stream
func (p *Process) SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer) {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.updateProcessState(ready)
	p.stream = stream
}

// ChangeSandbox changes sandbox of process
func (p *Process) ChangeSandbox(contractName, contractVersion, processName string) error {

	p.logger.Debugf("process [%s] is changing to [%s]...", p.processName, processName)

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	p.lock.Lock()
	defer p.lock.Unlock()

	if p.processState != idle {
		return fmt.Errorf("wrong state, current process state is %v, need %v", p.processState, idle)
	}

	if err := p.resetContext(contractName, contractVersion, processName); err != nil {
		return fmt.Errorf("failed to reset context, %v", err)
	}

	// TODO: kill process by send signal, reset exitCh, new process will never blocked
	if err := p.killProcess(); err != nil {
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

	if err := p.killProcess(); err != nil {
		return fmt.Errorf("failed to kill process, %v", err)
	}

	p.updateProcessState(closing)

	return nil
}

// handleTxRequest handle tx request from request group chan
func (p *Process) handleTxRequest(tx *protogo.DockerVMMessage) error {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.logger.Debugf("start handle tx req [%s]", tx.TxId)

	p.Tx = tx

	p.updateProcessState(busy)

	msg := &protogo.DockerVMMessage{
		TxId:         p.Tx.TxId,
		CrossContext: p.Tx.CrossContext,
		Request:      p.Tx.Request,
	}

	switch p.Tx.Request.Method {
	case initContract, upgradeContract:
		msg.Type = protogo.DockerVMType_INIT

	case invokeContract:
		msg.Type = protogo.DockerVMType_INVOKE

	default:
		return fmt.Errorf("invalid method: %s", p.Tx.Request.Method)
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

	return nil
}

// handleTimeout handle busy timeout (sandbox timeout) and ready timeout (tx chan empty)
func (p *Process) handleTimeout() error {

	switch p.processState {

	// busy timeout, restart, process state: busy -> timeout -> created -> ready, process manager keep busy
	case busy:
		p.lock.Lock()
		defer p.lock.Unlock()
		p.logger.Debugf("busy timeout, go to timeout")
		p.updateProcessState(timeout)
		if err := p.killProcess(); err != nil {
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

// release process success: true
// release process fail: false
func (p *Process) handleProcessExit(existErr *exitErr) bool {
	//
	//p.lock.Lock()
	//defer p.lock.Unlock()
	defer p.popTimer()

	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if existErr.err == utils.ContractExecError {

		// return error resp to chainmaker
		p.returnTxErrorResp(p.Tx.TxId, existErr.err.Error())

		// notify process manager to remove process cache
		p.logger.Debugf("start to release process")
		p.returnSandboxExitResp(existErr.err)

		return true
	}

	// 2. created fail, err from cmd.StdoutPipe() -> relaunch
	// 3. created fail, writeToFile fail -> relaunch
	if p.processState == created {
		p.logger.Warnf("failed to launch process: %s", existErr.err)
		go p.startProcess()
		return false
	}

	if p.processState == ready {
		p.logger.Warnf("process exited when ready: %s", existErr.err)
		p.returnSandboxExitResp(existErr.err)
		return true
	}

	if p.processState == idle {
		p.logger.Warnf("process exited when idle: %s", existErr.err)
		p.returnSandboxExitResp(existErr.err)
		return true
	}

	// 7. process panic, return error response and relaunch
	if p.processState == busy {
		p.logger.Warnf("process exited when busy: %s", existErr.err)
		p.returnSandboxExitResp(existErr.err)
		p.returnTxErrorResp(p.Tx.TxId, utils.RuntimePanicError.Error())
		p.updateProcessState(created)
		go p.startProcess()
		return true
	}

	//  ========= condition: after cmd.wait
	// 4. process change context, restart process
	if p.processState == changing {
		p.logger.Debugf("changing process to [%s]", p.processName)
		// restart process
		p.updateProcessState(created)
		p.Start()
		return true
	}

	//  ========= condition: after cmd.wait
	// 5. process killed because resource release
	if p.processState == closing {
		p.logger.Debugf("killed for periodic process cleaning")
		return true
	}

	// 6. process killed because of timeout, return error response and relaunch
	if p.processState == timeout {
		p.returnTxErrorResp(p.Tx.TxId, utils.TxTimeoutPanicError.Error())
		p.updateProcessState(created)
		go p.startProcess()
	}

	return false
}

// resetContext reset sandbox context to new request group
func (p *Process) resetContext(contractName, contractVersion, processName string) error {

	// reset request group
	var ok bool
	// GetRequestGroup is safe because waiting group exists -> request group exists
	if p.requestGroup, ok = p.requestScheduler.GetRequestGroup(contractName, contractVersion); !ok {
		return fmt.Errorf("failed to get request group")
	}

	// reset process info
	p.processName = processName
	p.contractName = contractName
	p.contractVersion = contractVersion
	p.logger = logger.NewDockerLogger(logger.GenerateProcessLoggerName(processName))

	// reset tx chan
	p.txCh = p.requestGroup.GetTxCh(p.isOrigProcess)

	return nil
}

// printContractLog print the sandbox cmd log
func (p *Process) printContractLog(contractPipe io.ReadCloser) {
	contractLogger := logger.NewDockerLogger(logger.MODULE_CONTRACT)
	rd := bufio.NewReader(contractPipe)
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			contractLogger.Info(err)
			return
		}
		str = strings.TrimSuffix(str, "\n")
		contractLogger.Debugf(str)
	}
}

// killProcess kills main process when process encounter error
func (p *Process) killProcess() error {
	<-p.cmdReadyCh
	p.logger.Debugf("start to kill process")
	if err := p.cmd.Process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process, %v", err)
	}
	return nil
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
	if state == ready {
		p.startReadyTimer()
	} else if state == busy {
		p.startBusyTimer()
	} else if state == idle {
		p.startIdleTimer()
	}
}

// returnTxErrorResp return error to request scheduler
func (p *Process) returnTxErrorResp(txId string, errMsg string) {
	errResp := &protogo.DockerVMMessage{
		Type: protogo.DockerVMType_ERROR,
		TxId: txId,
		Response: &protogo.TxResponse{
			Code:    protogo.DockerVMCode_FAIL,
			Result:  nil,
			Message: errMsg,
		},
	}
	p.logger.Errorf("return back error result for tx [%s]", p.Tx.TxId)
	_ = p.requestScheduler.PutMsg(errResp)
}

// returnTxErrorResp return error to request scheduler
func (p *Process) returnSandboxExitResp(err error) {
	errResp := &messages.SandboxExitMsg{
		ContractName:    p.Tx.Request.ContractName,
		ContractVersion: p.Tx.Request.ContractVersion,
		ProcessName:     p.processName,
		Err:             err,
	}
	_ = p.processManager.PutMsg(errResp)
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
	var txId string
	if p.Tx != nil {
		txId = p.Tx.TxId
	}
	p.logger.Debugf("start busy tx timer for tx [%s]", txId)
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.ExecTxTimeout)
}

// startReadyTimer start timer at ready state
// start when process ready, resp come
func (p *Process) startReadyTimer() {
	var txId string
	if p.Tx != nil {
		txId = p.Tx.TxId
	}
	p.logger.Debugf("start ready tx timer for tx [%s]", txId)
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.WaitingTxTime)
}

// startIdleTimer start timer at idle state
// start when process idle, len(txCh) > 0
func (p *Process) startIdleTimer() {
	var txId string
	if p.Tx != nil {
		txId = p.Tx.TxId
	}
	p.logger.Debugf("start idle tx timer for tx [%s]", txId)
	p.popTimer()
	p.timer.Reset(config.DockerVMConfig.Process.WaitingTxTime / readyToIdleTimeoutRatio)
}

func (p *Process) popTimer() {
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

// stopTimer stop timer
func (p *Process) stopTimer() {
	p.logger.Debugf("stop tx timer for tx [%s]", p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

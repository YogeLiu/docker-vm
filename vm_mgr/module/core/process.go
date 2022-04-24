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

const (
	created processState = iota
	busy
	ready
	changing
	idle
	timeout
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

	processState   processState
	isCrossProcess bool

	eventCh    chan interface{}
	cmdReadyCh chan bool
	exitCh     chan *exitErr
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
	manager interfaces.ProcessManager, scheduler interfaces.RequestScheduler, isCrossProcess bool) interfaces.Process {

	process := &Process{
		processName: processName,

		contractName:    contractName,
		contractVersion: contractVersion,

		cGroupPath: filepath.Join(security.CGroupRoot, security.ProcsFile),
		user:       user,

		processState:   created,
		isCrossProcess: isCrossProcess,

		eventCh:    make(chan interface{}, processManagerEventChSize),
		cmdReadyCh: make(chan bool, 1),
		exitCh:     make(chan *exitErr),
		respCh:     make(chan *protogo.DockerVMMessage, 1),
		timer:      time.NewTimer(math.MaxInt32 * time.Second), //initial tx timer, never triggered

		logger: logger.NewDockerLogger(logger.MODULE_PROCESS),

		processManager:   manager,
		requestScheduler: scheduler,

		lock: sync.RWMutex{},
	}

	process.requestGroup, _ = scheduler.GetRequestGroup(contractName, contractVersion)

	process.txCh = process.requestGroup.GetTxCh(isCrossProcess)

	return process
}

// PutMsg put invoking requests to chan, waiting for process to handle request
//  @param msg types include DockerVMType_TX_REQUEST, ChangeSandboxReqMsg and CloseSandboxReqMsg
func (p *Process) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		p.respCh <- msg.(*protogo.DockerVMMessage)

	case *messages.ChangeSandboxReqMsg, *messages.CloseSandboxReqMsg:
		p.eventCh <- msg

	default:
		p.logger.Errorf("unknown msg type")
	}
	return nil
}

// Start process, listen channels and exec cmd
func (p *Process) Start() {
	go p.listenProcess()
	p.startProcess()
}

// GetProcessName returns process name
func (p *Process) GetProcessName() string {

	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.processName
}

// GetContractName returns contract name
func (p *Process) GetContractName() string {

	p.lock.RLock()
	defer p.lock.RUnlock()

	return p.contractName
}

// GetContractVersion returns contract version
func (p *Process) GetContractVersion() string {

	p.lock.RLock()
	defer p.lock.RUnlock()

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

	p.stream = stream
}

// startProcess starts the process cmd
func (p *Process) startProcess() {
	p.updateProcessState(created)
	err := p.launchProcess()
	p.exitCh <- err
}

// launchProcess launch a new process
func (p *Process) launchProcess() *exitErr {

	p.logger.Debugf("launch process [%s]", p.processName)

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	cmd := exec.Cmd{
		Path: p.requestGroup.GetContractPath(),
		Args: []string{p.user.GetSockPath(), p.processName, p.contractName, p.contractVersion,
			config.DockerVMConfig.Log.SandboxLog.Level},
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
		p.logger.Errorf("[%s] fail to start process: %s", p.processName, err)
		return &exitErr{
			err:  utils.ContractExecError,
			desc: "",
		}
	}
	p.cmdReadyCh <- true

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, strconv.Itoa(cmd.Process.Pid)); err != nil {
		p.logger.Errorf("fail to add cgroup: %s", err)
		return &exitErr{
			err:  err,
			desc: "",
		}
	}
	p.logger.Debugf("add process [%s] to cgroup", p.processName)

	go p.printContractLog(contractOut)

	p.logger.Debugf("notify process [%s] started", p.processName)

	p.updateProcessState(ready)
	p.startReadyTimer()

	if err = cmd.Wait(); err != nil {
		p.logger.Warnf("process [%s] stopped for tx [%s], err is %v", p.processName, p.Tx.TxId, err)
		return &exitErr{
			err:  err,
			desc: stderr.String(),
		}
	}

	return nil
}

// listenProcess listen to eventCh, txCh, respCh and timer
func (p *Process) listenProcess() {
	for {
		if p.processState != created {
			select {
			case msg := <-p.eventCh:
				switch msg.(type) {

				// only when [idle]
				case *messages.ChangeSandboxReqMsg:
					m, _ := msg.(*messages.ChangeSandboxReqMsg)
					if err := p.handleChangeSandboxReq(m); err != nil {
						p.logger.Errorf("failed to handle change sandbox request, %v", err)
					}

				// only when [idle]
				case *messages.CloseSandboxReqMsg:
					if err := p.handleCloseSandboxReq(); err != nil {
						p.logger.Errorf("failed to handle close sandbox request, %v", err)
					}
				}

			// when [ready, idle]
			case tx := <-p.txCh:
				// condition: during cmd.wait
				if err := p.handleTxRequest(tx); err != nil {
					p.returnErrorResponse(tx.TxId, err.Error())
				}

			// when [busy]
			case resp := <-p.respCh:
				if err := p.handleTxResp(resp); err != nil {
					p.logger.Warnf("failed to handle tx response, %v", err)
				}

			// when [busy]
			case <-p.timer.C:
				if err := p.handleTimeout(); err != nil {
					p.logger.Errorf("failed to handle timeout timer, %v", err)
				}

			default:
				break
			}
		}

		select {
		// all
		case err := <-p.exitCh:
			processReleased := p.handleProcessExit(err)
			if processReleased {
				return
			}
		default:
			break
		}
	}
}

// handleChangeSandboxReq handle change sandbox request, change context, kill process, then restart process
func (p *Process) handleChangeSandboxReq(msg *messages.ChangeSandboxReqMsg) error {

	p.logger.Debugf("process [%s] is changing...", p.processName)

	p.lock.Lock()
	defer p.lock.Unlock()

	p.updateProcessState(changing)
	if err := p.resetContext(msg); err != nil {
		// if change sandbox failed, notify process manager to clean cache, then suicide
		p.processManager.PutMsg(&messages.SandboxExitRespMsg{
			ContractName:    msg.ContractName,
			ContractVersion: msg.ContractVersion,
			ProcessName:     msg.ProcessName,
			Err:             err,
		})
		return fmt.Errorf("failed to change sandbox, %v, process has been killed", err)
	}
	p.killProcess()
	return nil
}

// handleCloseSandboxReq handle close sandbox request
func (p *Process) handleCloseSandboxReq() error {

	p.logger.Debugf("process [%s] is killing...", p.processName)

	p.lock.Lock()
	defer p.lock.Unlock()

	p.killProcess()
	return nil
}

// handleTxRequest handle tx request from request group chan
func (p *Process) handleTxRequest(tx *protogo.DockerVMMessage) error {

	p.lock.Lock()
	defer p.lock.Unlock()

	p.logger.Debugf("process [%s] start handle tx req [%s]", p.processName, tx.TxId)

	// change state from ready to busy
	if err := p.processManager.ChangeProcessState(p.processName, true); err != nil {
		return fmt.Errorf("failed to change process [%s] state of tx [%s]", p.processName, tx.TxId)
	}
	p.updateProcessState(busy)

	p.Tx = tx

	// start busy timer to avoid process blocking
	p.startBusyTimer()

	msg := &protogo.DockerVMMessage{
		TxId:         p.Tx.TxId,
		CrossContext: &protogo.CrossContext{CurrentDepth: p.Tx.CrossContext.CurrentDepth},
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

	p.logger.Debugf("process [%s] start handle tx resp [%s]", p.processName, p.Tx.TxId)

	if msg.TxId != p.Tx.TxId {
		p.logger.Warnf("[%s] abandon tx response due to different tx id, response tx id [%s], "+
			"current tx id [%s]", p.processName, msg.TxId, p.Tx.TxId)
	}
	// after timeout, abandon tx response
	if p.processState != busy {
		p.logger.Warnf("[%s] abandon tx response due to busy timeout, tx id [%s]", p.processName, msg.TxId)
	}

	// change state from busy to ready
	if err := p.processManager.ChangeProcessState(p.processName, false); err != nil {
		return fmt.Errorf("failed to change process [%s] state of tx [%s]", p.processName, p.Tx.TxId)
	}
	p.stopTimer()
	p.startReadyTimer()
	p.updateProcessState(ready)

	return nil
}

// handleTimeout handle busy timeout (sandbox timeout) and ready timeout (tx chan empty)
func (p *Process) handleTimeout() error {

	p.lock.Lock()
	defer p.lock.Unlock()

	switch p.processState {

	// busy timeout, restart, process state: busy -> timeout -> created -> ready, process manager keep busy
	case busy:
		p.logger.Debugf("process [%s] busy timeout, go to timeout", p.processName)
		p.updateProcessState(timeout)
		p.killProcess()

	// ready timeout, process state: ready -> idle, process manager: busy -> idle
	case ready:
		p.logger.Debugf("process [%s] ready timeout, go to idle", p.processName)
		p.updateProcessState(idle)
		err := p.processManager.ChangeProcessState(p.processName, false)
		if err != nil {
			return fmt.Errorf("change process state error, %v", err)
		}

	default:
		return fmt.Errorf("process state should be running or ready")
	}
	return nil
}

// release process success: true
// release process fail: false
func (p *Process) handleProcessExit(existErr *exitErr) bool {

	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if existErr.err == utils.ContractExecError {

		// return error resp to chainmaker
		p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, p.Tx.TxId)
		p.returnErrorResponse(p.Tx.TxId, existErr.err.Error())

		// notify process manager to remove process cache
		p.logger.Debugf("release process: [%s]", p.processName)
		p.processManager.PutMsg(&messages.SandboxExitRespMsg{
			ContractName:    p.Tx.Request.ContractName,
			ContractVersion: p.Tx.Request.ContractVersion,
			ProcessName:     p.processName,
			Err:             existErr.err,
		})
		return true
	}

	// 2. created fail, err from cmd.StdoutPipe() -> relaunch
	// 3. created fail, writeToFile fail -> relaunch
	if p.processState == created {
		p.logger.Warnf("[%s] fail to launch process: %s", p.processName, existErr.err)
		go p.startProcess()
		return false
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
	if p.processState == idle {
		p.logger.Debugf("process [%s] killed for resource clean", p.processName)

		return true
	}

	var err error
	// 6. process killed because of timeout, return error response and relaunch
	if p.processState == timeout {
		err = utils.TxTimeoutPanicError
	}

	// 7. process panic, return error response and relaunch
	if p.processState == busy {
		err = utils.RuntimePanicError
		p.stopTimer()
		<-p.cmdReadyCh
	}

	p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, p.Tx.TxId)
	p.returnErrorResponse(p.Tx.TxId, err.Error())

	go p.startProcess()

	return false
}

// resetContext reset sandbox context to new request group
func (p *Process) resetContext(msg *messages.ChangeSandboxReqMsg) error {

	// reset process info
	p.processName = msg.ProcessName
	p.contractName = msg.ContractName
	p.contractVersion = msg.ContractVersion

	// reset request group
	var ok bool
	p.requestGroup, ok = p.requestScheduler.GetRequestGroup(msg.ContractName, msg.ContractVersion)
	if !ok {
		return fmt.Errorf("failed to get requets group")
	}

	// reset tx chan
	p.txCh = p.requestGroup.GetTxCh(p.isCrossProcess)
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
func (p *Process) killProcess() {
	<-p.cmdReadyCh
	p.logger.Debugf("kill process [%s]", p.processName)
	err := p.cmd.Process.Kill()
	if err != nil {
		p.logger.Warnf("fail to kill process [%s], %v", p.processName, err)
	}
}

// updateProcessState updates process state
func (p *Process) updateProcessState(state processState) {
	p.logger.Debugf("[%s] update process state: [%s]", p.processName, state)
	p.processState = state
}

// returnErrorResponse return error to request scheduler
func (p *Process) returnErrorResponse(txId string, errMsg string) {
	errResp := p.constructErrorResponse(txId, errMsg)
	if err := p.requestScheduler.PutMsg(errResp); err != nil {
		p.logger.Warnf("failed to return error response, %v", err)
	}
}

// sendMsg sends messages to sandbox
func (p *Process) sendMsg(msg *protogo.DockerVMMessage) error {
	p.logger.Debugf("send message [%s] process [%s]", msg, p.processName)
	return p.stream.Send(msg)
}

func (p *Process) constructErrorResponse(txId string, errMsg string) *protogo.DockerVMMessage {
	return &protogo.DockerVMMessage{
		Response: &protogo.TxResponse{
			TxId:    txId,
			Code:    protogo.DockerVMCode_FAIL,
			Result:  nil,
			Message: errMsg,
		},
	}
}

// startBusyTimer start timer at busy state
// start when new tx come
// stop when resp come
func (p *Process) startBusyTimer() {
	p.logger.Debugf("start tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
	p.timer.Reset(config.DockerVMConfig.GetBusyTimeout())
}

// startReadyTimer start timer at ready state
// start when process ready, resp come
// stop when new tx come
func (p *Process) startReadyTimer() {
	p.logger.Debugf("start tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
	p.timer.Reset(config.DockerVMConfig.GetReadyTimeout())
}

// stopTimer stop timer
func (p *Process) stopTimer() {
	p.logger.Debugf("stop tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

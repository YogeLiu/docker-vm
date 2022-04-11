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


type ExitErr struct {
	err  error
	desc string
}

// Process id of process is index of process in process list
// processName: contractName:contractVersion:index
// crossProcessName: txId:currentHeight
type Process struct {
	processName string

	contractName    string
	contractVersion string

	cGroupPath string
	user       *User
	cmd        *exec.Cmd

	ProcessState   protogo.ProcessState
	isCrossProcess bool

	eventCh    chan interface{}
	cmdReadyCh chan bool
	exitCh     chan *ExitErr
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
func NewProcess(user *User, contractName, contractVersion, processName string, manager interfaces.ProcessManager,
	scheduler interfaces.RequestScheduler, isCrossProcess bool) interfaces.Process {

	process := &Process{
		processName: processName,

		contractName:    contractName,
		contractVersion: contractVersion,

		cGroupPath: filepath.Join(security.CGroupRoot, security.ProcsFile),
		user:       user,

		ProcessState:   protogo.ProcessState_PROCESS_STATE_CREATED,
		isCrossProcess: isCrossProcess,

		eventCh:    make(chan interface{}, processManagerEventChSize),
		cmdReadyCh: make(chan bool, 1),
		exitCh:     make(chan *ExitErr),
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

func (p *Process) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage, *messages.ChangeSandboxReqMsg, *messages.CloseSandboxReqMsg:
		p.eventCh <- msg
	default:
		p.logger.Errorf("unknown msg type")
	}
	return nil
}

func (p *Process) Start() {
	go p.listenProcess()
	p.startProcess()
}

func (p *Process) listenProcess() {
	for {
		if p.ProcessState != protogo.ProcessState_PROCESS_STATE_CREATED {
			select {
			case msg := <-p.eventCh:
				switch msg.(type) {

				// only when [idle]
				case *messages.ChangeSandboxReqMsg:
					m, _ := msg.(*messages.ChangeSandboxReqMsg)
					if err := p.handleChangeSandboxReq(m); err != nil {
						p.logger.Errorf("failed to handle change sandbox request, %v", err)
					}

				// 10. release period expire
				// only when [idle]
				case *messages.CloseSandboxReqMsg:
					if err := p.handleCloseSandboxReq(); err != nil {
						p.logger.Errorf("failed to handle close sandbox request, %v", err)
					}

				// only when [busy]
				case *protogo.DockerVMMessage:
					m, _ := msg.(*protogo.DockerVMMessage)
					if err := p.handleMsg(m); err != nil {
						p.logger.Errorf("failed to handle message from sandbox, %v", err)
					}
				}

			// when [ready, idle]
			case tx := <-p.txCh:
				// condition: during cmd.wait
				// 7. created success, trigger new tx, previous state is created
				// 8. running success, next tx fail, trigger new tx, previous state is running
				// begin handle new tx
				// 9. handle new tx success
				if err := p.handleTxRequest(tx); err != nil {
					p.ReturnErrorResponse(tx.TxId, err.Error())
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

// startProcess starts the process cmd
func (p *Process) startProcess() {
	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_CREATED)
	err := p.LaunchProcess()
	p.exitCh <- err
}

// LaunchProcess launch a new process
func (p *Process) LaunchProcess() *ExitErr {
	p.logger.Debugf("[%s] launch process", p.processName)

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	cmd := exec.Cmd{
		Path: p.requestGroup.GetContractPath(),
		Args: []string{p.user.SockPath, p.processName, p.contractName, p.contractVersion,
			config.DockerVMConfig.Log.SandboxLog.Level},
		Stderr: &stderr,
	}

	contractOut, err := cmd.StdoutPipe()
	if err != nil {
		return &ExitErr{
			err:  err,
			desc: "",
		}
	}
	// these settings just working on linux,
	// but it doesn't affect running, because it will put into docker to run
	// setting pid namespace and allocate special uid for process
	cmd.SysProcAttr = &syscall.SysProcAttr{
		Credential: &syscall.Credential{
			Uid: uint32(p.user.Uid),
		},
		Cloneflags: syscall.CLONE_NEWPID,
	}
	p.cmd = &cmd

	if err = cmd.Start(); err != nil {
		p.logger.Errorf("[%s] fail to start process: %s", p.processName, err)
		return &ExitErr{
			err:  utils.ContractExecError,
			desc: "",
		}
	}
	p.cmdReadyCh <- true

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, strconv.Itoa(cmd.Process.Pid)); err != nil {
		p.logger.Errorf("fail to add cgroup: %s", err)
		return &ExitErr{
			err:  err,
			desc: "",
		}
	}
	p.logger.Debugf("[%s] add process to cgroup", p.processName)

	go p.printContractLog(contractOut)

	p.logger.Debugf("[%s] notify process started", p.processName)

	if err = cmd.Wait(); err != nil {
		p.logger.Warnf("[%s] process stopped for tx [%s], err is [%s], process state is [%s]",
			p.processName, p.Tx.TxId, err, p.ProcessState)
	}

	// cross process can only be killed: success finished or original process timeout
	if p.ProcessState != protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED {
		p.logger.Errorf("[%s] cross process fail: tx [%s], [%s], [%s]", p.processName,
			p.Tx.TxId, stderr.String(), err)
		return &ExitErr{
			err:  err,
			desc: "",
		}
	}
	return nil
}

// release process success: true
// release process fail: false
func (p *Process) handleProcessExit(existErr *ExitErr) bool {

	tx := p.Tx
	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if existErr.err == utils.ContractExecError {

		// return error resp to chainmaker
		p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, tx.TxId)
		p.ReturnErrorResponse(tx.TxId, existErr.err.Error())

		// notify process manager to remove process cache
		p.logger.Debugf("release process: [%s]", p.processName)
		p.processManager.PutMsg(&messages.SandboxExitRespMsg{
			ContractName:    tx.Request.ContractName,
			ContractVersion: tx.Request.ContractVersion,
			ProcessName:     p.processName,
			Err:             existErr.err,
		})
		return true
	}

	// 2. created fail, err from cmd.StdoutPipe() -> relaunch
	// 3. created fail, writeToFile fail -> relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_CREATED {
		p.logger.Warnf("[%s] fail to launch process: %s", p.processName, existErr.err)
		go p.startProcess()
		return false
	}

	//  ========= condition: after cmd.wait
	// 4. process release
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_IDLE {
		p.logger.Debugf("release process: [%s]", p.processName)

		return true
	}

	//p.logger.Errorf("[%s] process fail: tx [%s], [%s], [%s]", p.processName,
	//	p.Handler.TxRequest.TxId, existErr.desc, existErr.err)

	var err error
	// 5. process killed because of timeout, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT {
		err = utils.TxTimeoutPanicError
	}

	// 6. process panic, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_RUNNING {
		err = utils.RuntimePanicError
		p.stopTimer()
		<-p.cmdReadyCh
	}

	var errMsg string
	processDepth := p.Tx.ProcessInfo.CurrentDepth
	if processDepth == 0 {
		p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, tx.TxId)
		errMsg = err.Error()
	} else {
		errMsg = fmt.Sprintf("cross contract fail: err is:%s, cross processes: %s", err.Error(),
			p.Tx.ProcessInfo.CurrentProcessName)
		p.logger.Error(errMsg)
	}
	p.ReturnErrorResponse(tx.TxId, errMsg)

	go p.startProcess()

	return false
}

// handleTxRequest handle next tx or wait next available tx, process killed until expire time
// return triggered next tx successfully or not
func (p *Process) handleTxRequest(tx *protogo.DockerVMMessage) error {

	p.logger.Debugf("[%s] process start handle tx [%s]", p.processName, tx.TxId)
	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)

	p.Tx = tx
	p.startBusyTimer()
	defer p.stopTimer()

	msg := &protogo.DockerVMMessage{
		TxId:        p.Tx.TxId,
		ProcessInfo: &protogo.ProcessInfo{CurrentDepth: p.Tx.ProcessInfo.CurrentDepth},
		Request:     p.Tx.Request,
	}

	switch p.Tx.GetRequest().Method {
	case initContract, upgradeContract:
		msg.Type = protogo.DockerVMType_INIT
	case invokeContract:
		msg.Type = protogo.DockerVMType_INVOKE
	default:
		return fmt.Errorf("invalid method: %s", p.Tx.GetRequest().Method)
	}

	if err := p.sendMessage(msg); err != nil {
		return err
	}

	return nil
}

// handleChangeSandboxReq handle change sandbox request
func (p *Process) handleChangeSandboxReq(msg *messages.ChangeSandboxReqMsg) error {

	// if change sandbox failed, notify process manager to clean cache, then suicide
	if err := p.changeSandbox(msg); err != nil {
		p.processManager.PutMsg(&messages.SandboxExitRespMsg{
			ContractName:    msg.ContractName,
			ContractVersion: msg.ContractVersion,
			ProcessName:     msg.ProcessName,
			Err:             err,
		})
		if killErr := p.killProcess(); killErr != nil {
			return fmt.Errorf("failed to change sandbox, %v, "+
				"failed to kill process %s, %v", err, p.processName, killErr)
		}
		return fmt.Errorf("failed to change sandbox, %v, process has been killed", err)
	}

	p.exitCh <- &ExitErr{
		err:  nil,
		desc: "change sandbox context",
	}

	// restart process
	p.Start()

	return nil
}

// handleCloseSandboxReq handle close sandbox request
func (p *Process) handleCloseSandboxReq() error {

	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_RUNNING {
		return fmt.Errorf("can not change sandbox while process is running")
	}

	err := p.killProcess()
	if err != nil {
		return fmt.Errorf("failed to kill process %s, %v", p.processName, err)
	}
	return nil
}

// handleTimeout handle busy timeout (sandbox timeout) and ready timeout (tx chan empty)
func (p *Process) handleTimeout() error {

	switch p.ProcessState {
	case protogo.ProcessState_PROCESS_STATE_RUNNING:
		p.logger.Debugf("process [%s] busy timeout", p.processName)
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT)
		if err := p.killProcess(); err != nil {
			p.logger.Errorf("failed to kill process %s, %v", p.processName, err)
		}

	case protogo.ProcessState_PROCESS_STATE_READY:
		p.logger.Debugf("process [%s] ready timeout, go to idle", p.processName)
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_IDLE)
		err := p.processManager.ChangeProcessState(p.processName, false)
		if err != nil {
			return fmt.Errorf("change process state error, %v", err)
		}

	default:
		return fmt.Errorf("process state should be running or ready")
	}
	return nil
}

// handleTxResp handle tx response
func (p *Process) handleTxResp(msg *protogo.DockerVMMessage) error {
	if msg.TxId != p.Tx.TxId {
		return fmt.Errorf("[%s] abandon tx response due to different tx id, response tx id [%s], "+
			"current tx id [%s]", p.processName, msg.TxId, p.Tx.TxId)
	}
	// 11. after timeout, abandon tx response
	if p.ProcessState != protogo.ProcessState_PROCESS_STATE_RUNNING {
		return fmt.Errorf("[%s] abandon tx response due to busy timeout, tx id [%s]", p.processName, msg.TxId)
	}

	// 12. before timeout as success tx response, return response and trigger new tx
	p.stopTimer()
	p.startReadyTimer()
	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_READY)

	return nil
}

// handleBusyTimeout handle busy timeout timer
func (p *Process) handleBusyTimeout() error {
	// change process state to idle
	err := p.processManager.ChangeProcessState(p.processName, false)
	if err != nil {
		return fmt.Errorf("change process state error, %v", err)
	}
	return nil
}

// handleMsg handle all messages come from sandbox
func (p *Process) handleMsg(msg *protogo.DockerVMMessage) error {

	p.logger.Debugf("process [%s] handle msg [%s]", p.processName, msg)

	switch p.ProcessState {
	case protogo.ProcessState_PROCESS_STATE_CREATED:
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_READY)

	default:
		switch msg.Type {
		case protogo.DockerVMType_COMPLETED:
			p.respCh <- &protogo.DockerVMMessage{TxId: msg.TxId}

		default:
			return fmt.Errorf("handler cannot handle ready message [%s]", msg.Type)
		}
	}
	return nil
}

// changeSandbox change the sandbox context
func (p *Process) changeSandbox(msg *messages.ChangeSandboxReqMsg) error {

	if p.ProcessState != protogo.ProcessState_PROCESS_STATE_IDLE {
		return fmt.Errorf("can not change sandbox while not idle")
	}
	if err := p.killProcess(); err != nil {
		return fmt.Errorf("failed to kill process %s, %v", p.processName, err)
	}
	if err := p.resetContext(msg); err != nil {
		return fmt.Errorf("failed to reset context, %v", err)
	}
	return nil
}

// resetContext reset sandbox context to new request group
func (p *Process) resetContext(msg *messages.ChangeSandboxReqMsg) error {

	// reset process info
	p.processName = msg.ProcessName
	p.contractName = msg.ContractName
	p.contractVersion = msg.ContractVersion

	// reset request group
	var err error
	p.requestGroup, err = p.requestScheduler.GetRequestGroup(msg.ContractName, msg.ContractVersion)
	if err != nil {
		return fmt.Errorf("failed to get requets group, %v", err)
	}

	// reset tx chan
	p.txCh = p.requestGroup.GetTxCh(p.isCrossProcess)
	return nil
}

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

// kill main process when process encounter error
func (p *Process) killProcess() error {
	<-p.cmdReadyCh
	p.logger.Debugf("kill process [%s]", p.processName)
	err := p.cmd.Process.Kill()
	if err != nil {
		return fmt.Errorf("fail to kill process [%s], %v", p.processName, err)
	}
	return nil
}

// updateProcessState updates process state
func (p *Process) updateProcessState(state protogo.ProcessState) {
	p.logger.Debugf("[%s] update process state: [%s]", p.processName, state)
	p.ProcessState = state
}

// ReturnErrorResponse return error to request scheduler
func (p *Process) ReturnErrorResponse(txId string, errMsg string) {
	errResp := p.constructErrorResponse(txId, errMsg)
	if err := p.requestScheduler.PutMsg(errResp); err != nil {
		p.logger.Warnf("failed to return error response, %v", err)
	}
}

func (p *Process) SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer) {
	p.stream = stream
}

func (p *Process) sendMessage(msg *protogo.DockerVMMessage) error {
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

func (p *Process) GetProcessName() string {
	return p.processName
}

func (p *Process) GetContractName() string {
	return p.contractName
}

func (p *Process) GetContractVersion() string {
	return p.contractVersion
}

func (p *Process) GetUser() *User {
	return p.user
}

func (p *Process) startBusyTimer() {
	p.logger.Debugf("start tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
	p.timer.Reset(config.DockerVMConfig.GetBusyTimeout())
}

func (p *Process) startReadyTimer() {
	p.logger.Debugf("start tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
	p.timer.Reset(config.DockerVMConfig.GetReadyTimeout())
}

func (p *Process) stopTimer() {
	p.logger.Debugf("stop tx timer: process [%s], tx [%s]", p.processName, p.Tx.TxId)
	if !p.timer.Stop() && len(p.timer.C) > 0 {
		<-p.timer.C
	}
}

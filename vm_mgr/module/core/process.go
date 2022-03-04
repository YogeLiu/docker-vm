/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"go.uber.org/atomic"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

const (
	processWaitingTime      = 60 * 10
	processWaitingQueueSize = 1000
	triggerNewProcessSize   = 3
)

type ExitErr struct {
	err  error
	desc string
}

type ProcessMgrInterface interface {
	getProcessDepth(initialProcessName string) *ProcessDepth

	ReleaseProcess(processName string, user *security.User) bool
}

// Process id of process is index of process in process list
// processName: contractName:contractVersion:index
// crossProcessName: txId:currentHeight
type Process struct {
	txCount        *atomic.Uint64
	processName    string
	isCrossProcess bool

	contractName    string
	contractVersion string
	contractPath    string

	cGroupPath string
	user       *security.User
	cmd        *exec.Cmd

	ProcessState   protogo.ProcessState
	TxWaitingQueue chan *protogo.TxRequest
	responseCh     chan *protogo.TxResponse
	newTxTrigger   chan bool
	exitCh         chan *ExitErr
	expireTimer    *time.Timer // process waiting time
	Handler        *ProcessHandler
	notifyCh       chan bool

	logger *zap.SugaredLogger

	processMgr ProcessMgrInterface
	done       uint32
	mutex      sync.Mutex
}

// NewProcess new process, process working on main contract which is not called cross contract
func NewProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName, contractPath string, processPool ProcessMgrInterface) *Process {

	process := &Process{
		txCount:        atomic.NewUint64(0),
		processName:    processName,
		isCrossProcess: false,

		contractName:    txRequest.ContractName,
		contractVersion: txRequest.ContractVersion,
		contractPath:    contractPath,

		cGroupPath: filepath.Join(config.CGroupRoot, config.ProcsFile),
		user:       user,

		ProcessState:   protogo.ProcessState_PROCESS_STATE_CREATED,
		TxWaitingQueue: make(chan *protogo.TxRequest, processWaitingQueueSize),
		responseCh:     make(chan *protogo.TxResponse),
		newTxTrigger:   make(chan bool),
		exitCh:         make(chan *ExitErr),
		expireTimer:    time.NewTimer(processWaitingTime * time.Second),
		Handler:        nil,
		notifyCh:       make(chan bool, 1),

		logger: logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),

		processMgr: processPool,
	}

	processHandler := NewProcessHandler(txRequest, scheduler, processName, process)
	process.Handler = processHandler
	return process
}

// NewCrossProcess new cross process, process working on called cross process
func NewCrossProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName, contractPath string, processPool ProcessMgrInterface) *Process {

	process := &Process{
		txCount:         atomic.NewUint64(0),
		isCrossProcess:  true,
		processName:     processName,
		contractName:    txRequest.ContractName,
		contractVersion: txRequest.ContractVersion,
		ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
		responseCh:      make(chan *protogo.TxResponse),
		TxWaitingQueue:  nil,
		newTxTrigger:    nil,
		exitCh:          nil,
		expireTimer:     time.NewTimer(processWaitingTime * time.Second),
		logger:          logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),

		Handler:      nil,
		notifyCh:     make(chan bool, 1),
		user:         user,
		contractPath: contractPath,
		cGroupPath:   filepath.Join(config.CGroupRoot, config.ProcsFile),
		processMgr:   processPool,
	}

	processHandler := NewProcessHandler(txRequest, scheduler, processName, process)
	process.Handler = processHandler
	return process
}

func (p *Process) ProcessName() string {
	return p.processName
}

func (p *Process) ExecProcess() {

	go p.listenProcess()

	p.startProcess()
}

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

	var pn string
	if p.isCrossProcess {
		pn = utils.ConstructConcatOriginalAndCrossProcessName(p.Handler.TxRequest.TxContext.OriginalProcessName,
			p.processName)
	} else {
		pn = p.processName
	}

	cmd := exec.Cmd{
		Path:   p.contractPath,
		Args:   []string{p.user.SockPath, pn, p.contractName, p.contractVersion, config.SandBoxLogLevel},
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
	p.notifyCh <- true

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, cmd.Process.Pid); err != nil {
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
			p.processName, p.Handler.TxRequest.TxId, err, p.ProcessState)
	}

	if !p.isCrossProcess {
		return &ExitErr{
			err:  err,
			desc: stderr.String(),
		}
	}

	// cross process can only be killed: success finished or original process timeout
	if p.ProcessState != protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED {
		p.logger.Errorf("[%s] cross process fail: tx [%s], [%s], [%s]", p.processName,
			p.Handler.TxRequest.TxId, stderr.String(), err)
		return &ExitErr{
			err:  err,
			desc: "",
		}
	}
	return nil
}

func (p *Process) listenProcess() {
	for {
		select {
		case <-p.newTxTrigger:
			// condition: during cmd.wait
			// 7. created success, trigger nex tx, previous state is created
			// 8. running success, next tx fail, trigger new tx, previous state is running
			p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)
			p.resetProcessTimer()
			// begin handle new tx
			// 9. handle new tx success
			// 10. no tx in wait queue, util process expire
			currentTxId, err := p.handleNewTx()
			if err != nil {
				p.Handler.scheduler.ReturnErrorResponse(currentTxId, err.Error())
			}
		case txResponse := <-p.responseCh:
			// 11. after timeout, abandon tx response
			if p.ProcessState != protogo.ProcessState_PROCESS_STATE_RUNNING {
				continue
			}
			// 12. before timeout as success tx response, return response and trigger new tx
			p.Handler.stopTimer()
			p.resetProcessTimer()
			// return txResponse
			responseCh := p.Handler.scheduler.GetTxResponseCh()

			p.logger.Debugf("[%s] put tx response in response chan for in process [%s] with chan length[%d]",
				txResponse.TxId, p.processName, len(responseCh))

			responseCh <- txResponse

			p.logger.Debugf("[%s] end handle tx in process [%s]", txResponse.TxId, p.processName)
			// begin handle new tx
			currentTxId, err := p.handleNewTx()
			if err != nil {
				p.Handler.scheduler.ReturnErrorResponse(currentTxId, err.Error())
			}
		case <-p.Handler.txExpireTimer.C:
			p.StopProcess(false)
		case err := <-p.exitCh:
			processReleased := p.handleProcessExit(err)
			if processReleased {
				return
			}
		}
	}
}

// release process success: true
// release process fail: false
func (p *Process) handleProcessExit(existErr *ExitErr) bool {

	currentTx := p.Handler.TxRequest
	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if existErr.err == utils.ContractExecError {
		p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, currentTx.TxId)
		p.Handler.scheduler.ReturnErrorResponse(currentTx.TxId, existErr.err.Error())
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
	// 4. process expire, try to exit
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_EXPIRE {
		// when process timeout, release resources
		p.logger.Debugf("release process: [%s]", p.processName)

		if !p.processMgr.ReleaseProcess(p.processName, p.user) {
			go p.startProcess()
			return false
		}
		return true
	}

	p.logger.Errorf("[%s] process fail: tx [%s], [%s], [%s]", p.processName,
		p.Handler.TxRequest.TxId, existErr.desc, existErr.err)

	var err error
	// 5. process killed because of timeout, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT {
		err = utils.TxTimeoutPanicError
	}
	// 6. process panic, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_RUNNING {
		err = utils.RuntimePanicError
		p.Handler.stopTimer()
	}

	var errMsg string
	processDepth := p.processMgr.getProcessDepth(currentTx.TxContext.OriginalProcessName)
	if processDepth == nil {
		p.logger.Errorf("return back error result for process [%s] for tx [%s]", p.processName, currentTx.TxId)
		errMsg = err.Error()
	} else {
		errMsg = fmt.Sprintf("cross contract fail: err is:%s, cross processes: %s", err.Error(),
			processDepth.GetConcatProcessName())
		p.logger.Error(errMsg)
	}
	p.Handler.scheduler.ReturnErrorResponse(currentTx.TxId, errMsg)

	p.Handler.resetState()

	go p.startProcess()

	return false
}

// handleNewTx handle next tx or wait next available tx, process killed until expire time
// return triggered next tx successfully or not
func (p *Process) handleNewTx() (string, error) {

	select {
	case nextTx := <-p.TxWaitingQueue:
		p.logger.Debugf("[%s] process start handle tx [%s], waiting queue size [%d]", p.processName, nextTx.TxId, len(p.TxWaitingQueue))

		p.Handler.TxRequest = nextTx
		p.Handler.startTimer()
		p.disableProcessExpireTimer()

		err := p.Handler.HandleContract()

		// send tx msg fail or invalid method
		// valid method just have: initContract, invokeContract, upgradeContract
		if err != nil {
			p.logger.Errorf("[%s] process fail to invoke contract: %s", p.processName, err)
			p.Handler.stopTimer()
			p.resetProcessTimer()
			go p.triggerNewTx()
			return nextTx.TxId, err
		}
		return "", nil
	case <-p.expireTimer.C:
		p.StopProcess(true)
		return "", nil
	}
}

// AddTxWaitingQueue add tx with same contract to process waiting queue
func (p *Process) AddTxWaitingQueue(tx *protogo.TxRequest) {

	newCount := p.txCount.Add(1)
	tx.TxContext.OriginalProcessName = utils.ConstructOriginalProcessName(p.processName, newCount)
	p.logger.Debugf("[%s] update tx original name: [%s]", p.processName, tx.TxContext.OriginalProcessName)

	p.TxWaitingQueue <- tx
	p.logger.Debugf("[%s] add tx [%s] to waiting queue, new size [%d], "+
		"process state is [%s]", p.processName, tx.TxId, len(p.TxWaitingQueue), p.ProcessState)
}

func (p *Process) Size() int {
	return len(p.TxWaitingQueue)
}

func (p *Process) printContractLog(contractPipe io.ReadCloser) {
	contractLogger := logger.NewDockerLogger(logger.MODULE_CONTRACT, config.DockerLogDir)
	rd := bufio.NewReader(contractPipe)
	for {
		str, err := rd.ReadString('\n')
		if err != nil {
			return
		}
		str = strings.TrimSuffix(str, "\n")
		contractLogger.Debugf(str)
	}
}

// StopProcess stop process
func (p *Process) StopProcess(processTimeout bool) {
	p.logger.Debugf("[%s] stop process", p.processName)
	if processTimeout {
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_EXPIRE)
		p.killProcess(false)
	} else {
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT)
		p.killProcess(true)
	}
}

// kill cross process and free process in cross process table
func (p *Process) killCrossProcess() {
	<-p.notifyCh
	p.logger.Debugf("[%s] receive process notify and kill cross process", p.processName)
	err := p.cmd.Process.Kill()
	if err != nil {
		p.logger.Errorf("[%s] fail to kill cross process: [%s]", p.processName, err)
	}
}

// kill main process when process encounter error
func (p *Process) killProcess(isTxTimeout bool) {
	<-p.notifyCh
	p.logger.Debugf("[%s] kill original process", p.processName)
	err := p.cmd.Process.Kill()
	if err != nil {
		p.logger.Warnf("[%s] fail to kill corss process: %s", p.processName, err)
	}

	if !isTxTimeout {
		return
	}

	originalProcessName := p.Handler.TxRequest.TxContext.OriginalProcessName
	processDepth := p.processMgr.getProcessDepth(originalProcessName)

	if processDepth == nil {
		return
	}

	for depth, process := range processDepth.processes {
		if process != nil {
			p.logger.Debugf("[%s] kill cross process in depth [%d]", process.processName, depth)
			if err = process.cmd.Process.Kill(); err != nil {
				p.logger.Warnf("[%s] fail to kill corss process: %s", process.processName, err)
			}
		}
	}
}

func (p *Process) triggerNewTx() {
	p.logger.Debugf("[%s] trigger new tx for process", p.processName)
	p.newTxTrigger <- true
}

func (p *Process) returnTxResponse(txResponse *protogo.TxResponse) {
	p.logger.Debugf("[%s] return tx response to process [%s]", txResponse.TxId, p.processName)
	p.responseCh <- txResponse
}

func (p *Process) updateProcessState(state protogo.ProcessState) {
	p.logger.Debugf("[%s] update process state: [%s]", p.processName, state)
	p.ProcessState = state
}

// resetProcessTimer reset timer when tx finished
func (p *Process) resetProcessTimer() {
	p.logger.Debugf("[%s] reset process expire timer", p.processName)
	if !p.expireTimer.Stop() && len(p.expireTimer.C) > 0 {
		<-p.expireTimer.C
	}
	p.expireTimer.Reset(processWaitingTime * time.Second)
}

func (p *Process) disableProcessExpireTimer() {
	p.logger.Debugf("[%s] disable process expire timer", p.processName)
	if !p.expireTimer.Stop() && len(p.expireTimer.C) > 0 {
		<-p.expireTimer.C
	}
	p.expireTimer.Stop()
}

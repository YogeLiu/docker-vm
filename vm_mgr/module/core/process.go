/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"bufio"
	"bytes"
	"io"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
	"go.uber.org/zap/zapcore"

	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

const (
	processWaitingTime = 60 * 10
)

type ExitErr struct {
	err  error
	desc string
}

type ProcessMgr interface {
	GetProcessDepth(initialProcessName string) *ProcessDepth

	ReleaseProcess(processName string, user *security.User)

	CheckTxExpired(txID string) bool
}

type ProcessBalancer interface {
	GetTxQueue() chan *protogo.TxRequest
}

// Process
// id of process is index of process in process list
// processName: contractName:contractVersion:index
// crossProcessName: txId:currentHeight
type Process struct {
	txCount     uint64
	processName string
	//isCrossProcess bool

	contractName    string
	contractVersion string
	contractPath    string

	cGroupPath string
	user       *security.User
	cmd        *exec.Cmd

	ProcessState protogo.ProcessState
	responseCh   chan *protogo.TxResponse
	exitCh       chan *ExitErr
	newTxTrigger chan bool
	cmdReadyCh   chan bool
	expireTimer  *time.Timer // process waiting time

	crossResponseCh chan *SDKProtogo.DMSMessage

	Handler         *ProcessHandler
	processMgr      ProcessMgr
	processBalancer ProcessBalancer
	logger          *zap.SugaredLogger

	killOnce sync.Once

	ChainId string
}

// NewProcess new process, process working on main contract which is not called cross contract
func NewProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName, contractPath string, processMgr ProcessMgr, processBalancer ProcessBalancer) *Process {

	process := &Process{
		txCount:     0,
		processName: processName,
		//isCrossProcess: false,

		contractName:    txRequest.ContractName,
		contractVersion: txRequest.ContractVersion,
		contractPath:    contractPath,

		cGroupPath: filepath.Join(config.CGroupRoot, config.ProcsFile),
		user:       user,

		ProcessState: protogo.ProcessState_PROCESS_STATE_CREATED,
		responseCh:   make(chan *protogo.TxResponse),
		newTxTrigger: make(chan bool),
		exitCh:       make(chan *ExitErr),
		expireTimer:  time.NewTimer(processWaitingTime * time.Second),
		Handler:      nil,
		cmdReadyCh:   make(chan bool, 1),

		crossResponseCh: make(chan *SDKProtogo.DMSMessage),

		processMgr:      processMgr,
		processBalancer: processBalancer,
		logger:          logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),

		ChainId: txRequest.ChainId,
	}

	processHandler := NewProcessHandler(txRequest, scheduler, processName, process)
	process.Handler = processHandler
	return process
}

// NewCrossProcess new cross process, process working on called cross process
//func NewCrossProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
//	processName, contractPath string, processMgr ProcessMgr) *Process {
//
//	process := &Process{
//		txCount:        0,
//		processName:    processName,
//		isCrossProcess: true,
//
//		contractName:    txRequest.ContractName,
//		contractVersion: txRequest.ContractVersion,
//		contractPath:    contractPath,
//
//		cGroupPath: filepath.Join(config.CGroupRoot, config.ProcsFile),
//		user:       user,
//
//		ProcessState: protogo.ProcessState_PROCESS_STATE_CREATED,
//		responseCh:   make(chan *protogo.TxResponse),
//		newTxTrigger: nil,
//		exitCh:       nil,
//		expireTimer:  time.NewTimer(processWaitingTime * time.Second),
//		Handler:      nil,
//		cmdReadyCh:   make(chan bool, 1),
//
//		processMgr:      processMgr,
//		processBalancer: nil,
//		logger:          logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),
//
//		ChainId: txRequest.ChainId,
//	}
//
//	processHandler := NewProcessHandler(txRequest, scheduler, processName, process)
//	process.Handler = processHandler
//	return process
//}

func (p *Process) ProcessName() string {
	return p.processName
}

func (p *Process) ExecProcess() {

	go p.listenProcess()

	p.startProcess()
}

func (p *Process) startProcess() {
	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_CREATED)
	p.Handler.resetState()
	err := p.LaunchProcess()
	p.exitCh <- err
}

// LaunchProcess launch a new process
func (p *Process) LaunchProcess() *ExitErr {
	p.logger.Debugf("[%s] launch process", p.processName)

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	var pn string
	//if p.isCrossProcess {
	//	pn = utils.ConstructConcatOriginalAndCrossProcessName(p.Handler.TxRequest.TxContext.OriginalProcessName,
	//		p.processName)
	//} else {
	//	pn = p.processName
	//}

	pn = p.processName

	cmd := exec.Cmd{
		Path:   p.contractPath,
		Args:   []string{p.user.SockPath, pn, p.contractName, p.contractVersion, config.SandBoxLogLevel},
		Stderr: &stderr,
	}

	contractOut, err := cmd.StdoutPipe()
	if err != nil {
		return &ExitErr{
			err:  err,
			desc: "failed to get contract process stdout pipe",
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
			desc: "failed to start contract process",
		}
	}
	p.cmdReadyCh <- true

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, cmd.Process.Pid); err != nil {
		p.logger.Errorf("fail to add cgroup: %s", err)
		return &ExitErr{
			err:  err,
			desc: "failed to write cgroup",
		}
	}
	p.logger.Debugf("[%s] add process to cgroup", p.processName)

	go p.printContractLog(contractOut)

	p.logger.Debugf("[%s] notify process started", p.processName)

	if err = cmd.Wait(); err != nil {
		p.logger.Warnf("[%s] process stopped for tx [%s], err is [%s], process state is [%s]",
			p.processName, p.Handler.TxRequest.TxId, err, p.ProcessState)
	}

	// log stderr string
	stdErrStr := stderr.String()
	p.logger.Infof("process %s stderr is: %s", p.processName, stdErrStr)

	return &ExitErr{
		err:  err,
		desc: stdErrStr,
	}

	// cross process can only be killed: success finished or original process timeout
	//if p.ProcessState != protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED {
	//	p.logger.Errorf("[%s] cross process fail: tx [%s], [%s], [%s]", p.processName,
	//		p.Handler.TxRequest.TxId, stderr.String(), err)
	//	return &ExitErr{
	//		err:  err,
	//		desc: "",
	//	}
	//}
	//return nil
}

func (p *Process) listenProcess() {
	for {
		select {
		case <-p.newTxTrigger:
			// condition: during cmd.wait
			// 7. created success, trigger new tx, previous state is created
			// 8. running success, next tx fail, trigger new tx, previous state is running
			p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)
			p.resetProcessTimer()
			// begin handle new tx
			// 9. handle new tx success
			// 10. no tx in wait queue, util process expire
			currentTxId, err := p.handleNewTx()
			if err != nil {
				p.Handler.scheduler.ReturnErrorResponse(p.ChainId, currentTxId, err.Error())
			}
		case crossResponse := <-p.crossResponseCh:
			p.logger.Debugf("process [%s] handle cross contract completed message [%s]", p.processName, crossResponse.TxId)

			p.Handler.stopTimer()
			p.resetProcessTimer()

			responseChId := crossContractChKey(crossResponse.TxId, crossResponse.CurrentHeight)
			responseCh := p.Handler.scheduler.GetCrossContractResponseCh(p.Handler.TxRequest.ChainId, responseChId)
			if responseCh == nil {
				p.logger.Warnf("process [%s] fail to get response chan and abandon cross response [%s]",
					p.processName, p.Handler.TxRequest.TxId)
			} else {
				responseCh <- crossResponse
			}
			p.logger.Debugf("[%s] end handle cross tx in process [%s]", crossResponse.TxId, p.processName)

			// begin handle new tx
			currentTxId, err := p.handleNewTx()
			if err != nil {
				if p.Handler.TxRequest.TxContext.CurrentHeight > 0 {
					errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
						p.Handler.TxRequest.TxId, p.Handler.TxRequest.TxContext.CurrentHeight)
					p.Handler.scheduler.ReturnErrorCrossContractResponse(p.Handler.TxRequest, errResponse)
				} else {
					p.Handler.scheduler.ReturnErrorResponse(p.ChainId, currentTxId, err.Error())
				}
			}

		case txResponse := <-p.responseCh:
			if txResponse.TxId != p.Handler.TxRequest.TxId {
				p.logger.Warnf("[%s] abandon tx response due to different tx id, response tx id [%s], "+
					"current tx id [%s]", p.processName, txResponse.TxId, p.Handler.TxRequest.TxId)
				continue
			}
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

			// add tx call contract elapsed time to response
			txElapsedTime := p.Handler.scheduler.GetTxElapsedTime(txResponse.TxId)
			if txElapsedTime != nil {
				txResponse.TxElapsedTime.CrossCallCnt = txElapsedTime.CrossCallCnt
				txResponse.TxElapsedTime.CrossCallTime = txElapsedTime.CrossCallTime
			}
			// remove tx elapsed time
			p.Handler.scheduler.RemoveTxElapsedTime(p.Handler.TxRequest.TxId)

			responseCh <- txResponse

			p.logger.Debugf("[%s] end handle tx in process [%s]", txResponse.TxId, p.processName)
			// begin handle new tx
			currentTxId, err := p.handleNewTx()
			if err != nil {
				if p.Handler.TxRequest.TxContext.CurrentHeight > 0 {
					errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
						p.Handler.TxRequest.TxId, p.Handler.TxRequest.TxContext.CurrentHeight)
					p.Handler.scheduler.ReturnErrorCrossContractResponse(p.Handler.TxRequest, errResponse)
				} else {
					p.Handler.scheduler.ReturnErrorResponse(p.ChainId, currentTxId, err.Error())
				}
			}
		case <-p.Handler.txExpireTimer.C:
			// triggered by tx time out: config.SandBoxTimeout
			p.stopProcess(false)
		case err := <-p.exitCh:
			// triggered by process run into panic or exit
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
	p.logger.Debugf("[%s] handle process exist, current tx is: [%v]", currentTx.TxId, currentTx)
	// =========  condition: before cmd.wait
	// 1. created fail, ContractExecError -> return err and exit
	if existErr.err == utils.ContractExecError {
		p.Handler.scheduler.RemoveTxElapsedTime(currentTx.TxId)
		p.logger.Errorf("return back error result for process [%s] for tx [%s], %s, %s",
			p.processName, currentTx.TxId, existErr.desc, existErr.err)
		p.Handler.scheduler.ReturnErrorResponse(p.ChainId, currentTx.TxId, existErr.err.Error())

		p.logger.Debugf("release process: [%s]", p.processName)
		p.processMgr.ReleaseProcess(p.processName, p.user)
		return true
	}
	// 2. created fail, err from cmd.StdoutPipe() -> exit
	// 3. created fail, err from cgroup writeToFile fail -> exit
	// 4. created fail, contract init failed -> exit (tx will be timeout)
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_CREATED {
		p.logger.Errorf("[%s] fail to launch process: %s, %s", p.processName, existErr.desc, existErr.err)
		return true
	}
	//  ========= condition: after cmd.wait
	// 5. process expire, try to exit
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_EXPIRE {
		// when process timeout, release resources
		p.logger.Debugf("release process: [%s]", p.processName)

		p.processMgr.ReleaseProcess(p.processName, p.user)
		return true
	}

	p.logger.Errorf("[%s] process fail: tx [%s], [%s], [%s]", p.processName,
		p.Handler.TxRequest.TxId, existErr.desc, existErr.err)

	var err error
	// 6. process killed because of timeout, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT {
		err = utils.TxTimeoutPanicError
	}
	// 7. process panic, return error response and relaunch
	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_RUNNING {
		err = utils.RuntimePanicError
		p.Handler.stopTimer()
		<-p.cmdReadyCh
	}

	txElapsedTime := p.Handler.scheduler.GetTxElapsedTime(p.Handler.TxRequest.TxId)

	if currentTx.TxContext.CurrentHeight > 0 {
		p.logger.Warnf("process [%s] [%s] handle cross contract err message [%s]", p.processName,
			currentTx.TxId, existErr.err.Error())
		if txElapsedTime != nil {
			p.logger.Info(txElapsedTime.ToString(), txElapsedTime.PrintCallList())
		}
		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
			currentTx.TxId, currentTx.TxContext.CurrentHeight)
		p.Handler.scheduler.ReturnErrorCrossContractResponse(currentTx, errResponse)
		go p.startProcess()
		return false
	}

	p.Handler.scheduler.ReturnErrorResponse(p.ChainId, currentTx.TxId, err.Error())

	// time statistics, log tx spend times with detail, then remove statistics data
	if txElapsedTime != nil {
		p.logger.Info(txElapsedTime.ToString(), txElapsedTime.PrintCallList())
		p.Handler.scheduler.RemoveTxElapsedTime(p.Handler.TxRequest.TxId)
	}

	go p.startProcess()
	return false
}

// handleNewTx handle next tx or wait next available tx, process killed until expire time
// return triggered next tx successfully or not
func (p *Process) handleNewTx() (string, error) {

	select {
	case nextTx := <-p.processBalancer.GetTxQueue():
		if p.processMgr.CheckTxExpired(nextTx.TxId) {
			p.logger.Warnf("[%s] process meet expired tx[%s] before execution, removed",
				p.processName, nextTx.TxId)
			return "", nil
		}
		p.logger.Debugf("[%s] process start handle tx [%s], waiting queue size [%d]", p.processName,
			nextTx.TxId, len(p.processBalancer.GetTxQueue()))

		p.txCount++
		nextTx.TxContext.OriginalProcessName = utils.ConstructOriginalProcessName(p.processName, p.txCount)
		p.logger.Debugf("[%s] update tx original name: [%s]", p.processName, nextTx.TxContext.OriginalProcessName)

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
		p.stopProcess(true)
		return "", nil
	}
}

func (p *Process) printContractLog(contractPipe io.ReadCloser) {
	contractLogger := logger.NewDockerLogger(logger.MODULE_CONTRACT, config.DockerLogDir)
	zapLogger := contractLogger.Desugar()
	logLevel, _ := logger.TransformLogLevel(config.SandBoxLogLevel)
	if logLevel > zapcore.ErrorLevel {
		logLevel = zapcore.ErrorLevel
	}
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

// stopProcess stop process
func (p *Process) stopProcess(processTimeout bool) {
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
	p.killOnce.Do(func() {
		<-p.cmdReadyCh
		p.logger.Debugf("[%s] receive process notify and kill cross process", p.processName)
		err := p.cmd.Process.Kill()
		if err != nil {
			p.logger.Warnf("[%s] fail to kill cross process: [%s]", p.processName, err)
		}
	})
}

// kill main process when process encounter error
func (p *Process) killProcess(isTxTimeout bool) {
	<-p.cmdReadyCh
	p.logger.Debugf("[%s] kill original process", p.processName)
	err := p.cmd.Process.Kill()
	if err != nil {
		p.logger.Warnf("[%s] fail to kill corss process: %s", p.processName, err)
	}

	if !isTxTimeout {
		return
	}

	originalProcessName := p.Handler.TxRequest.TxContext.OriginalProcessName
	processDepth := p.processMgr.GetProcessDepth(originalProcessName)

	if processDepth == nil {
		return
	}

	for depth, process := range processDepth.processes {
		if process != nil {
			p.logger.Debugf("[%s] kill cross process in depth [%s]", process.processName, depth)
			process.killCrossProcess()
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

func (p *Process) returnCrossResponse(crossResponse *SDKProtogo.DMSMessage) {
	p.logger.Debugf("[%s] return cross tx response to process [%s]", crossResponse.TxId, p.processName)
	p.crossResponseCh <- crossResponse
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

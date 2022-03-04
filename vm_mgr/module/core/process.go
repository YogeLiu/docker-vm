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

type ProcessMgrInterface interface {
	getProcessDepth(initialProcessName string) *ProcessDepth
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
	txTrigger      chan bool
	expireTimer    *time.Timer // process waiting time
	Handler        *ProcessHandler
	notifyCh       chan bool

	logger *zap.SugaredLogger

	processMgr ProcessMgrInterface
	done       uint32
	mutex      sync.Mutex
}

func (p *Process) ProcessName() string {
	return p.processName
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
		txTrigger:      make(chan bool),
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
		TxWaitingQueue:  nil,
		txTrigger:       nil,
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

// LaunchProcess launch a new process
// a new process will start a cmd process and wait the process to end
// if process end because timeout, return nil
// if process end because of error, return runtime panic error and restart process
// if process end because of tx timeout, return tx timeout error and restart process
// after new process launched, it will trigger to handle tx,
// tx including init, upgrade, invoke based on the method of tx
func (p *Process) LaunchProcess() error {
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
		return err
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
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_FAIL)
		return err
	}

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, cmd.Process.Pid); err != nil {
		p.logger.Errorf("fail to add cgroup: %s", err)
		return err
	}
	p.logger.Debugf("[%s] add process to cgroup", p.processName)

	go p.printContractLog(contractOut)

	p.notifyCh <- true
	p.logger.Debugf("[%s] notify process started", p.processName)

	// wait process end, all err come from here
	// the life of wait including process initial, all running txs
	// any error will crash the process, will capture the error here
	// error including:
	// 1. running error: return runtime panic and return the tx result
	// 2. process timeout: do nothing
	// 3. tx timeout: return timeout error
	if err = cmd.Wait(); err != nil {

		p.logger.Warnf("[%s] process stopped for tx [%s], err is [%s], process state is [%s]",
			p.processName, p.Handler.TxRequest.TxId, err, p.ProcessState)

		// process exceed max process waiting time, exit process, we assume it as normal exit
		if p.ProcessState == protogo.ProcessState_PROCESS_STATE_EXPIRE {
			return nil
		}
		// cross process finished, docker manager kill it then to trigger it, we assume it as normal exit
		if p.isCrossProcess && p.ProcessState == protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED {
			return nil
		}

		if p.isCrossProcess {
			p.logger.Errorf("[%s] cross process fail: [%s], [%s]", stderr.String(), err)
		} else {
			p.logger.Errorf("[%s] process fail: tx [%s], [%s], [%s]", p.processName, p.Handler.TxRequest.TxId, stderr.String(), err)
		}

		// process fail because exceed main process max waiting time
		if p.ProcessState == protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT {
			err = utils.TxTimeoutPanicError
		}
		// process fail because of contract process error, return same error, runtime panic
		// for details, please check log
		if p.ProcessState == protogo.ProcessState_PROCESS_STATE_RUNNING {
			err = utils.RuntimePanicError
			p.Handler.stopTimer()
		}
	}

	// process exit because of err, reset process state
	p.Handler.resetHandler()
	return err
}

// InvokeProcess handle next tx or wait next available tx, process killed until expire time
// return triggered next tx successfully or not
func (p *Process) InvokeProcess() bool {

	select {
	case nextTx := <-p.TxWaitingQueue:
		p.logger.Debugf("[%s] process start handle tx [%s], waiting queue size [%d]", p.processName, nextTx.TxId, len(p.TxWaitingQueue))

		p.disableProcessExpireTimer()
		p.Handler.TxRequest = nextTx
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)

		err := p.Handler.HandleContract()
		if err != nil {
			p.logger.Errorf("[%s] process fail to invoke contract: %s", p.processName, err)
		}
		return true
	case <-p.expireTimer.C:
		p.StopProcess(true)
		return false
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
	//p.logger.Debugf("[%s] get process size, queue is [%+v]", p.processName, p.TxWaitingQueue)
	return len(p.TxWaitingQueue)
}

func (p *Process) printContractLog(contractPipe io.ReadCloser) {
	contractLogger := logger.NewDockerLogger(logger.MODULE_CONTRACT, config.DockerLogDir)

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
	close(p.notifyCh)

}

// kill main process when process encounter error
func (p *Process) killProcess(isTxTimeout bool) {

	if isTxTimeout {
		originalProcessName := p.Handler.TxRequest.TxContext.OriginalProcessName
		processDepth := p.processMgr.getProcessDepth(originalProcessName)
		if processDepth != nil {
			for depth, process := range processDepth.processes {
				if process != nil {
					p.logger.Debugf("[%s] kill cross process in depth [%d]", process.processName, depth)
					_ = process.cmd.Process.Kill()
				}
			}
		}
	}
	p.logger.Debugf("[%s] kill original process", p.processName)
	_ = p.cmd.Process.Kill()

}

func (p *Process) triggerProcessState() {
	p.logger.Debugf("[%s] trigger next tx for process", p.processName)
	p.txTrigger <- true
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

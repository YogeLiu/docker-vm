/*
	Copyright (C) BABEC. All rights reserved.
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

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"
	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/protocol"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module/security"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/pb/protogo"
)

const (
	processWaitingTime      = 60 * 10
	processWaitingQueueSize = 1000
)

type ProcessPoolInterface interface {
	RetrieveProcessContext(initialProcessName string) *ProcessContext
}

type Process struct {
	processName          string
	contractName         string
	contractVersion      string
	contractPath         string
	cGroupPath           string
	ProcessState         protogo.ProcessState
	TxWaitingQueue       chan *protogo.TxRequest
	txTrigger            chan bool
	expireTimer          *time.Timer // process waiting time
	logger               *zap.SugaredLogger
	Handler              *ProcessHandler
	user                 *security.User
	cmd                  *exec.Cmd
	processPoolInterface ProcessPoolInterface
	isCrossProcess       bool
	done                 uint32
	mutex                sync.Mutex
}

// NewProcess new process, process working on main contract which is not called cross contract
func NewProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName, contractPath string, processPool ProcessPoolInterface) *Process {

	process := &Process{
		isCrossProcess:  false,
		processName:     processName,
		contractName:    txRequest.ContractName,
		contractVersion: txRequest.ContractVersion,
		ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
		TxWaitingQueue:  make(chan *protogo.TxRequest, processWaitingQueueSize),
		txTrigger:       make(chan bool),
		expireTimer:     time.NewTimer(processWaitingTime * time.Second),
		logger:          logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),

		Handler:              nil,
		user:                 user,
		contractPath:         contractPath,
		cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
		processPoolInterface: processPool,
	}

	processHandler := NewProcessHandler(txRequest, scheduler, process)
	process.Handler = processHandler
	return process
}

// NewCrossProcess new cross process, process working on called cross process
func NewCrossProcess(user *security.User, txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName, contractPath string, processPool ProcessPoolInterface) *Process {

	process := &Process{
		isCrossProcess:  true,
		processName:     processName,
		contractName:    txRequest.ContractName,
		contractVersion: txRequest.ContractVersion,
		ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
		TxWaitingQueue:  nil,
		txTrigger:       nil,
		expireTimer:     time.NewTimer(processWaitingTime * time.Second),
		logger:          logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir),

		Handler:              nil,
		user:                 user,
		contractPath:         contractPath,
		cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
		processPoolInterface: processPool,
	}

	processHandler := NewProcessHandler(txRequest, scheduler, process)
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
	p.logger.Debugf("launch process")

	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)

	var err error           // process global error
	var stderr bytes.Buffer // used to capture the error message from contract

	cmd := exec.Cmd{
		Path:   p.contractPath,
		Args:   []string{p.user.SockPath, p.processName, p.contractName, p.contractVersion, config.SandBoxLogLevel},
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

	// start process
	if err = cmd.Start(); err != nil {
		p.logger.Errorf("fail to start process: %s", err)
		return err
	}

	// add control group
	if err = utils.WriteToFile(p.cGroupPath, cmd.Process.Pid); err != nil {
		p.logger.Errorf("fail to add cgroup: %s", err)
		return err
	}
	p.logger.Debugf("Add Process [%s] to cgroup", p.processName)

	go p.printContractLog(contractOut)

	// wait process end, all err come from here
	// the life of wait including process initial, all running txs
	// any error will crash the process, will capture the error here
	// error including:
	// 1. running error: return runtime panic and return the tx result
	// 2. process timeout: do nothing
	// 3. tx timeout: return timeout error
	if err = cmd.Wait(); err != nil {

		// process exceed max process waiting time, exit process, we assume it as normal exit
		if p.ProcessState == protogo.ProcessState_PROCESS_STATE_EXPIRE {
			return nil
		}
		// cross process finished, docker manager kill it then to trigger it, we assume it as normal exit
		if p.isCrossProcess && p.ProcessState == protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED {
			return nil
		}

		if p.isCrossProcess {
			p.logger.Errorf("cross process fail: [%s], [%s]", stderr.String(), err)
		} else {
			p.logger.Errorf("tx fail: [%s], [%s]", stderr.String(), err)
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

// InvokeProcess handle next tx
func (p *Process) InvokeProcess() {

	if len(p.TxWaitingQueue) == 0 {
		p.logger.Debugf("empty waiting queue")
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_WAITING)
		return
	}

	nextTx := <-p.TxWaitingQueue
	p.logger.Debugf("handle tx [%s]", nextTx.TxId[:5])

	// update tx in handler
	p.Handler.TxRequest = nextTx
	err := p.Handler.HandleContract()
	if err != nil {
		p.logger.Errorf("fail to invoke contract: %s", err)
	}

}

// AddTxWaitingQueue add tx with same contract to process waiting queue
func (p *Process) AddTxWaitingQueue(tx *protogo.TxRequest) {
	p.logger.Debugf("add tx to waiting queue")

	p.mutex.Lock()
	defer p.mutex.Unlock()

	tx.TxContext.OriginalProcessName = p.processName
	p.TxWaitingQueue <- tx

	if p.ProcessState == protogo.ProcessState_PROCESS_STATE_WAITING {
		p.triggerProcessState()
	}
}

// todo check need close after process end
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
	if processTimeout {
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_EXPIRE)
	} else {
		p.updateProcessState(protogo.ProcessState_PROCESS_STATE_TX_TIMEOUT)
	}
	p.killProcess()
}

// kill cross process and free process in cross process table
func (p *Process) killCrossProcess() {
	p.logger.Debugf("kill cross process: %s", p.processName)
	_ = p.cmd.Process.Kill()

}

// kill main process when process encounter error
func (p *Process) killProcess() {

	processContext := p.processPoolInterface.RetrieveProcessContext(p.processName)

	for _, process := range processContext.processList {
		if process != nil {
			p.logger.Debugf("kill process: %s", process.processName)
			_ = process.cmd.Process.Kill()
		}
	}
}

func (p *Process) triggerProcessState() {
	p.logger.Debugf("trigger next tx")
	p.updateProcessState(protogo.ProcessState_PROCESS_STATE_RUNNING)
	p.txTrigger <- true
}

func (p *Process) updateProcessState(state protogo.ProcessState) {
	p.logger.Debugf("update process state: [%s]", state)
	p.ProcessState = state
}

// resetProcessTimer reset timer when tx finished
func (p *Process) resetProcessTimer() {
	if !p.expireTimer.Stop() && len(p.expireTimer.C) > 0 {
		<-p.expireTimer.C
	}
	p.expireTimer.Reset(processWaitingTime * time.Second)
}

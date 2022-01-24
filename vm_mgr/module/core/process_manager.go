/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"fmt"
	"go.uber.org/zap"
	"strconv"
	"strings"
	"sync"
)

const (
	defaultMaxProcess = 10
)

type ProcessManager struct {
	logger *zap.SugaredLogger
	mutex  sync.Mutex

	contractManager *ContractManager
	usersManager    protocol.UserController
	scheduler       protocol.Scheduler
	// map[string]*ProcessBalance:
	// string: processNamePrefix: contractName:contractVersion
	// peerBalance:
	// sync.Map: map[index]*Process
	// int: index
	// processName: contractName:contractVersion#index (original process)
	balanceTable sync.Map

	// map[string1]map[int]*CrossProcess  sync.Map[map]
	// string1: originalProcessName: contractName:contractVersion#index#txCount - related to tx
	// int: height
	// processName: txId:height#txCount (cross process)
	crossTable sync.Map
}

func NewProcessManager(usersManager *UsersManager, contractManager *ContractManager) *ProcessManager {
	pmLogger := logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER, config.DockerLogDir)
	return &ProcessManager{
		usersManager:    usersManager,
		contractManager: contractManager,
		logger:          pmLogger,
		balanceTable:    sync.Map{},
		crossTable:      sync.Map{},
	}
}

func (pm *ProcessManager) SetScheduler(scheduler protocol.Scheduler) {
	pm.scheduler = scheduler
}

func (pm *ProcessManager) AddTx(txRequest *protogo.TxRequest) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	// processNamePrefix: contractName:contractVersion
	contractKey := utils.ConstructContractKey(txRequest.ContractName, txRequest.ContractVersion)
	// process exist, put current tx into process waiting queue and return
	processBalance := pm.getProcessBalance(contractKey)
	if processBalance == nil {
		newProcessBalance := NewProcessBalance()
		pm.balanceTable.Store(contractKey, newProcessBalance)
		pm.logger.Debugf("new process balance for contract [%s]", contractKey)
		processBalance = newProcessBalance
	}
	return pm.addTxToProcessBalance(txRequest, processBalance)
}

func (pm *ProcessManager) addTxToProcessBalance(txRequest *protogo.TxRequest, processBalance *ProcessBalance) error {
	process, err := processBalance.GetAvailableProcess()
	if err != nil {
		return err
	}
	if process != nil {
		process.AddTxWaitingQueue(txRequest)
		return nil
	}
	processName := utils.ConstructProcessName(txRequest.ContractName, txRequest.ContractVersion,
		processBalance.GetNextProcessIndex())
	process, err = pm.createNewProcess(processName, txRequest)
	if err == utils.ContractFileError {
		return err
	}
	if err != nil {
		return fmt.Errorf("faild to create process, err is: %s, txId: %s", err, txRequest.TxId)
	}
	processBalance.AddProcess(process, processName)
	process.AddTxWaitingQueue(txRequest)
	go pm.runningProcess(process)

	return nil
}

// CreateNewProcess create a new process
func (pm *ProcessManager) createNewProcess(processName string, txRequest *protogo.TxRequest) (*Process, error) {
	var (
		err          error
		user         *security.User
		contractPath string
	)

	user, err = pm.usersManager.GetAvailableUser()
	if err != nil {
		pm.logger.Errorf("fail to get available user, error: %s, txId: %s", err, txRequest.TxId)
		return nil, err
	}

	// get contract deploy path
	contractKey := utils.ConstructContractKey(txRequest.ContractName, txRequest.ContractVersion)
	contractPath, err = pm.contractManager.GetContract(txRequest.TxId, contractKey)
	if err != nil || len(contractPath) == 0 {
		pm.logger.Errorf("fail to get contract path, contractName is [%s], err is [%s]", contractKey, err)
		return nil, utils.ContractFileError
	}

	return NewProcess(user, txRequest, pm.scheduler, processName, contractPath, pm), nil
}

func (pm *ProcessManager) handleCallCrossContract(crossContractTx *protogo.TxRequest) {
	// validate contract deployed or not
	contractKey := utils.ConstructContractKey(crossContractTx.ContractName, crossContractTx.ContractVersion)
	contractPath, exist := pm.contractManager.checkContractDeployed(contractKey)

	if !exist {
		pm.logger.Errorf(utils.ContractNotDeployedError.Error())
		errResponse := constructCallContractErrorResponse(utils.ContractNotDeployedError.Error(), crossContractTx.TxId,
			crossContractTx.TxContext.CurrentHeight)
		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}
	// new process, process just for one tx
	user, err := pm.usersManager.GetAvailableUser()
	if err != nil {
		errMsg := fmt.Sprintf("fail to get available user: %s", err)
		pm.logger.Errorf(errMsg)
		errResponse := constructCallContractErrorResponse(errMsg, crossContractTx.TxId,
			crossContractTx.TxContext.CurrentHeight)
		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}

	processName := utils.ConstructCrossContractProcessName(crossContractTx.TxId,
		uint64(crossContractTx.TxContext.CurrentHeight))

	newProcess := NewCrossProcess(user, crossContractTx, pm.scheduler, processName, contractPath, pm)

	// register cross process
	pm.RegisterCrossProcess(crossContractTx.TxContext.OriginalProcessName, crossContractTx.TxContext.CurrentHeight,
		newProcess)

	err = newProcess.LaunchProcess()
	if err != nil {
		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
			crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
	}

	txContext := newProcess.Handler.TxRequest.TxContext
	pm.ReleaseCrossProcess(newProcess.processName, txContext.OriginalProcessName, txContext.CurrentHeight)
	_ = pm.usersManager.FreeUser(newProcess.user)
}

// ReleaseProcess release original process
// @param: processName: contract:version#index
func (pm *ProcessManager) ReleaseProcess(processName string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.logger.Infof("release process: [%s]", processName)

	nameList := strings.Split(processName, "#")
	contractKey := nameList[0]
	pm.removeProcessFromProcessBalance(contractKey, processName)
}

func (pm *ProcessManager) RegisterCrossProcess(originalProcessName string, depth uint32, calledProcess *Process) {
	pm.logger.Debugf("register cross process [%s], original process name [%s]",
		calledProcess.processName, originalProcessName)
	pm.addCrossProcessIntoDepth(originalProcessName, depth, calledProcess)
	calledProcess.Handler.processName = calledProcess.processName
}

func (pm *ProcessManager) ReleaseCrossProcess(crossProcessName string, originalProcessName string, depth uint32) {
	pm.logger.Debugf("release cross process [%s], original process name [%s]",
		crossProcessName, originalProcessName)
	pm.removeCrosseProcessFromDepth(originalProcessName, depth)
}

func (pm *ProcessManager) runningProcess(process *Process) {
	// execute contract method, including init, invoke
	go pm.listenProcessInvoke(process)
	// launch process wait block until process finished
runProcess:
	err := process.LaunchProcess()

	if err != nil && process.ProcessState != protogo.ProcessState_PROCESS_STATE_EXPIRE {
		currentTx := process.Handler.TxRequest

		pm.logger.Warnf("scheduler noticed process [%s] stop, tx [%s], err [%s]", process.processName, process.Handler.TxRequest.TxId, err)

		peerDepth := pm.getProcessDepth(currentTx.TxContext.OriginalProcessName)

		//todo: re check which child process is fail
		if len(peerDepth) > 1 {
			lastChildProcessHeight := len(peerDepth)
			lastContractName := peerDepth[uint32(lastChildProcessHeight)].contractName
			errMsg := fmt.Sprintf("%s fail: %s", lastContractName, err.Error())
			pm.scheduler.ReturnErrorResponse(currentTx.TxId, errMsg)
		} else {
			pm.logger.Errorf("return back error result for process [%s] for tx [%s]", process.processName, currentTx.TxId)
			pm.scheduler.ReturnErrorResponse(currentTx.TxId, err.Error())
		}

		if process.ProcessState != protogo.ProcessState_PROCESS_STATE_FAIL {
			// restart process and trigger next
			pm.logger.Debugf("restart process [%s]", process.processName)
			goto runProcess
		}
	}

	// when process timeout, release resources
	pm.logger.Debugf("release process: [%s]", process.processName)

	pm.ReleaseProcess(process.processName)
	_ = pm.usersManager.FreeUser(process.user)
}

func (pm *ProcessManager) listenProcessInvoke(process *Process) {
	for {
		select {
		case <-process.txTrigger:
			if process.ProcessState != protogo.ProcessState_PROCESS_STATE_READY {
				continue
			}
			success := process.InvokeProcess()
			if !success {
				return
			}
		case <-process.Handler.txExpireTimer.C:
			process.StopProcess(false)
		}
	}
}

// GetProcess retrieve process from process manager, could be original process or cross process:
// contractName:contractVersion#timestamp:index#txCount#depth
// original process: contractName:contractVersion#timestamp:index
func (pm *ProcessManager) GetProcess(processName string) *Process {
	pm.logger.Debugf("get process [%s]", processName)
	pm.mutex.Lock()
	defer pm.mutex.Unlock()
	nameList := strings.Split(processName, "#")
	if len(nameList) == 4 {
		crossKey := strings.Join(nameList[:3], "#")
		depth, _ := strconv.Atoi(nameList[3])
		cd, _ := pm.crossTable.Load(crossKey)
		crossDepth := cd.(map[uint32]*Process)
		return crossDepth[uint32(depth)]
	}

	contractKey := nameList[0]

	processBalance := pm.getProcessBalance(contractKey)
	return processBalance.GetProcess(processName)
}

func (pm *ProcessManager) getProcessBalance(key string) *ProcessBalance {
	processBalance, ok := pm.balanceTable.Load(key)
	if ok {
		return processBalance.(*ProcessBalance)
	}
	return nil
}

func (pm *ProcessManager) removeProcessFromProcessBalance(contractKey string, processName string) {
	pb, ok := pm.balanceTable.Load(contractKey)
	if !ok {
		return
	}

	processBalance := pb.(*ProcessBalance)
	processBalance.RemoveProcess(processName)

	if processBalance.Size() == 0 {
		pm.balanceTable.Delete(contractKey)
	}
}

func (pm *ProcessManager) addCrossProcessIntoDepth(originalProcessName string, height uint32, calledProcess *Process) {

	pm.logger.Debugf("add cross process %s with initial process name %s",
		calledProcess.processName, originalProcessName)

	processDepth, ok := pm.crossTable.Load(originalProcessName)
	if !ok {
		newProcessDepth := make(map[uint32]*Process)
		newProcessDepth[height] = calledProcess
		pm.crossTable.Store(originalProcessName, newProcessDepth)
		processDepth = newProcessDepth
	}

	pd := processDepth.(map[uint32]*Process)
	pd[height] = calledProcess
}

func (pm *ProcessManager) removeCrosseProcessFromDepth(originalProcessName string, depth uint32) {
	pm.logger.Debugf("release cross process with position: %v", depth)
	processDepth, ok := pm.crossTable.Load(originalProcessName)
	if ok {
		pd := processDepth.(map[uint32]*Process)
		delete(pd, depth)
		if len(pd) == 0 {
			pm.crossTable.Delete(originalProcessName)
		}
	}
}

func (pm *ProcessManager) getProcessDepth(originalProcessName string) map[uint32]*Process {
	depthTable, ok := pm.crossTable.Load(originalProcessName)
	if ok {
		return depthTable.(map[uint32]*Process)
	}
	return nil
}

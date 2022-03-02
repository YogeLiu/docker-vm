/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"go.uber.org/zap"
)

type ProcessManager struct {
	logger         *zap.SugaredLogger
	balanceRWMutex sync.RWMutex
	crossRWMutex   sync.RWMutex

	contractManager *ContractManager
	usersManager    protocol.UserController
	scheduler       protocol.Scheduler
	// map[string]*ProcessBalance:
	// string: processNamePrefix: contractName:contractVersion
	// peerBalance:
	// sync.Map: map[index]*Process
	// int: index
	// processName: contractName:contractVersion#index (original process)
	balanceTable map[string]*ProcessBalance

	// map[string1]map[int]*CrossProcess  sync.Map[map]
	// string1: originalProcessName: contractName:contractVersion#index#txCount - related to tx
	// int: height
	// processName: txId:height#txCount (cross process)
	crossTable map[string]*ProcessDepth
}

func NewProcessManager(usersManager *UsersManager, contractManager *ContractManager) *ProcessManager {
	pmLogger := logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER, config.DockerLogDir)
	return &ProcessManager{
		usersManager:    usersManager,
		contractManager: contractManager,
		logger:          pmLogger,
		balanceTable:    make(map[string]*ProcessBalance),
		crossTable:      make(map[string]*ProcessDepth),
	}
}

func (pm *ProcessManager) SetScheduler(scheduler protocol.Scheduler) {
	pm.scheduler = scheduler
}

func (pm *ProcessManager) AddTx(txRequest *protogo.TxRequest) error {
	pm.balanceRWMutex.Lock()
	defer pm.balanceRWMutex.Unlock()
	// processNamePrefix: contractName:contractVersion
	contractKey := utils.ConstructContractKey(txRequest.ContractName, txRequest.ContractVersion)
	// process exist, put current tx into process waiting queue and return
	processBalance := pm.balanceTable[contractKey]
	if processBalance == nil {
		newProcessBalance := NewProcessBalance()
		pm.balanceTable[contractKey] = newProcessBalance
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

func (pm *ProcessManager) removeProcessFromProcessBalance(contractKey string, processName string) bool {
	pm.balanceRWMutex.Lock()
	defer pm.balanceRWMutex.Unlock()
	processBalance, ok := pm.balanceTable[contractKey]
	if !ok {
		return true
	}
	process := processBalance.GetProcess(processName)
	if process == nil {
		return true
	}
	if process.Size() > 0 {
		return false
	}
	processBalance.RemoveProcess(processName)
	if processBalance.Size() == 0 {
		delete(pm.balanceTable, contractKey)
	}
	return true
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

	newCrossProcess := NewCrossProcess(user, crossContractTx, pm.scheduler, processName, contractPath, pm)

	// register cross process
	pm.RegisterCrossProcess(crossContractTx.TxContext.OriginalProcessName, newCrossProcess)

	err = newCrossProcess.LaunchProcess()
	if err != nil {
		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
			crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
	}

	txContext := newCrossProcess.Handler.TxRequest.TxContext
	pm.ReleaseCrossProcess(newCrossProcess.processName, txContext.OriginalProcessName)
	_ = pm.usersManager.FreeUser(newCrossProcess.user)
}

// ReleaseProcess release balance process
// @param: processName: contract:version#timestamp:index
func (pm *ProcessManager) ReleaseProcess(processName string) bool {
	pm.logger.Infof("release process: [%s]", processName)
	contractKey := utils.GetContractKeyFromProcessName(processName)
	return pm.removeProcessFromProcessBalance(contractKey, processName)
}

func (pm *ProcessManager) runningProcess(process *Process) {
	// execute contract method, including init, invoke
	go pm.listenProcessInvoke(process)
	// launch process wait block until process finished
runProcess:
	err := process.LaunchProcess()

	if err != nil && process.ProcessState != protogo.ProcessState_PROCESS_STATE_EXPIRE {
		currentTx := process.Handler.TxRequest

		pm.logger.Warnf("scheduler noticed process [%s] stop, tx [%s], err [%s]", process.processName,
			process.Handler.TxRequest.TxId, err)

		processDepth := pm.getProcessDepth(currentTx.TxContext.OriginalProcessName)
		if processDepth == nil {
			pm.logger.Warnf("return back error result for process [%s] for tx [%s]", process.processName, currentTx.TxId)
			pm.scheduler.ReturnErrorResponse(currentTx.TxId, err.Error())
		} else {
			errMsg := fmt.Sprintf("cross contract fail: err is:%s, cross processes: %s", err.Error(),
				processDepth.GetConcatProcessName())
			pm.logger.Warn(errMsg)
			pm.scheduler.ReturnErrorResponse(currentTx.TxId, errMsg)
		}

		if process.ProcessState != protogo.ProcessState_PROCESS_STATE_FAIL {
			// restart process and trigger next
			pm.logger.Debugf("restart process [%s]", process.processName)
			goto runProcess
		}
	}

	// when process timeout, release resources
	pm.logger.Debugf("release process: [%s]", process.processName)

	if !pm.ReleaseProcess(process.processName) {
		goto runProcess
	}
	_ = pm.usersManager.FreeUser(process.user)
}

func (pm *ProcessManager) listenProcessInvoke(process *Process) {
	for {
		select {
		case <-process.txTrigger:
			if process.ProcessState != protogo.ProcessState_PROCESS_STATE_READY {
				continue
			}
			success, currentTxId, err := process.InvokeProcess()

			if !success {
				return
			}

			if err != nil {
				pm.scheduler.ReturnErrorResponse(currentTxId, err.Error())
			}

		case <-process.Handler.txExpireTimer.C:
			process.StopProcess(false)
		}
	}
}

// GetProcess retrieve process from process manager, could be original process or cross process:
// cross process: contractName:contractVersion#timestamp:index#txCount#depth
// original process: contractName:contractVersion#timestamp:index
func (pm *ProcessManager) GetProcess(processName string) *Process {
	pm.logger.Debugf("get process [%s]", processName)
	isCrossProcess, processName1, processName2 := utils.TrySplitCrossProcessNames(processName)
	if isCrossProcess {
		pm.crossRWMutex.RLock()
		defer pm.crossRWMutex.RUnlock()
		crossDepth := pm.crossTable[processName1]
		return crossDepth.GetProcess(processName2)
	}
	pm.balanceRWMutex.RLock()
	defer pm.balanceRWMutex.RUnlock()
	processBalance := pm.balanceTable[processName1]
	return processBalance.GetProcess(processName2)
}

func (pm *ProcessManager) RegisterCrossProcess(originalProcessName string, crossProcess *Process) {
	pm.logger.Debugf("register cross process [%s], original process name [%s]",
		crossProcess.processName, originalProcessName)
	pm.crossRWMutex.Lock()
	defer pm.crossRWMutex.Unlock()
	processDepth, ok := pm.crossTable[originalProcessName]
	if !ok {
		newProcessDepth := NewProcessDepth()
		pm.crossTable[originalProcessName] = newProcessDepth
		processDepth = newProcessDepth
	}
	processDepth.AddProcess(crossProcess.processName, crossProcess)
}

func (pm *ProcessManager) ReleaseCrossProcess(crossProcessName string, originalProcessName string) {
	pm.logger.Debugf("release cross process [%s], original process name [%s]",
		crossProcessName, originalProcessName)
	pm.crossRWMutex.Lock()
	defer pm.crossRWMutex.Unlock()
	processDepth, ok := pm.crossTable[originalProcessName]
	if !ok {
		return
	}
	processDepth.RemoveProcess(crossProcessName)
	if processDepth.Size() == 0 {
		delete(pm.crossTable, originalProcessName)
	}
}

func (pm *ProcessManager) getProcessDepth(originalProcessName string) *ProcessDepth {
	pm.crossRWMutex.RLock()
	defer pm.crossRWMutex.RUnlock()
	return pm.crossTable[originalProcessName]
}

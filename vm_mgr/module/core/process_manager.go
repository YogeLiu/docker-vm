/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/tx_requests"
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
	// processNamePrefix: chainId#contractName#contractVersion
	contractKey := utils.ConstructContractKey(txRequest.ChainId, txRequest.ContractName, txRequest.ContractVersion)
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

	processBalance.AddTx(txRequest)

	needCreateNewProcess := processBalance.needCreateNewProcess()
	pm.logger.Debugf("[%s] need create new process: %v", txRequest.TxId, needCreateNewProcess)
	if !needCreateNewProcess {
		return nil
	}
	// todo add create process time statistics
	processName := utils.ConstructProcessName(txRequest.ChainId, txRequest.ContractName, txRequest.ContractVersion,
		processBalance.GetNextProcessIndex())
	process, err := pm.createNewProcess(processName, txRequest, processBalance)
	if err == utils.ContractFileError {
		// using existing process to handle tx
		if processBalance.Size() > 0 {
			return nil
		}
		// remove tx request from queue
		<-processBalance.GetTxQueue()
		return err
	}
	if err != nil {
		return fmt.Errorf("faild to create process, err is: %s, txId: %s", err, txRequest.TxId)
	}
	processBalance.AddProcess(process, processName)
	pm.logger.Debugf("[%s] update process length: %d", txRequest.TxId, processBalance.Size())
	go process.ExecProcess()

	return nil
}

// CreateNewProcess create a new process
func (pm *ProcessManager) createNewProcess(processName string, txRequest *protogo.TxRequest,
	processBalance *ProcessBalance) (*Process, error) {
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
	contractKey := utils.ConstructContractKey(txRequest.ChainId, txRequest.ContractName, txRequest.ContractVersion)
	contractPath, err = pm.contractManager.GetContract(txRequest.ChainId, txRequest.TxId, contractKey)
	if err != nil || len(contractPath) == 0 {
		pm.logger.Errorf("fail to get contract path, contractName is [%s], err is [%s]", contractKey, err)
		return nil, utils.ContractFileError
	}

	return NewProcess(user, txRequest, pm.scheduler, processName, contractPath, pm, processBalance), nil
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

// ReleaseProcess release balance process
// @param: processName: contract:version#timestamp:index
func (pm *ProcessManager) ReleaseProcess(processName string, user *security.User) {
	pm.logger.Infof("release process: [%s]", processName)
	contractKey := utils.GetContractKeyFromProcessName(processName)
	//released := pm.removeProcessFromProcessBalance(contractKey, processName)
	pm.removeProcessFromProcessBalance(contractKey, processName)
	_ = pm.usersManager.FreeUser(user)
}

func (pm *ProcessManager) removeProcessFromProcessBalance(contractKey string, processName string) {
	pm.balanceRWMutex.Lock()
	defer pm.balanceRWMutex.Unlock()
	processBalance, ok := pm.balanceTable[contractKey]
	if !ok {
		return
	}
	process := processBalance.GetProcess(processName)
	if process == nil {
		return
	}
	processBalance.RemoveProcess(processName)
	if processBalance.Size() == 0 {
		delete(pm.balanceTable, contractKey)
	}
}

//func (pm *ProcessManager) handleCallCrossContract(crossContractTx *protogo.TxRequest) {
//	// validate contract deployed or not
//	contractKey := utils.ConstructContractKey(crossContractTx.ContractName, crossContractTx.ContractVersion)
//	contractPath, err := pm.contractManager.GetContract(crossContractTx.ChainId, crossContractTx.TxId, contractKey)
//	if err != nil {
//		pm.logger.Errorf(err.Error())
//		errResponse := constructCallContractErrorResponse(err.Error(), crossContractTx.TxId,
//			crossContractTx.TxContext.CurrentHeight)
//		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
//		return
//	}
//	// new process, process just for one tx
//	user, err := pm.usersManager.GetAvailableUser()
//	if err != nil {
//		errMsg := fmt.Sprintf("fail to get available user: %s", err)
//		pm.logger.Errorf(errMsg)
//		errResponse := constructCallContractErrorResponse(errMsg, crossContractTx.TxId,
//			crossContractTx.TxContext.CurrentHeight)
//		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
//		return
//	}
//
//	processName := utils.ConstructCrossContractProcessName(crossContractTx.ChainId, crossContractTx.TxId,
//		uint64(crossContractTx.TxContext.CurrentHeight))
//
//	newCrossProcess := NewCrossProcess(user, crossContractTx, pm.scheduler, processName, contractPath, pm)
//
//	// register cross process
//	pm.RegisterCrossProcess(crossContractTx.TxContext.OriginalProcessName, newCrossProcess)
//
//	// 1. success finished
//	// 2. panic
//	// 3. timeout
//	exitErr := newCrossProcess.LaunchProcess()
//	if exitErr != nil {
//		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
//			crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
//		pm.scheduler.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
//	}
//
//	txContext := newCrossProcess.Handler.TxRequest.TxContext
//	pm.ReleaseCrossProcess(newCrossProcess.processName, txContext.OriginalProcessName)
//	_ = pm.usersManager.FreeUser(newCrossProcess.user)
//}

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

func (pm *ProcessManager) GetProcessDepth(originalProcessName string) *ProcessDepth {
	pm.crossRWMutex.RLock()
	defer pm.crossRWMutex.RUnlock()
	return pm.crossTable[originalProcessName]
}

func (pm *ProcessManager) ModifyContractName(txRequest *protogo.TxRequest) error {
	sysCallStart := time.Now()
	defer func() {
		// add time statistics
		spend := time.Since(sysCallStart).Nanoseconds()
		sysCallElapsedTime := tx_requests.NewSysCallElapsedTime(protogo.CDMType_CDM_TYPE_GET_CONTRACT_NAME, sysCallStart.UnixNano(), spend)
		pm.scheduler.AddTxSysCallElapsedTime(txRequest.TxId, sysCallElapsedTime)
	}()

	contractName, err := pm.contractManager.GetContractName(txRequest.ChainId, txRequest.TxId, txRequest.ContractName)
	if err != nil {
		return err
	}
	pm.logger.Debugf("replace txrequest contract name from %s to %s", txRequest.ContractName, contractName)
	txRequest.ContractName = contractName
	return nil
}

// CheckTxExpired return true for tx expired, false for not
func (pm *ProcessManager) CheckTxExpired(txID string) bool {
	if txTime := pm.scheduler.GetTxElapsedTime(txID); txTime != nil {
		// time from scheduler received to process received > expired time limit
		if time.Now().UnixNano()-txTime.StartTime > config.TxExpireTime*time.Second.Nanoseconds() {
			return true
		}
	}
	return false
}

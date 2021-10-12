/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/utils"

	SDKProtogo "chainmaker.org/chainmaker-contract-sdk-docker-go/pb_sdk/protogo"

	"golang.org/x/sync/singleflight"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/protocol"
	"go.uber.org/zap"
)

const (
	// ReqChanSize tx request chan size
	ReqChanSize = 1000
	// ResponseChanSize tx response chan size
	ResponseChanSize = 1000

	crossContractsChanSize = 50
)

type DockerScheduler struct {
	lock            sync.Mutex
	logger          *zap.SugaredLogger
	singleFlight    singleflight.Group
	userController  protocol.UserController
	contractManager *ContractManager
	processPool     *ProcessPool

	txReqCh          chan *protogo.TxRequest
	txResponseCh     chan *protogo.TxResponse
	getStateReqCh    chan *protogo.CDMMessage
	getByteCodeReqCh chan *protogo.CDMMessage
	responseChMap    sync.Map

	crossContractsCh chan *protogo.TxRequest
}

// NewDockerScheduler new docker scheduler
func NewDockerScheduler(userController protocol.UserController, processPool *ProcessPool) *DockerScheduler {

	contractManager := NewContractManager()

	scheduler := &DockerScheduler{
		userController:  userController,
		logger:          logger.NewDockerLogger(logger.MODULE_SCHEDULER),
		processPool:     processPool,
		contractManager: contractManager,

		txReqCh:          make(chan *protogo.TxRequest, ReqChanSize),
		txResponseCh:     make(chan *protogo.TxResponse, ResponseChanSize),
		getStateReqCh:    make(chan *protogo.CDMMessage, ReqChanSize*8),
		getByteCodeReqCh: make(chan *protogo.CDMMessage, ReqChanSize),
		crossContractsCh: make(chan *protogo.TxRequest, crossContractsChanSize),
		responseChMap:    sync.Map{},
	}

	contractManager.scheduler = scheduler

	return scheduler
}

// GetTxReqCh get tx request chan
func (s *DockerScheduler) GetTxReqCh() chan *protogo.TxRequest {
	return s.txReqCh
}

// GetTxResponseCh get tx response ch
func (s *DockerScheduler) GetTxResponseCh() chan *protogo.TxResponse {
	return s.txResponseCh
}

// GetGetStateReqCh retrieve get state request chan
func (s *DockerScheduler) GetGetStateReqCh() chan *protogo.CDMMessage {
	return s.getStateReqCh
}

// GetCrossContractReqCh get cross contract request chan
func (s *DockerScheduler) GetCrossContractReqCh() chan *protogo.TxRequest {
	return s.crossContractsCh
}

// GetByteCodeReqCh get bytecode request chan
func (s *DockerScheduler) GetByteCodeReqCh() chan *protogo.CDMMessage {
	return s.getByteCodeReqCh
}

// RegisterResponseCh register response chan
func (s *DockerScheduler) RegisterResponseCh(responseId string, responseCh chan *protogo.CDMMessage) {
	s.responseChMap.Store(responseId, responseCh)
}

// GetResponseChByTxId get response chan by tx id
func (s *DockerScheduler) GetResponseChByTxId(txId string) chan *protogo.CDMMessage {

	responseCh, _ := s.responseChMap.Load(txId)
	s.responseChMap.Delete(txId)
	return responseCh.(chan *protogo.CDMMessage)
}

// RegisterCrossContractResponseCh register cross contract response chan
func (s *DockerScheduler) RegisterCrossContractResponseCh(responseId string, responseCh chan *SDKProtogo.DMSMessage) {
	s.responseChMap.Store(responseId, responseCh)
}

// GetCrossContractResponseCh get cross contract response chan
func (s *DockerScheduler) GetCrossContractResponseCh(responseId string) chan *SDKProtogo.DMSMessage {

	responseCh, _ := s.responseChMap.Load(responseId)
	s.responseChMap.Delete(responseId)
	return responseCh.(chan *SDKProtogo.DMSMessage)
}

// StartScheduler start docker scheduler
func (s *DockerScheduler) StartScheduler() {

	s.logger.Infof("start docker scheduler")

	go s.listenIncomingTxRequest()

}

// StopScheduler todo may doesn't need
func (s *DockerScheduler) StopScheduler() {
	s.logger.Debugf("stop docker scheduler")
	close(s.txResponseCh)
	close(s.txReqCh)
	close(s.getStateReqCh)
	close(s.getByteCodeReqCh)
}

func (s *DockerScheduler) listenIncomingTxRequest() {
	s.logger.Debugf("start listen incoming tx request")

	for {
		select {
		case txRequest := <-s.txReqCh:
			go s.handleTx(txRequest)
		case crossContractMsg := <-s.crossContractsCh:
			go s.handleCallCrossContract(crossContractMsg)
		}
	}
}

func (s *DockerScheduler) handleTx(txRequest *protogo.TxRequest) {

	var err error
	var process *Process

	processName := s.constructProcessName(txRequest)
	process, exist := s.processPool.CheckProcessExist(processName)

	// process exist, put current tx into process waiting queue and return
	// only one goroutine can init process for same contract
	if exist {
		process.AddTxWaitingQueue(txRequest)
		return
	}

	// -----  process doesn't exist, init new process  ----
	proc, err, _ := s.singleFlight.Do(processName, func() (interface{}, error) {
		defer s.singleFlight.Forget(processName)

		user, err := s.userController.GetAvailableUser()
		if err != nil {
			s.logger.Errorf("fail to get available user: %s", err)
			return nil, err
		}

		// get contract deploy path
		contractKey := s.constructContractKey(txRequest.ContractName, txRequest.ContractVersion)
		contractPath, err := s.contractManager.GetContract(txRequest.TxId, contractKey)
		if err != nil || len(contractPath) == 0 {
			s.logger.Errorf("fail to get contract path, contractName is [%s], err is [%s]", contractKey, err)
			return nil, err
		}

		newProcess := NewProcess(user, txRequest, s, processName, contractPath, s.processPool)
		s.processPool.RegisterNewProcess(processName, newProcess)

		return newProcess, nil
	})
	if err != nil {
		s.returnErrorTxResponse(txRequest.TxId, err.Error())
		return
	}

	process = proc.(*Process)
	process.AddTxWaitingQueue(txRequest)

	// only one goroutine can launch process
	// reset will return
	if atomic.LoadUint32(&process.done) == 0 {
		process.mutex.Lock()
		defer process.mutex.Unlock()
		if process.done == 0 {
			atomic.StoreUint32(&process.done, 1)
			go s.initProcess(process)
		}
	}

}

func (s *DockerScheduler) initProcess(process *Process) {
	// execute contract method, including init, invoke
	go s.listenProcessInvoke(process)
	// launch process wait block until process finished
runProcess:
	err := process.LaunchProcess()

	if err != nil && process.ProcessState != protogo.ProcessState_PROCESS_STATE_EXPIRE {
		currentTx := process.Handler.TxRequest

		processContext := s.processPool.RetrieveProcessContext(process.processName)
		if processContext.size > 1 {
			lastContractName := processContext.processList[processContext.size-1].contractName
			errMsg := fmt.Sprintf("%s fail: %s", lastContractName, err.Error())
			s.returnErrorTxResponse(currentTx.TxId, errMsg)
		} else {
			s.returnErrorTxResponse(currentTx.TxId, err.Error())
		}
		// restart process and trigger next
		goto runProcess
	}

	// when process timeout, release resources
	s.logger.Debugf("release process: [%s]", process.processName)

	s.processPool.ReleaseProcess(process.processName)
	_ = s.userController.FreeUser(process.user)

}

func (s *DockerScheduler) listenProcessInvoke(process *Process) {

	for {
		select {
		case <-process.txTrigger:
			process.InvokeProcess()
		case <-process.Handler.txExpireTimer.C:
			process.StopProcess(false)
		case <-process.expireTimer.C:
			process.StopProcess(true)
			return
		}
	}
}

func (s *DockerScheduler) handleCallCrossContract(crossContractTx *protogo.TxRequest) {

	// validate contract deployed or not
	contractKey := s.constructContractKey(crossContractTx.ContractName, crossContractTx.ContractVersion)
	contractPath, exist := s.contractManager.checkContractDeployed(contractKey)

	if !exist {
		s.logger.Errorf(utils.ContractNotDeployedError.Error())
		errResponse := constructCallContractErrorResponse(utils.ContractNotDeployedError.Error(), crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		s.returnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}
	// new process, process just for one tx
	user, err := s.userController.GetAvailableUser()
	if err != nil {
		errMsg := fmt.Sprintf("fail to get available user: %s", err)
		s.logger.Errorf(errMsg)
		errResponse := constructCallContractErrorResponse(errMsg, crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		s.returnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}

	processName := s.constructCrossContractProcessName(crossContractTx)

	newProcess := NewCrossProcess(user, crossContractTx, s, processName, contractPath, s.processPool)
	s.processPool.RegisterNewProcess(processName, newProcess)

	// register cross process
	s.processPool.RegisterCrossProcess(crossContractTx.TxContext.OriginalProcessName, newProcess)

	err = newProcess.LaunchProcess()
	if err != nil {
		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(), crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		s.returnErrorCrossContractResponse(crossContractTx, errResponse)
	}

	txContext := newProcess.Handler.TxRequest.TxContext
	s.processPool.ReleaseCrossProcess(txContext.OriginalProcessName, txContext.CurrentHeight)
	_ = s.userController.FreeUser(newProcess.user)
}

func (s *DockerScheduler) returnErrorTxResponse(txId string, errMsg string) {
	errTxResponse := s.constructErrorResponse(txId, errMsg)
	s.txResponseCh <- errTxResponse
}

func (s *DockerScheduler) constructErrorResponse(txId string, errMsg string) *protogo.TxResponse {
	return &protogo.TxResponse{
		TxId:    txId,
		Code:    protogo.ContractResultCode_FAIL,
		Result:  nil,
		Message: errMsg,
	}
}

func (s *DockerScheduler) returnErrorCrossContractResponse(crossContractTx *protogo.TxRequest, errResponse *SDKProtogo.DMSMessage) {

	responseChId := crossContractChKey(crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
	responseCh := s.GetCrossContractResponseCh(responseChId)

	responseCh <- errResponse
}

// processName: contractName:contractVersion
func (s *DockerScheduler) constructProcessName(tx *protogo.TxRequest) string {
	handlerName := tx.ContractName + ":" + tx.ContractVersion
	return handlerName
}

func (s *DockerScheduler) constructCrossContractProcessName(tx *protogo.TxRequest) string {
	return tx.TxId + ":" + strconv.FormatUint(uint64(tx.TxContext.CurrentHeight), 10)
}

// constructContractKey contractKey: contractName:contractVersion
func (s *DockerScheduler) constructContractKey(contractName, contractVersion string) string {
	return contractName + "#" + contractVersion
}

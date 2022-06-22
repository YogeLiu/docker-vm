/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"

	"golang.org/x/sync/singleflight"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
)

const (
	// ReqChanSize tx request chan size
	ReqChanSize = 15000
	// ResponseChanSize tx response chan size
	ResponseChanSize = 15000

	crossContractsChanSize = 50
)

type DockerScheduler struct {
	lock           sync.Mutex
	logger         *zap.SugaredLogger
	singleFlight   singleflight.Group
	processManager *ProcessManager

	txReqCh          chan *protogo.TxRequest
	txResponseCh     chan *protogo.TxResponse
	getStateReqCh    chan *protogo.CDMMessage
	getByteCodeReqCh chan *protogo.CDMMessage
	responseChMap    sync.Map

	crossContractsCh chan *protogo.TxRequest

	// TxRequestMgr key: tx unique id
	TxRequestMgr map[string]*TxElapsedTime
}

// NewDockerScheduler new docker scheduler
func NewDockerScheduler(processManager *ProcessManager) *DockerScheduler {
	scheduler := &DockerScheduler{
		logger:         logger.NewDockerLogger(logger.MODULE_SCHEDULER, config.DockerLogDir),
		processManager: processManager,

		txReqCh:          make(chan *protogo.TxRequest, ReqChanSize),
		txResponseCh:     make(chan *protogo.TxResponse, ResponseChanSize),
		getStateReqCh:    make(chan *protogo.CDMMessage, ReqChanSize*8),
		getByteCodeReqCh: make(chan *protogo.CDMMessage, ReqChanSize),
		crossContractsCh: make(chan *protogo.TxRequest, crossContractsChanSize),
		responseChMap:    sync.Map{},
		// TxRequestMgr key: tx unique id
		TxRequestMgr: make(map[string]*TxElapsedTime),
	}

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
func (s *DockerScheduler) RegisterResponseCh(chainId, responseId string, responseCh chan *protogo.CDMMessage) {
	schedulerKey := utils.ConstructSchedulerKey(chainId, responseId)
	s.responseChMap.Store(schedulerKey, responseCh)
}

// GetResponseChByTxId get response chan by tx id
func (s *DockerScheduler) GetResponseChByTxId(chainId, txId string) chan *protogo.CDMMessage {
	schedulerKey := utils.ConstructSchedulerKey(chainId, txId)
	responseCh, _ := s.responseChMap.LoadAndDelete(schedulerKey)
	return responseCh.(chan *protogo.CDMMessage)
}

// RegisterCrossContractResponseCh register cross contract response chan
func (s *DockerScheduler) RegisterCrossContractResponseCh(chainId, responseId string, responseCh chan *SDKProtogo.DMSMessage) {
	schedulerKey := utils.ConstructSchedulerKey(chainId, responseId)
	s.responseChMap.Store(schedulerKey, responseCh)
}

// GetCrossContractResponseCh get cross contract response chan
func (s *DockerScheduler) GetCrossContractResponseCh(chainId, responseId string) chan *SDKProtogo.DMSMessage {
	schedulerKey := utils.ConstructSchedulerKey(chainId, responseId)

	responseCh, loaded := s.responseChMap.LoadAndDelete(schedulerKey)
	if !loaded {
		return nil
	}
	return responseCh.(chan *SDKProtogo.DMSMessage)
}

// StartScheduler start docker scheduler
func (s *DockerScheduler) StartScheduler() {

	s.logger.Debugf("start docker scheduler")

	go s.listenIncomingTxRequest()

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
	s.logger.Debugf("[%s] docker scheduler handle tx", txRequest.TxId)
	s.RegisterTxElapsedTime(txRequest, time.Now().UnixNano())
	err := s.processManager.AddTx(txRequest)
	if err == utils.ContractFileError {
		s.logger.Errorf("failed to add tx, err is :%s, txId: %s",
			err, txRequest.TxId)
		s.ReturnErrorResponse(txRequest.ChainId, txRequest.TxId, err.Error())
		return
	}
	if err != nil {
		s.logger.Warnf("add tx warning: err is :%s, txId: %s",
			err, txRequest.TxId)
		return
	}
}

func (s *DockerScheduler) handleCallCrossContract(crossContractTx *protogo.TxRequest) {
	s.logger.Debugf("[%s] docker scheduler handle cross contract tx", crossContractTx.TxId)

	err := s.processManager.ModifyContractName(crossContractTx)
	if err != nil {
		s.logger.Warnf("cant get cross contract name: err is :%s, txId: %s",
			err, crossContractTx.TxId)
		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
			crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		s.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}

	err = s.processManager.AddTx(crossContractTx)
	if err == utils.ContractFileError {
		s.logger.Errorf("failed to add tx, err is :%s, txId: %s",
			err, crossContractTx.TxId)

		errResponse := constructCallContractErrorResponse(utils.CrossContractRuntimePanicError.Error(),
			crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
		s.ReturnErrorCrossContractResponse(crossContractTx, errResponse)
		return
	}
	if err != nil {
		s.logger.Warnf("add cross contract tx warning: err is :%s, txId: %s",
			err, crossContractTx.TxId)
		return
	}
}

func (s *DockerScheduler) ReturnErrorResponse(chainId, txId string, errMsg string) {
	errTxResponse := s.constructErrorResponse(chainId, txId, errMsg)
	s.txResponseCh <- errTxResponse
}

func (s *DockerScheduler) constructErrorResponse(chainId, txId string, errMsg string) *protogo.TxResponse {
	return &protogo.TxResponse{
		TxId:    txId,
		Code:    protogo.ContractResultCode_FAIL,
		Result:  nil,
		Message: errMsg,
		ChainId: chainId,
	}
}

func (s *DockerScheduler) ReturnErrorCrossContractResponse(crossContractTx *protogo.TxRequest,
	errResponse *SDKProtogo.DMSMessage) {

	responseChId := crossContractChKey(crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
	responseCh := s.GetCrossContractResponseCh(crossContractTx.ChainId, responseChId)
	if responseCh == nil {
		s.logger.Warnf("scheduler fail to get response chan and abandon cross err response [%s]",
			errResponse.TxId)
		return
	}
	responseCh <- errResponse
}

func (s *DockerScheduler) RegisterTxElapsedTime(txRequest *protogo.TxRequest, startTime int64) {
	if s.TxRequestMgr[txRequest.TxId] != nil {
		s.logger.Debugf("duplicated tx, txid already exists: %s", txRequest.TxId)
		return
	}
	s.TxRequestMgr[txRequest.TxId] = NewTxElapsedTime(txRequest.TxId, startTime)
	return
}

func (s *DockerScheduler) AddTxSysCallElapsedTime(txId string, sysCallElapsedTime *SysCallElapsedTime) {
	if s.TxRequestMgr[txId] == nil {
		return
	}
	s.TxRequestMgr[txId].AddSysCallElapsedTime(sysCallElapsedTime)
	return
}

func (s *DockerScheduler) AddTxCallContractElapsedTime(txId string, sysCallElapsedTime *SysCallElapsedTime) {
	if s.TxRequestMgr[txId] == nil {
		return
	}
	s.TxRequestMgr[txId].AddCallContractElapsedTime(sysCallElapsedTime)
	return
}

func (s *DockerScheduler) RemoveTxElapsedTime(txId string) {
	delete(s.TxRequestMgr, txId)
	return
}

func (s *DockerScheduler) GetTxElapsedTime(txId string) *TxElapsedTime {
	return s.TxRequestMgr[txId]
}

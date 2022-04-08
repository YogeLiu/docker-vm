/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"

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
	logger *zap.SugaredLogger

	txReqCh          chan *protogo.TxRequest
	txResponseCh     chan *protogo.TxResponse
	getStateReqCh    chan *protogo.CDMMessage
	getByteCodeReqCh chan *protogo.CDMMessage
	responseChMap    sync.Map

	crossContractsCh chan *protogo.TxRequest
}

// NewDockerScheduler new docker scheduler
func NewDockerScheduler() *DockerScheduler {
	scheduler := &DockerScheduler{
		logger: logger.NewDockerLogger(logger.MODULE_SCHEDULER, config.DockerLogDir),

		txReqCh:          make(chan *protogo.TxRequest, ReqChanSize),
		txResponseCh:     make(chan *protogo.TxResponse, ResponseChanSize),
		getStateReqCh:    make(chan *protogo.CDMMessage, ReqChanSize*8),
		getByteCodeReqCh: make(chan *protogo.CDMMessage, ReqChanSize),
		crossContractsCh: make(chan *protogo.TxRequest, crossContractsChanSize),
		responseChMap:    sync.Map{},
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
func (s *DockerScheduler) RegisterResponseCh(responseId string, responseCh chan *protogo.CDMMessage) {
	s.responseChMap.Store(responseId, responseCh)
}

// GetResponseChByTxId get response chan by tx id
func (s *DockerScheduler) GetResponseChByTxId(txId string) chan *protogo.CDMMessage {

	responseCh, _ := s.responseChMap.LoadAndDelete(txId)
	return responseCh.(chan *protogo.CDMMessage)
}

// RegisterCrossContractResponseCh register cross contract response chan
func (s *DockerScheduler) RegisterCrossContractResponseCh(responseId string, responseCh chan *SDKProtogo.DMSMessage) {
	s.responseChMap.Store(responseId, responseCh)
}

// GetCrossContractResponseCh get cross contract response chan
func (s *DockerScheduler) GetCrossContractResponseCh(responseId string) chan *SDKProtogo.DMSMessage {
	responseCh, loaded := s.responseChMap.LoadAndDelete(responseId)
	if !loaded {
		return nil
	}
	return responseCh.(chan *SDKProtogo.DMSMessage)
}

func (s *DockerScheduler) ReturnErrorResponse(txId string, errMsg string) {
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

func (s *DockerScheduler) ReturnErrorCrossContractResponse(crossContractTx *protogo.TxRequest,
	errResponse *SDKProtogo.DMSMessage) {

	responseChId := crossContractChKey(crossContractTx.TxId, crossContractTx.TxContext.CurrentHeight)
	responseCh := s.GetCrossContractResponseCh(responseChId)
	if responseCh == nil {
		s.logger.Warnf("scheduler fail to get response chan and abandon cross err response [%s]",
			errResponse.TxId)
		return
	}
	responseCh <- errResponse
}

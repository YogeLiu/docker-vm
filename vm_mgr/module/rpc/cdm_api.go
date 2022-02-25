/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"errors"
	"io"
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"go.uber.org/zap"
)

type CDMApi struct {
	logger    *zap.SugaredLogger
	scheduler protocol.Scheduler
	stream    protogo.CDMRpc_CDMCommunicateServer
	stop      chan struct{}
	wg        *sync.WaitGroup
}

func NewCDMApi(scheduler protocol.Scheduler) *CDMApi {
	return &CDMApi{
		scheduler: scheduler,
		logger:    logger.NewDockerLogger(logger.MODULE_CDM_API, config.DockerLogDir),
		stream:    nil,
		stop:      nil,
		wg:        new(sync.WaitGroup),
	}
}

// CDMCommunicate docker manager stream function
func (cdm *CDMApi) CDMCommunicate(stream protogo.CDMRpc_CDMCommunicateServer) error {

	// init cdm api stop chan and stream
	cdm.stop = make(chan struct{})
	cdm.stream = stream

	cdm.wg.Add(2)

	go cdm.recvMsgRoutine()

	go cdm.sendMsgRoutine()

	cdm.wg.Wait()

	cdm.logger.Infof("cdm connection end")
	return nil
}

// recv three types of msg
// type1: txRequest
// type2: get_state response
// type3: get_bytecode response or path
func (cdm *CDMApi) recvMsgRoutine() {

	cdm.logger.Infof("start receiving cdm message ")

	var err error

	for {

		revMsg, revErr := cdm.stream.Recv()

		if revErr == io.EOF {
			cdm.logger.Errorf("cdm server receive eof and exit receive goroutine")
			close(cdm.stop)
			cdm.wg.Done()
			return
		}

		if revErr != nil {
			cdm.logger.Errorf("cdm server receive err and exit receive goroutine %s", revErr)
			close(cdm.stop)
			cdm.wg.Done()
			return
		}

		cdm.logger.Debugf("cdm server recv msg [%s]", revMsg)

		switch revMsg.Type {
		case protogo.CDMType_CDM_TYPE_TX_REQUEST:
			err = cdm.handleTxRequest(revMsg)
		case protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE:
			err = cdm.handleGetStateResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE:
			err = cdm.handleGetByteCodeResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR_RESPONSE:
			err = cdm.handleCreateKvIteratorResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR_RESPONSE:
			err = cdm.handleConsumeKvIteratorResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_TER_RESPONSE:
			err = cdm.handleCreateKeyHistoryKvIterResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER_RESPONSE:
			err = cdm.handleConsumeKeyHistoryKvIterResponse(revMsg)
		case protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS_RESPONSE:
			err = cdm.handleGetSenderAddrResponse(revMsg)
		default:
			err = errors.New("unknown message type")
		}

		if err != nil {
			cdm.logger.Errorf("fail to recv msg in handler: [%s]", err)
		}

	}
}

func (cdm *CDMApi) sendMsgRoutine() {

	cdm.logger.Infof("start sending cdm message")

	var err error

	for {
		select {
		case txResponseMsg := <-cdm.scheduler.GetTxResponseCh():
			cdm.logger.Debugf("[%s] retrieve response from chan and send to chain", txResponseMsg.TxId)
			cdmMsg := cdm.constructCDMMessage(txResponseMsg)
			err = cdm.sendMessage(cdmMsg)
		case getStateReqMsg := <-cdm.scheduler.GetGetStateReqCh():
			err = cdm.sendMessage(getStateReqMsg)
		case getByteCodeReqMsg := <-cdm.scheduler.GetByteCodeReqCh():
			err = cdm.sendMessage(getByteCodeReqMsg)
		case <-cdm.stop:
			cdm.wg.Done()
			cdm.logger.Debugf("stop sending cdm message ")
			return
		}

		if err != nil {
			cdm.logger.Errorf("fail to send msg: [%s]", err)
		}

	}

}

func (cdm *CDMApi) constructCDMMessage(txResponseMsg *protogo.TxResponse) *protogo.CDMMessage {

	//payload, _ := txResponseMsg.Marshal()

	cdmMsg := &protogo.CDMMessage{
		TxId:       txResponseMsg.TxId,
		Type:       protogo.CDMType_CDM_TYPE_TX_RESPONSE,
		TxResponse: txResponseMsg,
	}

	return cdmMsg
}

func (cdm *CDMApi) sendMessage(msg *protogo.CDMMessage) error {
	cdm.logger.Debugf("cdm send message [%s]", msg)
	return cdm.stream.Send(msg)
}

// unmarshal cdm message to send txRequest to txReqCh
func (cdm *CDMApi) handleTxRequest(cdmMessage *protogo.CDMMessage) error {

	txRequest := cdmMessage.TxRequest
	cdm.logger.Debugf("[%s] cdm server receive tx from chain", txRequest.TxId)

	cdm.scheduler.GetTxReqCh() <- txRequest

	return nil
}

func (cdm *CDMApi) handleGetStateResponse(cdmMessage *protogo.CDMMessage) error {

	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleGetByteCodeResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleCreateKvIteratorResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleConsumeKvIteratorResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleCreateKeyHistoryKvIterResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleConsumeKeyHistoryKvIterResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleGetSenderAddrResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

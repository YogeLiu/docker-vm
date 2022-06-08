/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"sync"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"go.uber.org/zap"
)

type CDMApi struct {
	logger    *zap.SugaredLogger
	scheduler protocol.Scheduler
}

type CommunicateConn struct {
	Stream          protogo.CDMRpc_CDMCommunicateServer
	StopSendChan    chan struct{}
	StopReceiveChan chan struct{}
	Wg              *sync.WaitGroup
}

func NewCDMApi(scheduler protocol.Scheduler) *CDMApi {
	return &CDMApi{
		scheduler: scheduler,
		logger:    logger.NewDockerLogger(logger.MODULE_CDM_API, config.DockerLogDir),
	}
}

// CDMCommunicate docker manager stream function
func (cdm *CDMApi) CDMCommunicate(stream protogo.CDMRpc_CDMCommunicateServer) error {

	communicateConn := &CommunicateConn{
		Stream:          stream,
		StopSendChan:    make(chan struct{}),
		StopReceiveChan: make(chan struct{}),
		Wg:              new(sync.WaitGroup),
	}

	communicateConn.Wg.Add(2)

	go cdm.receiveMsgRoutine(communicateConn)

	go cdm.sendMsgRoutine(communicateConn)

	communicateConn.Wg.Wait()

	cdm.logger.Infof("cdm connection end")
	return nil
}

// recv three types of msg
// type1: txRequest
// type2: get_state response
// type3: get_bytecode response or path
func (cdm *CDMApi) receiveMsgRoutine(c *CommunicateConn) {

	cdm.logger.Infof("start receiving cdm message ")

	var err error

	for {
		select {
		case <-c.StopReceiveChan:
			cdm.logger.Debugf("close cdm server receive goroutine")
			c.Wg.Done()
			return
		default:
			receivedMsg, revErr := c.Stream.Recv()

			if revErr != nil {
				cdm.logger.Errorf("cdm server receive err and exit receive goroutine %s", revErr)
				close(c.StopSendChan)
				c.Wg.Done()
				return
			}

			cdm.logger.Debugf("cdm server recv msg %s[%s] , type: [%s]", receivedMsg.ChainId, receivedMsg.TxId, receivedMsg.Type)

			switch receivedMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_REQUEST:
				err = cdm.handleTxRequest(receivedMsg)
			case protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE:
				err = cdm.handleGetStateResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_GET_BATCH_STATE_RESPONSE:
				err = cdm.handleGetBatchStateResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE:
				err = cdm.handleGetByteCodeResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR_RESPONSE:
				err = cdm.handleCreateKvIteratorResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR_RESPONSE:
				err = cdm.handleConsumeKvIteratorResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_TER_RESPONSE:
				err = cdm.handleCreateKeyHistoryKvIterResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER_RESPONSE:
				err = cdm.handleConsumeKeyHistoryKvIterResponse(receivedMsg)
			case protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS_RESPONSE:
				err = cdm.handleGetSenderAddrResponse(receivedMsg)
			default:
				errMsg := fmt.Sprintf("unknown message type, received msg: [%s]", receivedMsg)
				err = errors.New(errMsg)
			}

			if err != nil {
				cdm.logger.Errorf("fail to recv msg in handler: [%s]", err)
			}
		}
	}
}

func (cdm *CDMApi) sendMsgRoutine(c *CommunicateConn) {

	cdm.logger.Infof("start sending cdm message, goid: %d", cdm.getGoID())

	var err error

	for {
		select {
		case txResponseMsg := <-cdm.scheduler.GetTxResponseCh():
			cdm.logger.Debugf("[%s] send tx resp, chan len: [%d]", txResponseMsg.TxId,
				len(cdm.scheduler.GetTxResponseCh()))
			cdmMsg := cdm.constructCDMMessage(txResponseMsg)
			err = c.Stream.Send(cdmMsg)
		case getStateReqMsg := <-cdm.scheduler.GetGetStateReqCh():
			cdm.logger.Debugf("[%s] send syscall req, chan len: [%d]", getStateReqMsg.TxId,
				len(cdm.scheduler.GetGetStateReqCh()))
			err = c.Stream.Send(getStateReqMsg)
		case getByteCodeReqMsg := <-cdm.scheduler.GetByteCodeReqCh():
			err = c.Stream.Send(getByteCodeReqMsg)
		case <-c.StopSendChan:
			c.Wg.Done()
			cdm.logger.Debugf("stop cdm server send goroutine")
			return
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			cdm.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				close(c.StopReceiveChan)
				c.Wg.Done()
				return
			}
		}
	}
}

func (cdm *CDMApi) constructCDMMessage(txResponseMsg *protogo.TxResponse) *protogo.CDMMessage {
	cdmMsg := &protogo.CDMMessage{
		TxId:       txResponseMsg.TxId,
		Type:       protogo.CDMType_CDM_TYPE_TX_RESPONSE,
		TxResponse: txResponseMsg,
		ChainId:    txResponseMsg.ChainId,
	}
	return cdmMsg
}

// temporarily, just for debug
func (cdm *CDMApi) getGoID() uint64 {
	b := make([]byte, 64)
	b = b[:runtime.Stack(b, false)]
	b = bytes.TrimPrefix(b, []byte("goroutine "))
	b = b[:bytes.IndexByte(b, ' ')]
	n, _ := strconv.ParseUint(string(b), 10, 64)
	return n
}

// unmarshal cdm message to send txRequest to txReqCh
func (cdm *CDMApi) handleTxRequest(cdmMessage *protogo.CDMMessage) error {

	txRequest := cdmMessage.TxRequest
	cdm.logger.Debugf("[%s] cdm server receive tx from chain", txRequest.TxId)

	cdm.scheduler.GetTxReqCh() <- txRequest

	return nil
}

func (cdm *CDMApi) handleGetStateResponse(cdmMessage *protogo.CDMMessage) error {

	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleGetBatchStateResponse(cdmMessage *protogo.CDMMessage) error {

	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleGetByteCodeResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleCreateKvIteratorResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleConsumeKvIteratorResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleCreateKeyHistoryKvIterResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleConsumeKeyHistoryKvIterResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

func (cdm *CDMApi) handleGetSenderAddrResponse(cdmMessage *protogo.CDMMessage) error {
	responseCh := cdm.scheduler.GetResponseChByTxId(cdmMessage.ChainId, cdmMessage.TxId)

	responseCh <- cdmMessage

	return nil
}

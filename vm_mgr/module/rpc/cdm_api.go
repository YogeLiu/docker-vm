/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"bytes"
	"errors"
	"fmt"
	"io"
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
	logger      *zap.SugaredLogger
	scheduler   protocol.Scheduler
	stopSend    chan struct{}
	stopReceive chan struct{}
	wg          *sync.WaitGroup
}

func NewCDMApi(scheduler protocol.Scheduler) *CDMApi {
	return &CDMApi{
		scheduler:   scheduler,
		logger:      logger.NewDockerLogger(logger.MODULE_CDM_API, config.DockerLogDir),
		stopSend:    make(chan struct{}),
		stopReceive: make(chan struct{}),
		wg:          new(sync.WaitGroup),
	}
}

// CDMCommunicate docker manager stream function
func (cdm *CDMApi) CDMCommunicate(stream protogo.CDMRpc_CDMCommunicateServer) error {

	cdm.logger.Infof("new cdm connection start")

	cdm.wg.Add(2)

	go cdm.receiveMsgRoutine(stream)

	go cdm.sendMsgRoutine(stream)

	cdm.wg.Wait()

	cdm.logger.Infof("cdm connection end")
	return nil
}

// recv three types of msg
// type1: txRequest
// type2: get_state response
// type3: get_bytecode response or path
func (cdm *CDMApi) receiveMsgRoutine(stream protogo.CDMRpc_CDMCommunicateServer) {

	cdm.logger.Infof("start receiving cdm message ")

	var err error

	for {
		select {
		case <-cdm.stopReceive:
			cdm.logger.Debugf("close cdm server receive goroutine")
			cdm.wg.Done()
			return
		default:
			receivedMsg, revErr := stream.Recv()

			if revErr == io.EOF {
				cdm.logger.Errorf("cdm server receive eof and exit receive goroutine")
				close(cdm.stopSend)
				cdm.wg.Done()
				return
			}

			if revErr != nil {
				cdm.logger.Errorf("cdm server receive err and exit receive goroutine %s", revErr)
				close(cdm.stopSend)
				cdm.wg.Done()
				return
			}

			cdm.logger.Debugf("cdm server recv msg [%s]", receivedMsg)

			switch receivedMsg.Type {
			case protogo.CDMType_CDM_TYPE_TX_REQUEST:
				err = cdm.handleTxRequest(receivedMsg)
			case protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE:
				err = cdm.handleGetStateResponse(receivedMsg)
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

func (cdm *CDMApi) sendMsgRoutine(stream protogo.CDMRpc_CDMCommunicateServer) {

	cdm.logger.Infof("start sending cdm message, goid: %d", cdm.getGoID())

	var err error

	for {
		select {
		case txResponseMsg := <-cdm.scheduler.GetTxResponseCh():
			//cdm.logger.Infof("[%s] send tx resp, chan len: [%d]", txResponseMsg.TxId,
			//	len(cdm.scheduler.GetTxResponseCh()))
			cdmMsg := cdm.constructCDMMessage(txResponseMsg)
			err = cdm.sendMessage(cdmMsg, stream)
		case getStateReqMsg := <-cdm.scheduler.GetGetStateReqCh():
			//cdm.logger.Infof("[%s] send syscall req, chan len: [%d]", getStateReqMsg.TxId,
			//	len(cdm.scheduler.GetGetStateReqCh()))
			err = cdm.sendMessage(getStateReqMsg, stream)
		case getByteCodeReqMsg := <-cdm.scheduler.GetByteCodeReqCh():
			err = cdm.sendMessage(getByteCodeReqMsg, stream)
		case <-cdm.stopSend:
			cdm.wg.Done()
			cdm.logger.Debugf("stop cdm server send goroutine")
			return
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			cdm.logger.Errorf("fail to send msg: err: %s, err massage: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				close(cdm.stopReceive)
				cdm.wg.Done()
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
	}
	return cdmMsg
}

func (cdm *CDMApi) sendMessage(msg *protogo.CDMMessage, stream protogo.CDMRpc_CDMCommunicateServer) error {
	cdm.logger.Debugf("cdm send message [%s]", msg)
	return stream.Send(msg)
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

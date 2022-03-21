/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"bytes"
	"io"
	"runtime"
	"strconv"
	"sync"

	"go.uber.org/atomic"
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
	stream    protogo.CDMRpc_CDMCommunicateServer
	wg        *sync.WaitGroup
}

func NewCDMApi(scheduler protocol.Scheduler) *CDMApi {
	return &CDMApi{
		scheduler: scheduler,
		logger:    logger.NewDockerLogger(logger.MODULE_CDM_API, config.DockerLogDir),
		stream:    nil,
		wg:        new(sync.WaitGroup),
	}
}

// CDMCommunicate docker manager stream function
func (cdm *CDMApi) CDMCommunicate(stream protogo.CDMRpc_CDMCommunicateServer) error {

	cdm.stream = stream

	cdm.wg.Add(1)

	go cdm.receiveMsgRoutine()

	cdm.wg.Wait()

	cdm.logger.Infof("cdm connection end")
	return nil
}

// recv three types of msg
// type1: txRequest
// type2: get_state response
// type3: get_bytecode response or path
func (cdm *CDMApi) receiveMsgRoutine() {

	cdm.logger.Infof("start receiving cdm message ")

	var cnt atomic.Int64
	cnt.Store(0)

	for {
		recvMsg, err := cdm.stream.Recv()
		cdm.logger.Infof("msg size: ", recvMsg.Size())

		if err == io.EOF {
			cdm.logger.Errorf("cdm server receive eof and exit receive goroutine")
			cdm.wg.Done()
			return
		}

		if err != nil {
			cdm.logger.Errorf("cdm server receive err and exit receive goroutine %s", err)
			cdm.wg.Done()
			return
		}

		cdm.logger.Debugf("cdm server recv msg index [%d]", cnt.Add(1))

	}

}

func (cdm *CDMApi) sendMsgRoutine() {

	cdm.logger.Infof("start sending cdm message, goid: %d", cdm.getGoID())

	var err error

	for {
		select {
		case txResponseMsg := <-cdm.scheduler.GetTxResponseCh():
			cdm.logger.Debugf("[%s] send tx resp, chan len: [%d]", txResponseMsg.TxId,
				len(cdm.scheduler.GetTxResponseCh()))
			cdmMsg := cdm.constructCDMMessage(txResponseMsg)
			err = cdm.sendMessage(cdmMsg)
		case getStateReqMsg := <-cdm.scheduler.GetGetStateReqCh():
			cdm.logger.Debugf("[%s] send syscall req, chan len: [%d]", getStateReqMsg.TxId,
				len(cdm.scheduler.GetGetStateReqCh()))
			err = cdm.sendMessage(getStateReqMsg)
		case getByteCodeReqMsg := <-cdm.scheduler.GetByteCodeReqCh():
			err = cdm.sendMessage(getByteCodeReqMsg)
		}

		if err != nil {
			errStatus, _ := status.FromError(err)
			cdm.logger.Errorf("fail to send msg: err: %s, err message: %s, err code: %s", err,
				errStatus.Message(), errStatus.Code())
			if errStatus.Code() != codes.ResourceExhausted {
				cdm.wg.Done()
				return
			}
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
	cdm.logger.Debugf("cdm send message [%s]", msg.TxId)
	return cdm.stream.Send(msg)
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

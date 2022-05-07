/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"math"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"

	chainProtocol "chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"github.com/golang/protobuf/proto"
	"go.uber.org/zap"
)

type state string

const (
	created  state = "created"
	prepared state = "prepared"
	ready    state = "ready"

	// todo update here
	initContract    = "init_contract"
	invokeContract  = "invoke_contract"
	upgradeContract = "upgrade"

	maxTxDuration = math.MaxInt32
)

type ProcessInterface interface {
	triggerNewTx()

	returnTxResponse(txResponse *protogo.TxResponse)

	returnCrossResponse(crossResponse *SDKProtogo.DMSMessage)

	updateProcessState(state protogo.ProcessState)

	resetProcessTimer()

	killCrossProcess()
}

// ProcessHandler used to handle each contract's message
// to deal with each contract message
type ProcessHandler struct {
	processName   string
	state         state
	logger        *zap.SugaredLogger
	TxRequest     *protogo.TxRequest
	stream        SDKProtogo.DMSRpc_DMSCommunicateServer
	scheduler     protocol.Scheduler
	process       ProcessInterface
	txExpireTimer *time.Timer
}

func NewProcessHandler(txRequest *protogo.TxRequest, scheduler protocol.Scheduler,
	processName string, process ProcessInterface) *ProcessHandler {
	handler := &ProcessHandler{
		processName:   processName,
		logger:        logger.NewDockerLogger(logger.MODULE_DMS_HANDLER, config.DockerLogDir),
		TxRequest:     txRequest,
		state:         created,
		scheduler:     scheduler,
		process:       process,
		txExpireTimer: time.NewTimer(maxTxDuration * time.Second), //initial tx timer, never triggered
	}

	return handler
}

func (h *ProcessHandler) SetStream(stream SDKProtogo.DMSRpc_DMSCommunicateServer) {
	h.stream = stream
}

func (h *ProcessHandler) sendMessage(msg *SDKProtogo.DMSMessage) error {
	h.logger.Debugf("send message [%s], type: [%s], process [%s]", msg.TxId, msg.Type, h.processName)
	return h.stream.Send(msg)
}

// HandleMessage handle incoming message from contract
func (h *ProcessHandler) HandleMessage(msg *SDKProtogo.DMSMessage) error {
	h.logger.Debugf("process [%s] handle msg [%s]\n in state [%s]", h.processName, msg, h.state)

	switch h.state {
	case created:
		return h.handleCreated(msg)
	case prepared:
		return h.handlePrepare(msg)
	case ready:
		return h.handleReady(msg)
	}
	return nil
}

// ---------------------- prepare stage ---------------------

func (h *ProcessHandler) handleCreated(registerMsg *SDKProtogo.DMSMessage) error {
	if registerMsg.Type !=
		SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTER {
		return fmt.Errorf("handler cannot handle message (%s) while in state: %s",
			registerMsg, h.state)
	}

	registeredMsg := &SDKProtogo.DMSMessage{
		Type:    SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTERED,
		Payload: nil,
	}

	if err := h.sendMessage(registeredMsg); err != nil {
		h.logger.Errorf("fail to send message : [%v]", registeredMsg)
		return err
	}
	h.state = prepared

	return nil
}

// handlePrepare when sandbox send fist ready to server
// handler update state to ready
// and update process state
func (h *ProcessHandler) handlePrepare(readyMsg *SDKProtogo.DMSMessage) error {
	if readyMsg.Type != SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_READY {
		return fmt.Errorf("handler cannot handle message (%s) while in state: %s",
			readyMsg, h.state)
	}
	h.state = ready

	// cross contract call
	//if h.TxRequest.TxContext.CurrentHeight > 0 {
	//	return h.HandleContract()
	//}

	h.process.triggerNewTx()
	return nil
}

func (h *ProcessHandler) handleReady(readyMsg *SDKProtogo.DMSMessage) error {

	switch readyMsg.Type {
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_STATE_REQUEST:
		return h.handleGetState(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CALL_CONTRACT_REQUEST:
		return h.handleCallContract(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_COMPLETED:
		return h.handleCompleted(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CREATE_KV_ITERATOR_REQUEST:
		return h.handleCreateKvIterator(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CONSUME_KV_ITERATOR_REQUEST:
		return h.handleConsumeKvIterator(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CREATE_KEY_HISTORY_ITER_REQUEST:
		return h.handleCreateKeyHistoryIter(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CONSUME_KEY_HISTORY_ITER_REQUEST:
		return h.handleConsumeKeyHistoryIter(readyMsg)
	case SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_SENDER_ADDRESS_REQUEST:
		return h.handleGetSenderAddr(readyMsg)

	default:
		return fmt.Errorf("handler cannot handle ready message (%s) while in state: %s",
			readyMsg.Type, h.state)
	}

}

// HandleContract handle init, invoke, upgrade contract
func (h *ProcessHandler) HandleContract() error {

	switch h.TxRequest.Method {
	case initContract:
		return h.sendInit()
	case upgradeContract:
		return h.sendInit()
	case invokeContract:
		return h.sendInvoke()
	default:
		return fmt.Errorf("invalid method: %s", h.TxRequest.Method)
	}
}

// todo change init can be called only once
func (h *ProcessHandler) sendInit() error {

	input := &SDKProtogo.Input{Args: h.TxRequest.Parameters}
	inputPayload, _ := proto.Marshal(input)

	initMsg := &SDKProtogo.DMSMessage{
		TxId:          h.TxRequest.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_INIT,
		CurrentHeight: 0,
		Payload:       inputPayload,
	}

	return h.sendMessage(initMsg)
}

func (h *ProcessHandler) sendInvoke() error {

	input := &SDKProtogo.Input{Args: h.TxRequest.Parameters}

	inputPayload, _ := proto.Marshal(input)
	invokeMsg := &SDKProtogo.DMSMessage{
		TxId:          h.TxRequest.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_INVOKE,
		CurrentHeight: h.TxRequest.TxContext.CurrentHeight,
		Payload:       inputPayload,
	}

	return h.sendMessage(invokeMsg)
}

func (h *ProcessHandler) handleGetState(getStateMsg *SDKProtogo.DMSMessage) error {

	// get data from chain maker
	key := getStateMsg.Payload

	getStateReqMsg := &protogo.CDMMessage{
		TxId:    getStateMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_GET_STATE,
		Payload: key,
		ChainId: h.TxRequest.ChainId,
	}
	getStateResponseCh := make(chan *protogo.CDMMessage)

	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, getStateResponseCh)

	// wait to get state response
	h.scheduler.GetGetStateReqCh() <- getStateReqMsg

	getStateResponse := <-getStateResponseCh

	responseMsg := &SDKProtogo.DMSMessage{
		TxId:          getStateMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_STATE_RESPONSE,
		CurrentHeight: getStateMsg.CurrentHeight,
		ResultCode:    getStateResponse.ResultCode,
		Payload:       getStateResponse.Payload,
		Message:       getStateResponse.Message,
	}

	return h.sendMessage(responseMsg)
}

func constructCallContractErrorResponse(errMsg string, txId string, currentHeight uint32) *SDKProtogo.DMSMessage {

	contractErrorResponse := &SDKProtogo.ContractResponse{
		Response: &SDKProtogo.Response{
			Status:  1,
			Message: errMsg,
			Payload: nil,
		},
		WriteMap: nil,
		ReadMap:  nil,
		Events:   nil,
	}

	errResponsePayload, _ := proto.Marshal(contractErrorResponse)

	// construct cross contract response
	crossContractErrorResponse := &SDKProtogo.DMSMessage{
		TxId:          txId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CALL_CONTRACT_RESPONSE,
		CurrentHeight: currentHeight,
		Payload:       errResponsePayload,
	}

	return crossContractErrorResponse
}

func (h *ProcessHandler) handleCallContract(callContractMsg *SDKProtogo.DMSMessage) error {

	// validate cross contract params
	var callContractRequest SDKProtogo.CallContractRequest
	err := proto.Unmarshal(callContractMsg.Payload, &callContractRequest)
	if err != nil {
		errorResponse := constructCallContractErrorResponse(err.Error(), callContractMsg.TxId,
			callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	contractName := callContractRequest.ContractName
	contractVersion := callContractRequest.ContractVersion

	//if contractName == h.process.getContractName() && contractVersion == h.process.getContractVersion() &&
	//	h.TxRequest.ChainId == h.process.getContractChainId() {
	//	h.logger.Errorf(utils.CalledSameContractError.Error())
	//	errorResponse := constructCallContractErrorResponse(utils.MissingContractNameError.Error(),
	//		callContractMsg.TxId, callContractMsg.CurrentHeight)
	//	return h.sendMessage(errorResponse)
	//}

	if len(contractName) == 0 {
		h.logger.Errorf(utils.MissingContractNameError.Error())
		errorResponse := constructCallContractErrorResponse(utils.MissingContractNameError.Error(),
			callContractMsg.TxId, callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	if len(contractVersion) == 0 {
		h.logger.Errorf(utils.MissingContractVersionError.Error())
		errorResponse := constructCallContractErrorResponse(utils.MissingContractVersionError.Error(),
			callContractMsg.TxId, callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	if callContractMsg.CurrentHeight >= chainProtocol.CallContractDepth {
		h.logger.Errorf(utils.ExceedMaxDepthError.Error())
		errorResponse := constructCallContractErrorResponse(utils.ExceedMaxDepthError.Error(),
			callContractMsg.TxId, callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	// construct new tx
	callContractTx := &protogo.TxRequest{
		ChainId:         h.TxRequest.ChainId,
		TxId:            callContractMsg.TxId,
		ContractName:    contractName,
		ContractVersion: contractVersion,
		Method:          invokeContract,
		Parameters:      callContractRequest.Args,
		TxContext: &protogo.TxContext{
			CurrentHeight:       callContractMsg.CurrentHeight + 1,
			OriginalProcessName: h.TxRequest.TxContext.OriginalProcessName, // todo: doesn't have original now
			WriteMap:            nil,
			ReadMap:             nil,
		},
	}

	// register response chan, key = txID + contract height
	responseChId := crossContractChKey(callContractTx.TxId, callContractTx.TxContext.CurrentHeight)
	responseCh := make(chan *SDKProtogo.DMSMessage)
	h.scheduler.RegisterCrossContractResponseCh(callContractTx.ChainId, responseChId, responseCh)

	// pass msg to docker manager
	h.scheduler.GetCrossContractReqCh() <- callContractTx

	// wait docker manager response
	calledContractResponse := <-responseCh

	// check response has error or not
	var contractResponse SDKProtogo.ContractResponse
	err = proto.Unmarshal(calledContractResponse.Payload, &contractResponse)

	if err != nil {
		errorResponse := constructCallContractErrorResponse(err.Error(), callContractMsg.TxId,
			callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	if contractResponse.Response.Status != 200 {
		errorResponse := constructCallContractErrorResponse(contractResponse.Response.Message, callContractMsg.TxId,
			callContractMsg.CurrentHeight)
		return h.sendMessage(errorResponse)
	}

	// construct cross contract response
	crossContractResponse := &SDKProtogo.DMSMessage{
		TxId:          callContractMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CALL_CONTRACT_RESPONSE,
		CurrentHeight: callContractMsg.CurrentHeight,
		Payload:       calledContractResponse.Payload,
	}

	// give back response, could be normal response or error response
	return h.sendMessage(crossContractResponse)
}

func (h *ProcessHandler) handleCompleted(completedMsg *SDKProtogo.DMSMessage) error {

	// check current height,
	// if current height > 0, which is cross contract, just return result to previous contract
	if h.TxRequest.TxContext.CurrentHeight > 0 {
		h.process.returnCrossResponse(completedMsg)
		//h.logger.Debugf("process [%s] handle cross contract completed message [%s]", h.processName, completedMsg.TxId)
		//
		//responseChId := crossContractChKey(h.TxRequest.TxId, h.TxRequest.TxContext.CurrentHeight)
		//responseCh := h.scheduler.GetCrossContractResponseCh(h.TxRequest.ChainId, responseChId)
		//if responseCh == nil {
		//	h.logger.Warnf("process [%s] fail to get response chan and abandon cross response [%s]",
		//		h.processName, h.TxRequest.TxId)
		//	return nil
		//}
		//responseCh <- completedMsg

		//h.process.updateProcessState(protogo.ProcessState_PROCESS_STATE_CROSS_FINISHED)
		//h.process.killCrossProcess()

		return nil
	}

	var contractResponse SDKProtogo.ContractResponse
	_ = proto.Unmarshal(completedMsg.Payload, &contractResponse)

	//merge write map
	txResponse := &protogo.TxResponse{
		TxId:    h.TxRequest.TxId,
		ChainId: h.TxRequest.ChainId,
	}

	if contractResponse.Response.Status == 200 {

		txResponse.Code = protogo.ContractResultCode_OK
		txResponse.Result = contractResponse.Response.Payload
		txResponse.Message = "Success"
		txResponse.WriteMap = contractResponse.WriteMap
		txResponse.ReadMap = contractResponse.ReadMap

		var events []*protogo.DockerContractEvent
		for _, event := range contractResponse.Events {
			events = append(events, &protogo.DockerContractEvent{
				Topic:           event.Topic,
				ContractName:    event.ContractName,
				ContractVersion: event.ContractVersion,
				Data:            event.Data,
			})
		}

		txResponse.Events = events

	} else {
		txResponse.Code = protogo.ContractResultCode_FAIL
		txResponse.Result = []byte(contractResponse.Response.Message)
		txResponse.Message = "Fail"
		txResponse.WriteMap = nil
		txResponse.ReadMap = nil
		txResponse.Events = nil
	}

	h.process.returnTxResponse(txResponse)
	return nil
}

func (h *ProcessHandler) handleCreateKvIterator(createKvIteratorMsg *SDKProtogo.DMSMessage) error {
	keyList := createKvIteratorMsg.Payload

	createKvIteratorReqMsg := &protogo.CDMMessage{
		TxId:    createKvIteratorMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR,
		Payload: keyList,
		ChainId: h.TxRequest.ChainId,
	}

	createKvIteratorResponseCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, createKvIteratorResponseCh)

	h.scheduler.GetGetStateReqCh() <- createKvIteratorReqMsg

	createKvIteratorResponse := <-createKvIteratorResponseCh

	responseMsg := &SDKProtogo.DMSMessage{
		TxId:          createKvIteratorMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CREATE_KV_ITERATOR_RESPONSE,
		CurrentHeight: createKvIteratorMsg.CurrentHeight,
		ResultCode:    createKvIteratorResponse.ResultCode,
		Payload:       createKvIteratorResponse.Payload,
		Message:       createKvIteratorResponse.Message,
	}

	return h.sendMessage(responseMsg)
}

func (h *ProcessHandler) handleConsumeKvIterator(consumeKvIteratorMsg *SDKProtogo.DMSMessage) error {
	KeyList := consumeKvIteratorMsg.Payload

	consumeKvIteratorReqMsg := &protogo.CDMMessage{
		TxId:    consumeKvIteratorMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR,
		Payload: KeyList,
		ChainId: h.TxRequest.ChainId,
	}

	consumeKvIteratorResponseCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, consumeKvIteratorResponseCh)

	h.scheduler.GetGetStateReqCh() <- consumeKvIteratorReqMsg
	consumeKvIteratorResponse := <-consumeKvIteratorResponseCh

	responseMsg := &SDKProtogo.DMSMessage{
		TxId:          consumeKvIteratorMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CONSUME_KV_ITERATOR_RESPONSE,
		CurrentHeight: consumeKvIteratorMsg.CurrentHeight,
		ResultCode:    consumeKvIteratorResponse.ResultCode,
		Payload:       consumeKvIteratorResponse.Payload,
		Message:       consumeKvIteratorResponse.Message,
	}

	return h.sendMessage(responseMsg)
}

func (h *ProcessHandler) handleCreateKeyHistoryIter(createKeyHistoryIterMsg *SDKProtogo.DMSMessage) error {
	keyList := createKeyHistoryIterMsg.Payload

	createKeyHistoryIterReqMsg := &protogo.CDMMessage{
		TxId:    createKeyHistoryIterMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER,
		Payload: keyList,
		ChainId: h.TxRequest.ChainId,
	}

	respCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, respCh)

	h.scheduler.GetGetStateReqCh() <- createKeyHistoryIterReqMsg

	resp := <-respCh

	respMsg := &SDKProtogo.DMSMessage{
		TxId:          createKeyHistoryIterMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CREATE_KEY_HISTORY_ITER_RESPONSE,
		CurrentHeight: createKeyHistoryIterMsg.CurrentHeight,
		ResultCode:    resp.ResultCode,
		Payload:       resp.Payload,
		Message:       resp.Message,
	}

	return h.sendMessage(respMsg)
}

func (h *ProcessHandler) handleConsumeKeyHistoryIter(consumeKeyHistoryIterMsg *SDKProtogo.DMSMessage) error {
	keyList := consumeKeyHistoryIterMsg.Payload

	consumeKeyHistoryIterReqMsg := &protogo.CDMMessage{
		TxId:    consumeKeyHistoryIterMsg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER,
		Payload: keyList,
		ChainId: h.TxRequest.ChainId,
	}

	respCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, respCh)

	h.scheduler.GetGetStateReqCh() <- consumeKeyHistoryIterReqMsg

	resp := <-respCh

	respMsg := &SDKProtogo.DMSMessage{
		TxId:          consumeKeyHistoryIterMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_CONSUME_KEY_HISTORY_ITER_RESPONSE,
		CurrentHeight: consumeKeyHistoryIterMsg.CurrentHeight,
		ResultCode:    resp.ResultCode,
		Payload:       resp.Payload,
		Message:       resp.Message,
	}

	return h.sendMessage(respMsg)
}

func (h *ProcessHandler) handleGetSenderAddr(msg *SDKProtogo.DMSMessage) error {
	getSenderAddrReqMsg := &protogo.CDMMessage{
		TxId:    msg.TxId,
		Type:    protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS,
		Payload: nil,
		ChainId: h.TxRequest.ChainId,
	}

	respCh := make(chan *protogo.CDMMessage)
	h.scheduler.RegisterResponseCh(h.TxRequest.ChainId, h.TxRequest.TxId, respCh)
	h.scheduler.GetGetStateReqCh() <- getSenderAddrReqMsg

	resp := <-respCh

	respMsg := &SDKProtogo.DMSMessage{
		TxId:          getSenderAddrReqMsg.TxId,
		Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_SENDER_ADDRESS_RESPONSE,
		CurrentHeight: msg.CurrentHeight,
		ResultCode:    resp.ResultCode,
		Payload:       resp.Payload,
		Message:       resp.Message,
	}

	return h.sendMessage(respMsg)
}

func (h *ProcessHandler) resetState() {
	h.state = created
}

func (h *ProcessHandler) startTimer() {
	h.logger.Debugf("start tx timer: process [%s], tx [%s]", h.processName, h.TxRequest.TxId)
	if !h.txExpireTimer.Stop() && len(h.txExpireTimer.C) > 0 {
		<-h.txExpireTimer.C
	}
	h.txExpireTimer.Reset(time.Duration(config.SandBoxTimeout) * time.Second)
}

func (h *ProcessHandler) stopTimer() {
	h.logger.Debugf("stop tx timer: process [%s], tx [%s]", h.processName, h.TxRequest.TxId)
	if !h.txExpireTimer.Stop() && len(h.txExpireTimer.C) > 0 {
		<-h.txExpireTimer.C
	}
}

// cross contract chan key: txId#current_height
func crossContractChKey(txId string, currentHeight uint32) string {
	return txId + "#" + strconv.FormatUint(uint64(currentHeight), 10)
}

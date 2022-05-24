/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"errors"
	"fmt"
	"strings"
	"time"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/gas"
	"chainmaker.org/chainmaker/vm-docker-go/v2/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
)

const (
	mountContractDir = "contracts"
	msgIterIsNil     = "iterator is nil"
	timeout          = 10000 // tx execution timeout(milliseconds)
)

var (
	chainConfigContractName = syscontract.SystemContract_CHAIN_CONFIG.String()
	keyChainConfig          = chainConfigContractName
)

// RuntimeInstance docker-go runtime
type RuntimeInstance struct {
	rowIndex        int32                              // iterator index
	chainId         string                             // chain id
	logger          protocol.Logger                    //
	sendSysResponse func(msg *protogo.DockerVMMessage) //
	event           []*commonPb.ContractEvent          // tx event cache
	clientMgr       interfaces.ContractEngineClientMgr //
	runtimeService  interfaces.RuntimeService          //

	sandboxMsgCh chan *protogo.DockerVMMessage
	//sandboxMsgCh := make(chan *protogo.DockerVMMessage, 1)
	contractEngineMsgCh chan *protogo.DockerVMMessage
	//contractEngineMsgCh := make(chan *protogo.DockerVMMessage, 1)
}

func (r *RuntimeInstance) contractEngineMsgNotify(msg *protogo.DockerVMMessage) {
	r.contractEngineMsgCh <- msg
}

func (r *RuntimeInstance) sandboxMsgNotify(msg *protogo.DockerVMMessage, sendF func(msg *protogo.DockerVMMessage)) {
	r.sandboxMsgCh <- msg
	r.sendSysResponse = sendF
}

// Invoke process one tx in docker and return result
// nolint: gocyclo, revive
func (r *RuntimeInstance) Invoke(
	contract *commonPb.Contract,
	method string,
	byteCode []byte,
	parameters map[string][]byte,
	txSimContext protocol.TxSimContext,
	gasUsed uint64,
) (contractResult *commonPb.ContractResult, execOrderTxType protocol.ExecOrderTxType) {

	originalTxId := txSimContext.GetTx().Payload.TxId
	uniqueTxKey := r.clientMgr.GetUniqueTxKey(originalTxId)
	r.logger.Debugf("start handling tx [%s]", originalTxId)

	// contract response
	contractResult = &commonPb.ContractResult{
		// TODO
		Code:    uint32(1),
		Result:  nil,
		Message: "",
	}

	if !r.clientMgr.HasActiveConnections() {
		r.logger.Errorf("contract engine client stream not ready, waiting reconnect, tx id: %s", originalTxId)
		err := errors.New("contract engine client not connected")
		return r.errorResult(contractResult, err, err.Error())
	}

	specialTxType := protocol.ExecOrderTxTypeNormal

	var err error
	// init func gas used calc and check gas limit
	if gasUsed, err = gas.InitFuncGasUsed(gasUsed, parameters,
		gas.ContractParamCreatorOrgId,
		gas.ContractParamCreatorRole,
		gas.ContractParamCreatorPk,
		gas.ContractParamSenderOrgId,
		gas.ContractParamSenderRole,
		gas.ContractParamSenderPk,
		gas.ContractParamBlockHeight,
		gas.ContractParamTxId,
		gas.ContractParamTxTimeStamp,
	); err != nil {
		contractResult.GasUsed = gasUsed
		return r.errorResult(contractResult, err, err.Error())
	}

	//init contract gas used calc and check gas limit
	gasUsed, err = gas.ContractGasUsed(gasUsed, method, contract.Name, byteCode)
	if err != nil {
		contractResult.GasUsed = gasUsed
		return r.errorResult(contractResult, err, err.Error())
	}

	for key := range parameters {
		if strings.Contains(key, "CONTRACT") {
			delete(parameters, key)
		}
	}

	// construct DockerVMMessage
	txRequest := &protogo.TxRequest{
		ContractName:    contract.Name,
		ContractVersion: contract.Version,
		Method:          method,
		Parameters:      parameters,
		TxContext: &protogo.TxContext{
			WriteMap: nil,
			ReadMap:  nil,
		},
	}

	crossCtx := &protogo.CrossContext{
		CrossInfo:    txSimContext.GetCrossInfo(),
		CurrentDepth: uint32(txSimContext.GetDepth()),
	}

	dockerVMMsg := &protogo.DockerVMMessage{
		TxId:         uniqueTxKey,
		Type:         protogo.DockerVMType_TX_REQUEST,
		Request:      txRequest,
		CrossContext: crossCtx,
	}

	// register notify for sandbox msg
	err = r.runtimeService.RegisterSandboxMsgNotify(r.chainId, uniqueTxKey, r.sandboxMsgNotify)
	if err != nil {
		return r.errorResult(contractResult, err, err.Error())
	}

	// register receive notify
	err = r.clientMgr.PutTxRequestWithNotify(dockerVMMsg, r.chainId, r.contractEngineMsgNotify)
	if err != nil {
		return r.errorResult(contractResult, err, err.Error())
	}

	// send message to tx chan
	r.logger.Debugf("[%s] put tx in send chan with length [%d]", dockerVMMsg.TxId, r.clientMgr.GetTxSendChLen())

	defer func() {
		_ = r.runtimeService.DeleteSandboxMsgNotify(r.chainId, uniqueTxKey)
		_ = r.clientMgr.DeleteNotify(r.chainId, uniqueTxKey)
	}()

	timeoutC := time.After(timeout * time.Millisecond)

	// wait this chan
	for {
		select {
		case recvMsg := <-r.contractEngineMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				r.logger.Debugf("tx [%s] start get bytecode", uniqueTxKey)
				getByteCodeResponse := r.handleGetByteCodeRequest(uniqueTxKey, recvMsg, byteCode)
				r.clientMgr.PutByteCodeResp(getByteCodeResponse)
				r.logger.Debugf("tx [%s] finish get bytecode", uniqueTxKey)
			case protogo.DockerVMType_ERROR:
				r.logger.Debugf("handle tx [%s] failed, err: [%s]", originalTxId, recvMsg.Response.Message)
				return r.errorResult(
					contractResult,
					fmt.Errorf("tx timeout"),
					recvMsg.Response.Message,
				)
			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknown msg type"),
					"unknown msg type",
				)
			}

			// TODO: 超时时间自定义
		case <-timeoutC:
			r.logger.Debugf(
				"handle tx [%s] failed, fail to receive response in %d seconds and return timeout response",
				originalTxId,
				r.clientMgr.GetVMConfig().TxTimeLimit,
			)
			contractResult.GasUsed = gasUsed
			return r.errorResult(
				contractResult,
				fmt.Errorf("tx timeout"),
				"tx timeout",
			)

		case recvMsg := <-r.sandboxMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_STATE_REQUEST:
				r.logger.Debugf("tx [%s] start get state", uniqueTxKey)
				var getStateResponse *protogo.DockerVMMessage
				getStateResponse, gasUsed = r.handleGetStateRequest(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(getStateResponse)
				r.logger.Debugf("tx [%s] finish get state", uniqueTxKey)
			case protogo.DockerVMType_TX_RESPONSE:
				r.logger.Debugf(" finish handle tx [%s]", originalTxId)
				return r.handleTxResponse(recvMsg.TxId, recvMsg, txSimContext, gasUsed, specialTxType)

			case protogo.DockerVMType_CALL_CONTRACT_REQUEST:
				r.logger.Debugf("tx [%s] start call contract", uniqueTxKey)
				var callContractResponse *protogo.DockerVMMessage
				var crossTxType protocol.ExecOrderTxType
				callContractResponse, gasUsed, crossTxType = r.handlerCallContract(
					uniqueTxKey,
					recvMsg,
					txSimContext,
					gasUsed,
					contract.Name,
				)
				if crossTxType != protocol.ExecOrderTxTypeNormal {
					specialTxType = crossTxType
				}
				r.sendSysResponse(callContractResponse)

			case protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST:
				r.logger.Debugf("tx [%s] start create kv iterator", uniqueTxKey)
				var createKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKvIteratorResponse, gasUsed = r.handleCreateKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(createKvIteratorResponse)
				r.logger.Debugf("tx [%s] finish create kv iterator", uniqueTxKey)

			case protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST:
				r.logger.Debugf("tx [%s] start consume kv iterator", uniqueTxKey)
				var consumeKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKvIteratorResponse, gasUsed = r.handleConsumeKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(consumeKvIteratorResponse)
				r.logger.Debugf("tx [%s] finish consume kv iterator", uniqueTxKey)

			case protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST:
				r.logger.Debugf("tx [%s] start create key history iterator", uniqueTxKey)
				var createKeyHistoryIterResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKeyHistoryIterResp, gasUsed = r.handleCreateKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendSysResponse(createKeyHistoryIterResp)
				r.logger.Debugf("tx [%s] finish create key history iterator", uniqueTxKey)

			case protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST:
				r.logger.Debugf("tx [%s] start consume key history iterator", uniqueTxKey)
				var consumeKeyHistoryResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKeyHistoryResp, gasUsed = r.handleConsumeKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendSysResponse(consumeKeyHistoryResp)
				r.logger.Debugf("tx [%s] finish consume key history iteratorc", uniqueTxKey)

			case protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				r.logger.Debugf("tx [%s] start get sender address", uniqueTxKey)
				var getSenderAddressResp *protogo.DockerVMMessage
				getSenderAddressResp, gasUsed = r.handleGetSenderAddress(uniqueTxKey, txSimContext, gasUsed)
				r.sendSysResponse(getSenderAddressResp)
				r.logger.Debugf("tx [%s] finish get sender address", uniqueTxKey)

			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknow msg type"),
					"unknown msg type",
				)
			}
		}
	}
}

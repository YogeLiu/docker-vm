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
		TxId:    uniqueTxKey,
		Type:    protogo.DockerVMType_TX_REQUEST,
		Request: txRequest,
		// TODO:
		Response:       nil,
		SysCallMessage: nil,
		CrossContext:   crossCtx,
	}

	// register notify for sandbox msg
	// TODO: to method
	sandboxMsgCh := make(chan *protogo.DockerVMMessage, 1)
	sandboxMsgNotify := func(msg *protogo.DockerVMMessage, sendF func(msg *protogo.DockerVMMessage)) {
		sandboxMsgCh <- msg
		r.sendSysResponse = sendF
	}
	err = r.runtimeService.RegisterSandboxMsgNotify(r.chainId, uniqueTxKey, sandboxMsgNotify)
	if err != nil {
		return r.errorResult(contractResult, err, err.Error())
	}

	// register receive notify
	// TODO: to method
	contractEngineMsgCh := make(chan *protogo.DockerVMMessage, 1)
	contractEngineMsgNotify := func(msg *protogo.DockerVMMessage) {
		contractEngineMsgCh <- msg
	}
	err = r.clientMgr.PutTxRequestWithNotify(dockerVMMsg, r.chainId, contractEngineMsgNotify)
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
		case recvMsg := <-contractEngineMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				r.logger.Debugf("tx [%s] start get bytecode [%v]", uniqueTxKey, recvMsg)
				getByteCodeResponse := r.handleGetByteCodeRequest(uniqueTxKey, recvMsg, byteCode)
				r.clientMgr.PutByteCodeResp(getByteCodeResponse)
				r.logger.Debugf("tx [%s] finish get bytecode [%v]", uniqueTxKey, getByteCodeResponse)
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

		case recvMsg := <-sandboxMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_STATE_REQUEST:
				r.logger.Debugf("tx [%s] start get state [%v]", uniqueTxKey, recvMsg)
				// TODO: pass to error, gasUsed 在内部处理并返回
				getStateResponse, pass := r.handleGetStateRequest(uniqueTxKey, recvMsg, txSimContext)

				if pass {
					bytes, err := getStateResponse.SysCallMessage.Marshal()
					if err != nil {
						getStateResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
						getStateResponse.SysCallMessage.Payload = nil
						getStateResponse.SysCallMessage.Message = err.Error()
					}
					gasUsed, err = gas.GetStateGasUsed(gasUsed, bytes)
					if err != nil {
						getStateResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
						getStateResponse.SysCallMessage.Payload = nil
						getStateResponse.SysCallMessage.Message = err.Error()
					}
				}

				r.sendSysResponse(getStateResponse)
				r.logger.Debugf("tx [%s] finish get state [%v]", uniqueTxKey, getStateResponse)
			case protogo.DockerVMType_TX_RESPONSE:
				r.logger.Debugf(" finish handle tx [%s]", originalTxId)
				return r.handleTxResponse(recvMsg.TxId, recvMsg, txSimContext, gasUsed, specialTxType)

			case protogo.DockerVMType_CALL_CONTRACT_REQUEST:
				r.logger.Debugf("tx [%s] start call contract [%v]", uniqueTxKey, recvMsg)
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
				r.logger.Debugf("tx [%s] start create kv iterator [%v]", uniqueTxKey, recvMsg)
				var createKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKvIteratorResponse, gasUsed = r.handleCreateKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(createKvIteratorResponse)
				r.logger.Debugf("tx [%s] finish create kv iterator [%v]", uniqueTxKey, createKvIteratorResponse)

			case protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST:
				r.logger.Debugf("tx [%s] start consume kv iterator [%v]", uniqueTxKey, recvMsg)
				var consumeKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKvIteratorResponse, gasUsed = r.handleConsumeKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(consumeKvIteratorResponse)
				r.logger.Debugf("tx [%s] finish consume kv iterator [%v]", uniqueTxKey, consumeKvIteratorResponse)

			case protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST:
				r.logger.Debugf("tx [%s] start create key history iterator [%v]", uniqueTxKey, recvMsg)
				var createKeyHistoryIterResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKeyHistoryIterResp, gasUsed = r.handleCreateKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendSysResponse(createKeyHistoryIterResp)
				r.logger.Debugf("tx [%s] finish create key history iterator [%v]", uniqueTxKey, createKeyHistoryIterResp)

			case protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST:
				r.logger.Debugf("tx [%s] start consume key history iterator [%v]", uniqueTxKey, recvMsg)
				var consumeKeyHistoryResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKeyHistoryResp, gasUsed = r.handleConsumeKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendSysResponse(consumeKeyHistoryResp)
				r.logger.Debugf("tx [%s] finish consume key history iterator [%v]", uniqueTxKey, consumeKeyHistoryResp)

			case protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				r.logger.Debugf("tx [%s] start get sender address [%v]", uniqueTxKey, recvMsg)
				var getSenderAddressResp *protogo.DockerVMMessage
				getSenderAddressResp, gasUsed = r.handleGetSenderAddress(uniqueTxKey, txSimContext, gasUsed)
				r.sendSysResponse(getSenderAddressResp)
				r.logger.Debugf("tx [%s] finish get sender address [%v]", uniqueTxKey, getSenderAddressResp)

			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknow msg type"),
					"unknown msg type",
				)
			}

			// map[depth + count]channel
			// TODO: 监听区块调度超时的信号(交易积压问题)
			// a.
			// 	1. chain端清理当前高度的所有交易
			//	2. 发送信号通知engine清理当前高度交易
			//  3. 发送信号通知sandbox清理当前高度交易 (是否能做到丢掉当前正在处理的交易信息，重制整个sandbox，监听新的消息)
			// b. 重启docker

			// TODO: process name
			// contract_name#contract_version#1652356841624880612:0
		}
	}
}

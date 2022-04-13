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
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/gas"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/rpc"
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

type CDMClient interface {
	PutTxRequest(msg *protogo.DockerVMMessage)
	PutTxRequestWithNotify(msg *protogo.DockerVMMessage, notify func(msg *protogo.DockerVMMessage)) error
	GetTxSendChLen() int
	PutResponse(msg *protogo.DockerVMMessage)
	GetResponseSendChLen() int
	DeleteNotify(txId string) bool
	GetCMConfig() *config.DockerVMConfig
	GetUniqueTxKey(txId string) string
}

// RuntimeInstance docker-go runtime
type RuntimeInstance struct {
	rowIndex     int32  // iterator index
	ChainId      string // chain id
	Client       CDMClient
	Logger       protocol.Logger
	sendResponse func(msg *protogo.DockerVMMessage)
}

// Invoke process one tx in docker and return result
// nolint: gocyclo, revive
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte, txSimContext protocol.TxSimContext,
	gasUsed uint64) (contractResult *commonPb.ContractResult, execOrderTxType protocol.ExecOrderTxType) {
	originalTxId := txSimContext.GetTx().Payload.TxId
	uniqueTxKey := r.Client.GetUniqueTxKey(originalTxId)

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    uint32(1),
		Result:  nil,
		Message: "",
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

	// construct dockervm message
	txRequest := &protogo.TxRequest{
		ContractName: contract.Name,
		Method:       method,
		Parameters:   parameters,
		TxContext: &protogo.TxContext{
			WriteMap: nil,
			ReadMap:  nil,
		},
	}

	// ContractEngine 通过crossCtx判断是否是跨合约调用，以及跨合约调用的类型
	// TODO: 返回值应当携带该信息，外部跨合约调用直接返回，内部跨合约调用应当返回到指定sandbox中
	// 内部跨合约调用额外携带了 currentProcessName 和 parentProcessName
	crossCtx := &protogo.CrossContext{
		CurrentDepth: 0,
		CrossType:    protogo.CrossType_INTERNAL,
	}
	if txSimContext.GetDepth() > 0 {
		crossCtx.CurrentDepth = uint32(txSimContext.GetDepth())
		switch txSimContext.GetCallerRuntimeType() {
		case commonPb.RuntimeType_DOCKER_GO:
			crossCtx.CrossType = protogo.CrossType_INTERNAL
		case commonPb.RuntimeType_INVALID,
			commonPb.RuntimeType_NATIVE,
			commonPb.RuntimeType_WASMER,
			commonPb.RuntimeType_WXVM,
			commonPb.RuntimeType_GASM,
			commonPb.RuntimeType_EVM,
			commonPb.RuntimeType_DOCKER_JAVA:
			crossCtx.CrossType = protogo.CrossType_EXTERNAL
		default:
			contractResult.GasUsed = gasUsed
			err := errors.New("invalid caller runtime type")
			r.Logger.Debugf(err.Error())
			return r.errorResult(contractResult, err, err.Error())
		}
	}

	dockervmMsg := &protogo.DockerVMMessage{
		TxId:           uniqueTxKey,
		Type:           protogo.DockerVMType_TX_REQUEST,
		Request:        txRequest,
		Response:       nil,
		SysCallMessage: nil,
	}

	// register result chan
	contractEngineMsgCh := make(chan *protogo.DockerVMMessage, 1)
	notify := func(msg *protogo.DockerVMMessage) {
		contractEngineMsgCh <- msg
	}
	err = r.Client.PutTxRequestWithNotify(dockervmMsg, notify)
	if err != nil {
		return r.errorResult(contractResult, err, err.Error())
	}
	// send message to tx chan
	r.Logger.Debugf("[%s] put tx in send chan with length [%d]", dockervmMsg.TxId, r.Client.GetTxSendChLen())

	runtimeMsgCh := make(chan *protogo.DockerVMMessage, 1)
	responseNotify := func(msg *protogo.DockerVMMessage, sendF func(msg *protogo.DockerVMMessage)) {
		runtimeMsgCh <- msg
		r.sendResponse = sendF
	}
	err = rpc.GetRuntimeServiceInstance().RegisterNotifyForSandbox(uniqueTxKey, responseNotify)
	if err != nil {
		return r.errorResult(contractResult, err, err.Error())
	}

	timeoutC := time.After(timeout * time.Millisecond)

	r.Logger.Debugf("start tx [%s] in runtime", dockervmMsg.TxId)
	// wait this chan
	for {
		select {
		case recvMsg := <-contractEngineMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				r.Logger.Debugf("tx [%s] start get bytecode [%v]", uniqueTxKey, recvMsg)
				getByteCodeResponse := r.handleGetByteCodeRequest(uniqueTxKey, recvMsg, byteCode)
				r.Client.PutResponse(getByteCodeResponse)
				r.Logger.Debugf("tx [%s] finish get bytecode [%v]", uniqueTxKey, getByteCodeResponse)
			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknow type"),
					"fail to receive request",
				)
			}

		case <-timeoutC:
			deleted := r.Client.DeleteNotify(uniqueTxKey)
			if deleted {
				r.Logger.Errorf("[%s] fail to receive response in 10 seconds and return timeout response",
					uniqueTxKey)
				contractResult.GasUsed = gasUsed
				return r.errorResult(contractResult, fmt.Errorf("tx timeout"),
					"fail to receive response",
				)
			}

		case recvMsg := <-runtimeMsgCh:
			switch recvMsg.Type {
			case protogo.DockerVMType_GET_STATE_REQUEST:
				r.Logger.Debugf("tx [%s] start get state [%v]", uniqueTxKey, recvMsg)
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

				r.sendResponse(getStateResponse)
				r.Logger.Debugf("tx [%s] finish get state [%v]", uniqueTxKey, getStateResponse)
			case protogo.DockerVMType_TX_RESPONSE:
				r.Logger.Debugf("[%s] finish handle response [%v]", uniqueTxKey, contractResult)
				return r.handleTxResponse(recvMsg.TxId, recvMsg, txSimContext, gasUsed, specialTxType)

			case protogo.DockerVMType_CALL_CONTRACT_REQUEST:
				r.Logger.Debugf("tx [%s] start call contract [%v]", uniqueTxKey, recvMsg)
				var callContractResponse *protogo.DockerVMMessage
				callContractResponse, gasUsed = r.handlerCallContract(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendResponse(callContractResponse)

			case protogo.DockerVMType_CREATE_KV_ITERATOR_REQUEST:
				r.Logger.Debugf("tx [%s] start create kv iterator [%v]", uniqueTxKey, recvMsg)
				var createKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKvIteratorResponse, gasUsed = r.handleCreateKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendResponse(createKvIteratorResponse)
				r.Logger.Debugf("tx [%s] finish create kv iterator [%v]", uniqueTxKey, createKvIteratorResponse)

			case protogo.DockerVMType_CONSUME_KV_ITERATOR_REQUEST:
				r.Logger.Debugf("tx [%s] start consume kv iterator [%v]", uniqueTxKey, recvMsg)
				var consumeKvIteratorResponse *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKvIteratorResponse, gasUsed = r.handleConsumeKvIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendResponse(consumeKvIteratorResponse)
				r.Logger.Debugf("tx [%s] finish consume kv iterator [%v]", uniqueTxKey, consumeKvIteratorResponse)

			case protogo.DockerVMType_CREATE_KEY_HISTORY_ITER_REQUEST:
				r.Logger.Debugf("tx [%s] start create key history iterator [%v]", uniqueTxKey, recvMsg)
				var createKeyHistoryIterResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKeyHistoryIterResp, gasUsed = r.handleCreateKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendResponse(createKeyHistoryIterResp)
				r.Logger.Debugf("tx [%s] finish create key history iterator [%v]", uniqueTxKey, createKeyHistoryIterResp)

			case protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_REQUEST:
				r.Logger.Debugf("tx [%s] start consume key history iterator [%v]", uniqueTxKey, recvMsg)
				var consumeKeyHistoryResp *protogo.DockerVMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				consumeKeyHistoryResp, gasUsed = r.handleConsumeKeyHistoryIterator(uniqueTxKey, recvMsg, txSimContext, gasUsed)
				r.sendResponse(consumeKeyHistoryResp)
				r.Logger.Debugf("tx [%s] finish consume key history iterator [%v]", uniqueTxKey, consumeKeyHistoryResp)

			case protogo.DockerVMType_GET_SENDER_ADDRESS_REQUEST:
				r.Logger.Debugf("tx [%s] start get sender address [%v]", uniqueTxKey, recvMsg)
				var getSenderAddressResp *protogo.DockerVMMessage
				getSenderAddressResp, gasUsed = r.handleGetSenderAddress(uniqueTxKey, txSimContext, gasUsed)
				r.sendResponse(getSenderAddressResp)
				r.Logger.Debugf("tx [%s] finish get sender address [%v]", uniqueTxKey, getSenderAddressResp)

			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknow type"),
					"fail to receive request",
				)
			}
		}
	}
}

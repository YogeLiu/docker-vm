/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coocood/freecache"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/gas"
	"chainmaker.org/chainmaker/vm-engine/v2/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/utils"
)

const (
	mountContractDir = "contract-bins"
	msgIterIsNil     = "iterator is nil"
)

var dockerVMMsgPool = sync.Pool{
	New: func() interface{} {
		return &protogo.DockerVMMessage{
			Request: &protogo.TxRequest{
				TxContext: &protogo.TxContext{
					WriteMap: nil,
					ReadMap:  nil,
				},
			},
			CrossContext:  &protogo.CrossContext{},
			StepDurations: make([]*protogo.StepDuration, 0, 4),
		}
	},
}

// RuntimeInstance docker-go runtime
type RuntimeInstance struct {
	rowIndex        int32                              // iterator index
	chainId         string                             // chain id
	logger          protocol.Logger                    //
	sendSysResponse func(msg *protogo.DockerVMMessage) //
	event           []*commonPb.ContractEvent          // tx event cache
	clientMgr       interfaces.ContractEngineClientMgr //
	runtimeService  interfaces.RuntimeService          //

	sandboxMsgCh        chan *protogo.DockerVMMessage
	contractEngineMsgCh chan *protogo.DockerVMMessage
	DockerManager       *InstancesManager
	txDuration          *utils.TxDuration
	DefaultGasCache     *freecache.Cache
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
	if gasUsed, err = gas.InitFuncGasUsed(gasUsed, r.getChainConfigDefaultGas(txSimContext)); err != nil {
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

	dockerVMMsg, _ := dockerVMMsgPool.Get().(*protogo.DockerVMMessage)
	dockerVMMsg.ChainId = r.chainId
	dockerVMMsg.TxId = uniqueTxKey
	dockerVMMsg.Type = protogo.DockerVMType_TX_REQUEST
	dockerVMMsg.Request.ChainId = r.chainId
	dockerVMMsg.Request.ContractName = contract.Name
	dockerVMMsg.Request.ContractVersion = contract.Version
	dockerVMMsg.Request.Method = method
	dockerVMMsg.Request.Parameters = parameters
	dockerVMMsg.CrossContext.CrossInfo = txSimContext.GetCrossInfo()
	dockerVMMsg.CrossContext.CurrentDepth = uint32(txSimContext.GetDepth())
	dockerVMMsg.StepDurations = dockerVMMsg.StepDurations[:0]
	defer func() {
		dockerVMMsgPool.Put(dockerVMMsg)
		for _, dur := range dockerVMMsg.StepDurations {
			utils.TxStepPool.Put(dur)
		}
	}()

	utils.EnterNextStep(dockerVMMsg, protogo.StepType_RUNTIME_PREPARE_TX_REQUEST, "")

	// init time statistics
	startTime := time.Now()
	r.txDuration = utils.NewTxDuration(originalTxId, uniqueTxKey, startTime.UnixNano())

	// add time statistics
	fingerprint := txSimContext.GetBlockFingerprint()
	// if it is a query tx, fingerprint is "", not record this tx
	if fingerprint != "" {
		r.DockerManager.BlockDurationMgr.AddTx(fingerprint, r.txDuration)
	}

	defer func() {
		r.txDuration.TotalDuration = time.Since(startTime).Nanoseconds()
		r.DockerManager.BlockDurationMgr.FinishTx(fingerprint, r.txDuration)
		//r.logger.Debugf(r.txDuration.PrintSysCallList())
	}()

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

	timeoutC := time.After(time.Duration(config.VMConfig.TxTimeout) * time.Second)

	// wait this chan
	for {
		select {
		case recvMsg := <-r.contractEngineMsgCh:

			r.txDuration.StartSysCall(recvMsg.Type)

			switch recvMsg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				r.logger.Debugf("tx [%s] start get bytecode", uniqueTxKey)
				getByteCodeResponse := r.handleGetByteCodeRequest(uniqueTxKey, txSimContext, recvMsg, byteCode)
				r.clientMgr.PutByteCodeResp(getByteCodeResponse)
				r.logger.Debugf("tx [%s] finish get bytecode", uniqueTxKey)
				if err = r.txDuration.EndSysCall(recvMsg); err != nil {
					r.logger.Warnf("failed to end syscall, %v", err)
				}

			case protogo.DockerVMType_ERROR:
				r.logger.Warnf("handle tx [%s] failed, err: [%s]", originalTxId, recvMsg.Response.Message)
				contractResult.GasUsed = gasUsed
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
			r.logger.Errorf(
				"handle tx [%s] failed, fail to receive response in %d secs and return timeout response, %s",
				originalTxId, config.VMConfig.TxTimeout, utils.PrintTxSteps(dockerVMMsg))
			r.logger.Infof(r.txDuration.ToString())
			r.logger.InfoDynamic(func() string {
				return r.txDuration.PrintSysCallList()
			})
			contractResult.GasUsed = gasUsed
			return r.errorResult(
				contractResult,
				fmt.Errorf("tx timeout"),
				"tx timeout",
			)

		case recvMsg := <-r.sandboxMsgCh:

			r.txDuration.StartSysCall(recvMsg.Type)

			switch recvMsg.Type {
			case protogo.DockerVMType_GET_STATE_REQUEST:
				r.logger.Debugf("tx [%s] start get state", uniqueTxKey)
				var getStateResponse *protogo.DockerVMMessage
				getStateResponse, gasUsed = r.handleGetStateRequest(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(getStateResponse)
				r.logger.Debugf("tx [%s] finish get state", uniqueTxKey)

			case protogo.DockerVMType_GET_BATCH_STATE_REQUEST:
				r.logger.Debugf("tx [%s] start get batch state [%v]", uniqueTxKey)
				var getStateResponse *protogo.DockerVMMessage
				getStateResponse, gasUsed = r.handleGetBatchStateRequest(uniqueTxKey, recvMsg, txSimContext, gasUsed)

				r.sendSysResponse(getStateResponse)
				r.logger.Debugf("tx [%s] finish get batch state", uniqueTxKey)

			case protogo.DockerVMType_TX_RESPONSE:
				result, txType := r.handleTxResponse(originalTxId, recvMsg, txSimContext,
					gasUsed, specialTxType, contractResult)
				r.logger.Debugf("tx [%s] finish handle response", originalTxId)
				if err = r.txDuration.EndSysCall(recvMsg); err != nil {
					r.logger.Warnf("failed to end syscall, %v", err)
				}
				r.logger.Debugf("tx [%s] do some work after receive response", originalTxId)
				return result, txType

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
					contract.Address,
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
				r.logger.Debugf("tx [%s] finish consume key history iterator", uniqueTxKey)

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
			if err = r.txDuration.EndSysCall(recvMsg); err != nil {
				r.logger.Warnf("failed to end syscall, %v", err)
			}
		}
	}
}

func (r *RuntimeInstance) getChainConfigDefaultGas(txSimContext protocol.TxSimContext) uint64 {
	fingerprintBytes := []byte(txSimContext.GetBlockFingerprint())
	v, err := r.DefaultGasCache.Get(fingerprintBytes)
	if err == nil {
		return binary.BigEndian.Uint64(v)
	}
	chainConfig, err := txSimContext.GetBlockchainStore().GetLastChainConfig()
	if err != nil {
		r.logger.Debugf("get last chain config err [%v]", err.Error())
		return 0
	}

	var defaultGas uint64
	if chainConfig.AccountConfig != nil && chainConfig.AccountConfig.DefaultGas > 0 {
		defaultGas = chainConfig.AccountConfig.DefaultGas
	} else {
		r.logger.Debug("account config not set default gas value")
		defaultGas = 0
	}

	var buf = make([]byte, 8)
	binary.BigEndian.PutUint64(buf, defaultGas)
	if err = r.DefaultGasCache.Set(fingerprintBytes, buf, 64); err != nil {
		r.logger.Debugf("failed to put default gas cache for fingerprint: %s", txSimContext.GetBlockFingerprint())
	}
	return defaultGas
}

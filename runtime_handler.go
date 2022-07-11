package docker_go

import (
	"bytes"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	commonCrt "chainmaker.org/chainmaker/common/v2/cert"
	"chainmaker.org/chainmaker/common/v2/crypto"
	"chainmaker.org/chainmaker/common/v2/crypto/asym"
	bcx509 "chainmaker.org/chainmaker/common/v2/crypto/x509"
	"chainmaker.org/chainmaker/common/v2/evmutils"
	"chainmaker.org/chainmaker/pb-go/v2/accesscontrol"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	configPb "chainmaker.org/chainmaker/pb-go/v2/config"
	"chainmaker.org/chainmaker/pb-go/v2/store"
	"chainmaker.org/chainmaker/pb-go/v2/syscontract"
	vmPb "chainmaker.org/chainmaker/pb-go/v2/vm"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/gas"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
	"github.com/gogo/protobuf/proto"
)

func (r *RuntimeInstance) handleTxResponse(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64, txType protocol.ExecOrderTxType) (
	contractResult *commonPb.ContractResult, execOrderTxType protocol.ExecOrderTxType) {

	var err error
	txResponse := recvMsg.Response

	contractResult = new(commonPb.ContractResult)
	// tx fail, just return without merge read write map and events
	if txResponse.Code != 0 {
		contractResult.Code = 1
		contractResult.Result = txResponse.Result
		contractResult.Message = txResponse.Message
		contractResult.GasUsed = gasUsed

		return contractResult, protocol.ExecOrderTxTypeNormal
	}

	contractResult.Code = 0
	contractResult.Result = txResponse.Result
	contractResult.Message = txResponse.Message

	// merge read map to sim context
	if err = r.mergeSimContextReadMap(txSimContext, txResponse.GetReadMap()); err != nil {
		r.logger.Errorf("fail to merge tx[%s] sim context read map, %v", txId, err)
		return r.errorResult(contractResult, err, "fail to put in sim context")
	}
	r.logger.Debugf("merge tx[%s] sim context read map succeed", txId)

	// merge write map to sim context
	gasUsed, err = r.mergeSimContextWriteMap(txSimContext, txResponse.GetWriteMap(), gasUsed)
	if err != nil {
		r.logger.Errorf("fail to merge tx[%s] sim context write map, %v", txId, err)
		contractResult.GasUsed = gasUsed
		return r.errorResult(contractResult, err, "fail to put in sim context")
	}
	r.logger.Debugf("merge tx[%s] sim context write map succeed", txId)

	// merge events
	if len(txResponse.Events) > protocol.EventDataMaxCount-1 {
		err = fmt.Errorf("too many event data")
		return r.errorResult(contractResult, err, "fail to put event data")
	}

	for _, event := range txResponse.Events {
		contractEvent := &commonPb.ContractEvent{
			Topic:        event.Topic,
			TxId:         txId,
			ContractName: event.ContractName,
			EventData:    event.Data,
		}

		// emit event gas used calc and check gas limit
		gasUsed, err = gas.EmitEventGasUsed(gasUsed, contractEvent)
		if err != nil {
			contractResult.GasUsed = gasUsed
			return r.errorResult(contractResult, err, err.Error())
		}

		r.event = append(r.event, contractEvent)
	}

	contractResult.GasUsed = gasUsed
	contractResult.ContractEvent = r.event

	return contractResult, txType
}

func (r *RuntimeInstance) mergeSimContextReadMap(txSimContext protocol.TxSimContext,
	readMap map[string][]byte) error {

	for key, value := range readMap {
		var contractName string
		var contractKey string
		var contractField string
		keyList := strings.Split(key, "#")
		keyLen := len(keyList)
		if keyLen < 2 {
			return fmt.Errorf("%s's key list length == %d, needs to be >= 2", key, keyLen)
		}
		contractName = keyList[0]
		contractKey = keyList[1]
		if keyLen == 3 {
			contractField = keyList[2]
		}

		txSimContext.PutIntoReadSet(contractName, protocol.GetKeyStr(contractKey, contractField), value)
	}
	return nil
}

func (r *RuntimeInstance) handlerCallContract(
	txId string,
	recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext,
	gasUsed uint64,
	currentContractName string,
) (*protogo.DockerVMMessage, uint64, protocol.ExecOrderTxType) {

	response := r.newEmptyResponse(txId, protogo.DockerVMType_CALL_CONTRACT_RESPONSE)
	specialTxType := protocol.ExecOrderTxTypeNormal
	// validate cross contract params
	callContractPayload := recvMsg.SysCallMessage.Payload[config.KeyCallContractReq]
	var callContractReq protogo.CallContractRequest
	err := proto.Unmarshal(callContractPayload, &callContractReq)
	if err != nil {
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = err.Error()
		return response, gasUsed, specialTxType
	}

	contractName := callContractReq.ContractName
	if len(contractName) == 0 {
		errMsg := "missing contract name"
		r.logger.Error(errMsg)
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = errMsg
		return response, gasUsed, specialTxType
	}

	contractMethod := callContractReq.ContractMethod
	if len(contractMethod) == 0 {
		errMsg := "missing contract method"
		r.logger.Error(errMsg)
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = errMsg
		return response, gasUsed, specialTxType
	}

	if recvMsg.CrossContext.CurrentDepth > protocol.CallContractDepth {
		errMsg := "exceed max depth"
		r.logger.Error(errMsg)
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = errMsg
		return response, gasUsed, specialTxType
	}

	// construct new tx
	var result *commonPb.ContractResult
	var code commonPb.TxStatusCode
	var contract *commonPb.Contract
	contract, err = txSimContext.GetContractByName(contractName)
	if err != nil {
		errMsg := fmt.Sprintf(
			"[call contract] failed to get contract by [%s], err: %s",
			contractName,
			err.Error(),
		)
		r.logger.Error(errMsg)

		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = errMsg
		return response, gasUsed, specialTxType
	}

	result, specialTxType, code = txSimContext.CallContract(contract, contractMethod,
		nil, callContractReq.Args, gasUsed, txSimContext.GetTx().Payload.TxType)
	r.logger.Debugf("call contract result [%+v]", result)

	if code != commonPb.TxStatusCode_SUCCESS {
		errMsg := fmt.Sprintf("[call contract] execute error code: %s, msg: %s", code, result.Message)
		r.logger.Debugf("handle cross contract request failed, err: %s", errMsg)
		r.logger.Error(errMsg)
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = result.Message
		return response, gasUsed, specialTxType
	}

	response.SysCallMessage.Code = protogo.DockerVMCode_OK

	// merge event
	gasUsed = result.GasUsed
	for _, event := range result.ContractEvent {

		gasUsed, err = gas.EmitEventGasUsed(gasUsed, event)
		if err != nil {
			r.logger.Debugf("handle cross contract request failed, err: %s", err.Error())
			response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
			response.SysCallMessage.Message = err.Error()
			return response, gasUsed, specialTxType
		}

		r.event = append(r.event, event)
	}

	var callContractResponse *protogo.ContractResponse
	callContractResponse, err = constructCallContractResponse(result, currentContractName, txSimContext)
	if err != nil {
		r.logger.Debugf("handle cross contract request failed, err: %s", err.Error())
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = err.Error()
		return response, gasUsed, specialTxType
	}

	var respBytes []byte
	respBytes, err = callContractResponse.Marshal()
	if err != nil {
		r.logger.Debugf("handle cross contract request failed, err: %s", err.Error())
		response.SysCallMessage.Code = protogo.DockerVMCode_FAIL
		response.SysCallMessage.Message = err.Error()
		return response, gasUsed, specialTxType
	}

	response.SysCallMessage.Payload[config.KeyCallContractResp] = respBytes
	response.SysCallMessage.Code = protogo.DockerVMCode_OK
	//response.SysCallMessage.Message = "success"

	return response, gasUsed, specialTxType
}

func constructCallContractResponse(
	result *commonPb.ContractResult,
	contractName string,
	txSimContext protocol.TxSimContext,
) (*protogo.ContractResponse, error) {
	// get the latest status of the read / write set
	txReadMap, txWriteMap := txSimContext.GetTxRWMapByContractName(contractName)

	contractResponse := &protogo.ContractResponse{
		ReadMap:  make(map[string][]byte, len(txReadMap)),
		WriteMap: make(map[string][]byte, len(txWriteMap)),
		Response: &protogo.Response{
			Status:  int32(result.Code),
			Message: result.Message,
			Payload: result.Result,
		},
	}

	for readKey, txRead := range txReadMap {
		contractResponse.ReadMap[readKey] = txRead.Value
	}

	for writeKey, txWrite := range txWriteMap {
		contractResponse.WriteMap[writeKey] = txWrite.Value
	}

	return contractResponse, nil
}

func (r *RuntimeInstance) handleGetStateRequest(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {

	response := r.newEmptyResponse(txId, protogo.DockerVMType_GET_STATE_RESPONSE)

	var contractName string
	var value []byte
	var err error

	contractNameBytes, ok := recvMsg.SysCallMessage.Payload[config.KeyContractName]
	if !ok {
		err = errors.New("unknown contract name")
		r.logger.Errorf("%s", err)
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		return response, gasUsed
	}
	stateKey := recvMsg.SysCallMessage.Payload[config.KeyStateKey]

	contractName = string(contractNameBytes)

	value, err = txSimContext.Get(contractName, stateKey)

	if err != nil {
		r.logger.Errorf("fail to get state from sim context: %s", err)
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		return response, gasUsed
	}

	//r.logger.Debug("get value: ", string(value))
	r.logger.Debugf("[%s] get value: %s", txId, string(value))
	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyStateValue: value,
	}
	gasUsed, err = gas.GetStateGasUsed(gasUsed, value)
	if err != nil {
		r.logger.Errorf("%s", err)
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		return response, gasUsed
	}

	return response, gasUsed
}

func (r *RuntimeInstance) handleGetBatchStateRequest(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {

	response := r.newEmptyResponse(txId, protogo.DockerVMType_GET_BATCH_STATE_RESPONSE)

	var err error
	var payload []byte
	var getKeys []*vmPb.BatchKey

	keys := &vmPb.BatchKeys{}
	if err = keys.Unmarshal(recvMsg.SysCallMessage.Payload[config.KeyStateKey]); err != nil {
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		return response, gasUsed
	}

	getKeys, err = txSimContext.GetKeys(keys.Keys)
	if err != nil {
		response.SysCallMessage.Message = err.Error()
		return response, gasUsed
	}

	r.logger.Debugf("get batch keys values: %v", getKeys)
	resp := vmPb.BatchKeys{Keys: getKeys}
	payload, err = resp.Marshal()
	if err != nil {
		response.SysCallMessage.Message = err.Error()
		return response, gasUsed
	}

	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyStateValue: payload,
	}
	gasUsed, err = gas.GetBatchStateGasUsed(gasUsed, payload)
	if err != nil {
		r.logger.Errorf("%s", err)
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		return response, gasUsed
	}

	return response, gasUsed
}

func (r *RuntimeInstance) handleCreateKvIterator(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {

	createKvIteratorResponse := r.newEmptyResponse(txId, protogo.DockerVMType_CREATE_KV_ITERATOR_RESPONSE)

	/*
		|	index	|			desc			|
		|	----	|			----			|
		|	 0  	|		contractName		|
		|	 1  	|	createKvIteratorFunc	|
		|	 2  	|		startKey			|
		|	 3  	|		startField			|
		|	 4  	|		limitKey			|
		|	 5  	|		limitField			|
		|	 6  	|	  writeMapCache			|
	*/
	calledContractName := recvMsg.SysCallMessage.Payload[config.KeyContractName]
	createFunc := recvMsg.SysCallMessage.Payload[config.KeyIteratorFuncName]
	startKey := recvMsg.SysCallMessage.Payload[config.KeyIterStartKey]
	startField := recvMsg.SysCallMessage.Payload[config.KeyIterStartField]
	writeMapBytes := recvMsg.SysCallMessage.Payload[config.KeyWriteMap]

	writeMap := make(map[string][]byte)
	var err error
	if err = json.Unmarshal(writeMapBytes, &writeMap); err != nil {
		r.logger.Errorf("get WriteMap failed, %s", err.Error())
		createKvIteratorResponse.SysCallMessage.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	gasUsed, err = r.mergeSimContextWriteMap(txSimContext, writeMap, gasUsed)
	if err != nil {
		r.logger.Errorf("merge the sim context write map failed, %s", err.Error())
		createKvIteratorResponse.SysCallMessage.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	if err = protocol.CheckKeyFieldStr(string(startKey), string(startField)); err != nil {
		r.logger.Errorf("invalid key field str, %s", err.Error())
		createKvIteratorResponse.SysCallMessage.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	key := protocol.GetKeyStr(string(startKey), string(startField))

	var iter protocol.StateIterator
	switch string(createFunc) {
	case config.FuncKvIteratorCreate:
		limitKey := string(recvMsg.SysCallMessage.Payload[config.KeyIterLimitKey])
		limitField := string(recvMsg.SysCallMessage.Payload[config.KeyIterLimitField])
		iter, gasUsed, err = kvIteratorCreate(txSimContext, string(calledContractName), key, limitKey, limitField, gasUsed)
		if err != nil {
			r.logger.Errorf("failed to create kv iterator, %s", err.Error())
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Message = err.Error()
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	case config.FuncKvPreIteratorCreate:
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Message = err.Error()
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}

		keyStr := string(key)
		limitLast := keyStr[len(keyStr)-1] + 1
		limit := keyStr[:len(keyStr)-1] + string(limitLast)
		iter, err = txSimContext.Select(string(calledContractName), key, []byte(limit))
		if err != nil {
			r.logger.Errorf("failed to create kv pre iterator, %s", err.Error())
			createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.SysCallMessage.Message = err.Error()
			createKvIteratorResponse.SysCallMessage.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	index := atomic.AddInt32(&r.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)

	r.logger.Debug("create kv iterator: ", index)
	createKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	createKvIteratorResponse.SysCallMessage.Payload[config.KeyIterIndex] = bytehelper.IntToBytes(index)

	return createKvIteratorResponse, gasUsed
}

func (r *RuntimeInstance) handleConsumeKvIterator(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {

	consumeKvIteratorResponse := r.newEmptyResponse(txId, protogo.DockerVMType_CONSUME_KV_ITERATOR_RESPONSE)

	/*
		|	index	|			desc				|
		|	----	|			----  				|
		|	 0  	|	consumeKvIteratorFunc		|
		|	 1  	|		rsIndex					|
	*/
	consumeKvIteratorFunc := recvMsg.SysCallMessage.Payload[config.KeyIteratorFuncName]
	kvIteratorIndex, err := bytehelper.BytesToInt(recvMsg.SysCallMessage.Payload[config.KeyIterIndex])
	if err != nil {
		r.logger.Errorf("failed to get iterator index, %s", err.Error())
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.SysCallMessage.Message = err.Error()
			consumeKvIteratorResponse.SysCallMessage.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	iter, ok := txSimContext.GetIterHandle(kvIteratorIndex)
	if !ok {
		r.logger.Errorf("[kv iterator consume] can not found iterator index [%d]", kvIteratorIndex)
		consumeKvIteratorResponse.SysCallMessage.Message = fmt.Sprintf(
			"[kv iterator consume] can not found iterator index [%d]", kvIteratorIndex,
		)
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.SysCallMessage.Message = err.Error()
			consumeKvIteratorResponse.SysCallMessage.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	kvIterator, ok := iter.(protocol.StateIterator)
	if !ok {
		r.logger.Errorf("assertion failed")
		consumeKvIteratorResponse.SysCallMessage.Message = fmt.Sprintf(
			"[kv iterator consume] failed, iterator %d assertion failed", kvIteratorIndex,
		)
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.SysCallMessage.Message = err.Error()
			consumeKvIteratorResponse.SysCallMessage.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	switch string(consumeKvIteratorFunc) {
	case config.FuncKvIteratorHasNext:
		return kvIteratorHasNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case config.FuncKvIteratorNext:
		return kvIteratorNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case config.FuncKvIteratorClose:
		return kvIteratorClose(kvIterator, gasUsed, consumeKvIteratorResponse)

	default:
		consumeKvIteratorResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKvIteratorResponse.SysCallMessage.Message = fmt.Sprintf("%s not found", consumeKvIteratorFunc)
		consumeKvIteratorResponse.SysCallMessage.Payload = nil
		return consumeKvIteratorResponse, gasUsed
	}
}

func (r *RuntimeInstance) handleCreateKeyHistoryIterator(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {

	createKeyHistoryIterResponse := r.newEmptyResponse(txId, protogo.DockerVMType_CREATE_KEY_HISTORY_TER_RESPONSE)

	/*
		| index | desc          |
		| ----  | ----          |
		| 0     | contractName  |
		| 1     | key           |
		| 2     | field         |
		| 3     | writeMapCache |
	*/
	calledContractName := recvMsg.SysCallMessage.Payload[config.KeyContractName]
	keyStr := recvMsg.SysCallMessage.Payload[config.KeyHistoryIterKey]
	field := recvMsg.SysCallMessage.Payload[config.KeyHistoryIterField]
	writeMapBytes := recvMsg.SysCallMessage.Payload[config.KeyWriteMap]

	writeMap := make(map[string][]byte)
	var err error

	gasUsed, err = gas.CreateKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		createKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	if err = json.Unmarshal(writeMapBytes, &writeMap); err != nil {
		r.logger.Errorf("get write map failed, %s", err.Error())
		createKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	gasUsed, err = r.mergeSimContextWriteMap(txSimContext, writeMap, gasUsed)
	if err != nil {
		r.logger.Errorf("merge the sim context write map failed, %s", err.Error())
		createKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	if err = protocol.CheckKeyFieldStr(string(keyStr), string(field)); err != nil {
		r.logger.Errorf("invalid key field str, %s", err.Error())
		createKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	key := protocol.GetKeyStr(string(keyStr), string(field))

	iter, err := txSimContext.GetHistoryIterForKey(string(calledContractName), key)
	if err != nil {
		createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		createKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	index := atomic.AddInt32(&r.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)

	r.logger.Debug("create key history iterator: ", index)

	createKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	createKeyHistoryIterResponse.SysCallMessage.Payload = map[string][]byte{
		config.KeyIterIndex: bytehelper.IntToBytes(index),
	}

	return createKeyHistoryIterResponse, gasUsed
}

func (r *RuntimeInstance) handleConsumeKeyHistoryIterator(txId string, recvMsg *protogo.DockerVMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {
	consumeKeyHistoryIterResponse := r.newEmptyResponse(txId, protogo.DockerVMType_CONSUME_KEY_HISTORY_ITER_RESPONSE)

	currentGasUsed, err := gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		consumeKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		consumeKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	/*
		|	index	|			desc				|
		|	----	|			----  				|
		|	 0  	|	consumeKvIteratorFunc		|
		|	 1  	|		rsIndex					|
	*/

	consumeKeyHistoryIteratorFunc := recvMsg.SysCallMessage.Payload[config.KeyIteratorFuncName]
	keyHistoryIterIndex, err := bytehelper.BytesToInt(recvMsg.SysCallMessage.Payload[config.KeyIterIndex])
	if err != nil {
		r.logger.Errorf("failed to get iterator index, %s", err.Error())
		consumeKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.SysCallMessage.Message = err.Error()
		consumeKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	iter, ok := txSimContext.GetIterHandle(keyHistoryIterIndex)
	if !ok {
		errMsg := fmt.Sprintf("[key history iterator consume] can not found iterator index [%d]", keyHistoryIterIndex)
		r.logger.Error(errMsg)

		consumeKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.SysCallMessage.Message = errMsg
		consumeKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	keyHistoryIterator, ok := iter.(protocol.KeyHistoryIterator)
	if !ok {
		errMsg := "assertion failed"
		r.logger.Error(errMsg)

		consumeKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.SysCallMessage.Message = errMsg
		consumeKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	switch string(consumeKeyHistoryIteratorFunc) {
	case config.FuncKeyHistoryIterHasNext:
		return keyHistoryIterHasNext(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)

	case config.FuncKeyHistoryIterNext:
		return keyHistoryIterNext(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)

	case config.FuncKeyHistoryIterClose:
		return keyHistoryIterClose(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)
	default:
		consumeKeyHistoryIterResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.SysCallMessage.Message = fmt.Sprintf("%s not found", consumeKeyHistoryIteratorFunc)
		consumeKeyHistoryIterResponse.SysCallMessage.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}
}

func (r *RuntimeInstance) handleGetSenderAddress(txId string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.DockerVMMessage, uint64) {
	getSenderAddressResponse := r.newEmptyResponse(txId, protogo.DockerVMType_GET_SENDER_ADDRESS_RESPONSE)

	var err error
	gasUsed, err = gas.GetSenderAddressGasUsed(gasUsed)
	if err != nil {
		getSenderAddressResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.SysCallMessage.Message = err.Error()
		getSenderAddressResponse.SysCallMessage.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	var bytes []byte
	bytes, err = txSimContext.Get(chainConfigContractName, []byte(keyChainConfig))
	if err != nil {
		r.logger.Errorf("txSimContext get failed, name[%s] key[%s] err: %s",
			chainConfigContractName, keyChainConfig, err.Error())
		getSenderAddressResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.SysCallMessage.Message = err.Error()
		getSenderAddressResponse.SysCallMessage.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	var chainConfig configPb.ChainConfig
	if err = proto.Unmarshal(bytes, &chainConfig); err != nil {
		r.logger.Errorf("unmarshal chainConfig failed, contractName %s err: %+v", chainConfigContractName, err)
		getSenderAddressResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.SysCallMessage.Message = err.Error()
		getSenderAddressResponse.SysCallMessage.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	/*
		| memberType            | memberInfo |
		| ---                   | ---        |
		| MemberType_CERT       | PEM        |
		| MemberType_CERT_HASH  | HASH       |
		| MemberType_PUBLIC_KEY | PEM        |
		| MemberType_ALIAS      | ALIAS      |
	*/

	var address string
	address, err = r.getSenderAddress(chainConfig, txSimContext)
	if err != nil {
		r.logger.Error(err.Error())
		getSenderAddressResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.SysCallMessage.Message = err.Error()
		getSenderAddressResponse.SysCallMessage.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	r.logger.Debug("get sender address: ", address)
	getSenderAddressResponse.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	getSenderAddressResponse.SysCallMessage.Payload = map[string][]byte{
		config.KeySenderAddr: []byte(address),
	}

	return getSenderAddressResponse, gasUsed
}

func (r *RuntimeInstance) getSenderAddress(
	chainConfig configPb.ChainConfig,
	txSimContext protocol.TxSimContext,
) (string, error) {

	var address string
	var err error

	sender := txSimContext.GetSender()

	switch sender.MemberType {
	case accesscontrol.MemberType_CERT:
		address, err = r.getSenderAddressFromCert(sender.MemberInfo, chainConfig.Vm.AddrType)
		if err != nil {
			r.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			return "", err
		}
	case accesscontrol.MemberType_CERT_HASH,
		accesscontrol.MemberType_ALIAS:
		address, err = r.getSenderAddressFromCertHash(sender.MemberInfo, chainConfig.Vm.AddrType, txSimContext)
		if err != nil {
			r.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			return "", err
		}

	case accesscontrol.MemberType_PUBLIC_KEY:
		address, err = r.getSenderAddressFromPublicKeyPEM(
			sender.MemberInfo,
			chainConfig.Vm.AddrType,
			crypto.HashAlgoMap[chainConfig.GetCrypto().Hash],
		)
		if err != nil {
			r.logger.Errorf("getSenderAddressFromPublicKeyPEM failed, %s", err.Error())
			return "", err
		}

	default:
		r.logger.Errorf("getSenderAddress failed, invalid member type")
		return "", err
	}

	return address, nil
}

func (r *RuntimeInstance) getSenderAddressFromCertHash(
	memberInfo []byte,
	addressType configPb.AddrType,
	txSimContext protocol.TxSimContext,
) (string, error) {

	var certBytes []byte
	var err error
	certBytes, err = r.getCertFromChain(memberInfo, txSimContext)
	if err != nil {
		return "", err
	}

	var address string
	address, err = r.getSenderAddressFromCert(certBytes, addressType)
	if err != nil {
		r.logger.Errorf("getSenderAddressFromCert failed, %s", err.Error())
		return "", err
	}

	return address, nil
}

func (r *RuntimeInstance) getCertFromChain(memberInfo []byte, txSimContext protocol.TxSimContext) ([]byte, error) {
	certHashKey := hex.EncodeToString(memberInfo)
	certBytes, err := txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
	if err != nil {
		r.logger.Errorf("get cert from chain failed, %s", err.Error())
		return nil, err
	}

	return certBytes, nil
}

func (r *RuntimeInstance) getSenderAddressFromCert(certPem []byte, addressType configPb.AddrType) (string, error) {
	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromCertificatePEM(certPem)
		if err != nil {
			return "", fmt.Errorf("ParseCertificate failed, %s", err.Error())
		}

		return address, nil
	}

	if addressType == configPb.AddrType_CHAINMAKER {
		return r.calculateCertAddr(certPem)
	}

	return "", errors.New("invalid address type")
}

func (r *RuntimeInstance) calculateCertAddr(certPem []byte) (string, error) {
	blockCrt, _ := pem.Decode(certPem)
	crt, err := bcx509.ParseCertificate(blockCrt.Bytes)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	ski := hex.EncodeToString(crt.SubjectKeyId)
	addrInt, err := evmutils.MakeAddressFromHex(ski)
	if err != nil {
		return "", fmt.Errorf("MakeAddressFromHex failed, %s", err.Error())
	}

	addr := evmutils.BigToAddress(addrInt)
	addrBytes := addr[:]

	return hex.EncodeToString(addrBytes), nil
}

func (r *RuntimeInstance) getSenderAddressFromPublicKeyPEM(
	publicKeyPem []byte,
	addressType configPb.AddrType,
	hashType crypto.HashType,
) (string, error) {

	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromPublicKeyPEM(publicKeyPem)
		if err != nil {
			r.logger.Errorf("ZXAddressFromPublicKeyPEM, failed, %s", err.Error())
		}
		return address, err
	}

	if addressType == configPb.AddrType_CHAINMAKER {
		return r.calculatePubKeyAddr(publicKeyPem, hashType)
	}

	return "", errors.New("invalid address type")
}

func (r *RuntimeInstance) calculatePubKeyAddr(publicKeyPem []byte, hashType crypto.HashType) (string, error) {
	publicKey, err := asym.PublicKeyFromPEM(publicKeyPem)
	if err != nil {
		return "", fmt.Errorf("ParsePublicKey failed, %s", err.Error())
	}

	ski, err := commonCrt.ComputeSKI(hashType, publicKey.ToStandardKey())
	if err != nil {
		return "", fmt.Errorf("computeSKI from public key failed, %s", err.Error())
	}

	addrInt, err := evmutils.MakeAddressFromHex(hex.EncodeToString(ski))
	if err != nil {
		return "", fmt.Errorf("make address from public key failed, %s", err)
	}

	addr := evmutils.BigToAddress(addrInt)
	addrBytes := addr[:]

	return hex.EncodeToString(addrBytes), nil
}

func kvIteratorCreate(txSimContext protocol.TxSimContext, calledContractName string,
	key []byte, limitKey, limitField string, gasUsed uint64) (protocol.StateIterator, uint64, error) {
	var err error
	gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
	if err != nil {
		return nil, gasUsed, err
	}

	if err = protocol.CheckKeyFieldStr(limitKey, limitField); err != nil {
		return nil, gasUsed, err
	}
	limit := protocol.GetKeyStr(limitKey, limitField)
	var iter protocol.StateIterator
	iter, err = txSimContext.Select(calledContractName, key, limit)
	if err != nil {
		return nil, gasUsed, err
	}

	return iter, gasUsed, err
}

func kvIteratorClose(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	kvIterator.Release()
	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = nil

	return response, gasUsed
}

func (r *RuntimeInstance) mergeSimContextWriteMap(txSimContext protocol.TxSimContext,
	writeMap map[string][]byte, gasUsed uint64) (uint64, error) {
	// merge the sim context write map

	for key, value := range writeMap {
		var contractName string
		var contractKey string
		var contractField string
		keyList := strings.Split(key, "#")
		keyLen := len(keyList)
		if keyLen < 2 {
			return gasUsed, fmt.Errorf("key list length == %d, needs to be >= 2", keyLen)
		}
		contractName = keyList[0]
		contractKey = keyList[1]
		if keyLen == 3 {
			contractField = keyList[2]
		}
		// put state gas used calc and check gas limit
		var err error
		gasUsed, err = gas.PutStateGasUsed(gasUsed, contractName, contractKey, contractField, value)
		if err != nil {
			return gasUsed, err
		}

		err = txSimContext.Put(contractName, protocol.GetKeyStr(contractKey, contractField), value)
		if err != nil {
			return gasUsed, err
		}
	}

	return gasUsed, nil
}

func kvIteratorHasNext(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	hasNext := config.BoolFalse
	if kvIterator.Next() {
		hasNext = config.BoolTrue
	}

	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyIteratorHasNext: bytehelper.IntToBytes(int32(hasNext)),
	}

	return response, gasUsed
}

func kvIteratorNext(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	if kvIterator == nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = msgIterIsNil
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	var kvRow *store.KV
	kvRow, err = kvIterator.Value()
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	arrKey := strings.Split(string(kvRow.Key), "#")
	key := arrKey[0]
	field := ""
	if len(arrKey) > 1 {
		field = arrKey[1]
	}

	value := kvRow.Value

	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyUserKey:    []byte(key),
		config.KeyUserField:  []byte(field),
		config.KeyStateValue: value,
	}

	return response, gasUsed
}

func keyHistoryIterHasNext(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	hasNext := config.BoolFalse
	if iter.Next() {
		hasNext = config.BoolTrue
	}

	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyIteratorHasNext: bytehelper.IntToBytes(int32(hasNext)),
	}

	return response, gasUsed
}

func keyHistoryIterNext(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	if iter == nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = msgIterIsNil
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	var historyValue *store.KeyModification
	historyValue, err = iter.Value()
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	blockHeight := bytehelper.IntToBytes(int32(historyValue.BlockHeight))
	timestampStr := strconv.FormatInt(historyValue.Timestamp, 10)
	isDelete := config.BoolTrue
	if !historyValue.IsDelete {
		isDelete = config.BoolFalse
	}

	/*
		| index | desc        |
		| ---   | ---         |
		| 0     | txId        |
		| 1     | blockHeight |
		| 2     | value       |
		| 3     | isDelete    |
		| 4     | timestamp   |
	*/
	response.SysCallMessage.Payload = map[string][]byte{
		config.KeyTxId:        []byte(historyValue.TxId),
		config.KeyBlockHeight: blockHeight,
		config.KeyStateValue:  historyValue.Value,
		config.KeyIsDelete:    bytehelper.IntToBytes(int32(isDelete)),
		config.KeyTimestamp:   []byte(timestampStr),
	}

	return response, gasUsed
}

func keyHistoryIterClose(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.DockerVMMessage) (*protogo.DockerVMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.SysCallMessage.Code = protocol.ContractSdkSignalResultFail
		response.SysCallMessage.Message = err.Error()
		response.SysCallMessage.Payload = nil
		return response, gasUsed
	}

	iter.Release()
	response.SysCallMessage.Code = protocol.ContractSdkSignalResultSuccess
	response.SysCallMessage.Payload = nil

	return response, gasUsed
}

func (r *RuntimeInstance) handleGetByteCodeRequest(
	txId string,
	txSimContext protocol.TxSimContext,
	recvMsg *protogo.DockerVMMessage,
	byteCode []byte) *protogo.DockerVMMessage {

	response := &protogo.DockerVMMessage{
		TxId: txId,
		Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
		Response: &protogo.TxResponse{
			ChainId:         r.chainId,
			Result:          make([]byte, 1),
			ContractName:    recvMsg.Request.ContractName,
			ContractVersion: recvMsg.Request.ContractVersion,
		},
	}

	// ChainId#ContractName#ContractVersion
	// chain1#contract1#1.0.0
	contractFullName := constructContractKey(
		recvMsg.Request.ChainId,
		recvMsg.Request.ContractName,
		recvMsg.Request.ContractVersion,
	)

	contractName := recvMsg.Request.ContractName
	r.logger.Debugf("name: %s", contractName)
	r.logger.Debugf("full name: %s", contractFullName)

	var err error
	if len(byteCode) == 0 {
		r.logger.Warnf("[%s] bytecode is missing", txId)
		byteCode, err = txSimContext.GetContractBytecode(contractName)
		if err != nil || len(byteCode) == 0 {
			r.logger.Errorf("[%s] fail to get contract bytecode: %s, required contract name is: [%s]", txId, err,
				contractName)
			if err != nil {
				response.Response.Message = err.Error()
			} else {
				response.Response.Message = "contract byte is nil"
			}
			return response
		}
	}

	hostMountPath := r.clientMgr.GetVMConfig().DockerVMMountPath
	contractDir := filepath.Join(hostMountPath, mountContractDir)

	contractZipPath := filepath.Join(contractDir, fmt.Sprintf("%s.7z", contractName)) // contract1.7z
	contractPathWithoutVersion := filepath.Join(contractDir, contractName)
	contractFullNamePath := filepath.Join(contractDir, contractFullName)

	// save bytecode to disk
	err = r.saveBytesToDisk(byteCode, contractZipPath)
	if err != nil {
		r.logger.Errorf("fail to save bytecode to disk: %s", err)
		response.Response.Code = protogo.DockerVMCode_FAIL
		response.Response.Message = err.Error()
		return response
	}

	// extract 7z file
	unzipCommand := fmt.Sprintf("7z e %s -o%s -y", contractZipPath, contractDir) // contract1
	err = r.runCmd(unzipCommand)
	if err != nil {
		r.logger.Errorf("fail to extract contract, %v", err)
		response.Response.Code = protogo.DockerVMCode_FAIL
		response.Response.Message = err.Error()
		return response
	}

	// remove 7z file
	err = os.Remove(contractZipPath)
	if err != nil {
		r.logger.Errorf("fail to remove zipped file: %s", err)
		response.Response.Code = protogo.DockerVMCode_FAIL
		response.Response.Message = err.Error()
		return response
	}

	// replace contract name to contractName:version
	err = os.Rename(contractPathWithoutVersion, contractFullNamePath)
	if err != nil {
		r.logger.Errorf("fail to rename original file name: %s, "+
			"please make sure contract name should be same as zipped file", err)
		response.Response.Code = protogo.DockerVMCode_FAIL
		response.Response.Message = err.Error()
		return response
	}

	response.Response.Code = protogo.DockerVMCode_OK

	if r.clientMgr.NeedSendContractByteCode() {
		contractByteCode, err := ioutil.ReadFile(contractFullNamePath)
		if err != nil {
			r.logger.Errorf("fail to load contract executable file: %s, ", err)
			response.Response.Code = protogo.DockerVMCode_FAIL
			response.Response.Message = err.Error()
			return response
		}

		// remove contract file
		err = os.Remove(contractFullNamePath)
		if err != nil {
			r.logger.Errorf("fail to remove zipped file: %s", err)
		}

		response.Response.Code = protogo.DockerVMCode_OK
		response.Response.Result = contractByteCode
	}

	return response
}

// constructContractKey chainId#contractName#contractVersion
func constructContractKey(chainID, contractName, contractVersion string) string {
	var sb strings.Builder
	sb.WriteString(chainID)
	sb.WriteString("#")
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

func (r *RuntimeInstance) errorResult(
	contractResult *commonPb.ContractResult,
	err error,
	errMsg string) (*commonPb.ContractResult, protocol.ExecOrderTxType) {
	contractResult.Code = uint32(1)
	//if err != nil {
	//	errMsg += ", " + err.Error()
	//}
	contractResult.Message = errMsg
	//r.logger.Error(errMsg)
	return contractResult, protocol.ExecOrderTxTypeNormal
}

func (r *RuntimeInstance) saveBytesToDisk(bytes []byte, newFilePath string) error {

	f, err := os.Create(newFilePath)
	if err != nil {
		return err
	}
	defer func(f *os.File) {
		err = f.Close()
		if err != nil {
			return
		}
	}(f)

	_, err = f.Write(bytes)
	if err != nil {
		return err
	}

	return f.Sync()
}

// RunCmd exec cmd
func (r *RuntimeInstance) runCmd(command string) error {
	var stderr bytes.Buffer
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...) // #nosec
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		r.logger.Errorf("failed to run cmd %s start, %v, %v", command, err, stderr.String())
		return err
	}

	if err := cmd.Wait(); err != nil {
		r.logger.Errorf("failed to run cmd %s wait, %v, %v", command, err, stderr.String())
		return err
	}
	return nil
}

func (r *RuntimeInstance) newEmptyResponse(txId string, msgType protogo.DockerVMType) *protogo.DockerVMMessage {
	return &protogo.DockerVMMessage{
		TxId: txId,
		Type: msgType,
		SysCallMessage: &protogo.SysCallMessage{
			Payload: map[string][]byte{},
		},
		Response: nil,
		Request:  nil,
	}
}

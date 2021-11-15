/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync/atomic"

	"chainmaker.org/chainmaker/pb-go/v2/store"

	"chainmaker.org/chainmaker/common/v2/bytehelper"

	"chainmaker.org/chainmaker/vm-docker-go/config"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/gas"
	"chainmaker.org/chainmaker/vm-docker-go/pb/protogo"
	"github.com/gogo/protobuf/proto"
)

const (
	mountContractDir = "contracts"
)

type CDMClient interface {
	GetTxSendCh() chan *protogo.CDMMessage

	GetStateResponseSendCh() chan *protogo.CDMMessage

	RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage)

	GetCMConfig() *config.DockerVMConfig
}

// RuntimeInstance docker-go runtime
type RuntimeInstance struct {
	rowIndex int32  // iterator index
	ChainId  string // chain id
	Client   CDMClient
	Log      protocol.Logger
}

// Invoke process one tx in docker and return result
// nolint: gocyclo, revive
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte, txSimContext protocol.TxSimContext,
	gasUsed uint64) (contractResult *commonPb.ContractResult) {
	txId := txSimContext.GetTx().Payload.TxId

	// contract response
	contractResult = &commonPb.ContractResult{
		Code:    uint32(1),
		Result:  nil,
		Message: "",
	}

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
	gasUsed, err = gas.ContractGasUsed(gasUsed, method, contract.Name, byteCode, txSimContext)
	if err != nil {
		contractResult.GasUsed = gasUsed
		return r.errorResult(contractResult, err, err.Error())
	}

	for key := range parameters {
		if strings.Contains(key, "CONTRACT") {
			delete(parameters, key)
		}
	}

	// construct cdm message
	txRequest := &protogo.TxRequest{
		TxId:            txId,
		ContractName:    contract.Name,
		ContractVersion: contract.Version,
		Method:          method,
		Parameters:      parameters,
		TxContext: &protogo.TxContext{
			CurrentHeight:       0,
			OriginalProcessName: "",
			WriteMap:            nil,
			ReadMap:             nil,
		},
	}
	txRequestPayload, _ := proto.Marshal(txRequest)
	cdmMessage := &protogo.CDMMessage{
		TxId:    txId,
		Type:    protogo.CDMType_CDM_TYPE_TX_REQUEST,
		Payload: txRequestPayload,
	}

	// register result chan
	responseCh := make(chan *protogo.CDMMessage)
	r.Client.RegisterRecvChan(txId, responseCh)

	// send message to tx chan
	r.Client.GetTxSendCh() <- cdmMessage

	// wait this chan
	for {
		recvMsg := <-responseCh

		switch recvMsg.Type {
		case protogo.CDMType_CDM_TYPE_GET_BYTECODE:

			getByteCodeResponse := r.handleGetByteCodeRequest(txId, recvMsg, byteCode)
			r.Client.GetStateResponseSendCh() <- getByteCodeResponse

		case protogo.CDMType_CDM_TYPE_GET_STATE:

			getStateResponse, pass := r.handleGetStateRequest(txId, recvMsg, txSimContext)

			if pass {
				gasUsed, err = gas.GetStateGasUsed(gasUsed, getStateResponse.Payload)
				if err != nil {
					getStateResponse.ResultCode = protocol.ContractSdkSignalResultFail
					getStateResponse.Payload = nil
					getStateResponse.Message = err.Error()
				}
			}

			r.Client.GetStateResponseSendCh() <- getStateResponse

		case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
			// construct response
			var txResponse protogo.TxResponse
			_ = proto.UnmarshalMerge(recvMsg.Payload, &txResponse)

			// tx fail, just return without merge read write map and events
			if txResponse.Code != 0 {
				contractResult.Code = 1
				contractResult.Result = txResponse.Result
				contractResult.Message = txResponse.Message
				contractResult.GasUsed = gasUsed

				return contractResult
			}

			contractResult.Code = 0
			contractResult.Result = txResponse.Result
			contractResult.Message = txResponse.Message

			// merge the sim context write map

			for key, value := range txResponse.WriteMap {
				var contractName string
				var contractKey string
				var contractField string
				keyList := strings.Split(key, "#")
				contractName = keyList[0]
				contractKey = keyList[1]
				if len(keyList) == 3 {
					contractField = keyList[2]
				}
				// put state gas used calc and check gas limit
				gasUsed, err = gas.PutStateGasUsed(gasUsed, contractName, contractKey, contractField, value)
				if err != nil {
					contractResult.GasUsed = gasUsed
					return r.errorResult(contractResult, err, err.Error())
				}

				err = txSimContext.Put(contractName, protocol.GetKeyStr(contractKey, contractField), value)
				if err != nil {
					contractResult.GasUsed = gasUsed
					return r.errorResult(contractResult, err, "fail to put in sim context")
				}
			}

			// merge events
			var contractEvents []*commonPb.ContractEvent

			if len(txResponse.Events) > protocol.EventDataMaxCount-1 {
				err = fmt.Errorf("too many event data")
				return r.errorResult(contractResult, err, "fail to put event data")
			}

			for _, event := range txResponse.Events {
				contractEvent := &commonPb.ContractEvent{
					Topic:           event.Topic,
					TxId:            txId,
					ContractName:    event.ContractName,
					ContractVersion: event.ContractVersion,
					EventData:       event.Data,
				}

				// emit event gas used calc and check gas limit
				gasUsed, err = gas.EmitEventGasUsed(gasUsed, contractEvent)
				if err != nil {
					contractResult.GasUsed = gasUsed
					return r.errorResult(contractResult, err, err.Error())
				}

				contractEvents = append(contractEvents, contractEvent)
			}

			contractResult.GasUsed = gasUsed
			contractResult.ContractEvent = contractEvents

			close(responseCh)
			return contractResult

		case protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR:
			var createKvIteratorResponse *protogo.CDMMessage
			createKvIteratorResponse, gasUsed = r.handleCreateKvIterator(txId, recvMsg, txSimContext, gasUsed)

			r.Client.GetStateResponseSendCh() <- createKvIteratorResponse

		case protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR:
			var consumeKvIteratorResponse *protogo.CDMMessage
			consumeKvIteratorResponse, gasUsed = r.handleConsumeKvIterator(txId, recvMsg, txSimContext, gasUsed)

			r.Client.GetStateResponseSendCh() <- consumeKvIteratorResponse

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

func (r *RuntimeInstance) newEmptyResponse(txId string, msgType protogo.CDMType) *protogo.CDMMessage {
	return &protogo.CDMMessage{
		TxId:       txId,
		Type:       msgType,
		ResultCode: protocol.ContractSdkSignalResultFail,
		Payload:    nil,
		Message:    "",
	}
}

type Bool int32

const (
	FuncKvIteratorCreate    = "createKvIterator"
	FuncKvPreIteratorCreate = "createKvPreIterator"
	FuncKvIteratorHasNext   = "kvIteratorHasNext"
	FuncKvIteratorNext      = "kvIteratorNext"
	FuncKvIteratorClose     = "kvIteratorClose"

	boolTrue  Bool = 1
	boolFalse Bool = 0
)

func (r *RuntimeInstance) handleConsumeKvIterator(txId string, recvMsg *protogo.CDMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.CDMMessage, uint64) {

	consumeKvIteratorResponse := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR_RESPONSE)

	/*
		|	index	|			desc				|
		|	----	|			----  				|
		|	 0  	|	consumeKvIteratorFunc		|
		|	 1  	|		rsIndex					|
	*/

	keyList := strings.Split(string(recvMsg.Payload), "#")
	consumeKvIteratorFunc := keyList[0]
	kvIteratorIndex, err := bytehelper.BytesToInt([]byte(keyList[1]))
	if err != nil {
		r.Log.Errorf("failed to get iterator index, %s", err.Error())
		gasUsed, err = gas.KvIteratorConsumeGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.Message = err.Error()
			consumeKvIteratorResponse.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	kvIterator, ok := txSimContext.GetStateKvHandle(kvIteratorIndex)
	if !ok {
		r.Log.Errorf("GetStateKvHandle failed, %s", err.Error())
		consumeKvIteratorResponse.Message = fmt.Sprintf(
			"[kv iterator has next] can not found rs_index[%d]", kvIteratorIndex,
		)
		gasUsed, err = gas.KvIteratorConsumeGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.Message = err.Error()
			consumeKvIteratorResponse.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	switch consumeKvIteratorFunc {
	case FuncKvIteratorHasNext:
		return kvIteratorHasNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case FuncKvIteratorNext:
		return kvIteratorNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case FuncKvIteratorClose:
		return kvIteratorClose(kvIterator, gasUsed, consumeKvIteratorResponse)

	default:
		consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKvIteratorResponse.Message = fmt.Sprintf("%s not found", consumeKvIteratorFunc)
		consumeKvIteratorResponse.Payload = nil
		return consumeKvIteratorResponse, gasUsed
	}
}

func kvIteratorHasNext(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.KvIteratorConsumeGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	hasNext := boolFalse
	if kvIterator.Next() {
		hasNext = boolTrue
	}

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = bytehelper.IntToBytes(int32(hasNext))

	return response, gasUsed
}

func kvIteratorNext(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.KvIteratorConsumeGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	if kvIterator == nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = "iterator is nil"
		response.Payload = nil
		return response, gasUsed
	}

	var kvRow *store.KV
	kvRow, err = kvIterator.Value()
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	arrKey := strings.Split(string(kvRow.Key), "#")
	key := arrKey[0]
	field := ""
	if len(arrKey) > 1 {
		field = arrKey[1]
	}

	value := kvRow.Value

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = func() []byte {
		str := key + "#" + field + "#" + string(value)
		return []byte(str)
	}()

	return response, gasUsed
}

func kvIteratorClose(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.KvIteratorConsumeGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	kvIterator.Release()
	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = nil

	return response, gasUsed
}

func (r *RuntimeInstance) handleCreateKvIterator(txId string, recvMsg *protogo.CDMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.CDMMessage, uint64) {

	createKvIteratorResponse := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR_RESPONSE)

	/*
		|	index	|			desc			|
		|	----	|			----			|
		|	 0  	|		contractName		|
		|	 1  	|	createKvIteratorFunc	|
		|	 2  	|		startKey			|
		|	 3  	|		startField			|
		|	 4  	|		limitKey			|
		|	 5  	|		limitField			|
	*/
	keyList := strings.Split(string(recvMsg.Payload), "#")
	calledContractName := keyList[0]
	createFunc := keyList[1]
	startKey := keyList[2]
	startField := keyList[3]

	if err := protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		r.Log.Errorf("invalid key field str, %s", err.Error())
		createKvIteratorResponse.Message = err.Error()
		gasUsed, err = gas.KvIteratorCreateGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	key := protocol.GetKeyStr(startKey, startField)

	var iter protocol.StateIterator
	var err error
	switch createFunc {
	case FuncKvIteratorCreate:
		gasUsed, err = gas.KvIteratorCreateGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
		limitKey := keyList[4]
		limitField := keyList[5]
		if err = protocol.CheckKeyFieldStr(limitKey, limitField); err != nil {
			r.Log.Errorf("invalid key field str, %s", err.Error())
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
		limit := protocol.GetKeyStr(limitKey, limitField)
		iter, err = txSimContext.Select(calledContractName, key, limit)
		if err != nil {
			r.Log.Errorf("failed to create kv iterator, %s", err.Error())
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}

	case FuncKvPreIteratorCreate:
		gasUsed, err = gas.KvIteratorCreateGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}

		keyStr := string(key)
		limitLast := keyStr[len(keyStr)-1] + 1
		limit := keyStr[:len(keyStr)-1] + string(limitLast)
		iter, err = txSimContext.Select(calledContractName, key, []byte(limit))
		if err != nil {
			r.Log.Errorf("failed to create kv pre iterator, %s", err.Error())
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	index := atomic.AddInt32(&r.rowIndex, 1)
	txSimContext.SetStateKvHandle(index, iter)

	r.Log.Debug("create kv iterator: ", index)
	createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultSuccess
	createKvIteratorResponse.Payload = bytehelper.IntToBytes(index)

	return createKvIteratorResponse, gasUsed
}

func (r *RuntimeInstance) handleGetByteCodeRequest(txId string, recvMsg *protogo.CDMMessage,
	byteCode []byte) *protogo.CDMMessage {

	response := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE)

	contractFullName := string(recvMsg.Payload)             // contract1#1.0.0
	contractName := strings.Split(contractFullName, "#")[0] // contract1

	hostMountPath := r.Client.GetCMConfig().DockerVMMountPath
	hostMountPath = filepath.Join(hostMountPath, r.ChainId)

	contractDir := filepath.Join(hostMountPath, mountContractDir)
	contractZipPath := filepath.Join(contractDir, fmt.Sprintf("%s.7z", contractName)) // contract1.7z
	contractPathWithoutVersion := filepath.Join(contractDir, contractName)
	contractPathWithVersion := filepath.Join(contractDir, contractFullName)

	// save bytecode to disk
	err := r.saveBytesToDisk(byteCode, contractZipPath)
	if err != nil {
		r.Log.Errorf("fail to save bytecode to disk: %s", err)
		response.Message = err.Error()
		return response
	}

	// extract 7z file
	unzipCommand := fmt.Sprintf("7z e %s -o%s -y", contractZipPath, contractDir) // contract1
	err = r.runCmd(unzipCommand)
	if err != nil {
		r.Log.Errorf("fail to extract contract: %s", err)
		response.Message = err.Error()
		return response
	}

	// remove 7z file
	err = os.Remove(contractZipPath)
	if err != nil {
		r.Log.Errorf("fail to remove zipped file: %s", err)
		response.Message = err.Error()
		return response
	}

	// replace contract name to contractName:version
	err = os.Rename(contractPathWithoutVersion, contractPathWithVersion)
	if err != nil {
		r.Log.Errorf("fail to rename original file name: %s, "+
			"please make sure contract name should be same as zipped file", err)
		response.Message = err.Error()
		return response
	}

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = []byte(contractFullName)

	return response
}

func (r *RuntimeInstance) handleGetStateRequest(txId string, recvMsg *protogo.CDMMessage,
	txSimContext protocol.TxSimContext) (*protogo.CDMMessage, bool) {

	response := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE)

	var contractName string
	var contractKey string
	var contractField string
	var value []byte
	var err error

	keyList := strings.Split(string(recvMsg.Payload), "#")
	contractName = keyList[0]
	contractKey = keyList[1]
	if len(keyList) == 3 {
		contractField = keyList[2]
	}

	value, err = txSimContext.Get(contractName, protocol.GetKeyStr(contractKey, contractField))

	if err != nil {
		r.Log.Errorf("fail to get state from sim context: %s", err)
		response.Message = err.Error()
		return response, false
	}

	if len(value) == 0 {
		r.Log.Errorf("fail to get state from sim context: %s", "value is empty")
		response.Message = "value is empty"
		return response, false
	}

	r.Log.Debug("get value: ", string(value))
	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = value
	return response, true
}

func (r *RuntimeInstance) errorResult(contractResult *commonPb.ContractResult,
	err error, errMsg string) *commonPb.ContractResult {
	contractResult.Code = uint32(1)
	if err != nil {
		errMsg += ", " + err.Error()
	}
	contractResult.Message = errMsg
	r.Log.Error(errMsg)
	return contractResult
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
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...) // #nosec

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

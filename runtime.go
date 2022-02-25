/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync/atomic"
	"time"

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
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/gas"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"github.com/gogo/protobuf/proto"
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
// todo byteCode 参数在非安装调用的逻辑时 为空，需要从存储中读取
// todo 升级如果失败，会存在文件已经被覆盖删掉了，导致旧版本的合约也没办法调用
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte, txSimContext protocol.TxSimContext,
	gasUsed uint64) (contractResult *commonPb.ContractResult, execOrderTxType protocol.ExecOrderTxType) {
	txId := txSimContext.GetTx().Payload.TxId

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
	cdmMessage := &protogo.CDMMessage{
		TxId:      txId,
		Type:      protogo.CDMType_CDM_TYPE_TX_REQUEST,
		TxRequest: txRequest,
	}

	// register result chan
	responseCh := make(chan *protogo.CDMMessage)
	r.Client.RegisterRecvChan(txId, responseCh)

	// send message to tx chan
	sendCh := r.Client.GetTxSendCh()
	r.Log.Debugf("[%s] put tx in send chan with length [%d]", txRequest.TxId, len(sendCh))
	sendCh <- cdmMessage

	timeoutC := time.After(timeout * time.Millisecond)

	r.Log.Debugf("start tx [%s] in runtime", txRequest.TxId)
	// wait this chan
	for {
		select {
		case recvMsg := <-responseCh:
			switch recvMsg.Type {
			case protogo.CDMType_CDM_TYPE_GET_BYTECODE:
				r.Log.Debugf("tx [%s] start get bytecode [%v]", txId, recvMsg)
				getByteCodeResponse := r.handleGetByteCodeRequest(txId, recvMsg, byteCode)
				r.Client.GetStateResponseSendCh() <- getByteCodeResponse
				r.Log.Debugf("tx [%s] finish get bytecode [%v]", txId, getByteCodeResponse)

			case protogo.CDMType_CDM_TYPE_GET_STATE:
				r.Log.Debugf("tx [%s] start get state [%v]", txId, recvMsg)
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
				r.Log.Debugf("tx [%s] finish get state [%v]", txId, getStateResponse)

			case protogo.CDMType_CDM_TYPE_TX_RESPONSE:
				r.Log.Debugf("[%s] start handle response [%v]", txId, recvMsg)
				// construct response
				txResponse := recvMsg.TxResponse
				// tx fail, just return without merge read write map and events
				if txResponse.Code != 0 {
					contractResult.Code = 1
					contractResult.Result = txResponse.Result
					contractResult.Message = txResponse.Message
					contractResult.GasUsed = gasUsed
					r.Log.Errorf("[%s] return error response [%v]", txId, contractResult)
					return contractResult, protocol.ExecOrderTxTypeNormal
				}

				contractResult.Code = 0
				contractResult.Result = txResponse.Result
				contractResult.Message = txResponse.Message

				// merge the sim context write map
				gasUsed, err = r.mergeSimContextWriteMap(txSimContext, txResponse.GetWriteMap(), gasUsed)
				if err != nil {
					contractResult.GasUsed = gasUsed
					r.Log.Errorf("[%s] return error response [%v]", txId, contractResult)
					return r.errorResult(contractResult, err, "fail to put in sim context")
				}

				// merge events
				var contractEvents []*commonPb.ContractEvent

				if len(txResponse.Events) > protocol.EventDataMaxCount-1 {
					err = fmt.Errorf("too many event data")
					r.Log.Errorf("[%s] return error response [%v]", txId, contractResult)
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
						r.Log.Errorf("[%s] return error response [%v]", txId, contractResult)
						contractResult.GasUsed = gasUsed
						return r.errorResult(contractResult, err, err.Error())
					}

					contractEvents = append(contractEvents, contractEvent)
				}

				contractResult.GasUsed = gasUsed
				contractResult.ContractEvent = contractEvents

				close(responseCh)

				r.Log.Debugf("[%s] finish handle response [%v]", txId, contractResult)
				return contractResult, specialTxType

			case protogo.CDMType_CDM_TYPE_CREATE_KV_ITERATOR:
				r.Log.Debugf("tx [%s] start create kv iterator [%v]", txId, recvMsg)
				var createKvIteratorResponse *protogo.CDMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKvIteratorResponse, gasUsed = r.handleCreateKvIterator(txId, recvMsg, txSimContext, gasUsed)

				r.Client.GetStateResponseSendCh() <- createKvIteratorResponse
				r.Log.Debugf("tx [%s] finish create kv iterator [%v]", txId, createKvIteratorResponse)

			case protogo.CDMType_CDM_TYPE_CONSUME_KV_ITERATOR:
				r.Log.Debugf("tx [%s] start consume kv iterator [%v]", txId, recvMsg)
				var consumeKvIteratorResponse *protogo.CDMMessage
				consumeKvIteratorResponse, gasUsed = r.handleConsumeKvIterator(txId, recvMsg, txSimContext, gasUsed)

				r.Client.GetStateResponseSendCh() <- consumeKvIteratorResponse
				r.Log.Debugf("tx [%s] finish consume kv iterator [%v]", txId, consumeKvIteratorResponse)

			case protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_ITER:
				r.Log.Debugf("tx [%s] start create key history iterator [%v]", txId, recvMsg)
				var createKeyHistoryIterResp *protogo.CDMMessage
				specialTxType = protocol.ExecOrderTxTypeIterator
				createKeyHistoryIterResp, gasUsed = r.handleCreateKeyHistoryIterator(txId, recvMsg, txSimContext, gasUsed)
				r.Client.GetStateResponseSendCh() <- createKeyHistoryIterResp
				r.Log.Debugf("tx [%s] finish create key history iterator [%v]", txId, createKeyHistoryIterResp)

			case protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER:
				r.Log.Debugf("tx [%s] start consume key history iterator [%v]", txId, recvMsg)
				var consumeKeyHistoryResp *protogo.CDMMessage
				consumeKeyHistoryResp, gasUsed = r.handleConsumeKeyHistoryIterator(txId, recvMsg, txSimContext, gasUsed)
				r.Client.GetStateResponseSendCh() <- consumeKeyHistoryResp
				r.Log.Debugf("tx [%s] finish consume key history iterator [%v]", txId, consumeKeyHistoryResp)

			case protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS:
				r.Log.Debugf("tx [%s] start get sender address [%v]", txId, recvMsg)
				var getSenderAddressResp *protogo.CDMMessage
				getSenderAddressResp, gasUsed = r.handleGetSenderAddress(txId, txSimContext, gasUsed)
				r.Client.GetStateResponseSendCh() <- getSenderAddressResp
				r.Log.Debugf("tx [%s] finish get sender address [%v]", txId, getSenderAddressResp)

			default:
				contractResult.GasUsed = gasUsed
				return r.errorResult(
					contractResult,
					fmt.Errorf("unknow type"),
					"fail to receive request",
				)
			}
		case <-timeoutC:
			contractResult.GasUsed = gasUsed
			return r.errorResult(
				contractResult,
				fmt.Errorf("docker-vm-go timeout"),
				"fail to receive response",
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

func (r *RuntimeInstance) handleGetSenderAddress(txId string,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.CDMMessage, uint64) {
	getSenderAddressResponse := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_GET_SENDER_ADDRESS_RESPONSE)

	var err error
	gasUsed, err = gas.GetSenderAddressGasUsed(gasUsed)
	if err != nil {
		getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.Message = err.Error()
		getSenderAddressResponse.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	var bytes []byte
	bytes, err = txSimContext.Get(chainConfigContractName, []byte(keyChainConfig))
	if err != nil {
		r.Log.Errorf("txSimContext get failed, name[%s] key[%s] err: %s",
			chainConfigContractName, keyChainConfig, err.Error())
		getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.Message = err.Error()
		getSenderAddressResponse.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	var chainConfig configPb.ChainConfig
	if err = proto.Unmarshal(bytes, &chainConfig); err != nil {
		r.Log.Errorf("unmarshal chainConfig failed, contractName %s err: %+v", chainConfigContractName, err)
		getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.Message = err.Error()
		getSenderAddressResponse.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	/*
		| memberType            | memberInfo |
		| ---                   | ---        |
		| MemberType_CERT       | PEM        |
		| MemberType_CERT_HASH  | HASH       |
		| MemberType_PUBLIC_KEY | PEM        |
	*/

	var address string
	sender := txSimContext.GetSender()
	switch sender.MemberType {
	case accesscontrol.MemberType_CERT:
		address, err = getSenderAddressFromCert(sender.MemberInfo, chainConfig.GetVm().GetAddrType())
		if err != nil {
			r.Log.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
			getSenderAddressResponse.Message = err.Error()
			getSenderAddressResponse.Payload = nil
			return getSenderAddressResponse, gasUsed
		}

	case accesscontrol.MemberType_CERT_HASH:
		certHashKey := hex.EncodeToString(sender.MemberInfo)
		var certBytes []byte
		certBytes, err = txSimContext.Get(syscontract.SystemContract_CERT_MANAGE.String(), []byte(certHashKey))
		if err != nil {
			r.Log.Errorf("get cert from chain fialed, %s", err.Error())
			getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
			getSenderAddressResponse.Message = err.Error()
			getSenderAddressResponse.Payload = nil
			return getSenderAddressResponse, gasUsed
		}

		address, err = getSenderAddressFromCert(certBytes, chainConfig.GetVm().GetAddrType())
		if err != nil {
			r.Log.Errorf("getSenderAddressFromCert failed, %s", err.Error())
			getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
			getSenderAddressResponse.Message = err.Error()
			getSenderAddressResponse.Payload = nil
			return getSenderAddressResponse, gasUsed
		}

	case accesscontrol.MemberType_PUBLIC_KEY:
		address, err = getSenderAddressFromPublicKeyPEM(sender.MemberInfo, chainConfig.GetVm().GetAddrType(),
			crypto.HashAlgoMap[chainConfig.GetCrypto().Hash])
		if err != nil {
			r.Log.Errorf("getSenderAddressFromPublicKeyPEM failed, %s", err.Error())
			getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
			getSenderAddressResponse.Message = err.Error()
			getSenderAddressResponse.Payload = nil
			return getSenderAddressResponse, gasUsed
		}

	default:
		r.Log.Errorf("handleGetSenderAddress failed, invalid member type")
		getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultFail
		getSenderAddressResponse.Message = err.Error()
		getSenderAddressResponse.Payload = nil
		return getSenderAddressResponse, gasUsed
	}

	r.Log.Debug("get sender address: ", address)
	getSenderAddressResponse.ResultCode = protocol.ContractSdkSignalResultSuccess
	getSenderAddressResponse.Payload = []byte(address)

	return getSenderAddressResponse, gasUsed
}

func getSenderAddressFromCert(certPem []byte, addressType configPb.AddrType) (string, error) {
	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromCertificatePEM(certPem)
		if err != nil {
			return "", fmt.Errorf("ParseCertificate failed, %s", err.Error())
		}

		return address, nil
	} else if addressType == configPb.AddrType_ETHEREUM {
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

		return addrInt.String(), nil
	} else {
		return "", errors.New("invalid address type")
	}
}

func getSenderAddressFromPublicKeyPEM(publicKeyPem []byte, addressType configPb.AddrType,
	hashType crypto.HashType) (string, error) {
	if addressType == configPb.AddrType_ZXL {
		address, err := evmutils.ZXAddressFromPublicKeyPEM(publicKeyPem)
		if err != nil {
			return "", fmt.Errorf("ZXAddressFromPublicKeyPEM, failed, %s", err.Error())
		}
		return address, nil
	} else if addressType == configPb.AddrType_ETHEREUM {
		publicKey, err := asym.PublicKeyFromPEM(publicKeyPem)
		if err != nil {
			return "", fmt.Errorf("ParsePublicKey failed, %s", err.Error())
		}

		ski, err := commonCrt.ComputeSKI(hashType, publicKey.ToStandardKey())
		if err != nil {
			return "", fmt.Errorf("computeSKI from public key failed, %s", err.Error())
		}

		addr, err := evmutils.MakeAddressFromHex(hex.EncodeToString(ski))
		if err != nil {
			return "", fmt.Errorf("make address from cert SKI failed, %s", err)
		}
		return addr.String(), nil
	} else {
		return "", errors.New("invalid address type")
	}
}

func (r *RuntimeInstance) handleCreateKeyHistoryIterator(txId string, recvMsg *protogo.CDMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.CDMMessage, uint64) {

	createKeyHistoryIterResponse := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_CREATE_KEY_HISTORY_TER_RESPONSE)

	/*
		| index | desc          |
		| ----  | ----          |
		| 0     | contractName  |
		| 1     | key           |
		| 2     | field         |
		| 3     | writeMapCache |
	*/
	keyList := strings.SplitN(string(recvMsg.Payload), "#", 4)
	calledContractName := keyList[0]
	keyStr := keyList[1]
	field := keyList[2]
	writeMapBytes := keyList[3]

	writeMap := make(map[string][]byte)
	var err error

	gasUsed, err = gas.CreateKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.Message = err.Error()
		createKeyHistoryIterResponse.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	if err = json.Unmarshal([]byte(writeMapBytes), &writeMap); err != nil {
		r.Log.Errorf("get write map failed, %s", err.Error())
		createKeyHistoryIterResponse.Message = err.Error()
		createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	gasUsed, err = r.mergeSimContextWriteMap(txSimContext, writeMap, gasUsed)
	if err != nil {
		r.Log.Errorf("merge the sim context write map failed, %s", err.Error())
		createKeyHistoryIterResponse.Message = err.Error()
		createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	if err = protocol.CheckKeyFieldStr(keyStr, field); err != nil {
		r.Log.Errorf("invalid key field str, %s", err.Error())
		createKeyHistoryIterResponse.Message = err.Error()
		createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	key := protocol.GetKeyStr(keyStr, field)

	iter, err := txSimContext.GetHistoryIterForKey(calledContractName, key)
	if err != nil {
		createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		createKeyHistoryIterResponse.Payload = nil
		return createKeyHistoryIterResponse, gasUsed
	}

	index := atomic.AddInt32(&r.rowIndex, 1)
	txSimContext.SetIterHandle(index, iter)

	r.Log.Debug("create key history iterator: ", index)

	createKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultSuccess
	createKeyHistoryIterResponse.Payload = bytehelper.IntToBytes(index)

	return createKeyHistoryIterResponse, gasUsed
}

func (r *RuntimeInstance) handleConsumeKeyHistoryIterator(txId string, recvMsg *protogo.CDMMessage,
	txSimContext protocol.TxSimContext, gasUsed uint64) (*protogo.CDMMessage, uint64) {
	consumeKeyHistoryIterResponse := r.newEmptyResponse(txId, protogo.CDMType_CDM_TYPE_CONSUME_KEY_HISTORY_ITER_RESPONSE)

	currentGasUsed, err := gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		consumeKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.Message = err.Error()
		consumeKeyHistoryIterResponse.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	/*
		|	index	|			desc				|
		|	----	|			----  				|
		|	 0  	|	consumeKvIteratorFunc		|
		|	 1  	|		rsIndex					|
	*/

	keyList := strings.Split(string(recvMsg.Payload), "#")
	consumeKeyHistoryIteratorFunc := keyList[0]
	keyHistoryIterIndex, err := bytehelper.BytesToInt([]byte(keyList[1]))
	if err != nil {
		r.Log.Errorf("failed to get iterator index, %s", err.Error())
		consumeKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.Message = err.Error()
		consumeKeyHistoryIterResponse.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	iter, ok := txSimContext.GetIterHandle(keyHistoryIterIndex)
	if !ok {
		errMsg := fmt.Sprintf("[key history iterator consume] can not found iterator index [%d]", keyHistoryIterIndex)
		r.Log.Error(errMsg)

		consumeKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.Message = errMsg
		consumeKeyHistoryIterResponse.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	keyHistoryIterator, ok := iter.(protocol.KeyHistoryIterator)
	if !ok {
		errMsg := "assertion failed"
		r.Log.Error(errMsg)

		consumeKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.Message = errMsg
		consumeKeyHistoryIterResponse.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}

	switch consumeKeyHistoryIteratorFunc {
	case config.FuncKeyHistoryIterHasNext:
		return keyHistoryIterHasNext(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)

	case config.FuncKeyHistoryIterNext:
		return keyHistoryIterNext(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)

	case config.FuncKeyHistoryIterClose:
		return keyHistoryIterClose(keyHistoryIterator, gasUsed, consumeKeyHistoryIterResponse)
	default:
		consumeKeyHistoryIterResponse.ResultCode = protocol.ContractSdkSignalResultFail
		consumeKeyHistoryIterResponse.Message = fmt.Sprintf("%s not found", consumeKeyHistoryIteratorFunc)
		consumeKeyHistoryIterResponse.Payload = nil
		return consumeKeyHistoryIterResponse, currentGasUsed
	}
}

func keyHistoryIterHasNext(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	hasNext := config.BoolFalse
	if iter.Next() {
		hasNext = config.BoolTrue
	}

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = bytehelper.IntToBytes(int32(hasNext))

	return response, gasUsed
}

func keyHistoryIterNext(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	if iter == nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = msgIterIsNil
		response.Payload = nil
		return response, gasUsed
	}

	var historyValue *store.KeyModification
	historyValue, err = iter.Value()
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
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
	response.Payload = func() []byte {
		str := historyValue.TxId + "#" +
			string(blockHeight) + "#" +
			string(historyValue.Value) + "#" +
			string(bytehelper.IntToBytes(int32(isDelete))) + "#" +
			timestampStr
		return []byte(str)
	}()

	return response, gasUsed
}

func keyHistoryIterClose(iter protocol.KeyHistoryIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKeyHistoryIterGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	iter.Release()
	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = nil

	return response, gasUsed
}

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
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.Message = err.Error()
			consumeKvIteratorResponse.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	iter, ok := txSimContext.GetIterHandle(kvIteratorIndex)
	if !ok {
		r.Log.Errorf("[kv iterator consume] can not found iterator index [%d]", kvIteratorIndex)
		consumeKvIteratorResponse.Message = fmt.Sprintf(
			"[kv iterator consume] can not found iterator index [%d]", kvIteratorIndex,
		)
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.Message = err.Error()
			consumeKvIteratorResponse.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	kvIterator, ok := iter.(protocol.StateIterator)
	if !ok {
		r.Log.Errorf("assertion failed")
		consumeKvIteratorResponse.Message = fmt.Sprintf(
			"[kv iterator consume] failed, iterator %d assertion failed", kvIteratorIndex,
		)
		gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
		if err != nil {
			consumeKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			consumeKvIteratorResponse.Message = err.Error()
			consumeKvIteratorResponse.Payload = nil
			return consumeKvIteratorResponse, gasUsed
		}
		return consumeKvIteratorResponse, gasUsed
	}

	switch consumeKvIteratorFunc {
	case config.FuncKvIteratorHasNext:
		return kvIteratorHasNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case config.FuncKvIteratorNext:
		return kvIteratorNext(kvIterator, gasUsed, consumeKvIteratorResponse)

	case config.FuncKvIteratorClose:
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
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	hasNext := config.BoolFalse
	if kvIterator.Next() {
		hasNext = config.BoolTrue
	}

	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = bytehelper.IntToBytes(int32(hasNext))

	return response, gasUsed
}

func kvIteratorNext(kvIterator protocol.StateIterator, gasUsed uint64,
	response *protogo.CDMMessage) (*protogo.CDMMessage, uint64) {
	var err error
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
	if err != nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = err.Error()
		response.Payload = nil
		return response, gasUsed
	}

	if kvIterator == nil {
		response.ResultCode = protocol.ContractSdkSignalResultFail
		response.Message = msgIterIsNil
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
	gasUsed, err = gas.ConsumeKvIteratorGasUsed(gasUsed)
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

func (r *RuntimeInstance) mergeSimContextWriteMap(txSimContext protocol.TxSimContext,
	writeMap map[string][]byte, gasUsed uint64) (uint64, error) {
	// merge the sim context write map

	for key, value := range writeMap {
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
		|	 6  	|	  writeMapCache			|
	*/
	keyList := strings.SplitN(string(recvMsg.Payload), "#", 7)
	calledContractName := keyList[0]
	createFunc := keyList[1]
	startKey := keyList[2]
	startField := keyList[3]
	writeMapBytes := keyList[6]

	writeMap := make(map[string][]byte)
	var err error
	if err = json.Unmarshal([]byte(writeMapBytes), &writeMap); err != nil {
		r.Log.Errorf("get WriteMap failed, %s", err.Error())
		createKvIteratorResponse.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	gasUsed, err = r.mergeSimContextWriteMap(txSimContext, writeMap, gasUsed)
	if err != nil {
		r.Log.Errorf("merge the sim context write map failed, %s", err.Error())
		createKvIteratorResponse.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	if err = protocol.CheckKeyFieldStr(startKey, startField); err != nil {
		r.Log.Errorf("invalid key field str, %s", err.Error())
		createKvIteratorResponse.Message = err.Error()
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
		if err != nil {
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	}

	key := protocol.GetKeyStr(startKey, startField)

	var iter protocol.StateIterator
	switch createFunc {
	case config.FuncKvIteratorCreate:
		limitKey := keyList[4]
		limitField := keyList[5]
		iter, gasUsed, err = kvIteratorCreate(txSimContext, calledContractName, key, limitKey, limitField, gasUsed)
		if err != nil {
			r.Log.Errorf("failed to create kv iterator, %s", err.Error())
			createKvIteratorResponse.ResultCode = protocol.ContractSdkSignalResultFail
			createKvIteratorResponse.Message = err.Error()
			createKvIteratorResponse.Payload = nil
			return createKvIteratorResponse, gasUsed
		}
	case config.FuncKvPreIteratorCreate:
		gasUsed, err = gas.CreateKvIteratorGasUsed(gasUsed)
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
	txSimContext.SetIterHandle(index, iter)

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

	r.Log.Debug("get value: ", string(value))
	response.ResultCode = protocol.ContractSdkSignalResultSuccess
	response.Payload = value
	return response, true
}

func (r *RuntimeInstance) errorResult(contractResult *commonPb.ContractResult,
	err error, errMsg string) (*commonPb.ContractResult, protocol.ExecOrderTxType) {
	contractResult.Code = uint32(1)
	if err != nil {
		errMsg += ", " + err.Error()
	}
	contractResult.Message = errMsg
	r.Log.Error(errMsg)
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
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...) // #nosec

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

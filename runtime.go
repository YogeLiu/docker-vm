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
	ChainId string // chain id
	Client  CDMClient
	Log     protocol.Logger
}

// Invoke process one tx in docker and return result
func (r *RuntimeInstance) Invoke(contract *commonPb.Contract, method string,
	byteCode []byte, parameters map[string][]byte, txSimContext protocol.TxSimContext,
	gasUsed uint64) (contractResult *commonPb.ContractResult, ExecOrderTxType protocol.ExecOrderTxType) {
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

				return contractResult, protocol.ExecOrderTxTypeNormal
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
			return contractResult, protocol.ExecOrderTxTypeNormal
		default:
			contractResult.GasUsed = gasUsed
			return r.errorResult(contractResult, fmt.Errorf("unknow type"), "fail to receive request")
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

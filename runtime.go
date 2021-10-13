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

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/gas"
	"github.com/gogo/protobuf/proto"
)

const (
	mountContractDir = "contracts"
)

type CDMClient interface {
	GetTxSendCh() chan *protogo.CDMMessage

	GetStateResponseSendCh() chan *protogo.CDMMessage

	RegisterRecvChan(txId string, recvCh chan *protogo.CDMMessage)
}

// RuntimeInstance docker-go runtime
type RuntimeInstance struct {
	ChainId string // chain id
	Client  CDMClient
	Log     *logger.CMLogger
}

// Invoke process one tx in docker and return result
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
	); err != nil {
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
		case protogo.CDMType_CDM_TYPE_GET_STATE:

			getStateResponse := &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_STATE_RESPONSE,
				Payload: nil,
			}
			keyList := strings.Split(string(recvMsg.Payload), "#")
			calledContractName := keyList[0]
			calledContractKey := keyList[1]
			value, err := txSimContext.Get(calledContractName, []byte(calledContractKey))
			if err != nil {
				// if it has error, return payload is nil
				r.Log.Errorf("fail to get state from sim context: %s", err)
				r.Client.GetStateResponseSendCh() <- getStateResponse
				continue
			}

			r.Log.Debug("get value: ", string(value))
			getStateResponse.Payload = value

			// get state gas used calc and check gas limit
			gasUsed, err = gas.GetStateGasUsed(gasUsed, value)
			if err != nil {
				contractResult.GasUsed = gasUsed
				return r.errorResult(contractResult, err, err.Error())
			}

			r.Client.GetStateResponseSendCh() <- getStateResponse

		case protogo.CDMType_CDM_TYPE_GET_BYTECODE:

			getBytecodeResponse := &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE_RESPONSE,
				Payload: nil,
			}

			contractFullName := string(recvMsg.Payload)             // contract1#1.0.0
			contractName := strings.Split(contractFullName, "#")[0] // contract1

			// init contract gas used calc and check gas limit
			//gasUsed, err = gas.ContractGasUsed(gasUsed, method, contractName, byteCode, txSimContext)
			//if err != nil {
			//	contractResult.GasUsed = gasUsed
			//	return r.errorResult(contractResult, err, err.Error())
			//}

			dockerConfig := localconf.ChainMakerConfig.DockerConfig
			hostMountPath := dockerConfig.MountPath

			contractDir := filepath.Join(hostMountPath, mountContractDir)
			contractZipPath := filepath.Join(contractDir, fmt.Sprintf("%s.7z", contractName)) // contract1.7z
			contractPathWithoutVersion := filepath.Join(contractDir, contractName)
			contractPathWithVersion := filepath.Join(contractDir, contractFullName)

			// save bytecode to disk
			err := r.saveBytesToDisk(byteCode, contractZipPath)
			if err != nil {
				r.Log.Errorf("fail to save bytecode to disk: %s", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}
			r.Log.Debug("write zip file: ", contractZipPath)

			// extract 7z file
			unzipCommand := fmt.Sprintf("7z e %s -o%s -y", contractZipPath, contractDir) // contract1
			err = r.runCmd(unzipCommand)
			if err != nil {
				r.Log.Errorf("fail to extract contract: %s", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}
			r.Log.Debug("extract zip file: ", contractZipPath)

			// remove 7z file
			err = os.Remove(contractZipPath)
			if err != nil {
				contractResult.GasUsed = gasUsed
				return r.errorResult(contractResult, err, "fail to remove zipped file")
			}

			// replace contract name to contractName:version
			err = os.Rename(contractPathWithoutVersion, contractPathWithVersion)
			if err != nil {
				r.Log.Errorf("fail to rename original file name: %s, "+
					"please make sure contract name should be same as zipped file", err)
				r.Client.GetStateResponseSendCh() <- getBytecodeResponse
				continue
			}

			getBytecodeResponse.Payload = []byte(contractFullName)

			r.Client.GetStateResponseSendCh() <- getBytecodeResponse

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
				keyList := strings.Split(key, "#")
				calledContractName := keyList[0]
				calledContractKey := keyList[1]
				// put state gas used calc and check gas limit
				gasUsed, err = gas.PutStateGasUsed(gasUsed, calledContractName, calledContractKey, value)
				if err != nil {
					contractResult.GasUsed = gasUsed
					return r.errorResult(contractResult, err, err.Error())
				}

				err := txSimContext.Put(calledContractName, []byte(calledContractKey), value)
				if err != nil {
					contractResult.GasUsed = gasUsed
					return r.errorResult(contractResult, err, "fail to put in sim context")
				}
			}

			// merge events
			var contractEvents []*commonPb.ContractEvent

			if len(txResponse.Events) > protocol.EventDataMaxCount-1 {
				err := fmt.Errorf("too many event data")
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
		default:
			contractResult.GasUsed = gasUsed
			return r.errorResult(contractResult, fmt.Errorf("unknow type"), "fail to receive request")

		}

	}

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
	cmd := exec.Command(commands[0], commands[1:]...)

	if err := cmd.Start(); err != nil {
		return err
	}

	return cmd.Wait()
}

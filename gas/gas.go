/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package gas

import (
	"encoding/json"
	"errors"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/utils/v2"
)

const (
	// function list gas price
	GetArgsGasPrice               uint64 = 1
	GetStateGasPrice              uint64 = 1
	PutStateGasPrice              uint64 = 10
	DelStateGasPrice              uint64 = 10
	EmitEventGasPrice             uint64 = 5
	LogGasPrice                   uint64 = 5
	KvIteratorCreateGasPrice      uint64 = 1
	KvPreIteratorCreateGasPrice   uint64 = 1
	KvIteratorHasNextGasPrice     uint64 = 1
	KvIteratorNextGasPrice        uint64 = 1
	KvIteratorCloseGasPrice       uint64 = 1
	KeyHistoryIterCreateGasPrice  uint64 = 1
	KeyHistoryIterHasNextGasPrice uint64 = 1
	KeyHistoryIterNextGasPrice    uint64 = 1
	KeyHistoryIterCloseGasPrice   uint64 = 1
	GetSenderAddressGasPrice      uint64 = 1

	// method
	initContract    = "init_contract"
	upgradeContract = "upgrade"

	// default gas used
	defaultGas uint64 = 100000

	// init function gas used
	initFuncGas uint64 = 1250
)

func GetArgsGasUsed(gasUsed uint64, args map[string]string) (uint64, error) {
	argsBytes, err := json.Marshal(args)
	if err != nil {
		return 0, err
	}
	gasUsed += uint64(len(argsBytes)) * GetArgsGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func GetSenderAddressGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * GetSenderAddressGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func CreateKeyHistoryIterGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KeyHistoryIterCreateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKeyHistoryIterGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KeyHistoryIterHasNextGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func CreateKvIteratorGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KvIteratorCreateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}
	return gasUsed, nil
}

func ConsumeKvIteratorGasUsed(gasUsed uint64) (uint64, error) {
	gasUsed += 10 * KvIteratorNextGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited")
	}

	return gasUsed, nil
}

func GetStateGasUsed(gasUsed uint64, value []byte) (uint64, error) {
	gasUsed += uint64(len(value)) * GetStateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func PutStateGasUsed(gasUsed uint64, contractName, key, field string, value []byte) (uint64, error) {
	gasUsed += (uint64(len(value)) + uint64(len([]byte(contractName+key+field)))) * PutStateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func DelStateGasUsed(gasUsed uint64, value []byte) (uint64, error) {
	gasUsed += uint64(len(value)) * DelStateGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func EmitEventGasUsed(gasUsed uint64, contractEvent *common.ContractEvent) (uint64, error) {
	contractEventBytes, err := json.Marshal(contractEvent)
	if err != nil {
		return 0, err
	}

	gasUsed += uint64(len(contractEventBytes)) * EmitEventGasPrice
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func InitFuncGasUsed(gasUsed, configDefaultGas uint64) (uint64, error) {
	gasUsed = getInitFuncGasUsed(gasUsed, configDefaultGas)
	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}

	return gasUsed, nil

}

func ContractGasUsed(gasUsed uint64, method string, contractName string, byteCode []byte) (uint64, error) {
	if method == initContract {
		gasUsed += (uint64(len([]byte(contractName+utils.PrefixContractByteCode))) +
			uint64(len(byteCode))) * PutStateGasPrice
	}

	if method == upgradeContract {
		gasUsed += uint64(len(byteCode)) * PutStateGasPrice
	}

	if CheckGasLimit(gasUsed) {
		return 0, errors.New("over gas limited ")
	}
	return gasUsed, nil
}

func getInitFuncGasUsed(gasUsed, configDefaultGas uint64) uint64 {
	// if config not set default gas
	if configDefaultGas == 0 {
		return gasUsed + defaultGas + initFuncGas
	} else {
		return gasUsed + configDefaultGas + initFuncGas
	}
}

func CheckGasLimit(gasUsed uint64) bool {
	return gasUsed > protocol.GasLimit
}

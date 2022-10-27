/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"fmt"
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDockerGoCrossCall(t *testing.T) {
	setupTest(t)

	// success test
	parameters0 := generateInitParams()
	parameters0["contract_name"] = []byte(ContractNameTest)
	parameters0["contract_method"] = []byte("display")

	contractInfo := commonPb.Contract{
		Name:        ContractNameTest,
		RuntimeType: commonPb.RuntimeType_GO,
		Address:     ContractNameAddr,
	}

	invalidContractInfo := commonPb.Contract{
		Name:        "",
		RuntimeType: commonPb.RuntimeType_INVALID,
		Address:     "",
	}

	mockTxContext2 := initMockSimContext(t)
	mockGetLastChainConfig(mockTxContext2)
	mockCrossCallGetDepth(mockTxContext2)
	mockCrossCallGetCrossInfo(mockTxContext2)

	mockCallContract(mockTxContext2, parameters0)
	mockTxContext2.EXPECT().GetTxRWMapByContractName(gomock.Any()).Return(nil, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName(ContractNameTest).Return(&contractInfo, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName("").Return(&invalidContractInfo, nil).AnyTimes()

	result, _ := mockRuntimeInstance.Invoke(mockContractId, methodCrossContract, nil,
		parameters0, mockTxContext2, uint64(123))
	fmt.Printf("result -> [%+v]", result)
	assert.Equal(t, uint32(0), result.Code)
	assert.Equal(t, []byte("display successful"), result.Result)

	//// called contract out of range
	//parameters1 := generateInitParams()
	//parameters1["method"] = []byte("cross_contract")
	//parameters1["contract_name"] = []byte(ContractNameTest)
	//parameters1["contract_version"] = []byte("v1.0.0")
	//parameters1["contract_method"] = []byte("out_of_range")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters1, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//assert.Equal(t, []byte("cross contract runtime panic"), result.Result)
	//
	//fmt.Println("========== start testing missing contract name ============== ")
	//// missing contract name
	//parameters2 := generateInitParams()
	//parameters2["method"] = []byte("cross_contract")
	//parameters2["contract_name"] = []byte("")
	//parameters2["contract_version"] = []byte("v1.0.0")
	//parameters2["contract_method"] = []byte("display")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters2, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//assert.Equal(t, []byte("missing contract name"), result.Result)
	////
	//
	//fmt.Println("========== start testing missing contract version ============== ")
	//// missing contract version
	//parameters3 := generateInitParams()
	//parameters3["method"] = []byte("cross_contract")
	//parameters3["contract_name"] = []byte(ContractNameTest)
	//parameters3["contract_version"] = []byte("")
	//parameters3["contract_method"] = []byte("display")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters3, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//assert.Equal(t, []byte("missing contact version"), result.Result)
	//
	//// missing contract method
	//parameters4 := generateInitParams()
	//parameters4["method"] = []byte("cross_contract")
	//parameters4["contract_name"] = []byte(ContractNameTest)
	//parameters4["contract_version"] = []byte("v1.0.0")
	//parameters4["contract_method"] = []byte("")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters4, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//assert.Equal(t, []byte("unknown method"), result.Result)
	//
	//// wrong contract method
	//parameters5 := generateInitParams()
	//parameters5["method"] = []byte("cross_contract")
	//parameters5["contract_name"] = []byte(ContractNameTest)
	//parameters5["contract_version"] = []byte("v1.0.0")
	//parameters5["contract_method"] = []byte("random method")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters5, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//assert.Equal(t, []byte("unknown method"), result.Result)
	//
	//// call contract self
	//parameters6 := generateInitParams()
	//parameters6["method"] = []byte("cross_contract_self")
	//
	//result, _ = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
	//	parameters6, mockTxContext, uint64(123))
	//assert.Equal(t, uint32(1), result.Code)
	//////assert.Equal(t, []byte("exceed max depth"), result.Result)

	tearDownTest()

}

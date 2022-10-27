/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"testing"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDockerGoGetSenderAddr(t *testing.T) {
	setupTest(t)

	simContext := initMockSimContext(t)
	mockGetLastChainConfig(simContext)
	mockTxQueryCertFromChain(simContext)
	mockGetSender(simContext)
	mockGetStrAddrFromPbMember(simContext)
	// mockTxGetChainConf(simContext)
	mockGetBlockVersion(simContext)
	mockNormalGetrossInfo(simContext)
	mockNormalGetDepth(simContext)

	testData := []struct {
		/*
			| MemberType            | AddrType            |
			| ---                   | ---                 |
			| MemberType_CERT       | AddrType_ZXL        |
			| MemberType_CERT_HASH  | AddrType_ZXL        |
			| MemberType_PUBLIC_KEY | AddrType_ZXL        |
			| MemberType_ALIAS 		| AddrType_ZXL        |
			| MemberType_CERT       | AddrType_CHAINMAKER |
			| MemberType_CERT_HASH  | AddrType_CHAINMAKER |
			| MemberType_PUBLIC_KEY | AddrType_CHAINMAKER |
			| MemberType_ALIAS 		| AddrType_CHAINMAKER |
		*/
		wantAddr string
	}{
		{zxlCertAddressFromCert},
		// {zxlCertAddressFromCert},
		{zxlPKAddress},
		// {zxlCertAddressFromCert},

		{cmCertAddressFromCert},
		// {cmCertAddressFromCert},
		{cmPKAddress},
		// {cmCertAddressFromCert},
	}

	parameters := generateInitParams()
	method := "get_sender_address"

	for index, data := range testData {
		result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
			parameters, simContext, uint64(123))
		assert.Equal(t, uint32(0), result.GetCode())
		assert.Equal(t, data.wantAddr, string(result.GetResult()))
		t.Logf("addr[%d] : [%s]", index, result.GetResult())
	}

	tearDownTest()
}

func TestDockerGoGetCrossSender(t *testing.T) {
	setupTest(t)

	// success test
	parameters0 := generateInitParams()
	parameters0["contract_name"] = []byte(ContractNameTest)
	parameters0["contract_method"] = []byte("get_sender")

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
	assert.Equal(t, uint32(0), result.Code)
	assert.Equal(t, []byte(ContractNameAddr), result.Result)

	tearDownTest()
}

func TestDockerGoGetCrossSenderAddr(t *testing.T) {
	setupTest(t)

	// success test
	parameters0 := generateInitParams()
	parameters0["contract_name"] = []byte(ContractNameTest)
	parameters0["contract_method"] = []byte("get_sender_address")

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

	mockTxQueryCertFromChain(mockTxContext2)
	mockGetSender(mockTxContext2)
	mockGetStrAddrFromPbMember(mockTxContext2)

	mockCallContract(mockTxContext2, parameters0)
	mockTxContext2.EXPECT().GetTxRWMapByContractName(gomock.Any()).Return(nil, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName(ContractNameTest).Return(&contractInfo, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName("").Return(&invalidContractInfo, nil).AnyTimes()

	result, _ := mockRuntimeInstance.Invoke(mockContractId, methodCrossContract, nil,
		parameters0, mockTxContext2, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Equal(t, []byte(zxlCertAddressFromCert), result.Result)

	tearDownTest()
}

func TestDockerGoGetCrossOrigin(t *testing.T) {
	setupTest(t)

	// success test
	parameters0 := generateInitParams()
	parameters0["contract_name"] = []byte(ContractNameTest)
	parameters0["contract_method"] = []byte("get_origin")

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

	mockTxQueryCertFromChain(mockTxContext2)
	mockGetSender(mockTxContext2)
	mockGetStrAddrFromPbMember(mockTxContext2)

	mockCallContract(mockTxContext2, parameters0)
	mockTxContext2.EXPECT().GetTxRWMapByContractName(gomock.Any()).Return(nil, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName(ContractNameTest).Return(&contractInfo, nil).AnyTimes()
	mockTxContext2.EXPECT().GetContractByName("").Return(&invalidContractInfo, nil).AnyTimes()

	result, _ := mockRuntimeInstance.Invoke(mockContractId, methodCrossContract, nil,
		parameters0, mockTxContext2, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Equal(t, []byte(zxlPKAddress), result.Result)

	tearDownTest()
}

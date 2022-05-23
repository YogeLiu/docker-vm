/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDockerGoGetSenderAddr(t *testing.T) {
	setupTest(t)
	mockTxQueryCertFromChain(mockTxContext)
	mockGetSender(mockTxContext)
	mockTxGetChainConf(mockTxContext)

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
		{zxlCertAddressFromCert},
		{zxlPKAddress},
		{zxlCertAddressFromCert},
		{cmCertAddressFromCert},
		{cmCertAddressFromCert},
		{cmPKAddress},
		{cmCertAddressFromCert},
	}

	parameters := generateInitParams()
	parameters["method"] = []byte("get_sender_address")

	for index, data := range testData {
		result, _ := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
			parameters, mockTxContext, uint64(123))
		assert.Equal(t, uint32(0), result.GetCode())
		assert.Equal(t, data.wantAddr, string(result.GetResult()))
		t.Logf("addr[%d] : [%s]", index, result.GetResult())
	}

	tearDownTest()
}

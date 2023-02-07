/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/assert"
)

const (
	methodPutState = "put_state"
	methodGetState = "get_state"
	fixEmptyKey    = "fix_empty_key"
)

// put state and delete state testing
func TestDockerGoPutState(t *testing.T) {
	setupTest(t)
	method := methodPutState
	parameters := generateInitParams()
	parameters["key"] = []byte("key1")
	parameters["field"] = []byte("field1")
	parameters["value"] = []byte("500")

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key1"), []byte("field1")), []byte("500"))
	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Contains(t, tmpSimContextMap, fmt.Sprintf("%s::key1#field1", ContractNameTest))

	parameters1 := generateInitParams()
	parameters1["key"] = []byte("key2")
	parameters1["field"] = []byte("field2")
	parameters1["value"] = []byte("500")
	method = "put_state_byte"

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key2"), []byte("field2")), []byte("500"))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters1, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Contains(t, tmpSimContextMap, fmt.Sprintf("%s::key2#field2", ContractNameTest))

	parameters2 := generateInitParams()
	parameters2["key"] = []byte("key3")
	parameters2["value"] = []byte("300")
	parameters2["method"] = []byte("put_state_from_key_byte")
	method = invokeMethod

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key3"), nil), []byte("300"))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters2, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Contains(t, tmpSimContextMap, fmt.Sprintf("%s::key3", ContractNameTest))
	value, ok := tmpSimContextMap[fmt.Sprintf("%s::key3", ContractNameTest)]
	assert.True(t, ok)
	assert.Equal(t, []byte("300"), value)

	parameters3 := generateInitParams()
	parameters3["method"] = []byte("put_state_from_key_byte")
	parameters3["key"] = []byte("key4")
	parameters3["value"] = []byte("400")
	parameters3["method"] = []byte("put_state_from_key_byte")

	method = invokeMethod

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key4"), nil), []byte("400"))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters3, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	value, ok = tmpSimContextMap[fmt.Sprintf("%s::key4", ContractNameTest)]
	assert.True(t, ok)
	assert.Equal(t, []byte("400"), value)

	parameters4 := generateInitParams()
	parameters4["key"] = []byte(fixEmptyKey)
	parameters4["field"] = []byte("")
	parameters4["value"] = []byte("500")
	method = methodPutState

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte(fixEmptyKey), []byte("")), []byte("500"))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters4, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	value, ok = tmpSimContextMap[fmt.Sprintf("%s::%s", ContractNameTest, fixEmptyKey)]
	assert.True(t, ok)
	assert.Equal(t, []byte("500"), value)

	parameters5 := generateInitParams()
	parameters5["key"] = []byte("key1")
	parameters5["field"] = []byte("field1")
	method = methodPutState

	generateValue := func(size int) string {
		var sb strings.Builder
		for i := 0; i < size; i++ {
			sb.WriteString("a")
		}
		return sb.String()
	}
	fiveMData := generateValue(4000000)
	parameters5["value"] = []byte(fiveMData)

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key1"), []byte("field1")), []byte(fiveMData))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters5, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)

	tearDownTest()
}

func TestDockerGoGetState(t *testing.T) {
	setupTest(t)
	parameters := generateInitParams()
	parameters["key"] = []byte("key1")
	parameters["field"] = []byte("field1")
	parameters["value"] = []byte("500")
	method := methodPutState
	mockTxContext.EXPECT().GetBlockVersion().DoAndReturn(
		GetBlockVersion,
	).AnyTimes()

	mockPut(mockTxContext, ContractNameTest, protocol.GetKey([]byte("key1"), []byte("field1")), []byte("500"))
	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters, mockTxContext, uint64(123))
	fmt.Println(result)
	assert.Equal(t, uint32(0), result.Code)
	assert.Contains(t, tmpSimContextMap, fmt.Sprintf("%s::key1#field1", ContractNameTest))

	parameters6 := generateInitParams()
	parameters6["key"] = []byte("key1")
	parameters6["field"] = []byte("field1")
	method = methodGetState

	mockTxContext.EXPECT().Get(ContractNameTest, protocol.GetKey([]byte("key1"), []byte("field1"))).
		Return([]byte("500"), nil)
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters6, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Equal(t, []byte("500"), result.Result)

	parameters7 := generateInitParams()
	parameters7["key"] = []byte("key11111")
	parameters7["field"] = []byte("field1")
	method = methodGetState

	mockTxContext.EXPECT().Get(ContractNameTest, protocol.GetKey([]byte("key11111"), []byte("field1"))).
		Return([]byte(""), nil)
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters7, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Nil(t, result.Result)

	parameters8 := generateInitParams()
	parameters8["key"] = []byte(fixEmptyKey)
	parameters8["field"] = []byte("field1")
	method = methodGetState

	mockTxContext.EXPECT().Get(ContractNameTest, protocol.GetKey([]byte(fixEmptyKey), []byte("field1"))).Return([]byte(""), nil)
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters8, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	assert.Nil(t, result.Result)

	parameters9 := generateInitParams()
	parameters9["key"] = []byte("key4")
	parameters9["field"] = []byte("field4")
	method = methodGetState

	mockTxContext.EXPECT().Get(ContractNameTest, protocol.GetKey([]byte("key4"), []byte("field4"))).
		Return([]byte(""), errors.New("simContext fail"))
	result, _ = mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters9, mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Equal(t, []byte("simContext fail"), result.Result)

	tearDownTest()
}

func TestDockerGoTimeout(t *testing.T) {
	setupTest(t)

	parameters0 := generateInitParams()
	parameters0["method"] = []byte("time_out")
	result, _ := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters0, mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Nil(t, result.Result)
	fmt.Printf("%+v", result)
	assert.Equal(t, "tx time out", result.Message)
	assert.Nil(t, result.ContractEvent)
	tearDownTest()
}

func TestDockerGoOutRange(t *testing.T) {
	setupTest(t)

	parameters0 := generateInitParams()
	parameters0["method"] = []byte("out_of_range")
	method := "out_of_range"
	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters0, mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Nil(t, result.Result)
	assert.Equal(t, "runtime panic", result.Message)
	assert.Nil(t, result.ContractEvent)
	tearDownTest()

}

func TestDockerGoInvalidMethod(t *testing.T) {
	setupTest(t)

	fmt.Println("======= testing ======== ")

	parameters0 := generateInitParams()
	parameters0["method"] = []byte("fake_method")
	method := "FakeMethod"

	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters0, mockTxContext, uint64(123))
	assert.Equal(t, uint32(1), result.Code)
	assert.Equal(t, []byte("unknown method"), result.Result)
	fmt.Println(result)

	tearDownTest()
}

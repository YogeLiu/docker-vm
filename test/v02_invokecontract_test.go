package test

//
//import (
//	"errors"
//	"fmt"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//)
//
//// put state and delete state testing
//func TestDockerGoPutState(t *testing.T) {
//	setupTest(t)
//	parameters := generateInitParams()
//	parameters["method"] = []byte("put_state")
//	parameters["key"] = []byte("key1")
//	parameters["value"] = []byte("500")
//
//	mockPut(mockTxContext, ContractNameTest, []byte("key1"), []byte("500"))
//	result := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters, mockTxContext, uint64(123))
//	fmt.Println(result)
//	assert.Equal(t, uint32(0), result.Code)
//	assert.Contains(t, tmpSimContextMap, fmt.Sprintf("%s::key1", ContractNameTest))
//
//	parameters1 := generateInitParams()
//	parameters1["method"] = []byte("put_state")
//	parameters1["key"] = []byte("")
//	parameters1["value"] = []byte("500")
//	mockPut(mockTxContext, ContractNameTest, []byte(""), []byte("500"))
//	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters1, mockTxContext, uint64(123))
//	fmt.Println(result)
//	assert.Equal(t, uint32(0), result.Code)
//	_, ok := tmpSimContextMap[fmt.Sprintf("%s::", ContractNameTest)]
//	assert.True(t, ok)
//
//	parameters2 := generateInitParams()
//	parameters2["method"] = []byte("put_state")
//	parameters2["key"] = []byte("key2")
//	parameters2["value"] = []byte("")
//	mockPut(mockTxContext, ContractNameTest, []byte("key2"), []byte(""))
//	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters2, mockTxContext, uint64(123))
//	assert.Equal(t, uint32(0), result.Code)
//	newKey := fmt.Sprintf("%s::key2", ContractNameTest)
//	_, ok = tmpSimContextMap[newKey]
//	assert.True(t, ok)
//
//	//writeMapLength := len(tmpSimContextMap)
//	//parameters3 := generateInitParams()
//	//parameters3["method"] = []byte("del_state")
//	//parameters3["get_state_key"] = []byte("key2")
//	//mockDel(mockTxContext, ContractNameTest, []byte("key2"))
//	//result := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil, parameters3, mockTxContext, uint64(123))
//	//assert.Equal(t, uint32(0), result.Code)
//	//assert.Equal(t, writeMapLength, len(tmpSimContextMap))
//	//assert.Equal(t, []byte{}, tmpSimContextMap[fmt.Sprintf("%s::key2", ContractNameTest)])
//	tearDownTest()
//}
//
//func TestDockerGoGetState(t *testing.T) {
//	setupTest(t)
//
//	parameters0 := generateInitParams()
//	parameters0["method"] = []byte("get_state")
//	parameters0["get_state_key"] = []byte("test_key1")
//	mockTxContext.EXPECT().Get(ContractNameTest, []byte("test_key1")).Return([]byte("100"), nil)
//	result := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters0, mockTxContext, uint64(123))
//	assert.Equal(t, uint32(0), result.Code)
//	assert.Equal(t, []byte("100"), result.Result)
//
//	parameters1 := generateInitParams()
//	parameters1["method"] = []byte("get_state")
//	parameters1["get_state_key"] = []byte("key11111")
//	mockTxContext.EXPECT().Get(ContractNameTest, []byte("key11111")).Return([]byte(""), nil)
//	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters1, mockTxContext, uint64(123))
//	assert.Equal(t, uint32(1), result.Code)
//	assert.Equal(t, "Fail", result.Message)
//
//	parameters2 := generateInitParams()
//	parameters2["method"] = []byte("get_state")
//	parameters2["get_state_key"] = []byte("")
//	mockTxContext.EXPECT().Get(ContractNameTest, []byte("")).Return([]byte(""), nil)
//	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters2, mockTxContext, uint64(123))
//	assert.Equal(t, uint32(1), result.Code)
//	assert.Equal(t, "Fail", result.Message)
//
//	parameters3 := generateInitParams()
//	parameters3["method"] = []byte("get_state")
//	parameters3["get_state_key"] = []byte("key4")
//	mockTxContext.EXPECT().Get(ContractNameTest, []byte("key4")).Return([]byte(""), errors.New("simContext fail"))
//	result = mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
//		parameters3, mockTxContext, uint64(123))
//	assert.Equal(t, uint32(1), result.Code)
//	assert.Equal(t, []byte("simContext fail"), result.Result)
//
//	tearDownTest()
//}

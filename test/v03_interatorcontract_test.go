package test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"chainmaker.org/chainmaker/common/v2/serialize"

	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/common/v2/bytehelper"
	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/stretchr/testify/assert"
)

// create kvIterator and consume kvIterator testing
func TestDockerGoKvIterator1(t *testing.T) {
	setupTest(t)

	// test data
	iteratorWSets, _ = makeStringKeyMap()

	// create iter
	testCreateKvIterator(t)

	// consume iter with next
	testConsumeKvIteratorWithNext(t)

	resetKvIteratorCacheAndIndex()

	tearDownTest()
}

func TestDockerGoKvIterator2(t *testing.T) {
	setupTest(t)

	// test data
	iteratorWSets, _ = makeStringKeyMap()

	// create iter
	testCreateKvIterator(t)

	// consume iter with nextRow
	testConsumeKvIteratorWithNextRow(t)

	resetKvIteratorCacheAndIndex()

	tearDownTest()
}

func testCreateKvIterator(t *testing.T) {
	// NewIterator 1
	startKey1 := protocol.GetKeyStr("key2", "")
	limit1 := protocol.GetKeyStr("key4", "")
	mockSelect(mockTxContext, ContractNameTest, startKey1, limit1)
	mockTxContext.EXPECT().SetStateKvHandle(gomock.Any(), gomock.Any()).DoAndReturn(
		func(iteratorIndex int32, iterator protocol.StateIterator) {
			kvRowCache[atomic.AddInt32(&kvSetIndex, int32(1))] = iterator
		},
	).AnyTimes()

	parameters1 := generateInitParams()
	parameters1["method"] = []byte("new_iterator")
	parameters1["key"] = []byte("key2")
	parameters1["limit"] = []byte("key4")
	result1 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters1, mockTxContext, uint64(123))
	fmt.Println(result1)
	assert.Equal(t, uint32(0), result1.Code)
	index1, err := bytehelper.BytesToInt(result1.Result)
	assert.Nil(t, err)
	assert.Equal(t, int32(1), index1)

	// NewIteratorWithField 2
	startKey2 := protocol.GetKeyStr("key1", "field1")
	limit2 := protocol.GetKeyStr("key1", "field3")
	mockSelect(mockTxContext, ContractNameTest, startKey2, limit2)

	parameters2 := generateInitParams()
	parameters2["method"] = []byte("new_iterator_with_field")
	parameters2["key"] = []byte("key1")
	parameters2["field"] = []byte("field1")
	parameters2["limit"] = []byte("field3")
	result2 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters2, mockTxContext, uint64(123))
	fmt.Println(result2)
	assert.Equal(t, uint32(0), result2.Code)

	// NewIteratorPrefixWithKey 3
	startKey3 := protocol.GetKeyStr("key3", "")
	keyStr3 := string(startKey3)
	limitLast3 := keyStr3[len(keyStr3)-1] + 1
	limit3 := keyStr3[:len(keyStr3)-1] + string(limitLast3)
	mockSelect(mockTxContext, ContractNameTest, startKey3, []byte(limit3))

	parameters3 := generateInitParams()
	parameters3["method"] = []byte("new_iterator_prefix_with_key")
	parameters3["key"] = []byte("key3")
	result3 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters3, mockTxContext, uint64(123))
	fmt.Println(result3)
	assert.Equal(t, uint32(0), result3.Code)

	// NewIteratorPrefixWithKeyField 4
	startKey4 := protocol.GetKeyStr("key1", "field2")
	keyStr4 := string(startKey4)
	limitLast4 := keyStr4[len(keyStr4)-1] + 1
	limit4 := keyStr4[:len(keyStr4)-1] + string(limitLast4)
	mockSelect(mockTxContext, ContractNameTest, startKey4, []byte(limit4))

	parameters4 := generateInitParams()
	parameters4["method"] = []byte("new_iterator_prefix_with_key_field")
	parameters4["key"] = []byte("key1")
	parameters4["field"] = []byte("field2")
	result4 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters4, mockTxContext, uint64(123))
	fmt.Println(result4)
	assert.Equal(t, uint32(0), result4.Code)

	// consume kvIterator
	mockGetStateKvHandle(mockTxContext, int32(1))
	mockGetStateKvHandle(mockTxContext, int32(2))
	mockGetStateKvHandle(mockTxContext, int32(3))
	mockGetStateKvHandle(mockTxContext, int32(4))
}

func testConsumeKvIteratorWithNext(t *testing.T) {
	/*
		for {
			if hasNext {
				next
			} else {
				break
			}
		}
	*/
	for index := int32(1); index < 5; index++ {
		fmt.Println("===")
		fmt.Printf("=== iterator [%d] start\n", index)
		fmt.Println("===")
		for {
			// HasNext
			parameters5 := generateInitParams()
			parameters5["method"] = []byte("iterator_has_next")
			parameters5["index"] = bytehelper.IntToBytes(index)
			result5 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
				parameters5, mockTxContext, uint64(123))
			assert.Equal(t, uint32(0), result5.Code)
			fmt.Printf("=== result: %+v\n", *result5)

			has, err := bytehelper.BytesToInt(result5.Result)
			fmt.Printf("=== HasNext: %d\n", has)
			assert.Nil(t, err)
			if has != boolTrue {
				// Close
				parameters5 := generateInitParams()
				parameters5["method"] = []byte("iterator_release")
				parameters5["index"] = bytehelper.IntToBytes(index)
				result5 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
					parameters5, mockTxContext, uint64(123))
				assert.Equal(t, uint32(0), result5.Code)
				break
			}

			// Next
			parameters6 := generateInitParams()
			parameters6["method"] = []byte("iterator_next")
			parameters6["index"] = bytehelper.IntToBytes(index)
			result6 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
				parameters6, mockTxContext, uint64(123))
			assert.Equal(t, uint32(0), result6.Code)
			fmt.Printf("Next key, field, value : [%s]\n", result6.Result)
		}
	}
}

func testConsumeKvIteratorWithNextRow(t *testing.T) {
	/*
		for {
			if hasNext {
				nextRow
			} else {
				break
			}
		}
	*/
	for index := int32(1); index < 5; index++ {
		fmt.Println("===")
		fmt.Printf("=== iterator [%d] start\n", index)
		fmt.Println("===")
		for {
			// HasNext
			parameters5 := generateInitParams()
			parameters5["method"] = []byte("iterator_has_next")
			parameters5["index"] = bytehelper.IntToBytes(index)
			result5 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
				parameters5, mockTxContext, uint64(123))
			assert.Equal(t, uint32(0), result5.Code)
			fmt.Printf("=== result: %+v\n", *result5)

			has, err := bytehelper.BytesToInt(result5.Result)
			fmt.Printf("=== HasNext: %d\n", has)
			assert.Nil(t, err)
			if has == boolFalse {
				// Close
				parameters5 := generateInitParams()
				parameters5["method"] = []byte("iterator_release")
				parameters5["index"] = bytehelper.IntToBytes(index)
				result5 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
					parameters5, mockTxContext, uint64(123))
				assert.Equal(t, uint32(0), result5.Code)
				break
			}

			// NextRow
			parameters7 := generateInitParams()
			parameters7["method"] = []byte("iterator_next_row")
			parameters7["index"] = bytehelper.IntToBytes(index)
			result7 := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
				parameters7, mockTxContext, uint64(123))
			assert.Equal(t, uint32(0), result7.Code)
			element := *serialize.NewEasyCodecWithBytes(result7.Result)
			key, err := element.GetString("key")
			assert.Nil(t, err)
			field, err := element.GetString("field")
			assert.Nil(t, err)
			value, err := element.GetBytes("value")
			assert.Nil(t, err)
			fmt.Printf("NextRow element key: [%s]\n", key)
			fmt.Printf("NextRow element field: [%s]\n", field)
			fmt.Printf("NextRow element value: [%s]\n", value)
		}
	}
}

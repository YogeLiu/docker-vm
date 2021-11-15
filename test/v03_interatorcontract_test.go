package test

import (
	"fmt"
	"sync/atomic"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDockerGoKvIterator(t *testing.T) {
	setupTest(t)

	// test data
	iteratorWSets, _ = makeStringKeyMap()

	// NewIterator 1
	startKey1 := protocol.GetKeyStr("key2", "")
	limit1 := protocol.GetKeyStr("key4", "")
	mockSelect(mockTxContext, ContractNameTest, startKey1, limit1)
	mockTxContext.EXPECT().SetStateKvHandle(gomock.Any(), gomock.Any()).DoAndReturn(
		func(iteratorIndex int32, iterator protocol.StateIterator) {
			kvRowCache[atomic.AddInt32(&kvSetIndex, int32(1))] = iterator
		},
	).AnyTimes()

	// NewIteratorWithField 2
	startKey2 := protocol.GetKeyStr("key1", "field1")
	limit2 := protocol.GetKeyStr("key1", "field3")
	mockSelect(mockTxContext, ContractNameTest, startKey2, limit2)

	// NewIteratorPrefixWithKey 3
	startKey3 := protocol.GetKeyStr("key3", "")
	keyStr3 := string(startKey3)
	limitLast3 := keyStr3[len(keyStr3)-1] + 1
	limit3 := keyStr3[:len(keyStr3)-1] + string(limitLast3)
	mockSelect(mockTxContext, ContractNameTest, startKey3, []byte(limit3))

	// NewIteratorPrefixWithKeyField 4
	startKey4 := protocol.GetKeyStr("key1", "field2")
	keyStr4 := string(startKey4)
	limitLast4 := keyStr4[len(keyStr4)-1] + 1
	limit4 := keyStr4[:len(keyStr4)-1] + string(limitLast4)
	mockSelect(mockTxContext, ContractNameTest, startKey4, []byte(limit4))

	// consume kvIterator
	mockGetStateKvHandle(mockTxContext, int32(1))
	mockGetStateKvHandle(mockTxContext, int32(2))
	mockGetStateKvHandle(mockTxContext, int32(3))
	mockGetStateKvHandle(mockTxContext, int32(4))

	parameters := generateInitParams()
	parameters["method"] = []byte("kv_iterator_test")
	result := mockRuntimeInstance.Invoke(mockContractId, invokeMethod, nil,
		parameters, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.Code)
	fmt.Printf("%s\n", result.Result)
	fmt.Printf("%d\n", result.Code)
	fmt.Printf("%s\n", result.Message)

	resetKvIteratorCacheAndIndex()

	tearDownTest()
}

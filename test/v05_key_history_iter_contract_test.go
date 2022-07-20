/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"sync/atomic"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/assert"
)

func TestDockerGoKeyHistoryKvIterator(t *testing.T) {
	setupTest(t)

	keyHistoryData = makeKeyModificationMap()

	key := protocol.GetKeyStr("key1", "field1")
	mockGetHistoryIterForKey(mockTxContext, ContractNameTest, key)
	mockTxContext.EXPECT().SetIterHandle(gomock.Any(), gomock.Any()).DoAndReturn(
		func(iteratorIndex int32, iterator protocol.KeyHistoryIterator) {
			kvRowCache[atomic.AddInt32(&kvSetIndex, int32(1))] = iterator
		},
	).AnyTimes()

	mockGetKeyHistoryKVHandle(mockTxContext, int32(1))

	parameters := generateInitParams()
	parameters["key"] = []byte("key1")
	parameters["field"] = []byte("field1")
	method := "key_history_kv_iter"
	result, _ := mockRuntimeInstance.Invoke(mockContractId, method, nil,
		parameters, mockTxContext, uint64(123))
	assert.Equal(t, uint32(0), result.GetCode())
	resetIterCacheAndIndex()

	tearDownTest()
}

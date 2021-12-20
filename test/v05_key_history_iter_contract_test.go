/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package test

import (
	"sync/atomic"
	"testing"

	"github.com/golang/mock/gomock"

	"chainmaker.org/chainmaker/protocol/v2"
)

func TestDockerGoKeyHistoryKvIterator(t *testing.T) {
	setupTest(t)

	keyHistoryData = makeKeyModificationMap()

	key := protocol.GetKeyStr("key1", "field1")
	mockGetHistoryIterForKey(mockTxContext, ContractNameTest, key)
	mockTxContext.EXPECT().SetIterHandle(gomock.Any(), gomock.Any()).DoAndReturn(
		func(iteratorIndex int32, iterator protocol.StateIterator) {
			kvRowCache[atomic.AddInt32(&kvSetIndex, int32(1))] = iterator
		},
	).AnyTimes()

	mockGetKeyHistoryKVHandle(mockTxContext, int32(1))

	parameters := generateInitParams()
	parameters["method"] = []byte("key_history_kv_iter")
	parameters["key"] = []byte("key1")
	parameters["field"] = []byte("field1")

	resetIterCacheAndIndex()

	tearDownTest()
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	sync "sync"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
)

var pbPool = sync.Pool{
	New: func() interface{} {
		return &protogo.DockerVMMessage{}
	},
}

func ReturnToPool(m *protogo.DockerVMMessage) {
	if m != nil {
		m.Reset()
		pbPool.Put(m)
	}
}

func DockerVMMessageFromPool() *protogo.DockerVMMessage {
	return pbPool.Get().(*protogo.DockerVMMessage)
}

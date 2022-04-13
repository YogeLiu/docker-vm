/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interfaces

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

type RequestScheduler interface {
	Start()
	PutMsg(msg interface{}) error
	GetRequestGroup(contractName, contractVersion string) (RequestGroup, bool)
}

type RequestGroup interface {
	Start()
	PutMsg(msg interface{}) error
	GetContractPath() string
	GetTxCh(isOrig bool) chan *protogo.DockerVMMessage
}

type ProcessManager interface {
	Start()
	SetScheduler(RequestScheduler)
	PutMsg(msg interface{}) error
	GetProcessByName(processName string) (Process, bool)
	GetProcessNumByContractKey(contractName, contractVersion string) int
	ChangeProcessState(processName string, toBusy bool) error
}

type Process interface {
	PutMsg(msg interface{}) error
	Start()
	GetProcessName() string
	GetContractName() string
	GetContractVersion() string
	GetUser() *core.User
	SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer)
}

type UserManager interface {
	GetAvailableUser() (*core.User, error)
	FreeUser(user *core.User) error
	BatchCreateUsers() error
}
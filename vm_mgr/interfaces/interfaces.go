/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package interfaces

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

type Scheduler interface {
	GetRequestGroup(contractName string) (RequestGroup, error)
}

type RequestGroup interface {
	PutMsg(msg interface{}) error
}

type RequestScheduler interface {
	PutMsg(msg *protogo.DockerVMMessage) error
}

type ProcessManager interface {
	PutMsg(msg *protogo.DockerVMMessage) error
	GetProcessByName(name string) (Process, error)
}

type Process interface {
	PutMsg(msg *protogo.DockerVMMessage) error
	SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer)
	GetName() string
}

type UserController interface {
	// GetAvailableUser get available user
	GetAvailableUser() (*core.User, error)
	// FreeUser free user
	AddAvailableUser(user *core.User) error
}

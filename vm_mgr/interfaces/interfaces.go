/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package interfaces

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

type (
	RequestScheduler interface {
		Start()
		PutMsg(msg interface{}) error
		GetRequestGroup(chainID, contractName, contractVersion string) (RequestGroup, bool)
		GetContractManager() ContractManager
	}
	RequestGroup interface {
		Start()
		PutMsg(msg interface{}) error
		GetContractPath() string
		GetTxCh(isOrig bool) chan *protogo.DockerVMMessage
	}
	ProcessManager interface {
		Start()
		SetScheduler(RequestScheduler)
		PutMsg(msg interface{}) error
		GetProcessByName(processName string) (Process, bool)
		GetProcessNumByContractKey(chainID, contractName, contractVersion string) int
		ChangeProcessState(processName string, toBusy bool) error
	}
	Process interface {
		PutMsg(msg *protogo.DockerVMMessage)
		Start()
		GetProcessName() string
		GetChainID() string
		GetContractName() string
		GetContractVersion() string
		GetUser() User
		SetStream(stream protogo.DockerVMRpc_DockerVMCommunicateServer)
		ChangeSandbox(chainID, contractName, contractVersion, processName string) error
		CloseSandbox() error
	}
	UserManager interface {
		GetAvailableUser() (User, error)
		FreeUser(user User) error
		BatchCreateUsers() error
	}
	User interface {
		GetUid() int
		GetGid() int
		GetSockPath() string
		GetUserName() string
	}
	ChainRPCService interface {
		SetScheduler(scheduler RequestScheduler)
		PutMsg(msg interface{}) error
		DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error
	}
	ContractManager interface {
		Start()
		SetScheduler(RequestScheduler)
		PutMsg(msg interface{}) error
		GetContractMountDir() string
	}
)

/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package config

// DockerVMConfig match vm settings in chain maker yml
type DockerVMConfig struct {
	EnableDockerVM        bool   `mapstructure:"enable_dockervm"`
	DockerVMContainerName string `mapstructure:"dockervm_container_name"`
	DockerVMMountPath     string `mapstructure:"dockervm_mount_path"`
	DockerVMHost          string `mapstructure:"docker_vm_host"`
	DockerVMPort          uint32 `mapstructure:"docker_vm_port"`
	MaxSendMsgSize        uint32 `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize        uint32 `mapstructure:"max_recv_msg_size"`
}

// DockerContainerConfig docker container settings
type DockerContainerConfig struct {
	AttachStdOut bool
	AttachStderr bool
	ShowStdout   bool
	ShowStderr   bool

	VMMgrDir string

	HostMountDir string
}

type Bool int32

const (

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// SockDir dir save domain socket file
	SockDir = "sock"

	// stateKvIterator method
	FuncKvIteratorCreate    = "createKvIterator"
	FuncKvPreIteratorCreate = "createKvPreIterator"
	FuncKvIteratorHasNext   = "kvIteratorHasNext"
	FuncKvIteratorNext      = "kvIteratorNext"
	FuncKvIteratorClose     = "kvIteratorClose"

	// keyHistoryKvIterator method
	FuncKeyHistoryIterHasNext = "keyHistoryIterHasNext"
	FuncKeyHistoryIterNext    = "keyHistoryIterNext"
	FuncKeyHistoryIterClose   = "keyHistoryIterClose"

	// int32 representation of bool
	BoolTrue  Bool = 1
	BoolFalse Bool = 0
)

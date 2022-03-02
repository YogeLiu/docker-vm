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
	DockerVMLogPath       string `mapstructure:"dockervm_log_path"`
	LogInConsole          bool   `mapstructure:"log_in_console"`
	LogLevel              string `mapstructure:"log_level"`
	DockerVMHost          string `mapstructure:"docker_vm_host"`
	DockerVMPort          uint32 `mapstructure:"docker_vm_port"`
	UserNum               uint32 `mapstructure:"user_num"`
	TxTimeLimit           uint32 `mapstructure:"time_limit"`
	MaxConcurrency        uint32 `mapstructure:"max_concurrency"`
}

// DockerContainerConfig docker container settings
type DockerContainerConfig struct {
	AttachStdOut bool
	AttachStderr bool
	ShowStdout   bool
	ShowStderr   bool

	ImageName     string
	ContainerName string
	VMMgrDir      string

	DockerMountDir string
	DockerLogDir   string
	HostMountDir   string
	HostLogDir     string
}

type Bool int32

const (
	EnvEnablePprof = "ENV_ENABLE_PPROF"
	EnvPprofPort   = "ENV_PPROF_PORT"

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// SockDir dir save domain socket file
	SockDir = "sock"

	MaxSendSize = 100
	MaxRecvSize = 100

	PProfPort = "23356"
	SDKPort   = "24356"

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

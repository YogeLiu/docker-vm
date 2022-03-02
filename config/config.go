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
	DockerVMUDSOpen       bool   `mapstructure:"uds_open"`
	UserNum               uint32 `mapstructure:"user_num"`
	TxTimeLimit           uint32 `mapstructure:"time_limit"`
	MaxConcurrency        uint32 `mapstructure:"max_concurrency"`
	MaxSendMsgSize        uint32 `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize        uint32 `mapstructure:"max_recv_msg_size"`
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
	ENV_ENABLE_UDS        = "ENV_ENABLE_UDS"
	ENV_USER_NUM          = "ENV_USER_NUM"
	ENV_TX_TIME_LIMIT     = "ENV_TX_TIME_LIMIT"
	ENV_LOG_LEVEL         = "ENV_LOG_LEVEL"
	ENV_LOG_IN_CONSOLE    = "ENV_LOG_IN_CONSOLE"
	ENV_MAX_CONCURRENCY   = "ENV_MAX_CONCURRENCY"
	EnvEnablePprof        = "ENV_ENABLE_PPROF"
	EnvPprofPort          = "ENV_PPROF_PORT"
	ENV_MAX_SEND_MSG_SIZE = "ENV_MAX_SEND_MSG_SIZE"
	ENV_MAX_RECV_MSG_SIZE = "ENV_MAX_RECV_MSG_SIZE"

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// SockDir dir save domain socket file
	SockDir = "sock"
	// SockName domain socket file name
	SockName = "cdm.sock"

	TestPort  = "22356"
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

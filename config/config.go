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
	EnablePprof           bool   `mapstructure:"enable_pprof"`
	DockerVMPprofPort     uint32 `mapstructure:"docker_vm_pprof_port"`
	SandBoxPprofPort      uint32 `mapstructure:"sandbox_pprof_port"`
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

	TestPort = "22356"

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

const (
	KeyContractFullName = "KEY_CONTRACT_FULL_NAME"
	KeySenderAddr       = "KEY_SENDER_ADDR"

	KeyCallContractResp = "KEY_CALL_CONTRACT_RESPONSE"
	KeyCallContractReq  = "KEY_CALL_CONTRACT_REQUEST"

	KeyStateKey   = "KEY_STATE_KEY"
	KeyUserKey    = "KEY_USER_KEY"
	KeyUserField  = "KEY_USER_FIELD"
	KeyStateValue = "KEY_STATE_VALUE"

	KeyKVIterKey = "KEY_KV_ITERATOR_KEY"
	KeyIterIndex = "KEY_KV_ITERATOR_INDEX"

	KeyHistoryIterKey   = "KEY_HISTORY_ITERATOR_KEY"
	KeyHistoryIterField = "KEY_HISTORY_ITERATOR_FIELD"
	//KeyHistoryIterIndex = "KEY_HISTORY_ITERATOR_INDEX"

	KeyContractName     = "KEY_CONTRACT_NAME"
	KeyIteratorFuncName = "KEY_ITERATOR_FUNC_NAME"
	KeyIterStartKey     = "KEY_ITERATOR_START_KEY"
	KeyIterStartField   = "KEY_ITERATOR_START_FIELD"
	KeyIterLimitKey     = "KEY_ITERATOR_LIMIT_KEY"
	KeyIterLimitField   = "KEY_ITERATOR_LIMIT_FIELD"
	KeyWriteMap         = "KEY_WRITE_MAP"
	KeyIteratorHasNext  = "KEY_ITERATOR_HAS_NEXT"

	KeyTxId        = "KEY_TX_ID"
	KeyBlockHeight = "KEY_BLOCK_HEIGHT"
	KeyIsDelete    = "KEY_IS_DELETE"
	KeyTimestamp   = "KEY_TIMESTAMP"
)

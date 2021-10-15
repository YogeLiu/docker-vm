package config

type DockerVMConfig struct {
	EnableDockerVM             bool   `mapstructure:"enable_dockervm"`
	DockerVMContainerName      string `mapstructure:"dockervm_container_name"`
	DockerVMMountPath          string `mapstructure:"dockervm_mount_path"`
	DockerVMLogPath            string `mapstructure:"dockervm_log_path"`
	DockerVMUDSOpen            bool   `mapstructure:"uds_open"`
	DockerVMMaxSendMessageSize uint32 `mapstructure:"max_send_message_size"`
	DockerVMMaxRecvMessageSize uint32 `mapstructure:"max_recv_message_size"`
	TxSize                     uint32 `mapstructure:"tx_size"`
	UserNum                    uint32 `mapstructure:"user_num"`
	TxTimeLimit                uint32 `mapstructure:"tx_time_limit"`
}

const (
	ENV_ENABLE_UDS        = "ENV_ENABLE_UDS"
	ENV_MAX_SEND_MSG_SIZE = "ENV_MAX_SEND_MSG_SIZE"
	ENV_MAX_RECV_MSG_SIZE = "ENV_MAX_RECV_MSG_SIZE"
	ENV_TX_SIZE           = "ENV_TX_SIZE"
	ENV_USER_NUM          = "ENV_USER_NUM"
	ENV_TX_TIME_LIMIT     = "ENV_TX_TIME_LIMIT"
	ENV_LOG_LEVEL         = "ENV_LOG_LEVEL"
	ENV_LOG_IN_CONSOLE    = "ENV_LOG_IN_CONSOLE"

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// SockDir dir save domain socket file
	SockDir = "sock"
	// SockName domain socket file name
	SockName = "cdm.sock"
)

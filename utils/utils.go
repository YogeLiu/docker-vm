package utils

import (
	"strings"

	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
)

const (
	DefaultMaxSendSize    = 4
	DefaultMaxRecvSize    = 4
	DefaultUserNum        = 1000
	DefaultTxTimeLimit    = 5
	DefaultMaxConcurrency = 500
)

func SplitContractName(contractNameAndVersion string) string {
	contractName := strings.Split(contractNameAndVersion, "#")[0]
	return contractName
}

func GetMaxSendMsgSizeFromConfig(config *config.DockerVMConfig) uint32 {
	if config.MaxSendMsgSize < DefaultMaxSendSize {
		return DefaultMaxSendSize
	}
	return config.MaxSendMsgSize
}

func GetMaxRecvMsgSizeFromConfig(config *config.DockerVMConfig) uint32 {
	if config.MaxRecvMsgSize < DefaultMaxRecvSize {
		return DefaultMaxRecvSize
	}
	return config.MaxRecvMsgSize
}

func GetMaxConcurrencyFromConfig(config *config.DockerVMConfig) uint32 {
	if config.MaxConcurrency == 0 {
		return DefaultMaxConcurrency
	}
	return config.MaxConcurrency
}

func GetTxTimeLimitFromConfig(config *config.DockerVMConfig) uint32 {
	if config.TxTimeLimit == 0 {
		return DefaultTxTimeLimit
	}
	return config.TxTimeLimit
}

func GetUserNumFromConfig(config *config.DockerVMConfig) uint32 {
	if config.UserNum == 0 {
		return DefaultUserNum
	}
	return config.UserNum
}

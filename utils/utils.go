package utils

import (
	"strings"

	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
)

const (
	DefaultMaxConnection = 1
)

func SplitContractName(contractNameAndVersion string) string {
	contractName := strings.Split(contractNameAndVersion, "#")[0]
	return contractName
}

func GetMaxSendMsgSizeFromConfig(conf *config.DockerVMConfig) uint32 {
	if conf.MaxSendMsgSize < config.DefaultMaxSendSize {
		return config.DefaultMaxSendSize
	}
	return conf.MaxSendMsgSize
}

func GetMaxRecvMsgSizeFromConfig(conf *config.DockerVMConfig) uint32 {
	if conf.MaxRecvMsgSize < config.DefaultMaxRecvSize {
		return config.DefaultMaxRecvSize
	}
	return conf.MaxRecvMsgSize
}

func GetMaxConnectionFromConfig(config *config.DockerVMConfig) uint32 {
	if config.MaxConnection == 0 {
		return DefaultMaxConnection
	}
	return config.MaxConnection
}

// ConstructNotifyMapKey chainId#txId
func ConstructNotifyMapKey(names ...string) string {
	return strings.Join(names, "#")
}

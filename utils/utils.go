package utils

import (
	"strings"

	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
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
package utils

import (
	"fmt"
	"strings"

	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
)

const (
	DefaultMaxSendSize = 20
	DefaultMaxRecvSize = 20
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

func GetURLFromConfig(config *config.DockerVMConfig) string {
	ip := config.DockerVMHost
	port := config.DockerVMPort
	if ip == "" {
		ip = "127.0.0.1"
	}
	if port == 0 {
		port = 22351
	}
	url := fmt.Sprintf("%s:%d", ip, config.DockerVMPort)
	return url
}

package utils

import (
	"fmt"
	"os"
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
	if config.ContractEngine.MaxConnection == 0 {
		return DefaultMaxConnection
	}
	return uint32(config.ContractEngine.MaxConnection)
}

// ConstructNotifyMapKey chainId#txId
func ConstructNotifyMapKey(names ...string) string {
	return strings.Join(names, "#")
}

func CreateDir(directory string) error {
	exist, err := Exists(directory)
	if err != nil {
		return err
	}

	if !exist {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			return fmt.Errorf("failed to create [%s], err: [%s]", directory, err)
		}
	}

	return nil
}

// exists returns whether the given file or directory exists
func Exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

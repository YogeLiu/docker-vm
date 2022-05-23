/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"errors"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
)

var (
	mountDir string
)

// 找 链 拿合约执行文件，保存合约map

type ContractManager struct {
	lock            sync.RWMutex
	getContractLock singleflight.Group
	contractsMap    map[string]string
	logger          *zap.SugaredLogger
	scheduler       protocol.Scheduler
}

// NewContractManager new contract manager
func NewContractManager() *ContractManager {
	contractManager := &ContractManager{
		contractsMap: make(map[string]string),
		logger:       logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, config.DockerLogDir),
	}

	mountDir = config.DockerMountDir

	// todo 还需要 initial contract map 吗？？？
	//_ = contractManager.initialContractMap()
	return contractManager
}

func (cm *ContractManager) SetScheduler(scheduler protocol.Scheduler) {
	cm.scheduler = scheduler
}

// GetContract get contract path in volume,
// if it exists in volume, return path
// if not exist in volume, request from chain maker state library
// contractKey include version, e.g., chainId#fact#v1.0.0
func (cm *ContractManager) GetContract(chainId, txId, contractKey string) (string, error) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	// get contract path from map
	contractPath, ok := cm.contractsMap[contractKey]
	if ok {
		cm.logger.Debugf("get contract from memory [%s], path is [%s]", contractKey, contractPath)
		return contractPath, nil
	}

	// get contract path from chain maker
	cPath, err, _ := cm.getContractLock.Do(contractKey, func() (interface{}, error) {
		defer cm.getContractLock.Forget(contractKey)

		return cm.lookupContractFromDB(chainId, txId, contractKey)
	})
	if err != nil {
		cm.logger.Errorf("fail to get contract path from chain maker, contract name : [%s] -- txId [%s] ", contractKey, txId)
		return "", err
	}

	return cPath.(string), nil
}

// todo modify method name
func (cm *ContractManager) lookupContractFromDB(chainId, txId, contractKey string) (string, error) {

	enableUnixDomainSocket, _ := strconv.ParseBool(os.Getenv(config.ENV_ENABLE_UDS))

	contractPath := filepath.Join(mountDir, chainId, config.ContractsDir, contractKey)
	err := utils.CreateDir(filepath.Join(mountDir, chainId, config.ContractsDir))
	if err != nil {
		return "", err
	}

	_, err = os.Stat(contractPath)
	if err != nil {
		// if run into other errors
		if !errors.Is(err, os.ErrNotExist) {
			return "", errors.New("fail to get contract from path")
		}

		// file not exist then getByteCodeFromChain
		getByteCodeMsg := &protogo.CDMMessage{
			TxId:    txId,
			Type:    protogo.CDMType_CDM_TYPE_GET_BYTECODE,
			Payload: []byte(contractKey),
			ChainId: chainId,
		}
		// send request to chain maker
		responseChan := make(chan *protogo.CDMMessage)
		cm.scheduler.RegisterResponseCh(chainId, txId, responseChan)

		cm.scheduler.GetByteCodeReqCh() <- getByteCodeMsg

		returnMsg := <-responseChan

		if returnMsg.Payload == nil {
			return "", errors.New("fail to get bytecode")
		}

		if !enableUnixDomainSocket {
			content := returnMsg.Payload
			err := ioutil.WriteFile(contractPath, content, 0755)
			if err != nil {
				return "", err
			}
		}

		err = cm.setFileMod(contractPath)
		if err != nil {
			return "", err
		}
	}

	// save contract file path to map
	cm.contractsMap[contractKey] = contractPath
	//cm.logger.Debugf("get contract disk [%s], path is [%s]", contractName, contractPath)

	return contractPath, nil
}

// SetFileRunnable make file runnable, file permission is 755
func (cm *ContractManager) setFileMod(filePath string) error {

	err := os.Chmod(filePath, 0755)
	if err != nil {
		cm.logger.Errorf("fail to set contract mod , filePath : [%s] ", filePath)
		return err
	}

	return nil
}

func (cm *ContractManager) initialContractMap() error {

	files, err := ioutil.ReadDir(mountDir)
	if err != nil {
		cm.logger.Errorf("fail to scan contract dir")
		return err
	}
	for _, f := range files {
		contractName := f.Name()
		contractPath := filepath.Join(mountDir, contractName)
		cm.contractsMap[contractName] = contractPath
	}

	cm.logger.Debugf("init contract map with size [%d]", len(cm.contractsMap))

	return nil
}

func (cm *ContractManager) checkContractDeployed(contractName string) (string, bool) {
	cm.lock.RLock()
	defer cm.lock.RUnlock()

	contractPath, ok := cm.contractsMap[contractName]

	if ok {
		return contractPath, true
	}
	return "", false
}

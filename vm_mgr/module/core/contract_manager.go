/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"io/ioutil"
	"path/filepath"

	"go.uber.org/zap"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
)

const (
	ContractsDir               = "contracts" // ContractsDir dir save executable contract
	contractManagerEventChSize = 64
)

// ContractManager manage all contracts with LRU cache
type ContractManager struct {
	contractsLRU *utils.Cache                  // contract LRU cache, make sure the contracts doesn't take up too much disk space
	logger       *zap.SugaredLogger            // contract manager logger
	scheduler    interfaces.RequestScheduler   // request scheduler
	eventCh      chan *protogo.DockerVMMessage // contract invoking handler
	mountDir     string                        // contract mount Dir
}

// NewContractManager returns new contract manager
func NewContractManager() *ContractManager {
	contractManager := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}
	_ = contractManager.initContractLRU()
	return contractManager
}

// SetScheduler set request scheduler
func (cm *ContractManager) SetScheduler(scheduler interfaces.RequestScheduler) {
	cm.scheduler = scheduler
}

// Start contract manager, listen event chan,
// event chan msg types: DockerVMType_GET_BYTECODE_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (cm *ContractManager) Start() {
	go func() {
		select {
		case msg := <-cm.eventCh:
			switch msg.Type {
			case protogo.DockerVMType_GET_BYTECODE_REQUEST:
				// handle get contract request:
				// len(path) == 0: contract not found in lru cache, try to request bytecode from chain
				// len(path) != 0: contract found in lru cache, send contract ready signal to request group
				path, err := cm.handleGetContractReq(msg)
				if err != nil {
					cm.logger.Errorf("failed to handle get bytecode request, %v", err)
					break
				}
				if len(path) == 0 {
					cm.logger.Debugf("send get bytecode request to chain, contract name: [%s], " +
						"contract version: [%s], txId [%s] ", msg.GetRequest().GetContractName(),
						msg.GetRequest().GetContractVersion(), msg.TxId)
					break
				}
				if err = cm.sendContractReadySignal(msg.GetRequest().GetContractName(),
					msg.GetRequest().GetContractVersion()); err != nil {
					cm.logger.Errorf("failed to handle get bytecode request, %v", err)
				}
			case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
				err := cm.handleGetContractResp(msg)
				if err != nil {
					cm.logger.Errorf("failed to handle get bytecode response, %v", err)
					break
				}
			default:
				cm.logger.Errorf("unknown msg type")
			}
		}
	}()
}

// PutMsg put invoking requests into chan, waiting for contract manager to handle request
//  @param msg types include DockerVMType_GET_BYTECODE_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (cm *ContractManager) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		cm.eventCh <- m
	default:
		cm.logger.Errorf("unknown msg type")
	}
	return nil
}

// initContractLRU loads contract files from disk to lru
func (cm *ContractManager) initContractLRU() error {
	files, err := ioutil.ReadDir(cm.mountDir)
	if err != nil {
		return fmt.Errorf("fail to scan contract dir [%s], %v", cm.mountDir)
	}

	// contracts that exceed the limit will be cleaned up
	for i, f := range files {
		name := f.Name()
		path := filepath.Join(cm.mountDir, name)
		if i < cm.contractsLRU.MaxEntries {
			cm.contractsLRU.Add(name, path)
			continue
		}
		if err = utils.RemoveDir(path); err != nil {
			return fmt.Errorf("fail to clean contract files, file path: [%s], %v", path, err)
		}
	}
	cm.logger.Debugf("init contract LRU with size [%d]", cm.contractsLRU.Len())
	return nil
}

// handleGetContractReq return contract path,
// if it exists in contract LRU, return path
// if not exists, request from chain
func (cm *ContractManager) handleGetContractReq(msg *protogo.DockerVMMessage) (string, error) {
	// get contract path from lru, return path
	contractKey := utils.ConstructContractKey(msg.GetRequest().GetContractName(), msg.GetRequest().GetContractVersion())
	if contractPath, ok := cm.contractsLRU.Get(contractKey); ok {
		path := contractPath.(string)
		cm.logger.Debugf("get contract [%s] from memory, path: [%s]", contractKey, path)
		return path, nil
	}
	// request contract from chain, return ""
	err := cm.requestContractFromChain(msg)
	if err != nil {
		return "", fmt.Errorf("failed to request contract from chain, contract key : [%s], txId [%s] ",
			contractKey, msg.GetTxId())
	}
	return "", nil
}

// handleGetContractResp handle get contract msg, save in lru,
// if contract lru is full, pop oldest contracts from lru, delete from disk.
func (cm *ContractManager) handleGetContractResp(msg *protogo.DockerVMMessage) error {
	// check the response from chain
	if msg.GetResponse().GetCode() == protogo.DockerVMCode_FAIL {
		return fmt.Errorf("chain failed to load bytecode")
	}
	// if contracts lru is full, delete oldest contract
	if cm.contractsLRU.Len() == cm.contractsLRU.MaxEntries {
		oldestContractPath := cm.contractsLRU.GetOldest()
		if oldestContractPath == nil {
			return fmt.Errorf("oldest contract is nil")
		}

		cm.contractsLRU.RemoveOldest()

		if err := utils.RemoveDir(oldestContractPath.(string)); err != nil {
			return fmt.Errorf("failed to remove file, %v", err)
		}
		cm.logger.Debugf("removed oldest contract from disk and lru")
	}

	// save contract in lru (contract file already saved in disk by chain)
	path := filepath.Join(cm.mountDir, msg.GetResponse().GetContractName())
	cm.contractsLRU.Add(msg.GetResponse().GetContractName(), path)
	cm.logger.Debugf("new contract saved in lru and disk")

	// send contract ready signal to request group
	if err := cm.sendContractReadySignal(msg.GetRequest().GetContractName(),
		msg.GetRequest().GetContractVersion()); err != nil {
		cm.logger.Errorf("failed to handle get bytecode request, %s", err)
	}
	return nil
}

// requestContractFromChain request contract from chain
func (cm *ContractManager) requestContractFromChain(msg *protogo.DockerVMMessage) error {
	// send request to request scheduler
	if err := cm.scheduler.PutMsg(msg); err != nil {
		return err
	}
	return nil
}

// sendContractReadySignal send contract ready signal to request group, request group can request process now.
func (cm *ContractManager) sendContractReadySignal(contractName, contractVersion string) error {
	if cm.scheduler == nil {
		return fmt.Errorf("request scheduler have not been initialized")
	}
	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)
	requestGroup, err := cm.scheduler.GetRequestGroup(contractName, contractVersion)
	if err != nil {
		return fmt.Errorf("failed to get request group, %v", err)
	}
	err = requestGroup.PutMsg(groupKey)
	if err != nil {
		return fmt.Errorf("failed to put msg into request group's event chan")
	}
	return nil
}

// GetContractMountDir returns contract mount dir
func (cm *ContractManager) GetContractMountDir() string {
	return cm.mountDir
}
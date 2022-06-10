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
	sizePerContract            = 15 // MiB
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
func NewContractManager() (*ContractManager, error) {
	contractManager := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize / sizePerContract),
		logger:       logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}
	if err := contractManager.initContractLRU(); err != nil {
		return nil, err
	}
	return contractManager, nil
}

// SetScheduler set request scheduler
func (cm *ContractManager) SetScheduler(scheduler interfaces.RequestScheduler) {
	cm.scheduler = scheduler
}

// Start contract manager, listen event chan
func (cm *ContractManager) Start() {

	cm.logger.Debugf("start contract manager routine")

	go func() {
		for {
			select {
			case msg := <-cm.eventCh:
				switch msg.Type {

				case protogo.DockerVMType_GET_BYTECODE_REQUEST:
					if err := cm.handleGetContractReq(msg); err != nil {
						cm.logger.Errorf("failed to handle get bytecode request, %v", err)
					}

				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					err := cm.handleGetContractResp(msg)
					if err != nil {
						cm.logger.Errorf("failed to handle get bytecode response, %v", err)
						break
					}

				default:
					cm.logger.Errorf("unknown msg type, msg: %+v", msg)
				}
			}
		}
	}()
}

// PutMsg put invoking requests to chan, waiting for contract manager to handle request
//  @param req types include DockerVMType_GET_BYTECODE_REQUEST and DockerVMType_GET_BYTECODE_RESPONSE
func (cm *ContractManager) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		cm.eventCh <- m
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetContractMountDir returns contract mount dir
func (cm *ContractManager) GetContractMountDir() string {
	return cm.mountDir
}

// initContractLRU loads contract files from disk to lru
func (cm *ContractManager) initContractLRU() error {
	err := cm.initContractPath()
	if err != nil {
		return fmt.Errorf("failed to init contract path, %v", err)
	}

	files, err := ioutil.ReadDir(cm.mountDir)
	if err != nil {
		return fmt.Errorf("failed to read contract dir [%s], %v", cm.mountDir, err)
	}

	// contracts that exceed the limit will be cleaned up
	for i, f := range files {
		name := f.Name()
		path := filepath.Join(cm.mountDir, name)
		// file num < max entries
		if i < cm.contractsLRU.MaxEntries {
			cm.contractsLRU.Add(name, path)
			continue
		}
		// file num >= max entries
		if err = utils.RemoveDir(path); err != nil {
			return fmt.Errorf("failed to remove contract files, file path: [%s], %v", path, err)
		}
	}
	cm.logger.Debugf("init contract LRU with size [%d]", cm.contractsLRU.Len())
	return nil
}

// handleGetContractReq return contract path,
// if it exists in contract LRU, return path
// if not exists, request from chain
func (cm *ContractManager) handleGetContractReq(req *protogo.DockerVMMessage) error {

	cm.logger.Debugf("handle get contract request, txId: [%s]", req.TxId)

	if req.Request == nil {
		return fmt.Errorf("empty request payload")
	}

	contractKey := utils.ConstructContractKey(req.Request.ContractName, req.Request.ContractVersion)

	// contract path found in lru
	if contractPath, ok := cm.contractsLRU.Get(contractKey); ok {
		path := contractPath.(string)
		cm.logger.Debugf("get contract [%s] from memory, path: [%s]", contractKey, path)
		if err := cm.sendContractReadySignal(req.Request.ContractName, req.Request.ContractVersion); err != nil {
			return fmt.Errorf("failed to handle get bytecode request, %v", err)
		}
		return nil
	}

	// request contract from chain
	cm.requestContractFromChain(req)

	cm.logger.Debugf("send get bytecode request to chain, contract name: [%s], "+
		"contract version: [%s], txId [%s] ", req.Request.ContractName, req.Request.ContractVersion, req.TxId)

	return nil
}

// handleGetContractResp handle get contract req, save in lru,
// if contract lru is full, pop oldest contracts from lru, delete from disk.
func (cm *ContractManager) handleGetContractResp(resp *protogo.DockerVMMessage) error {

	cm.logger.Debugf("handle get contract response, txId: [%s]", resp.TxId)

	if resp.Response == nil {
		return fmt.Errorf("empty response payload")
	}

	// check the response from chain
	if resp.Response.Code == protogo.DockerVMCode_FAIL {
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
	groupKey := utils.ConstructContractKey(resp.Response.ContractName, resp.Response.ContractVersion)

	path := filepath.Join(cm.mountDir, groupKey)
	cm.contractsLRU.Add(groupKey, path)

	if config.DockerVMConfig.RPC.ChainRPCProtocol == config.TCP {
		if len(resp.Response.Result) == 0 {
			return fmt.Errorf("invalid contract, contract is nil")
		}

		err := ioutil.WriteFile(path, resp.Response.Result, 0755)
		if err != nil {
			return fmt.Errorf("failed to write contract file, [%s]", groupKey)
		}
	}

	cm.logger.Infof("contract [%s] saved in lru and dir [%s]", groupKey, path)

	// send contract ready signal to request group
	if err := cm.sendContractReadySignal(resp.Response.ContractName,
		resp.Response.ContractVersion); err != nil {
		return fmt.Errorf("failed to send contract ready signal, %v", err)
	}
	return nil
}

// requestContractFromChain request contract from chain
func (cm *ContractManager) requestContractFromChain(msg *protogo.DockerVMMessage) {
	// send request to request scheduler
	_ = cm.scheduler.PutMsg(msg)
}

// sendContractReadySignal send contract ready signal to request group, request group can request process now.
func (cm *ContractManager) sendContractReadySignal(contractName, contractVersion string) error {

	// check whether scheduler was initialized
	if cm.scheduler == nil {
		return fmt.Errorf("request scheduler has not been initialized")
	}

	// get request group
	// GetRequestGroup is safe because it's a new request group, no process exist trigger
	requestGroup, ok := cm.scheduler.GetRequestGroup(contractName, contractVersion)
	if !ok {
		return fmt.Errorf("failed to get request group")
	}
	_ = requestGroup.PutMsg(&protogo.DockerVMMessage{
		Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
	})
	return nil
}

func (cm *ContractManager) initContractPath() error {
	var err error
	// mkdir paths
	contractDir := filepath.Join(config.DockerMountDir, ContractsDir)
	err = utils.CreateDir(contractDir)
	if err != nil {
		return err
	}
	cm.logger.Debug("set contract dir: ", contractDir)

	return nil
}

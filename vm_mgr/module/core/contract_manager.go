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

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
)

const (
	ContractsDir                = "contract-bins" // ContractsDir dir save executable contract
	_contractManagerEventChSize = 64
	_sizePerContract            = 15 // MiB
)

// ContractManager manage all contracts with LRU cache
type ContractManager struct {
	contractsLRU *utils.Cache                // contract LRU cache, make sure the contracts doesn't take up too much disk space
	logger       *zap.SugaredLogger          // contract manager logger
	scheduler    interfaces.RequestScheduler // request scheduler
	eventCh      chan interface{}            // contract invoking handler
	mountDir     string                      // contract mount Dir
}

// check interface implement
var _ interfaces.ContractManager = (*ContractManager)(nil)

// NewContractManager returns new contract manager
func NewContractManager() (*ContractManager, error) {
	contractManager := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize / _sizePerContract),
		logger:       logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
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
				switch msg.(type) {
				case *protogo.DockerVMMessage:
					m := msg.(*protogo.DockerVMMessage)
					switch m.Type {
					case protogo.DockerVMType_GET_BYTECODE_REQUEST:
						if err := cm.handleGetContractReq(m); err != nil {
							cm.logger.Errorf("failed to handle get bytecode request, %v", err)
						}

					case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
						err := cm.handleGetContractResp(m)
						if err = cm.processContractResp(m, err); err != nil {
							cm.logger.Errorf("failed to handle [%s] get bytecode response, %v", m.TxId, err)
						}

					default:
						cm.logger.Errorf("unknown msg type, msg: %+v", msg)
					}

				case *messages.BadContractResp:
					err := cm.handleBadContractResp(msg.(*messages.BadContractResp))
					if err != nil {
						cm.logger.Errorf("failed to handle remove contract, %v", err)
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
	case *protogo.DockerVMMessage, *messages.BadContractResp:
		cm.eventCh <- msg
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

	contractKey := utils.ConstructContractKey(req.Request.ChainId, req.Request.ContractName, req.Request.ContractVersion)

	// contract path found in lru
	if contractPath, ok := cm.contractsLRU.Get(contractKey); ok {
		path := contractPath.(string)
		cm.logger.Debugf("get contract [%s] from memory, path: [%s]", contractKey, path)
		if err := cm.sendContractReadySignal(req.Request.ChainId, req.Request.ContractName,
			req.Request.ContractVersion, protogo.DockerVMCode_OK); err != nil {
			return fmt.Errorf("failed to handle get bytecode request, %v", err)
		}
		return nil
	}

	// request contract from chain
	if err := cm.requestContractFromChain(req); err != nil {
		return fmt.Errorf("failed to request contract from chain, %v", err)
	}

	cm.logger.Infof("send get bytecode request to chain [%s], contract name: [%s], contract version: [%s], "+
		"txId [%s] ", req.Request.ChainId, req.Request.ContractName, req.Request.ContractVersion, req.TxId)

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
	contractKey := utils.ConstructContractKey(resp.Response.ChainId, resp.Response.ContractName, resp.Response.ContractVersion)

	path := filepath.Join(cm.mountDir, contractKey)
	cm.contractsLRU.Add(contractKey, path)

	if config.DockerVMConfig.RPC.ChainRPCProtocol == config.TCP {
		if len(resp.Response.Result) == 0 {
			return fmt.Errorf("invalid contract, contract is nil")
		}

		err := ioutil.WriteFile(path, resp.Response.Result, 0755)
		if err != nil {
			return fmt.Errorf("failed to write contract file, [%s]", contractKey)
		}
	}

	cm.logger.Infof("contract [%s] saved in lru and dir [%s]", contractKey, path)

	return nil
}

// handleProcessContractResp process contract resp
func (cm *ContractManager) processContractResp(msg *protogo.DockerVMMessage, err error) error {

	var code protogo.DockerVMCode
	if err != nil {
		code = protogo.DockerVMCode_FAIL
	}

	// send contract ready signal to request group
	if sendErr := cm.sendContractReadySignal(msg.Response.ChainId, msg.Response.ContractName,
		msg.Response.ContractVersion, code); sendErr != nil {
		err = fmt.Errorf("%v, %v", err, sendErr)
	}

	return err
}

// handleBadContractResp removes contract from cache and disk
func (cm *ContractManager) handleBadContractResp(msg *messages.BadContractResp) error {

	cm.logger.Debugf("handle remove contract, txId: [%s]", msg.Tx.TxId)

	// construct contract key
	contractKey := utils.ConstructContractKey(msg.Tx.Request.ChainId, msg.Tx.Request.ContractName, msg.Tx.Request.ContractVersion)

	path, ok := cm.contractsLRU.Get(contractKey)
	if !ok {
		return fmt.Errorf("contract %s not exists in lru", contractKey)
	}

	cm.contractsLRU.Remove(contractKey)
	if err := utils.RemoveDir(path.(string)); err != nil {
		return fmt.Errorf("failed to remove file, %v", err)
	}

	cm.logger.Debugf("removed contract %s from disk and lru", contractKey)

	return nil
}

// requestContractFromChain request contract from chain
func (cm *ContractManager) requestContractFromChain(msg *protogo.DockerVMMessage) error {
	// send request to request scheduler
	if err := cm.scheduler.PutMsg(msg); err != nil {
		return fmt.Errorf("failed to invoke scheduler PutMsg, %v", err)
	}

	return nil
}

// sendContractReadySignal send contract ready signal to request group, request group can request process now.
func (cm *ContractManager) sendContractReadySignal(chainID, contractName, contractVersion string, status protogo.DockerVMCode) error {

	// check whether scheduler was initialized
	if cm.scheduler == nil {
		return fmt.Errorf("request scheduler has not been initialized")
	}

	// get request group
	// GetRequestGroup is safe because it's a new request group, no process exist trigger
	requestGroup, ok := cm.scheduler.GetRequestGroup(chainID, contractName, contractVersion)
	if !ok {
		return fmt.Errorf("failed to get request group, %s", utils.ConstructContractKey(chainID, contractName, contractVersion))
	}

	if err := requestGroup.PutMsg(&protogo.DockerVMMessage{
		ChainId: chainID,
		Type:    protogo.DockerVMType_GET_BYTECODE_RESPONSE,
		Response: &protogo.TxResponse{
			Code: status,
		},
	}); err != nil {
		return fmt.Errorf("failed to invoke request group PutMsg, %v", err)
	}

	return nil
}

func (cm *ContractManager) initContractPath() error {
	var err error
	// mkdir paths
	contractDir := filepath.Join(config.DockerMountDir, ContractsDir)
	err = utils.CreateDir(contractDir, 0777)
	if err != nil {
		return err
	}
	cm.logger.Debug("set contract dir: ", contractDir)

	return nil
}

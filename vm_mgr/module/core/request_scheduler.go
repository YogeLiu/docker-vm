/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"fmt"
	"go.uber.org/zap"
	"sync"
)

const (
	// requestSchedulerEventChSize is request scheduler event chan size
	requestSchedulerEventChSize = 15000
	// requestSchedulerEventChSize is close request group chan size
	closeChSize = 8
)

// RequestScheduler schedule all requests and responses between chain and contract engine, includes:
// get bytecode request (contract engine -> chain)
// get bytecode response (chain -> contract engine)
// tx request (chain -> contract engine)
// call contract request (chain -> contract engine)
// tx error (contract engine -> chain)
type RequestScheduler struct {
	logger *zap.SugaredLogger // request scheduler logger
	lock   sync.RWMutex       // request scheduler rw lock

	eventCh chan *protogo.DockerVMMessage  // request scheduler event handler chan
	closeCh chan *messages.RequestGroupKey // close request group chan

	requestGroups       map[string]interfaces.RequestGroup // contractName#contractVersion
	chainRPCService     interfaces.ChainRPCService         // chain rpc service
	contractManager     *ContractManager                   // contract manager
	origProcessManager  interfaces.ProcessManager          // manager for original process
	crossProcessManager interfaces.ProcessManager          // manager for cross process
}

// NewRequestScheduler new a request scheduler
func NewRequestScheduler(
	service interfaces.ChainRPCService,
	oriPMgr interfaces.ProcessManager,
	crossPMgr interfaces.ProcessManager,
	cMgr *ContractManager) *RequestScheduler {

	scheduler := &RequestScheduler{
		logger: logger.NewDockerLogger(logger.MODULE_REQUEST_SCHEDULER),
		lock:   sync.RWMutex{},

		eventCh: make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),

		requestGroups:       make(map[string]interfaces.RequestGroup),
		chainRPCService:     service,
		origProcessManager:  oriPMgr,
		crossProcessManager: crossPMgr,
		contractManager:     cMgr,
	}
	return scheduler
}

// Start starts docker scheduler
func (s *RequestScheduler) Start() {

	s.logger.Debugf("start request scheduler")

	go func() {
		for {
			select {
			case msg := <-s.eventCh:
				switch msg.Type {
				case protogo.DockerVMType_GET_BYTECODE_REQUEST:
					if err := s.handleGetContractReq(msg); err != nil {
						s.logger.Errorf("failed to handle get bytecode request, %v", err)
					}
				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					if err := s.handleGetContractResp(msg); err != nil {
						s.logger.Errorf("failed to handle get bytecode response, %v", err)
					}
				case protogo.DockerVMType_TX_REQUEST:
					if err := s.handleTxReq(msg); err != nil {
						s.logger.Errorf("failed to handle tx request, %v", err)
					}
				case protogo.DockerVMType_CALL_CONTRACT_REQUEST:
					if err := s.handleTxReq(msg); err != nil {
						s.logger.Errorf("failed to handle call contract request, %v", err)
					}
				case protogo.DockerVMType_ERROR:
					if err := s.handleErrResp(msg); err != nil {
						s.logger.Errorf("failed to handle error response, %v", err)
					}
				default:
					s.logger.Errorf("unknown msg type, %+v", msg)
				}
			case msg := <-s.closeCh:
				if err := s.handleCloseReq(msg); err != nil {
					s.logger.Warnf("close request group %v", err)
				}
			}
		}
	}()
}

// PutMsg puts invoking msgs to chain, waiting for request scheduler to handle request
func (s *RequestScheduler) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		s.eventCh <- m
	case *messages.RequestGroupKey:
		m, _ := msg.(*messages.RequestGroupKey)
		s.closeCh <- m
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetRequestGroup returns request group
func (s *RequestScheduler) GetRequestGroup(contractName, contractVersion string) (interfaces.RequestGroup, bool) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	groupKey := utils.ConstructContractKey(contractName, contractVersion)
	group, ok := s.requestGroups[groupKey]
	return group, ok
}

// handleGetContractReq handles get contract bytecode request, transfer to chain rpc service
func (s *RequestScheduler) handleGetContractReq(req *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle get contract request, txId: [%s]", req.TxId)

	if err := s.chainRPCService.PutMsg(req); err != nil {
		return err
	}
	return nil
}

// handleGetContractResp handles get contract bytecode response, transfer to contract manager
func (s *RequestScheduler) handleGetContractResp(resp *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle get contract response, txId: [%s]", resp.TxId)

	if err := s.contractManager.PutMsg(resp); err != nil {
		return err
	}
	return nil
}

// handleTxReq handles tx request from chain, transfer to request group
func (s *RequestScheduler) handleTxReq(req *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	s.logger.Debugf("handle tx request, txId: [%s]", req.TxId)

	if req.Request == nil {
		return fmt.Errorf("empty request payload")
	}

	// construct request group key from request
	contractName := req.Request.ContractName
	contractVersion := req.Request.ContractVersion
	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)

	// try to get request group, if not, add it
	group, ok := s.requestGroups[groupKey]
	if !ok {
		s.logger.Debugf("create new request group %s", groupKey)
		group = NewRequestGroup(contractName, contractVersion,
			s.origProcessManager, s.crossProcessManager, s.contractManager, s)
		group.Start()
		s.requestGroups[groupKey] = group
	}

	// put req to such request group
	if err := group.PutMsg(req); err != nil {
		return err
	}
	return nil
}

// handleErrResp handles tx failed error
func (s *RequestScheduler) handleErrResp(resp *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle err resp, txId: [%s]", resp.TxId)

	if err := s.chainRPCService.PutMsg(resp); err != nil {
		return err
	}
	return nil
}

// handleCloseReq handles close request group request
func (s *RequestScheduler) handleCloseReq(msg *messages.RequestGroupKey) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	s.logger.Debugf("handle close request group reqest, contract name: [%s], contract version: [%s]",
		msg.ContractName, msg.ContractVersion)

	groupKey := utils.ConstructRequestGroupKey(msg.ContractName, msg.ContractVersion)
	if _, ok := s.requestGroups[groupKey]; ok {
		_ = s.requestGroups[groupKey].PutMsg(&messages.CloseMsg{})
		delete(s.requestGroups, groupKey)
		return nil
	}
	return fmt.Errorf("request group %s not found", groupKey)
}

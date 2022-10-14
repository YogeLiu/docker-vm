/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"sync"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"go.uber.org/zap"
)

const (
	// _requestSchedulerTxChSize is request scheduler event chan size
	_requestSchedulerTxChSize = 50000
	// _requestSchedulerEventChSize is request scheduler event chan size
	_requestSchedulerEventChSize = 1000
	// _closeChSize is close request group chan size
	_closeChSize = 8
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
	txCh    chan *protogo.DockerVMMessage  // request scheduler event handler chan
	closeCh chan *messages.RequestGroupKey // close request group chan

	requestGroups       map[string]interfaces.RequestGroup // chainID#contractName#contractVersion
	chainRPCService     interfaces.ChainRPCService         // chain rpc service
	contractManager     interfaces.ContractManager         // contract manager
	origProcessManager  interfaces.ProcessManager          // manager for original process
	crossProcessManager interfaces.ProcessManager          // manager for cross process
}

// check interface implement
var _ interfaces.RequestScheduler = (*RequestScheduler)(nil)

// NewRequestScheduler new a request scheduler
func NewRequestScheduler(
	service interfaces.ChainRPCService,
	oriPMgr interfaces.ProcessManager,
	crossPMgr interfaces.ProcessManager,
	cMgr interfaces.ContractManager) *RequestScheduler {

	scheduler := &RequestScheduler{
		logger: logger.NewDockerLogger(logger.MODULE_REQUEST_SCHEDULER),
		lock:   sync.RWMutex{},

		eventCh: make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
		txCh:    make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
		closeCh: make(chan *messages.RequestGroupKey, _closeChSize),

		requestGroups:       make(map[string]interfaces.RequestGroup),
		chainRPCService:     service,
		origProcessManager:  oriPMgr,
		crossProcessManager: crossPMgr,
		contractManager:     cMgr,
	}
	return scheduler
}

// Start starts request scheduler
func (s *RequestScheduler) Start() {

	s.logger.Debugf("start request scheduler routine")

	go func() {
		for {
			select {
			case msg := <-s.eventCh:
				switch msg.Type {
				case protogo.DockerVMType_GET_BYTECODE_REQUEST:
					if err := s.handleGetContractReq(msg); err != nil {
						s.logger.Errorf("failed to handle get contract request, %v", err)
					}

				case protogo.DockerVMType_GET_BYTECODE_RESPONSE:
					s.handleGetContractResp(msg)

				case protogo.DockerVMType_ERROR:
					if err := s.handleErrResp(msg); err != nil {
						s.logger.Errorf("failed to handle error response, %v", err)
					}

				}
			case msg := <-s.txCh:
				if err := s.handleTxReq(msg); err != nil {
					s.logger.Errorf("failed to handle tx request, %v", err)
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
		switch m.Type {
		case protogo.DockerVMType_GET_BYTECODE_REQUEST, protogo.DockerVMType_GET_BYTECODE_RESPONSE, protogo.DockerVMType_ERROR:
			s.eventCh <- m
		case protogo.DockerVMType_TX_REQUEST:
			//utils.EnterNextStep(m, protogo.StepType_ENGINE_SCHEDULER_RECEIVE_TX_REQUEST, fmt.Sprintf(" scheduler tx chan size: %d", len(s.txCh)))
			s.txCh <- m
		default:
			return fmt.Errorf("unknown msg type, %+v", msg)
		}

	case *messages.RequestGroupKey:
		m, _ := msg.(*messages.RequestGroupKey)
		s.closeCh <- m

	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetRequestGroup returns request group
func (s *RequestScheduler) GetRequestGroup(chainID, contractName, contractVersion string) (interfaces.RequestGroup, bool) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	groupKey := utils.ConstructContractKey(chainID, contractName, contractVersion)
	group, ok := s.requestGroups[groupKey]
	return group, ok
}

// GetContractManager returns contract manager
func (s *RequestScheduler) GetContractManager() interfaces.ContractManager {

	return s.contractManager
}

// handleGetContractReq handles get contract bytecode request, transfer to chain rpc service
func (s *RequestScheduler) handleGetContractReq(req *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle get contract request, txId: [%s]", req.TxId)

	if err := s.chainRPCService.PutMsg(req); err != nil {
		return fmt.Errorf("failed to invoke chain rpc service PutMsg, %v", err)
	}
	return nil
}

// handleGetContractResp handles get contract bytecode response, transfer to contract manager
func (s *RequestScheduler) handleGetContractResp(resp *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle get contract response, txId: [%s]", resp.TxId)

	if err := s.contractManager.PutMsg(resp); err != nil {
		return fmt.Errorf("failed to invoke contract manager PutMsg, %v", err)
	}

	return nil
}

// handleTxReq handles tx request from chain, transfer to request group
func (s *RequestScheduler) handleTxReq(req *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	logger.DebugDynamic(s.logger, func() string {
		return fmt.Sprintf("handle tx request, txId: [%s]", req.TxId)
	})

	if req.Request == nil {
		return fmt.Errorf("empty request payload")
	}

	// construct request group key from request
	chainID := req.Request.ChainId
	contractName := req.Request.ContractName
	contractVersion := req.Request.ContractVersion
	groupKey := utils.ConstructContractKey(chainID, contractName, contractVersion)

	// try to get request group, if not, add it
	group, ok := s.requestGroups[groupKey]
	if !ok {
		s.logger.Debugf("create new request group %s", groupKey)
		group = NewRequestGroup(chainID, contractName, contractVersion,
			s.origProcessManager, s.crossProcessManager, s)
		group.Start()
		s.requestGroups[groupKey] = group
	}

	//utils.EnterNextStep(req, protogo.StepType_ENGINE_SCHEDULER_SEND_TX_REQUEST, "")
	// put req to such request group
	if err := group.PutMsg(req); err != nil {
		return fmt.Errorf("failed to invoke request group PutMsg, %v", err)
	}

	return nil
}

// handleErrResp handles tx failed error
func (s *RequestScheduler) handleErrResp(resp *protogo.DockerVMMessage) error {

	s.logger.Debugf("handle err resp, txId: [%s]", resp.TxId)

	if err := s.chainRPCService.PutMsg(resp); err != nil {
		return fmt.Errorf("failed to invoke chain rpc service PutMsg, %v", err)
	}

	return nil
}

// handleCloseReq handles close request group request
func (s *RequestScheduler) handleCloseReq(msg *messages.RequestGroupKey) error {

	s.logger.Debugf("handle close request group, chainID: [%s], "+
		"contract name: [%s], contract version: [%s]", msg.ChainID, msg.ContractName, msg.ContractVersion)

	//if s.origProcessManager.GetProcessNumByContractKey(msg.ChainID, msg.ContractName, msg.ContractVersion) != 0 ||
	//	s.crossProcessManager.GetProcessNumByContractKey(msg.ChainID, msg.ContractName, msg.ContractVersion) != 0 {
	//	s.logger.Debugf("process exists, stop to close request group")
	//	return nil
	//}

	groupKey := utils.ConstructContractKey(msg.ChainID, msg.ContractName, msg.ContractVersion)

	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.requestGroups[groupKey]; !ok {
		return fmt.Errorf("request group %s not found", groupKey)
	}
	if err := s.requestGroups[groupKey].PutMsg(&messages.CloseMsg{}); err != nil {
		return fmt.Errorf("failed to invoke request group PutMsg, %v", err)
	}
	delete(s.requestGroups, groupKey)
	return nil
}

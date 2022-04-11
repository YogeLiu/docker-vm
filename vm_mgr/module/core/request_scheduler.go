/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/rpc"
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

type RequestScheduler struct {
	logger *zap.SugaredLogger
	lock   sync.RWMutex

	eventCh             chan *protogo.DockerVMMessage
	closeCh             chan string
	requestGroups       map[string]interfaces.RequestGroup // contractName#contractVersion
	chainRPCService     *rpc.ChainRPCService
	contractManager     *ContractManager
	origProcessManager  interfaces.ProcessManager
	crossProcessManager interfaces.ProcessManager
}

// NewRequestScheduler new request scheduler
func NewRequestScheduler(
	service *rpc.ChainRPCService,
	oriPMgr interfaces.ProcessManager,
	crossPMgr interfaces.ProcessManager,
	cMgr *ContractManager) *RequestScheduler {
	scheduler := &RequestScheduler{
		logger: logger.NewDockerLogger(logger.MODULE_REQUEST_SCHEDULER),
		lock:   sync.RWMutex{},

		eventCh:             make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
		closeCh:             make(chan string, closeChSize),
		requestGroups:       make(map[string]interfaces.RequestGroup),
		chainRPCService:     service,
		origProcessManager:  oriPMgr,
		crossProcessManager: crossPMgr,
		contractManager:     cMgr,
	}
	return scheduler
}

// Start start docker scheduler
func (s *RequestScheduler) Start() {

	s.logger.Debugf("start request scheduler")

	go func() {
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
				s.logger.Errorf("unknown msg type")
			}
		case msg := <-s.closeCh:
			if err := s.handleCloseReq(msg); err != nil {
				s.logger.Warnf("close request group %v", err)
			}
		}
	}()
}

func (s *RequestScheduler) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage:
		m, _ := msg.(*protogo.DockerVMMessage)
		s.eventCh <- m
	case string:
		m, _ := msg.(string)
		s.closeCh <- m
	default:
		s.logger.Errorf("unknown msg type")
	}
	return nil
}

func (s *RequestScheduler) handleGetContractReq(req *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.chainRPCService.PutMsg(req); err != nil {
		return err
	}
	return nil
}

func (s *RequestScheduler) handleGetContractResp(resp *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.contractManager.PutMsg(resp); err != nil {
		return err
	}
	return nil
}

func (s *RequestScheduler) handleTxReq(req *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	contractName := req.GetRequest().GetContractName()
	contractVersion := req.GetRequest().GetContractVersion()
	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)
	group, ok := s.requestGroups[groupKey]
	if !ok {
		s.logger.Debugf("create new request group %s", contractName)
		group = NewRequestGroup(contractName, contractVersion, s.origProcessManager, s.crossProcessManager, s.contractManager, s)
		s.requestGroups[contractName] = group
	}
	if err := group.PutMsg(req); err != nil {
		return err
	}
	return nil
}

func (s *RequestScheduler) handleErrResp(resp *protogo.DockerVMMessage) error {

	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.chainRPCService.PutMsg(resp); err != nil {
		return err
	}
	return nil
}

func (s *RequestScheduler) handleCloseReq(groupKey string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if _, ok := s.requestGroups[groupKey]; ok {
		delete(s.requestGroups, groupKey)
		return nil
	}
	return fmt.Errorf("request group %s not found", groupKey)
}

func (s *RequestScheduler) GetRequestGroup(contractName, contractVersion string) (interfaces.RequestGroup, error) {

	s.lock.RLock()
	defer s.lock.RUnlock()

	groupKey := utils.ConstructContractKey(contractName, contractVersion)
	if group, ok := s.requestGroups[groupKey]; ok {
		return group, nil
	}
	return nil, fmt.Errorf("request group %s not found", groupKey)
}

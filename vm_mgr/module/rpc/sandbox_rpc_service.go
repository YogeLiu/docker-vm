/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
)

// SandboxRPCService handles all messages of sandboxes (1 to n).
// Received message will be put into process's event chan.
// Messages will be sent by process itself.
type SandboxRPCService struct {
	logger     *zap.SugaredLogger        // sandbox rpc service logger
	processMgr interfaces.ProcessManager // process manager
}

// NewSandboxRPCService returns a sandbox rpc service
//  @param manager is process manager
//  @return *SandboxRPCService
func NewSandboxRPCService(manager interfaces.ProcessManager) *SandboxRPCService {
	return &SandboxRPCService{
		logger:     logger.NewDockerLogger(logger.MODULE_SANDBOX_RPC_SERVER, config.DockerLogDir),
		processMgr: manager,
	}
}

// DockerVMCommunicate is docker vm stream for sandboxes:
// 1. receive message
// 2. set stream of process
// 3. put messages into process's event chan
//  @param stream is grpc stream
//  @return error
func (s *SandboxRPCService) DockerVMCommunicate(stream protogo.DockerVMRpc_DockerVMCommunicateServer) error {
	for {
		msg, err := stream.Recv()
		if err != nil {
			s.logger.Errorf("fail to receive msg: %s", err)
			return err
		}
		process, err := s.processMgr.GetProcessByName(msg.ProcessInfo.CurrentProcessName)
		if err != nil {
			s.logger.Errorf("fail to get process: [%v]", err)
			return err
		}
		process.SetStream(stream)
		switch msg {
		case nil:
			s.logger.Errorf("[%s] received nil message, ending contract stream", process.GetName())
			return err
		default:
			s.logger.Debugf("[%s] handle msg [%v]", process.GetName(), msg)
			err = process.PutMsg(msg)
			if err != nil {
				s.logger.Errorf("fail to put msg into process chan: [%c]", err)
				return err
			}
		}
	}
}

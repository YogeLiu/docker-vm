/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"fmt"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
)

// SandboxRPCService handles all messages of sandboxes (1 to n).
// Received message will be put into process's event chan.
// Messages will be sent by process itself.
type SandboxRPCService struct {
	logger          *zap.SugaredLogger        // sandbox rpc service logger
	origProcessMgr  interfaces.ProcessManager // process manager
	crossProcessMgr interfaces.ProcessManager // process manager
}

// NewSandboxRPCService returns a sandbox rpc service
//  @param manager is process manager
//  @return *SandboxRPCService
func NewSandboxRPCService(origProcessMgr, crossProcessMgr interfaces.ProcessManager) *SandboxRPCService {
	return &SandboxRPCService{
		logger:          logger.NewDockerLogger(logger.MODULE_SANDBOX_RPC_SERVICE),
		origProcessMgr:  origProcessMgr,
		crossProcessMgr: crossProcessMgr,
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
			return fmt.Errorf("failed to recv msg: %s", err)
		}

		if msg == nil {
			return fmt.Errorf("recv nil message, ending contract stream")
		}

		s.logger.Debugf("receive msg from sandbox[%s]", msg.TxId)

		var process interfaces.Process
		var ok bool
		// process may be created, busy, timeout, recreated
		// created: ok (regular)
		// busy: ok (regular)
		// timeout: ok (restart, process abandon tx)
		// recreated: ok (process abandon tx)
		process, ok = s.origProcessMgr.GetProcessByName(msg.CrossContext.ProcessName)
		if !ok {
			process, ok = s.crossProcessMgr.GetProcessByName(msg.CrossContext.ProcessName)
		}

		if !ok {
			return fmt.Errorf("failed to get process, %v", err)
		}

		if msg.Type == protogo.DockerVMType_REGISTER {
			s.logger.Debugf("try to set stream, %s", msg.TxId)
			process.SetStream(stream)
			continue
		}
		s.logger.Debugf("end recv msg, txId: %s", msg.TxId)
		process.PutMsg(msg)
	}
}

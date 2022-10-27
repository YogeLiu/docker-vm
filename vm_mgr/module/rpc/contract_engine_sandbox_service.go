/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"fmt"
	"io"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
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
	var process interfaces.Process
	for {
		msg := utils.DockerVMMessageFromPool()
		err := stream.RecvMsg(msg)

		if err != nil {
			if err == io.EOF || status.Code(err) == codes.Canceled {
				s.logger.Debugf("sandbox client grpc stream closed (context cancelled)")
			} else {
				s.logger.Errorf("failed to recv msg: %s", err)
			}
			return err
		}

		if msg == nil {
			err = fmt.Errorf("recv nil message, ending contract stream")
			s.logger.Errorf(err.Error())
			return err
		}

		logger.DebugDynamic(s.logger, func() string {
			return fmt.Sprintf("receive msg from sandbox[%s]", msg.TxId)
		})

		var ok bool
		// process may be created, busy, timeout, recreated
		// created: ok (regular)
		// busy: ok (regular)
		// timeout: ok (restart, process abandon tx)
		// recreated: ok (process abandon tx)
		if process == nil {
			if process, ok = s.origProcessMgr.GetProcessByName(msg.CrossContext.ProcessName); !ok {
				if process, ok = s.crossProcessMgr.GetProcessByName(msg.CrossContext.ProcessName); !ok {
					err = fmt.Errorf("failed to get process, %v", err)
					s.logger.Errorf(err.Error())
					return err
				}
			}
		}

		if msg.Type == protogo.DockerVMType_REGISTER {
			s.logger.Debugf("try to set stream, %s", msg.TxId)
			process.SetStream(stream)
			continue
		}
		logger.DebugDynamic(s.logger, func() string {
			return fmt.Sprintf("end recv msg, txId: %s", msg.TxId)
		})
		process.PutMsg(msg)
	}
}

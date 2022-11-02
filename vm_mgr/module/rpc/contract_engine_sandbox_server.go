/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"runtime"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// SandboxRPCServer is server of bidirectional streaming RPC (sandbox <=> contract engine)
type SandboxRPCServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger
}

// NewSandboxRPCServer build new chain to sandbox rpc server.
func NewSandboxRPCServer(sockDir string) (*SandboxRPCServer, error) {

	log := logger.NewDockerLogger(logger.MODULE_SANDBOX_RPC_SERVER)

	if err := utils.CreateDir(sockDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create sandbox sock dir %s", sockDir)
	}

	sandboxRPCSockPath := filepath.Join(sockDir, config.SandboxRPCSockName)

	log.Infof("new sandbox rpc server(UDS) %s", sandboxRPCSockPath)

	listenAddress, err := net.ResolveUnixAddr("unix", sandboxRPCSockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve unix addr, %v", err)
	}

	listener, err := CreateUnixListener(listenAddress, sandboxRPCSockPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create unix listener, %v", err)
	}

	//set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    config.DockerVMConfig.RPC.ServerKeepAliveTime,
		Timeout: config.DockerVMConfig.RPC.ServerKeepAliveTimeout,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	//set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime:             config.DockerVMConfig.RPC.ServerMinInterval,
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))
	serverOpts = append(serverOpts, grpc.ConnectionTimeout(config.DockerVMConfig.RPC.ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(config.DockerVMConfig.RPC.MaxSendMsgSize*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(config.DockerVMConfig.RPC.MaxRecvMsgSize*1024*1024))
	serverOpts = append(serverOpts, grpc.ReadBufferSize(4*1024*1024))
	serverOpts = append(serverOpts, grpc.WriteBufferSize(4*1024*1024))
	serverOpts = append(serverOpts, grpc.InitialWindowSize(64*1024*1024))
	serverOpts = append(serverOpts, grpc.InitialConnWindowSize(64*1024*1024))
	serverOpts = append(serverOpts, grpc.NumStreamWorkers(uint32(runtime.NumCPU()/2)))

	server := grpc.NewServer(serverOpts...)

	return &SandboxRPCServer{
		Listener: listener,
		Server:   server,
		logger:   log,
	}, nil
}

// CreateUnixListener create an unix listener
func CreateUnixListener(listenAddress *net.UnixAddr, sockPath string) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		if err = os.Remove(sockPath); err != nil {
			return nil, fmt.Errorf("failed to remove %s", sockPath)
		}
		goto start
	}
	if err = os.Chmod(sockPath, 0777); err != nil {
		return nil, fmt.Errorf("failed to chmod %s", sockPath)
	}
	return listener, nil
}

// StartSandboxRPCServer starts the server:
// 1. register sandbox_rpc_service to server
// 2. start a goroutine to serve
func (s *SandboxRPCServer) StartSandboxRPCServer(service *SandboxRPCService) error {

	s.logger.Info("start sandbox rpc server")

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.Server, service)

	go func() {
		if err := s.Server.Serve(s.Listener); err != nil {
			s.logger.Errorf("sandbox rpc server exited, %v", err)
		}
	}()

	return nil
}

// StopSandboxRPCServer stops the server
func (s *SandboxRPCServer) StopSandboxRPCServer() {
	s.logger.Debugf("stop sandbox rpc server")
	if s.Server != nil {
		s.Server.Stop()
	}
}

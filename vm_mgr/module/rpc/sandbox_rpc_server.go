/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"errors"
	"net"
	"os"
	"path/filepath"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
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
func NewSandboxRPCServer() (*SandboxRPCServer, error) {

	sandboxRPCSockPath := filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockPath)

	listenAddress, err := net.ResolveUnixAddr("unix", sandboxRPCSockPath)
	if err != nil {
		return nil, err
	}

	listener, err := CreateUnixListener(listenAddress, sandboxRPCSockPath)
	if err != nil {
		return nil, err
	}

	//set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    config.ServerKeepAliveTime,
		Timeout: config.ServerKeepAliveTimeout,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	//set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime:             config.ServerMinInterval,
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))
	serverOpts = append(serverOpts, grpc.ConnectionTimeout(config.ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(utils.GetMaxSendMsgSizeFromEnv()*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(utils.GetMaxRecvMsgSizeFromEnv()*1024*1024))

	server := grpc.NewServer(serverOpts...)

	return &SandboxRPCServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_SANDBOX_RPC_SERVER, config.DockerLogDir),
	}, nil
}

// CreateUnixListener create an unix listener
func CreateUnixListener(listenAddress *net.UnixAddr, sockPath string) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		if err = os.Remove(sockPath); err != nil {
			return nil, err
		}
		goto start
	}
	if err = os.Chmod(sockPath, 0777); err != nil {
		return nil, err
	}
	return listener, nil
}

// StartSandboxRPCServer starts the server:
// 1. register sandbox_rpc_service to server
// 2. start a goroutine to serve
func (s *SandboxRPCServer) StartSandboxRPCServer(service *SandboxRPCService) error {

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.Server, service)

	s.logger.Debugf("start sandbox_rpc_server...")

	go func() {
		if err := s.Server.Serve(s.Listener); err != nil {
			s.logger.Errorf("fail to start sandbox_rpc_server: %s", err)
		}
	}()

	return nil
}

// StopSandboxRPCServer stops the server
func (s *SandboxRPCServer) StopSandboxRPCServer() {
	s.logger.Debugf("stop sandbox_rpc_server...")
	if s.Server != nil {
		s.Server.Stop()
	}
}

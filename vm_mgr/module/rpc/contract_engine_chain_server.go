/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"errors"
	"fmt"
	"net"
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

const (
	ChainRPCDir      = "contract-engine-sock" // /mount/sock
	ChainRPCSockName = "chain.sock"           // /mount/sock/chain.sock
)

// ChainRPCServer is server of bidirectional streaming RPC (chain <=> contract engine)
type ChainRPCServer struct {
	Listener net.Listener       // chain network listener for stream-oriented protocols
	Server   *grpc.Server       // grpc server for chain
	logger   *zap.SugaredLogger // chain rpc server logger
}

// NewChainRPCServer build new chain to docker vm rpc server:
// 1. choose UDS/TCP grpc server
// 2. set max_send_msg_size, max_recv_msg_size, etc.
func NewChainRPCServer() (*ChainRPCServer, error) {

	log := logger.NewDockerLogger(logger.MODULE_CHAIN_RPC_SERVER)

	// choose unix domain socket or tcp
	netProtocol := config.DockerVMConfig.RPC.ChainRPCProtocol

	var listener net.Listener
	var err error

	if netProtocol == config.TCP {

		port := config.DockerVMConfig.RPC.ChainRPCPort

		endPoint := fmt.Sprintf(":%d", port)

		log.Infof("new chain rpc server(TCP) %s", endPoint)

		if listener, err = net.Listen("tcp", endPoint); err != nil {
			return nil, fmt.Errorf("failed to listen tcp, %v", err)
		}

	} else {

		sockDir := filepath.Join(config.DockerMountDir, ChainRPCDir)
		if err = utils.CreateDir(sockDir, 0755); err != nil {
			return nil, fmt.Errorf("failed to create chain rpc sock dir %s", sockDir)
		}

		absChainRPCUDSPath := filepath.Join(sockDir, ChainRPCSockName)

		log.Infof("new chain rpc server(UDS) %s", absChainRPCUDSPath)

		listenAddress, err := net.ResolveUnixAddr("unix", absChainRPCUDSPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve unix addr, %v", err)
		}

		if listener, err = CreateUnixListener(listenAddress, absChainRPCUDSPath); err != nil {
			return nil, fmt.Errorf("failed to create unix listener, %v", err)
		}
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

	return &ChainRPCServer{
		Listener: listener,
		Server:   server,
		logger:   log,
	}, nil
}

// StartChainRPCServer starts the server:
// 1. register chain_rpc_service to server
// 2. start a goroutine to serve
func (s *ChainRPCServer) StartChainRPCServer(service *ChainRPCService) error {

	s.logger.Infof("start chain rpc server")

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.Server, service)

	go func() {
		if err := s.Server.Serve(s.Listener); err != nil {
			s.logger.Errorf("chain rpc server exited: %s", err)
		}
	}()

	return nil
}

// StopChainRPCServer stops the server
func (s *ChainRPCServer) StopChainRPCServer() {
	s.logger.Debugf("stop chain rpc server")
	if s.Server != nil {
		s.Server.Stop()
	}
}

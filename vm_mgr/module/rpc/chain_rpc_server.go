/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

// ChainRPCServer is server of bidirectional streaming RPC
type ChainRPCServer struct {
	Listener net.Listener       // chain network listener for stream-oriented protocols
	Server   *grpc.Server       // grpc server for chain
	logger   *zap.SugaredLogger // chain rpc server logger
}

// NewChainRPCServer build new chain to docker vm rpc server:
// 1. choose UDS/TCP grpc server
// 2. set max_send_msg_size, max_recv_msg_size, etc.
func NewChainRPCServer() (*ChainRPCServer, error) {

	// choose unix domin socket or tcp
	enableUnixDomainSocket, _ := strconv.ParseBool(os.Getenv(config.ENV_ENABLE_UDS))

	var listener net.Listener
	var err error

	if !enableUnixDomainSocket {
		port := os.Getenv("Port")
		if port == "" {
			return nil, errors.New("server listen port not provided")
		}

		endPoint := fmt.Sprintf(":%s", port)

		if listener, err = net.Listen("tcp", endPoint); err != nil {
			return nil, err
		}
	} else {
		absChainRPCUDSPath := filepath.Join(config.SockBaseDir, config.ChainRPCSockName)

		listenAddress, err := net.ResolveUnixAddr("unix", absChainRPCUDSPath)
		if err != nil {
			return nil, err
		}

		if listener, err = CreateUnixListener(listenAddress, absChainRPCUDSPath); err != nil {
			return nil, err
		}
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

	return &ChainRPCServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_CHAIN_RPC_SERVER, config.DockerLogDir),
	}, nil
}

// StartChainRPCServer starts the server:
// 1. register chain_rpc_service to server
// 2. start a goroutine to serve
func (s *ChainRPCServer) StartChainRPCServer(service *ChainRPCService) error {

	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.Server, service)

	s.logger.Debugf("start chain_rpc_server...")

	go func() {
		if err := s.Server.Serve(s.Listener); err != nil {
			s.logger.Errorf("fail to start chain_rpc_server: %s", err)
		}
	}()

	return nil
}

// StopChainRPCServer stops the server
func (s *ChainRPCServer) StopChainRPCServer() {
	s.logger.Debugf("stop chain_rpc_server...")
	if s.Server != nil {
		s.Server.Stop()
	}
}

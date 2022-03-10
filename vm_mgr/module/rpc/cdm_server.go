/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type CDMServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger
}

// NewCDMServer build new chainmaker to docker manager rpc server
func NewCDMServer() (*CDMServer, error) {

	enableUnixDomainSocket, _ := strconv.ParseBool(os.Getenv(config.ENV_ENABLE_UDS))

	var listener net.Listener
	var err error

	if !enableUnixDomainSocket {
		port := os.Getenv("Port")

		if port == "" {
			return nil, errors.New("server listen port not provided")
		}

		endPoint := fmt.Sprintf(":%s", port)
		listener, err = net.Listen("tcp", endPoint)
		if err != nil {
			return nil, err
		}
	} else {

		absCdmUDSPath := filepath.Join(config.SockBaseDir, config.SockName)

		listenAddress, err := net.ResolveUnixAddr("unix", absCdmUDSPath)
		if err != nil {
			return nil, err
		}

		listener, err = CreateUnixListener(listenAddress, absCdmUDSPath)
		if err != nil {
			return nil, err
		}

	}

	//set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	//set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime:             ServerMinInterval,
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))
	serverOpts = append(serverOpts, grpc.ConnectionTimeout(ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(utils.GetMaxSendMsgSizeFromEnv()*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(utils.GetMaxRecvMsgSizeFromEnv()*1024*1024))

	server := grpc.NewServer(serverOpts...)

	return &CDMServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_CDM_SERVER, config.DockerLogDir),
	}, nil
}

// StartCDMServer Start the server
func (cdm *CDMServer) StartCDMServer(apiInstance *CDMApi) error {

	var err error

	if cdm.Listener == nil {
		return errors.New("nil listener")
	}

	if cdm.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterCDMRpcServer(cdm.Server, apiInstance)

	cdm.logger.Debugf("start cdm server")

	go func() {
		err = cdm.Server.Serve(cdm.Listener)
		if err != nil {
			cdm.logger.Errorf("cdm server fail to start: %s", err)
		}
	}()

	return nil
}

// StopCDMServer Stop the server
func (cdm *CDMServer) StopCDMServer() {
	cdm.logger.Infof("stop cdm server")
	if cdm.Server != nil {
		cdm.Server.Stop()
	}
}

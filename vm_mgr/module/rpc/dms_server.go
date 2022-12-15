/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"errors"
	"net"
	"os"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/pb_sdk/protogo"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type DMSServer struct {
	Listener net.Listener
	Server   *grpc.Server
	logger   *zap.SugaredLogger
}

// NewDMSServer build new docker manager to sandbox server, current: each server in charge of one sandbox
func NewDMSServer() (*DMSServer, error) {

	dmsSockPath := filepath.Join(config.DMSDir, config.DMSSockPath)

	listenAddress, err := net.ResolveUnixAddr("unix", dmsSockPath)
	if err != nil {
		return nil, err
	}

	listener, err := CreateUnixListener(listenAddress, dmsSockPath)
	if err != nil {
		return nil, err
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

	return &DMSServer{
		Listener: listener,
		Server:   server,
		logger:   logger.NewDockerLogger(logger.MODULE_DMS_SERVER, config.DockerLogDir),
	}, nil
}

func CreateUnixListener(listenAddress *net.UnixAddr, sockPath string) (*net.UnixListener, error) {
start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		err = os.Remove(sockPath)
		if err != nil {
			return nil, err
		}
		goto start
	}
	if err = os.Chmod(sockPath, 0777); err != nil {
		return nil, err
	}
	return listener, nil

}

// StartDMSServer Start the server
func (dms *DMSServer) StartDMSServer(dmsApi *DMSApi) error {

	if dms.Listener == nil {
		return errors.New("nil listener")
	}

	if dms.Server == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDMSRpcServer(dms.Server, dmsApi)

	dms.logger.Debugf("start dms server")

	go func() {
		err := dms.Server.Serve(dms.Listener)
		if err != nil {
			dms.logger.Errorf("dms server fail to start: %s", err)
		}
	}()

	return nil
}

// StopDMSServer Stop the server
func (dms *DMSServer) StopDMSServer() {

	dms.logger.Infof("stop dms server")

	if dms.Server != nil {
		dms.Server.Stop()
	}
}

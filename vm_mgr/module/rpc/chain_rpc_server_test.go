/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

// Package rpc includes 2 rpc servers, one for chainmaker client(1-1), the other one for sandbox (1-n)
package rpc

import (
	"net"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"google.golang.org/grpc/keepalive"

	"go.uber.org/zap"
	"google.golang.org/grpc"
)

func TestChainRPCServer_StartChainRPCServer(t *testing.T) {
	s := newMockScheduler(t)
	defer s.finish()
	scheduler := s.getScheduler()

	os.Setenv("Port", "8080")
	server, err := NewChainRPCServer()
	if err != nil {
		t.Error(err.Error())
		return
	}

	type fields struct {
		Listener net.Listener
		Server   *grpc.Server
		logger   *zap.SugaredLogger
	}

	type args struct {
		apiInstance *ChainRPCService
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testStartCDMServer",
			fields: fields{
				Listener: server.Listener,
				Server:   server.Server,
				logger:   utils.GetLogHandler(),
			},
			args: args{
				apiInstance: NewChainRPCService(scheduler),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}

			os.Setenv("UdsOpen", "true")
			if err := cdm.StartChainRPCServer(tt.args.apiInstance); (err != nil) != tt.wantErr {
				t.Errorf("StartChainRPCServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMServer_StopCDMServer(t *testing.T) {
	os.Setenv("Port", "8083")
	server, err := NewChainRPCServer()
	if err != nil {
		t.Error(err.Error())
		return
	}
	type fields struct {
		Listener net.Listener
		Server   *grpc.Server
		logger   *zap.SugaredLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "testStopCDMServer",
			fields: fields{
				Server: server.Server,
				logger: utils.GetLogHandler(),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}
			cdm.StopChainRPCServer()
		})
	}
}

func TestNewCDMServer(t *testing.T) {
	os.Setenv("Port", "8099")
	server, err := NewChainRPCServer()
	if err != nil {
		t.Error(err.Error())
		return
	}

	tests := []struct {
		name    string
		want    *ChainRPCServer
		wantErr bool
	}{
		{
			name:    "testNewCDMServe",
			want:    server,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("Port", "8089")
			got, err := NewChainRPCServer()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChainRPCServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if got == nil {
				t.Errorf("NewChainRPCServer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// NewChainRPCServer build new chainmaker to docker manager rpc server
func newCDMServer(defaultPort string) (*ChainRPCServer, error) {
	enableUnixDomainSocket, _ := strconv.ParseBool(os.Getenv("UdsOpen"))

	var listener net.Listener
	var err error

	if !enableUnixDomainSocket {
		port := os.Getenv("Port")

		if port == "" {
			port = defaultPort
		}

		listener, err = net.Listen("tcp", ":"+port)
		if err != nil {
			return nil, err
		}
	} else {

		absCdmUDSPath := filepath.Join(config.SockBaseDir, config.ChainRPCSockName)

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
		MinTime: config.ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	//set default connection timeout
	maxSendSizeConfig := os.Getenv("MaxSendMessageSize")
	maxRecvSizeConfig := os.Getenv("MaxRecvMessageSize")

	maxSendSize, _ := strconv.Atoi(maxSendSizeConfig)
	maxRecvSize, _ := strconv.Atoi(maxRecvSizeConfig)

	serverOpts = append(serverOpts, grpc.ConnectionTimeout(config.ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(maxSendSize*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(maxRecvSize*1024*1024))

	server := grpc.NewServer(serverOpts...)

	return &ChainRPCServer{
		Listener: listener,
		Server:   server,
		logger:   utils.GetLogHandler(),
	}, nil
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"net"
	"testing"
)

const (
	configFileName = "../../../test/testdata/config/vm.yml"
)

var port = 20001

func TestChainRPCServer_StartChainRPCServer(t *testing.T) {

	SetConfig()

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
		service *ChainRPCService
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCServer_StartChainRPCServer",
			fields: fields{
				Listener: server.Listener,
				Server:   server.Server,
				logger:   logger.NewTestDockerLogger(),
			},
			args: args{
				service: NewChainRPCService(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}
			if err := s.StartChainRPCServer(tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("StartChainRPCServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainRPCServer_StopChainRPCServer(t *testing.T) {

	SetConfig()

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
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "TestChainRPCServer_StopChainRPCServer",
			fields: fields{
				Listener: server.Listener,
				Server:   server.Server,
				logger:   logger.NewTestDockerLogger(),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}
			s.StopChainRPCServer()
		})
	}
}

func TestNewChainRPCServer(t *testing.T) {

	SetConfig()

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
			name:    "TestNewChainRPCServer",
			want:    server,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.DockerVMConfig.RPC.ChainRPCPort = 8016
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

func SetConfig() {
	_ = config.InitConfig(configFileName)

	config.DockerVMConfig.RPC.ChainRPCProtocol = config.TCP
	config.DockerVMConfig.RPC.ServerKeepAliveTime = 60
	config.DockerVMConfig.RPC.ServerKeepAliveTimeout = 20
	config.DockerVMConfig.RPC.ConnectionTimeout = 5
	config.DockerVMConfig.RPC.MaxSendMsgSize = 4
	config.DockerVMConfig.RPC.MaxRecvMsgSize = 4
	config.DockerVMConfig.RPC.ChainRPCPort = port
	port++
}

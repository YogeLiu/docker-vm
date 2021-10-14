/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/protocol"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	"golang.org/x/sync/singleflight"
	"os"
	"reflect"
	"sync"
	"testing"
)

const (
	contractName    = "contractName1"
	contractNameBad = "contractName2"
	contractValue   = "contractValue1"
	contractVersion = "contractVersion1"
	contractPath  = "contractPath1"
	sockPath  = "sockPath"
	testPath = "/test"
)

func TestContractManager_GetContract(t *testing.T) {
	currentPath, _ := os.Getwd()
	logPath := currentPath + testPath
	type fields struct {
		lock            sync.RWMutex
		getContractLock singleflight.Group
		contractsMap    map[string]string
		logger          *zap.SugaredLogger
		scheduler       *protocol.MockScheduler
	}

	type args struct {
		txId         string
		contractName string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "testGetContract",
			fields: fields{
				lock:            sync.RWMutex{},
				getContractLock: singleflight.Group{},
				contractsMap: map[string]string{
					contractName: contractValue,
				},
				logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath),
				scheduler: protocol.NewMockScheduler(gomock.NewController(t)),
			},
			args: args{
				txId:         "txId",
				contractName: contractName,
			},
			want:    contractValue,
			wantErr: false,
		},

		//{
		//	name: "testGetContractOtherBad",
		//	fields: fields{
		//		lock:            sync.RWMutex{},
		//		getContractLock: singleflight.Group{},
		//		contractsMap: map[string]string{
		//			contractName: contractValue,
		//		},
		//		logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, currentPath),
		//		scheduler: protocol.NewMockScheduler(gomock.NewController(t)),
		//	},
		//	args: args{
		//		txId:         "111",
		//		contractName: contractNameBad,
		//	},
		//	want:    contractValue,
		//	wantErr: true,
		//},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ContractManager{
				lock:            tt.fields.lock,
				getContractLock: tt.fields.getContractLock,
				contractsMap:    tt.fields.contractsMap,
				logger:          tt.fields.logger,
				scheduler:       tt.fields.scheduler,
			}
			got, err := cm.GetContract(tt.args.txId, tt.args.contractName)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetContract() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GetContract() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContractManager_checkContractDeployed(t *testing.T) {
	type fields struct {
		lock            sync.RWMutex
		getContractLock singleflight.Group
		contractsMap    map[string]string
		logger          *zap.SugaredLogger
		scheduler       protocol.Scheduler
	}
	type args struct {
		contractName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
		want1  bool
	}{
		{
			name: "good",
			fields: fields{
				lock:            sync.RWMutex{},
				getContractLock: singleflight.Group{},
				contractsMap: map[string]string{
					contractName: contractValue,
				},
				logger:    nil,
				scheduler: nil,
			},
			args: args{
				contractName: contractName,
			},
			want:  contractValue,
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ContractManager{
				lock:            tt.fields.lock,
				getContractLock: tt.fields.getContractLock,
				contractsMap:    tt.fields.contractsMap,
				logger:          tt.fields.logger,
				scheduler:       tt.fields.scheduler,
			}
			got, got1 := cm.checkContractDeployed(tt.args.contractName)
			if got != tt.want {
				t.Errorf("checkContractDeployed() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("checkContractDeployed() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestContractManager_initialContractMap(t *testing.T) {
	currentPath, _ := os.Getwd()
	logPath := currentPath + testPath
	type fields struct {
		lock            sync.RWMutex
		getContractLock singleflight.Group
		contractsMap    map[string]string
		logger          *zap.SugaredLogger
		scheduler       protocol.Scheduler
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "testInitialContractMap",
			fields: fields{
				lock:            sync.RWMutex{},
				getContractLock: singleflight.Group{},
				contractsMap: map[string]string{
					contractName: contractValue,
				},
				logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath),
				scheduler: nil,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ContractManager{
				lock:            tt.fields.lock,
				getContractLock: tt.fields.getContractLock,
				contractsMap:    tt.fields.contractsMap,
				logger:          tt.fields.logger,
				scheduler:       tt.fields.scheduler,
			}

			mountDir = currentPath
			if err := cm.initialContractMap(); (err != nil) != tt.wantErr {
				t.Errorf("initialContractMap() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContractManager_lookupContractFromDB(t *testing.T) {
	//currentPath, _ := os.Getwd()
	//logPath := currentPath + testPath
	type fields struct {
		lock            sync.RWMutex
		getContractLock singleflight.Group
		contractsMap    map[string]string
		logger          *zap.SugaredLogger
		scheduler       *protocol.MockScheduler
	}
	type args struct {
		txId         string
		contractName string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		//{
		//	name: "testGetContract",
		//	fields: fields{
		//		lock:            sync.RWMutex{},
		//		getContractLock: singleflight.Group{},
		//		contractsMap: map[string]string{
		//			contractName: contractValue,
		//		},
		//		logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath),
		//		scheduler: protocol.NewMockScheduler(gomock.NewController(t)),
		//	},
		//	args: args{
		//		txId:         "txId",
		//		contractName: contractName,
		//	},
		//	want:    contractValue,
		//	wantErr: false,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ContractManager{
				lock:            tt.fields.lock,
				getContractLock: tt.fields.getContractLock,
				contractsMap:    tt.fields.contractsMap,
				logger:          tt.fields.logger,
				scheduler:       tt.fields.scheduler,
			}
			got, err := cm.lookupContractFromDB(tt.args.txId, tt.args.contractName)
			if (err != nil) != tt.wantErr {
				t.Errorf("lookupContractFromDB() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("lookupContractFromDB() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestContractManager_setFileMod(t *testing.T) {
	currentPath, _ := os.Getwd()
	type fields struct {
		lock            sync.RWMutex
		getContractLock singleflight.Group
		contractsMap    map[string]string
		logger          *zap.SugaredLogger
		scheduler       protocol.Scheduler
	}
	type args struct {
		filePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testSetFileMod",
			fields: fields{
				lock:            sync.RWMutex{},
				getContractLock: singleflight.Group{},
				contractsMap:    nil,
				logger:          nil,
				scheduler:       nil,
			},
			args:    args{filePath: currentPath},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := &ContractManager{
				lock:            tt.fields.lock,
				getContractLock: tt.fields.getContractLock,
				contractsMap:    tt.fields.contractsMap,
				logger:          tt.fields.logger,
				scheduler:       tt.fields.scheduler,
			}
			if err := cm.setFileMod(tt.args.filePath); (err != nil) != tt.wantErr {
				t.Errorf("setFileMod() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewContractManager(t *testing.T) {
	currentPath, _ := os.Getwd()
	logPath := currentPath + testPath
	log := logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath)
	tests := []struct {
		name string
		want *ContractManager
	}{
		{
			name: "NewContractManager",
			want: &ContractManager{
				contractsMap: make(map[string]string),
				logger:       log,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mountDir = currentPath
			got := NewContractManager(currentPath)
			got.logger = log

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewContractManager() = %v, want %v", got, tt.want)
			}
		})
	}
}

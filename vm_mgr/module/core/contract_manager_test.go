///*
//Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
//
package core

//
//import (
//	"os"
//	"reflect"
//	"sync"
//	"testing"
//
//	"github.com/golang/mock/gomock"
//	"go.uber.org/zap"
//	"golang.org/x/sync/singleflight"
//
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
//)
//
//const (
//	contractName = "contractName1"
//	//contractNameBad = "contractName2"
//	contractValue   = "contractValue1"
//	contractVersion = "contractVersion1"
//	contractPath    = "contractPath1"
//	txId            = "0xb0ff781740fd5bc45f63c7d4f572384343c3c8e8a7e64d602d0c95651b804352"
//	payload         = "payload1"
//	sockPath        = "sockPath"
//	method          = "method1"
//	testPath        = "/"
//)
//
//func TestContractManager_GetContract(t *testing.T) {
//	c := gomock.NewController(t)
//	defer c.Finish()
//	responseChan := make(chan *protogo.CDMMessage)
//	scheduler := protocol.NewMockScheduler(c)
//	scheduler.EXPECT().RegisterResponseCh(txId, responseChan).Return().AnyTimes()
//	type fields struct {
//		lock            sync.RWMutex
//		getContractLock singleflight.Group
//		contractsLRU    map[string]string
//		logger          *zap.SugaredLogger
//		scheduler       *protocol.MockScheduler
//	}
//
//	type args struct {
//		txId         string
//		contractName string
//	}
//
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    string
//		wantErr bool
//	}{
//		{
//			name: "testGetContract",
//			fields: fields{
//				lock:            sync.RWMutex{},
//				getContractLock: singleflight.Group{},
//				contractsLRU: map[string]string{
//					contractName: contractValue,
//				},
//				logger:    utils.GetLogHandler(),
//				scheduler: scheduler,
//			},
//			args: args{
//				txId:         "txId",
//				contractName: contractName,
//			},
//			want:    contractValue,
//			wantErr: false,
//		},
//
//		//{
//		//	name: "testGetContractOtherBad",
//		//	fields: fields{
//		//		lock:            sync.RWMutex{},
//		//		getContractLock: singleflight.Group{},
//		//		contractsLRU: map[string]string{
//		//			contractName: contractValue,
//		//		},
//		//		logger:    utils.GetLogHandler(),
//		//		scheduler: scheduler,
//		//	},
//		//	args: args{
//		//		txId:         txId,
//		//		contractName: contractNameBad,
//		//	},
//		//	want:    contractValue,
//		//	wantErr: false,
//		//},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cm := &ContractManager{
//				lock:            tt.fields.lock,
//				getContractLock: tt.fields.getContractLock,
//				contractsLRU:    tt.fields.contractsLRU,
//				logger:          tt.fields.logger,
//				scheduler:       tt.fields.scheduler,
//			}
//			got, err := cm.handleGetContractReq(tt.args.txId, tt.args.contractName)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("handleGetContractReq() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("handleGetContractReq() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestContractManager_checkContractDeployed(t *testing.T) {
//	type fields struct {
//		lock            sync.RWMutex
//		getContractLock singleflight.Group
//		contractsLRU    map[string]string
//		logger          *zap.SugaredLogger
//		scheduler       protocol.Scheduler
//	}
//	type args struct {
//		contractName string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   string
//		want1  bool
//	}{
//		{
//			name: "good",
//			fields: fields{
//				lock:            sync.RWMutex{},
//				getContractLock: singleflight.Group{},
//				contractsLRU: map[string]string{
//					contractName: contractValue,
//				},
//				logger:    nil,
//				scheduler: nil,
//			},
//			args: args{
//				contractName: contractName,
//			},
//			want:  contractValue,
//			want1: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cm := &ContractManager{
//				lock:            tt.fields.lock,
//				getContractLock: tt.fields.getContractLock,
//				contractsLRU:    tt.fields.contractsLRU,
//				logger:          tt.fields.logger,
//				scheduler:       tt.fields.scheduler,
//			}
//			got, got1 := cm.checkContractDeployed(tt.args.contractName)
//			if got != tt.want {
//				t.Errorf("checkContractDeployed() got = %v, want %v", got, tt.want)
//			}
//			if got1 != tt.want1 {
//				t.Errorf("checkContractDeployed() got1 = %v, want %v", got1, tt.want1)
//			}
//		})
//	}
//}
//
//func TestContractManager_initialContractMap(t *testing.T) {
//	currentPath, _ := os.Getwd()
//	logPath := currentPath + testPath
//	type fields struct {
//		lock            sync.RWMutex
//		getContractLock singleflight.Group
//		contractsLRU    map[string]string
//		logger          *zap.SugaredLogger
//		scheduler       protocol.Scheduler
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		{
//			name: "testInitialContractMap",
//			fields: fields{
//				lock:            sync.RWMutex{},
//				getContractLock: singleflight.Group{},
//				contractsLRU: map[string]string{
//					contractName: contractValue,
//				},
//				logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath),
//				scheduler: nil,
//			},
//			wantErr: false,
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cm := &ContractManager{
//				lock:            tt.fields.lock,
//				getContractLock: tt.fields.getContractLock,
//				contractsLRU:    tt.fields.contractsLRU,
//				logger:          tt.fields.logger,
//				scheduler:       tt.fields.scheduler,
//			}
//
//			mountDir = currentPath
//			if err := cm.initContractLRU(); (err != nil) != tt.wantErr {
//				t.Errorf("initContractLRU() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestContractManager_lookupContractFromDB(t *testing.T) {
//	//currentPath, _ := os.Getwd()
//	//logPath := currentPath + testPath
//	type fields struct {
//		lock            sync.RWMutex
//		getContractLock singleflight.Group
//		contractsLRU    map[string]string
//		logger          *zap.SugaredLogger
//		scheduler       *protocol.MockScheduler
//	}
//	type args struct {
//		txId         string
//		contractName string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		want    string
//		wantErr bool
//	}{
//		//{
//		//	name: "testGetContract",
//		//	fields: fields{
//		//		lock:            sync.RWMutex{},
//		//		getContractLock: singleflight.Group{},
//		//		contractsLRU: map[string]string{
//		//			contractName: contractValue,
//		//		},
//		//		logger:    logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath),
//		//		scheduler: protocol.NewMockScheduler(gomock.NewController(t)),
//		//	},
//		//	args: args{
//		//		txId:         "txId",
//		//		contractName: contractName,
//		//	},
//		//	want:    contractValue,
//		//	wantErr: false,
//		//},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cm := &ContractManager{
//				lock:            tt.fields.lock,
//				getContractLock: tt.fields.getContractLock,
//				contractsLRU:    tt.fields.contractsLRU,
//				logger:          tt.fields.logger,
//				scheduler:       tt.fields.scheduler,
//			}
//			got, err := cm.requestContractFromChain(tt.args.txId, tt.args.contractName)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("requestContractFromChain() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if got != tt.want {
//				t.Errorf("requestContractFromChain() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestContractManager_setFileMod(t *testing.T) {
//	currentPath, _ := os.Getwd()
//	type fields struct {
//		lock            sync.RWMutex
//		getContractLock singleflight.Group
//		contractsLRU    map[string]string
//		logger          *zap.SugaredLogger
//		scheduler       protocol.Scheduler
//	}
//	type args struct {
//		filePath string
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "testSetFileMod",
//			fields: fields{
//				lock:            sync.RWMutex{},
//				getContractLock: singleflight.Group{},
//				contractsLRU:    nil,
//				logger:          nil,
//				scheduler:       nil,
//			},
//			args:    args{filePath: currentPath},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			cm := &ContractManager{
//				lock:            tt.fields.lock,
//				getContractLock: tt.fields.getContractLock,
//				contractsLRU:    tt.fields.contractsLRU,
//				logger:          tt.fields.logger,
//				scheduler:       tt.fields.scheduler,
//			}
//			if err := cm.setFileMod(tt.args.filePath); (err != nil) != tt.wantErr {
//				t.Errorf("setFileMod() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestNewContractManager(t *testing.T) {
//	currentPath, _ := os.Getwd()
//	logPath := currentPath + testPath
//	logger := logger.NewDockerLogger(logger.MODULE_CONTRACT_MANAGER, logPath)
//	tests := []struct {
//		name string
//		want *ContractManager
//	}{
//		{
//			name: "NewContractManager",
//			want: &ContractManager{
//				contractsLRU: make(map[string]string),
//				logger:       logger,
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//
//			mountDir = currentPath
//			got := NewContractManager(currentPath)
//			got.logger = logger
//
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewContractManager() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

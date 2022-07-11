/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"path/filepath"
	"testing"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
)

func TestContractManager_GetContractMountDir(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}

	type fields struct {
		cMgr *ContractManager
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestContractManager_GetContractMountDir",
			fields: fields{
				cMgr: cMgr,
			},
			want: "/mount/contracts",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			if got := cm.GetContractMountDir(); got != tt.want {
				t.Errorf("GetContractMountDir() = %v, wantInCache %v", got, tt.want)
			}
		})
	}
}

func TestContractManager_PutMsg(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}

	type fields struct {
		cMgr *ContractManager
	}
	type args struct {
		msg interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestContractManager_PutMsg_DockerVMMessage",
			fields: fields{
				cMgr: cMgr,
			},
			args:    args{msg: &protogo.DockerVMMessage{}},
			wantErr: false,
		},
		{
			name: "TestContractManager_PutMsg_String",
			fields: fields{
				cMgr: cMgr,
			},
			args:    args{msg: "test"},
			wantErr: true,
		},
		{
			name: "TestContractManager_PutMsg_Int",
			fields: fields{
				cMgr: cMgr,
			},
			args:    args{msg: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			if err := cm.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContractManager_SetScheduler(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}

	type fields struct {
		cMgr *ContractManager
	}

	type args struct {
		scheduler interfaces.RequestScheduler
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "TestContractManager_SetScheduler",
			fields: fields{
				cMgr: cMgr,
			},
			args: args{scheduler: &RequestScheduler{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			cm.SetScheduler(tt.args.scheduler)
		})
	}
}

func TestContractManager_Start(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
	}
	cMgr.Start()

	type fields struct {
		cMgr *ContractManager
	}

	type args struct {
		req *protogo.DockerVMMessage
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "TestContractManager_Start",
			fields: fields{cMgr: cMgr},
			args: args{req: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			}},
		},
		{
			name:   "TestContractManager_Start",
			fields: fields{cMgr: cMgr},
			args: args{req: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
			}},
		},
		{
			name:   "TestContractManager_Start",
			fields: fields{cMgr: cMgr},
			args: args{req: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_ERROR,
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			cm.PutMsg(tt.args.req)
		})
	}
}

func TestContractManager_handleGetContractReq(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			txCh:    make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
			eventCh: make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"chain1#testContractName#1.0.0": &RequestGroup{
				txCh:    make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
				eventCh: make(chan interface{}, _requestGroupEventChSize),
			}},
		},
	}
	cMgr.contractsLRU.Add("chain1#testContractName#1.0.0", "/mount/contracts/chain1#testContractName#1.0.0")

	type fields struct {
		cMgr *ContractManager
	}
	type args struct {
		req *protogo.DockerVMMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestContractManager_handleGetContractReq",
			fields: fields{cMgr: cMgr},
			args: args{req: &protogo.DockerVMMessage{
				TxId: "testTxId",
				Type: protogo.DockerVMType_TX_REQUEST,
				Request: &protogo.TxRequest{
					ChainId:         "chain1",
					ContractName:    "testContractName",
					ContractVersion: "1.0.0",
				},
			}},
			wantErr: false,
		},
		{
			name:   "TestContractManager_handleGetContractReq",
			fields: fields{cMgr: cMgr},
			args: args{req: &protogo.DockerVMMessage{
				TxId: "testTxId",
				Type: protogo.DockerVMType_TX_REQUEST,
				Request: &protogo.TxRequest{
					ChainId:         "chain1",
					ContractName:    "testContractName2",
					ContractVersion: "1.0.0",
				},
			}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			if err := cm.handleGetContractReq(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("handleGetContractReq() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestContractManager_handleGetContractResp(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			txCh:    make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
			eventCh: make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"chain1#testContractName#1.0.0": &RequestGroup{
				txCh:    make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
				eventCh: make(chan interface{}, _requestGroupEventChSize),
			}},
		},
	}

	type fields struct {
		cMgr *ContractManager
	}
	type args struct {
		resp *protogo.DockerVMMessage
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantExist bool
		wantErr   bool
	}{
		{
			name:   "TestContractManager_handleGetContractResp",
			fields: fields{cMgr: cMgr},
			args: args{resp: &protogo.DockerVMMessage{
				TxId: "testTxId",
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
				Response: &protogo.TxResponse{
					ChainId:         "chain1",
					ContractName:    "testContractName",
					ContractVersion: "1.0.0",
					Code:            protogo.DockerVMCode_FAIL,
				},
			}},
			wantExist: false,
			wantErr:   true,
		},
		{
			name:   "TestContractManager_handleGetContractResp",
			fields: fields{cMgr: cMgr},
			args: args{resp: &protogo.DockerVMMessage{
				TxId: "testTxId",
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
				Response: &protogo.TxResponse{
					ChainId:         "chain1",
					ContractName:    "testContractName",
					ContractVersion: "1.0.0",
				},
			}},
			wantExist: true,
			wantErr:   true,
		},
		{
			name: "TestContractManager_handleGetContractResp",
			fields: fields{cMgr: &ContractManager{
				contractsLRU: cMgr.contractsLRU,
				logger:       cMgr.logger,
				eventCh:      cMgr.eventCh,
				mountDir:     cMgr.mountDir,
			}},
			args: args{resp: &protogo.DockerVMMessage{
				TxId: "testTxId",
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
				Response: &protogo.TxResponse{
					ChainId:         "chain1",
					ContractName:    "testContractName",
					ContractVersion: "1.0.0",
				},
			}},
			wantExist: true,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			if err := cm.handleGetContractResp(tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("handleGetContractResp() error = %v, wantErr %v", err, tt.wantErr)
			}
			groupKey := utils.ConstructContractKey(
				tt.args.resp.Response.ChainId,
				tt.args.resp.Response.ContractName,
				tt.args.resp.Response.ContractVersion)
			_, exist := cm.contractsLRU.Get(groupKey)
			if exist != tt.wantExist {
				t.Errorf("handleGetContractResp() value existed = %v, wantExist %v", exist, tt.wantExist)
			}
		})
	}
}

func TestContractManager_sendContractReadySignal(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileSize),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan interface{}, _contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			txCh:    make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
			eventCh: make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"chain1#testContractName#1.0.0": &RequestGroup{
				txCh:    make(chan *protogo.DockerVMMessage, _requestGroupTxChSize),
				eventCh: make(chan interface{}, _requestGroupEventChSize),
			}},
		},
	}

	type fields struct {
		cMgr *ContractManager
	}

	type args struct {
		chainID         string
		contractName    string
		contractVersion string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestContractManager_sendContractReadySignal",
			fields: fields{cMgr: cMgr},
			args: args{
				chainID:         "chain1",
				contractName:    "testContractName",
				contractVersion: "1.0.0",
			},
			wantErr: false,
		},
		{
			name:   "TestContractManager_sendContractReadySignal",
			fields: fields{cMgr: cMgr},
			args: args{
				chainID:         "chain1",
				contractName:    "testContractName",
				contractVersion: "1.0.1",
			},
			wantErr: true,
		},
		{
			name: "TestContractManager_sendContractReadySignal",
			fields: fields{cMgr: &ContractManager{
				contractsLRU: cMgr.contractsLRU,
				logger:       cMgr.logger,
				eventCh:      cMgr.eventCh,
				mountDir:     cMgr.mountDir,
			}},
			args: args{
				chainID:         "chain1",
				contractName:    "testContractName",
				contractVersion: "1.0.0",
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := tt.fields.cMgr
			if err := cm.sendContractReadySignal(tt.args.chainID, tt.args.contractName, tt.args.contractVersion, protogo.DockerVMCode_OK); (err != nil) != tt.wantErr {
				t.Errorf("sendContractReadySignal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewContractManager(t *testing.T) {

	SetConfig()

	tests := []struct {
		name string
		//wantExist bool
		wantErr bool
	}{
		{
			name: "TestNewContractManager",
			//wantExist: true,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewContractManager()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewContractManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			//if (got != nil) != tt.wantExist {
			//	t.Errorf("NewContractManager() exist = %v, wantExist %v", got != nil, tt.wantExist)
			//	return
			//}
		})
	}
}

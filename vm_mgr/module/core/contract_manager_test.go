/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"path/filepath"
	"testing"
)

func TestContractManager_GetContractMountDir(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			eventCh: make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"testContractName#1.0.0": &RequestGroup{
				eventCh: make(chan *protogo.DockerVMMessage, requestGroupEventChSize),
			}},
		},
	}
	cMgr.contractsLRU.Add("testContractName#1.0.0", "/mount/contracts/testContractName#1.0.0")

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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			eventCh: make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"testContractName#1.0.0": &RequestGroup{
				eventCh: make(chan *protogo.DockerVMMessage, requestGroupEventChSize),
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
					ContractName:    "testContractName",
					ContractVersion: "1.0.0",
				},
			}},
			wantExist: true,
			wantErr:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := cMgr
			if err := cm.handleGetContractResp(tt.args.resp); (err != nil) != tt.wantErr {
				t.Errorf("handleGetContractResp() error = %v, wantErr %v", err, tt.wantErr)
			}
			groupKey := utils.ConstructContractKey(tt.args.resp.Response.ContractName,
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
		contractsLRU: utils.NewCache(config.DockerVMConfig.Contract.MaxFileNum),
		logger:       logger.NewTestDockerLogger(),
		eventCh:      make(chan *protogo.DockerVMMessage, contractManagerEventChSize),
		mountDir:     filepath.Join(config.DockerMountDir, ContractsDir),
		scheduler: &RequestScheduler{
			eventCh: make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
			requestGroups: map[string]interfaces.RequestGroup{"testContractName#1.0.0": &RequestGroup{
				eventCh: make(chan *protogo.DockerVMMessage, requestGroupEventChSize),
			}},
		},
	}

	type fields struct {
		cMgr *ContractManager
	}

	type args struct {
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
				contractName:    "testContractName",
				contractVersion: "1.0.0",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cm := cMgr
			if err := cm.sendContractReadySignal(tt.args.contractName, tt.args.contractVersion); (err != nil) != tt.wantErr {
				t.Errorf("sendContractReadySignal() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

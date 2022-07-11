/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/module/rpc"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"reflect"
	"testing"
)

func TestNewRequestScheduler(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()

	type args struct {
		service   interfaces.ChainRPCService
		oriPMgr   interfaces.ProcessManager
		crossPMgr interfaces.ProcessManager
		cMgr      *ContractManager
	}
	tests := []struct {
		name string
		args args
		want *RequestScheduler
	}{
		{
			name: "TestNewRequestScheduler",
			args: args{
				service:   nil,
				oriPMgr:   nil,
				crossPMgr: nil,
				cMgr:      nil,
			},
			want: &RequestScheduler{
				logger: log,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRequestScheduler(tt.args.service, tt.args.oriPMgr, tt.args.crossPMgr, tt.args.cMgr)
			got.logger = tt.want.logger
			got.eventCh = tt.want.eventCh
			got.txCh = tt.want.txCh
			got.closeCh = tt.want.closeCh
			got.requestGroups = tt.want.requestGroups
			got.chainRPCService = tt.want.chainRPCService
			got.origProcessManager = tt.want.origProcessManager
			got.crossProcessManager = tt.want.crossProcessManager
			got.contractManager = tt.want.contractManager

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRequestScheduler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestScheduler_GetRequestGroup(t *testing.T) {

	SetConfig()

	scheduler := newTestRequestScheduler(t)
	log := logger.NewTestDockerLogger()
	scheduler.logger = log

	requestGroup := &RequestGroup{}
	groupKey := utils.ConstructContractKey(testChainID, "testContractName1", "1.0.0")
	scheduler.requestGroups[groupKey] = requestGroup

	type fields struct {
		scheduler *RequestScheduler
	}

	type args struct {
		contractName    string
		contractVersion string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interfaces.RequestGroup
		want1  bool
	}{
		{
			name:   "TestRequestScheduler_GetRequestGroup_GoodCase",
			fields: fields{scheduler: scheduler},
			args:   args{contractName: "testContractName1", contractVersion: "1.0.0"},
			want:   requestGroup,
			want1:  true,
		},
		{
			name:   "TestRequestScheduler_GetRequestGroup_WrongContractName",
			fields: fields{scheduler: scheduler},
			args:   args{contractName: "testContractName2", contractVersion: "1.0.0"},
			want:   nil,
			want1:  false,
		},
		{
			name:   "TestRequestScheduler_GetRequestGroup_WrongContractVersion",
			fields: fields{scheduler: scheduler},
			args:   args{contractName: "testContractName1", contractVersion: "1.0.1"},
			want:   nil,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.fields.scheduler
			got, got1 := s.GetRequestGroup(testChainID, tt.args.contractName, tt.args.contractVersion)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetRequestGroup() got = %v, wantNum %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetRequestGroup() got1 = %v, wantNum %v", got1, tt.want1)
			}
		})
	}
}

func TestRequestScheduler_PutMsg(t *testing.T) {

	SetConfig()

	scheduler := newTestRequestScheduler(t)
	log := logger.NewTestDockerLogger()
	scheduler.logger = log

	type fields struct {
		scheduler *RequestScheduler
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
			name:    "TestRequestScheduler_PutMsg_DockerVMMessage",
			fields:  fields{scheduler: scheduler},
			args:    args{&protogo.DockerVMMessage{}},
			wantErr: true,
		},
		{
			name:    "TestRequestScheduler_PutMsg_RequestGroupKey",
			fields:  fields{scheduler: scheduler},
			args:    args{&messages.RequestGroupKey{}},
			wantErr: false,
		},
		{
			name:    "TestRequestScheduler_PutMsg_String",
			fields:  fields{scheduler: scheduler},
			args:    args{"test"},
			wantErr: true,
		},
		{
			name:    "TestRequestScheduler_PutMsg_Struct",
			fields:  fields{scheduler: scheduler},
			args:    args{struct{}{}},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := scheduler
			if err := s.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestScheduler_Start(t *testing.T) {

	SetConfig()

	scheduler := newTestRequestScheduler(t)
	log := logger.NewTestDockerLogger()
	scheduler.logger = log
	scheduler.Start()

	type fields struct {
		scheduler *RequestScheduler
	}
	type args struct {
		msg interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			}},
		},
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
			}},
		},
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_TX_REQUEST,
			}},
		},
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_CALL_CONTRACT_REQUEST,
			}},
		},
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_ERROR,
			}},
		},
		{
			name:   "TestRequestScheduler_Start",
			fields: fields{scheduler: scheduler},
			args: args{msg: &messages.RequestGroupKey{
				ChainID:         testChainID,
				ContractName:    "testContractName",
				ContractVersion: "v1.0.0",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.fields.scheduler
			s.PutMsg(tt.args.msg)
		})
	}
}

func TestRequestScheduler_handleCloseReq(t *testing.T) {

	SetConfig()

	scheduler := newTestRequestScheduler(t)
	log := logger.NewTestDockerLogger()
	scheduler.logger = log

	requestGroup := &RequestGroup{stopCh: make(chan struct{}, 1)}
	groupKey := utils.ConstructContractKey(testChainID, "testContractName1", "1.0.0")
	scheduler.requestGroups[groupKey] = requestGroup

	type fields struct {
		scheduler *RequestScheduler
	}

	type args struct {
		requestGroupKey *messages.RequestGroupKey
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestRequestScheduler_handleCloseReq_GoodCase",
			fields: fields{scheduler: scheduler},
			args: args{
				requestGroupKey: &messages.RequestGroupKey{
					ChainID:         testChainID,
					ContractName:    "testContractName1",
					ContractVersion: "1.0.0",
				},
			},
			wantErr: false,
		},
		{
			name:   "TestRequestScheduler_handleCloseReq_WrongContractName",
			fields: fields{scheduler: scheduler},
			args: args{
				requestGroupKey: &messages.RequestGroupKey{
					ChainID:         testChainID,
					ContractName:    "testContractName2",
					ContractVersion: "1.0.0",
				},
			},
			wantErr: true,
		},
		{
			name:   "TestRequestScheduler_handleCloseReq_WrongContractVersion",
			fields: fields{scheduler: scheduler},
			args: args{
				requestGroupKey: &messages.RequestGroupKey{
					ChainID:         testChainID,
					ContractName:    "testContractName1",
					ContractVersion: "1.0.1",
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.fields.scheduler
			if err := s.handleCloseReq(tt.args.requestGroupKey); (err != nil) != tt.wantErr {
				t.Errorf("handleCloseReq() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//func TestRequestScheduler_handleErrResp(t *testing.T) {
//
//	SetConfig()
//
//	scheduler := newTestRequestScheduler(t)
//	log := logger.NewTestDockerLogger()
//	scheduler.logger = log
//	scheduler.chainRPCService = rpc.NewChainRPCService()
//
//	type fields struct {
//		scheduler *RequestScheduler
//	}
//
//	type args struct {
//		msg *protogo.DockerVMMessage
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name:   "TestRequestScheduler_handleErrResp",
//			fields: fields{scheduler: scheduler},
//			args: args{
//				msg: &protogo.DockerVMMessage{
//					Type: protogo.DockerVMType_ERROR,
//				},
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := tt.fields.scheduler
//			if err := s.handleErrResp(tt.args.msg); (err != nil) != tt.wantErr {
//				t.Errorf("handleErrResp() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

//func TestRequestScheduler_handleGetContractReq(t *testing.T) {
//
//	SetConfig()
//
//	scheduler := newTestRequestScheduler(t)
//	log := logger.NewTestDockerLogger()
//	scheduler.logger = log
//	scheduler.chainRPCService = rpc.NewChainRPCService()
//
//	type fields struct {
//		scheduler *RequestScheduler
//	}
//
//	type args struct {
//		msg *protogo.DockerVMMessage
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name:   "TestRequestScheduler_handleGetContractReq",
//			fields: fields{scheduler: scheduler},
//			args: args{
//				msg: &protogo.DockerVMMessage{
//					Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
//				},
//			},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := tt.fields.scheduler
//			if err := s.handleGetContractReq(tt.args.msg); (err != nil) != tt.wantErr {
//				t.Errorf("handleErrResp() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestRequestScheduler_handleTxReq(t *testing.T) {

	SetConfig()

	scheduler := newTestRequestScheduler(t)
	log := logger.NewTestDockerLogger()
	scheduler.logger = log
	scheduler.contractManager = &ContractManager{eventCh: make(chan interface{}, _contractManagerEventChSize)}
	type fields struct {
		scheduler *RequestScheduler
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
			name:   "TestRequestScheduler_handleTxReq",
			fields: fields{scheduler: scheduler},
			args: args{
				req: &protogo.DockerVMMessage{
					Type: protogo.DockerVMType_TX_REQUEST,
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName1",
						ContractVersion: "1.0.0",
					},
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 0,
					},
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.fields.scheduler
			if err := s.handleTxReq(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("handleErrResp() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func newTestRequestScheduler(t *testing.T) *RequestScheduler {

	// new user controller
	usersManager := NewUsersManager()

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	// start chain rpc server
	chainRPCService := rpc.NewChainRPCService()

	// new scheduler
	scheduler := NewRequestScheduler(chainRPCService, origProcessManager, crossProcessManager,
		&ContractManager{eventCh: make(chan interface{}, _contractManagerEventChSize)})
	origProcessManager.SetScheduler(scheduler)
	crossProcessManager.SetScheduler(scheduler)
	chainRPCService.SetScheduler(scheduler)

	return scheduler
}

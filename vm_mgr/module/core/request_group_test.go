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
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
)

func TestNewRequestGroup(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()

	testContractName := "testContractName"
	testContractVersion := "1.0.0"
	eventCh := make(chan interface{}, _requestGroupEventChSize)
	txCh := make(chan *protogo.DockerVMMessage, _requestGroupEventChSize)
	stopCh := make(chan struct{})
	origTxCh := make(chan *messages.TxPayload, _origTxChSize)
	crossTxCh := make(chan *messages.TxPayload, _crossTxChSize)

	requestGroup := NewRequestGroup(testChainID, testContractName, testContractVersion, nil, nil, nil)
	requestGroup.eventCh = eventCh
	requestGroup.stopCh = stopCh
	requestGroup.txCh = txCh
	requestGroup.origTxController.txCh = origTxCh
	requestGroup.crossTxController.txCh = crossTxCh
	requestGroup.logger = log

	type args struct {
		contractName    string
		contractVersion string
		oriPMgr         interfaces.ProcessManager
		crossPMgr       interfaces.ProcessManager
		scheduler       *RequestScheduler
	}
	tests := []struct {
		name string
		args args
		want *RequestGroup
	}{
		{
			name: "TestNewRequestGroup",
			args: args{
				contractName:    testContractName,
				contractVersion: testContractVersion,
			},
			want: requestGroup,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewRequestGroup(
				testChainID,
				tt.args.contractName,
				tt.args.contractVersion,
				tt.args.oriPMgr,
				tt.args.crossPMgr,
				tt.args.scheduler,
			)
			got.eventCh = eventCh
			got.txCh = txCh
			got.stopCh = stopCh
			got.origTxController = tt.want.origTxController
			got.crossTxController = tt.want.crossTxController
			got.requestScheduler = tt.want.requestScheduler
			got.logger = log
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRequestGroup() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRequestGroup_GetContractPath(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()

	cMgr := &ContractManager{
		mountDir: filepath.Join(config.DockerMountDir, ContractsDir),
	}
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, &RequestScheduler{contractManager: cMgr})
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "TestRequestGroup_GetContractPath",
			fields: fields{
				group: requestGroup,
			},
			want: "/mount/contract-bins/chain1#testContractName#1.0.0",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			if got := r.GetContractPath(); got != tt.want {
				t.Errorf("GetContractPath() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestRequestGroup_GetTxCh(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	testOrigTxCh := make(chan *messages.TxPayload, _origTxChSize)
	testCrossTxCh := make(chan *messages.TxPayload, _crossTxChSize)
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log

	requestGroup.origTxController.txCh = testOrigTxCh
	requestGroup.crossTxController.txCh = testCrossTxCh

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		isOrig bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   chan *messages.TxPayload
	}{
		{
			name: "TestRequestGroup_GetTxCh_Orig",
			fields: fields{
				group: requestGroup,
			},
			args: args{isOrig: true},
			want: testOrigTxCh,
		},
		{
			name: "TestRequestGroup_GetTxCh_Cross",
			fields: fields{
				group: requestGroup,
			},
			args: args{isOrig: false},
			want: testCrossTxCh,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			if got := r.GetTxCh(tt.args.isOrig); got != tt.want {
				t.Errorf("GetTxCh() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestRequestGroup_PutMsg(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log
	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _crossTxChSize),
		processMgr: crossProcessManager,
	}

	requestGroup.Start()

	defer func() {
		requestGroup.stopCh <- struct{}{}
	}()

	type fields struct {
		group *RequestGroup
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
			name:   "TestRequestGroup_PutMsg_DockerVMMessage",
			fields: fields{group: requestGroup},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_TX_REQUEST,
			}},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_PutMsg_DockerVMMessage",
			fields: fields{group: requestGroup},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_RESPONSE,
			}},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_PutMsg_DockerVMMessage",
			fields: fields{group: requestGroup},
			args: args{msg: &protogo.DockerVMMessage{
				Type: protogo.DockerVMType_GET_BYTECODE_REQUEST,
			}},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_PutMsg_GetProcessRespMsg",
			fields: fields{group: requestGroup},
			args: args{msg: &messages.GetProcessRespMsg{
				IsOrig:     true,
				ProcessNum: 0,
			}},
			wantErr: false,
		},
		{
			name:    "TestRequestGroup_PutMsg_String",
			fields:  fields{group: requestGroup},
			args:    args{msg: "test"},
			wantErr: true,
		},
		{
			name:    "TestRequestGroup_PutMsg_Int",
			fields:  fields{group: requestGroup},
			args:    args{msg: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			if err := r.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestGroup_Start(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "TestRequestGroup_Start",
			fields: fields{group: requestGroup},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			r.Start()
		})
	}
}

func TestRequestGroup_getProcesses(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _crossTxChSize),
		processMgr: crossProcessManager,
	}
	requestGroup.origTxController.txCh <- &messages.TxPayload{
		Tx: &protogo.DockerVMMessage{},
	}

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		isOrig bool
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantNum   int
		wantState bool
		wantErr   bool
	}{
		{
			name: "TestRequestGroup_getProcesses_origTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{isOrig: true},
			wantNum:   1,
			wantState: true,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_getProcesses_crossTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{isOrig: false},
			wantNum:   0,
			wantState: false,
			wantErr:   false,
		},
		//{
		//	name: "TestRequestGroup_getProcesses_unknown_txtype",
		//	fields: fields{
		//		group: requestGroup,
		//	},
		//	args:      args{isOrig: 3},
		//	wantNum:   0,
		//	wantState: false,
		//	wantErr:   true,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			num, err := r.getProcesses(tt.args.isOrig)
			if (err != nil) != tt.wantErr {
				t.Errorf("getProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if num != tt.wantNum {
				t.Errorf("getProcesses() got = %v, wantNum %v", num, tt.wantNum)
			}
			if tt.args.isOrig {
				if r.origTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.origTxController.processWaiting, tt.wantState)
				}
			} else {
				if r.crossTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.crossTxController.processWaiting, tt.wantState)
				}
			}
		})
	}
}

func TestRequestGroup_handleContractReadyResp(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _crossTxChSize),
		processMgr: crossProcessManager,
	}

	type fields struct {
		group *RequestGroup
	}

	tests := []struct {
		name   string
		fields fields
		want   contractState
	}{
		{
			name:   "TestRequestGroup_handleContractReadyResp",
			fields: fields{group: requestGroup},
			want:   _contractReady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			r.handleContractReadyResp(&protogo.DockerVMMessage{})
			if r.contractState != tt.want {
				t.Errorf("handleContractReadyResp() got = %v, wantNum %v", r.contractState, tt.want)
			}
		})
	}
}

func TestRequestGroup_handleProcessReadyResp(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, nil)
	requestGroup.logger = log

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *messages.TxPayload, _crossTxChSize),
		processMgr: crossProcessManager,
	}
	requestGroup.origTxController.txCh <- &messages.TxPayload{
		Tx: &protogo.DockerVMMessage{},
	}

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		isOrig    bool
		toWaiting bool
	}
	tests := []struct {
		name      string
		fields    fields
		args      args
		wantState bool
		wantErr   bool
	}{
		{
			name: "TestRequestGroup_handleProcessReadyResp_origTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{isOrig: true, toWaiting: true},
			wantState: true,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_handleProcessReadyResp_crossTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{isOrig: false, toWaiting: false},
			wantState: false,
			wantErr:   false,
		},
		//{
		//	name: "TestRequestGroup_handleProcessReadyResp_unknown_txtype",
		//	fields: fields{
		//		group: requestGroup,
		//	},
		//	args:      args{isOrig: 3},
		//	wantState: false,
		//	wantErr:   true,
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			err := r.handleProcessReadyResp(&messages.GetProcessRespMsg{
				IsOrig:     tt.args.isOrig,
				ProcessNum: 0,
			})
			if (err != nil) != tt.wantErr {
				t.Errorf("getProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.args.isOrig {
				if r.origTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.origTxController.processWaiting, tt.wantState)
				}
			} else {
				if r.crossTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.crossTxController.processWaiting, tt.wantState)
				}
			}
		})
	}
}

func TestRequestGroup_handleTxReq(t *testing.T) {

	SetConfig()

	cMgr := &ContractManager{
		mountDir: filepath.Join(config.DockerMountDir, ContractsDir),
	}
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil,
		nil, &RequestScheduler{
			lock:            sync.RWMutex{},
			txCh:            make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
			eventCh:         make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
			contractManager: cMgr,
		})
	log := logger.NewTestDockerLogger()
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
	}

	type args struct {
		req   *protogo.DockerVMMessage
		state contractState
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:   "TestRequestGroup_putTxReqToCh_GoodCase_ContractEmpty",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 0,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: _contractEmpty,
			},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_putTxReqToCh_GoodCase_ContractWaiting",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 0,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: _contractWaiting,
			},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_putTxReqToCh_GoodCase_ContractReady_Orig",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 0,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: _contractReady,
			},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_putTxReqToCh_GoodCase_ContractReady_Cross",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 4,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: _contractReady,
			},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_putTxReqToCh_BadCase_ContractReady_Cross",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 6,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: _contractReady,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			r.contractState = tt.args.state
			if err := r.putTxReqToCh(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("handleTxReq() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestRequestGroup_putTxReqToCh(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup(testChainID, "testContractName", "1.0.0", nil, nil, &RequestScheduler{
		lock:    sync.RWMutex{},
		txCh:    make(chan *protogo.DockerVMMessage, _requestSchedulerTxChSize),
		eventCh: make(chan *protogo.DockerVMMessage, _requestSchedulerEventChSize),
	})
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
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
			name:   "TestRequestGroup_putTxReqToCh_GoodCase",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 0,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
			},
			wantErr: false,
		},
		{
			name:   "TestRequestGroup_putTxReqToCh_BadCase",
			fields: fields{group: requestGroup},
			args: args{
				req: &protogo.DockerVMMessage{
					TxId: "testTxID",
					Type: protogo.DockerVMType_TX_REQUEST,
					CrossContext: &protogo.CrossContext{
						CurrentDepth: 6,
					},
					Request: &protogo.TxRequest{
						ChainId:         testChainID,
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			if err := r.putTxReqToCh(tt.args.req); (err != nil) != tt.wantErr {
				t.Errorf("putTxReqToCh() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

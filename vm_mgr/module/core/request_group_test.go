package core

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
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
	eventCh := make(chan *protogo.DockerVMMessage, requestGroupEventChSize)
	origTxCh := make(chan *protogo.DockerVMMessage, origTxChSize)
	crossTxCh := make(chan *protogo.DockerVMMessage, crossTxChSize)

	requestGroup := NewRequestGroup(testContractName, testContractVersion, nil, nil, nil, nil)
	requestGroup.eventCh = eventCh
	requestGroup.origTxController.txCh = origTxCh
	requestGroup.crossTxController.txCh = crossTxCh
	requestGroup.logger = log

	type args struct {
		contractName    string
		contractVersion string
		oriPMgr         interfaces.ProcessManager
		crossPMgr       interfaces.ProcessManager
		cMgr            *ContractManager
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
				tt.args.contractName,
				tt.args.contractVersion,
				tt.args.oriPMgr,
				tt.args.crossPMgr,
				tt.args.cMgr,
				tt.args.scheduler,
			)
			got.eventCh = eventCh
			got.origTxController.txCh = origTxCh
			got.crossTxController.txCh = crossTxCh
			got.logger = log
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewRequestGroup() = %v, wantNum %v", got, tt.want)
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, cMgr, nil)
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
			want: "/mount/contracts/testContractName#1.0.0",
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
	testOrigTxCh := make(chan *protogo.DockerVMMessage, origTxChSize)
	testCrossTxCh := make(chan *protogo.DockerVMMessage, crossTxChSize)
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
	requestGroup.logger = log

	requestGroup.origTxController.txCh = testOrigTxCh
	requestGroup.crossTxController.txCh = testCrossTxCh

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		isCross bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   chan *protogo.DockerVMMessage
	}{
		{
			name: "TestRequestGroup_GetTxCh_Orig",
			fields: fields{
				group: requestGroup,
			},
			args: args{isCross: false},
			want: testOrigTxCh,
		},
		{
			name: "TestRequestGroup_GetTxCh_Cross",
			fields: fields{
				group: requestGroup,
			},
			args: args{isCross: true},
			want: testCrossTxCh,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			if got := r.GetTxCh(tt.args.isCross); got != tt.want {
				t.Errorf("GetTxCh() = %v, wantNum %v", got, tt.want)
			}
		})
	}
}

func TestRequestGroup_PutMsg(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
	requestGroup.logger = log

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
			name:    "TestRequestGroup_PutMsg_DockerVMMessage",
			fields:  fields{group: requestGroup},
			args:    args{msg: &protogo.DockerVMMessage{}},
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
	requestGroup.logger = log

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *protogo.DockerVMMessage, origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *protogo.DockerVMMessage, crossTxChSize),
		processMgr: crossProcessManager,
	}
	requestGroup.origTxController.txCh <- &protogo.DockerVMMessage{}

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		txType TxType
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
			args:      args{txType: origTx},
			wantNum:   1,
			wantState: true,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_getProcesses_crossTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{txType: crossTx},
			wantNum:   0,
			wantState: false,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_getProcesses_unknown_txtype",
			fields: fields{
				group: requestGroup,
			},
			args:      args{txType: 3},
			wantNum:   0,
			wantState: false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			num, err := r.getProcesses(tt.args.txType)
			if (err != nil) != tt.wantErr {
				t.Errorf("getProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if num != tt.wantNum {
				t.Errorf("getProcesses() got = %v, wantNum %v", num, tt.wantNum)
			}
			if tt.args.txType == origTx {
				if r.origTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.origTxController.processWaiting, tt.wantState)
				}
			} else if tt.args.txType == crossTx {
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
	requestGroup.logger = log

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
			want:   contractReady,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			r.handleContractReadyResp()
			if r.contractState != tt.want {
				t.Errorf("handleContractReadyResp() got = %v, wantNum %v", r.contractState, tt.want)
			}
		})
	}
}

func TestRequestGroup_handleProcessReadyResp(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil, nil, nil, nil)
	requestGroup.logger = log

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	usersManager := NewUsersManager()
	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)

	requestGroup.origTxController = &txController{
		txCh:       make(chan *protogo.DockerVMMessage, origTxChSize),
		processMgr: origProcessManager,
	}
	requestGroup.crossTxController = &txController{
		txCh:       make(chan *protogo.DockerVMMessage, crossTxChSize),
		processMgr: crossProcessManager,
	}
	requestGroup.origTxController.txCh <- &protogo.DockerVMMessage{}

	type fields struct {
		group *RequestGroup
	}
	type args struct {
		txType TxType
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
			args:      args{txType: origTx},
			wantState: true,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_handleProcessReadyResp_crossTx",
			fields: fields{
				group: requestGroup,
			},
			args:      args{txType: crossTx},
			wantState: false,
			wantErr:   false,
		},
		{
			name: "TestRequestGroup_handleProcessReadyResp_unknown_txtype",
			fields: fields{
				group: requestGroup,
			},
			args:      args{txType: 3},
			wantState: false,
			wantErr:   true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := tt.fields.group
			err := r.handleProcessReadyResp(tt.args.txType)
			if (err != nil) != tt.wantErr {
				t.Errorf("getProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			if tt.args.txType == origTx {
				if r.origTxController.processWaiting != tt.wantState {
					t.Errorf("getProcesses() got = %v, wantState %v",
						r.origTxController.processWaiting, tt.wantState)
				}
			} else if tt.args.txType == crossTx {
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil,
		nil, cMgr, &RequestScheduler{
			lock:                sync.RWMutex{},
			eventCh:             make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
		})
	log := logger.NewTestDockerLogger()
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
	}

	type args struct {
		req *protogo.DockerVMMessage
		state contractState
	}

	tests := []struct {
		name   string
		fields fields
		args   args
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
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: contractEmpty,
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
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: contractWaiting,
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
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: contractReady,
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
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: contractReady,
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
						ContractName:    "testContractName",
						ContractVersion: "1.0.0",
					},
				},
				state: contractReady,
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
	requestGroup := NewRequestGroup("testContractName", "1.0.0", nil,
		nil, nil, &RequestScheduler{
			lock:                sync.RWMutex{},
			eventCh:             make(chan *protogo.DockerVMMessage, requestSchedulerEventChSize),
		})
	requestGroup.logger = log

	type fields struct {
		group *RequestGroup
	}

	type args struct {
		req *protogo.DockerVMMessage
	}

	tests := []struct {
		name   string
		fields fields
		args   args
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

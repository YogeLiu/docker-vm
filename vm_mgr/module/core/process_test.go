package core
//
//import (
//	"chainmaker.org/chainmaker/protocol/v2"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/rpc"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
//	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
//	"testing"
//)
//
//const (
//	testContractName    = "testContractName"
//	testContractVersion = "v1.0.0"
//	testUid             = 20001
//)
//
//var (
//	groupKey        = utils.ConstructContractKey(testContractName, testContractVersion)
//	testProcessName = utils.ConstructProcessName(testContractName, testContractVersion, 0)
//)
//
//func TestNewProcess(t *testing.T) {
//
//	SetConfig()
//
//	type args struct {
//		isOrig bool
//	}
//	tests := []struct {
//		name string
//		args args
//	}{
//		{
//			name: "TestNewProcess_Orig",
//			args: args{isOrig: true},
//		},
//		{
//			name: "TestNewProcess_Cross",
//			args: args{isOrig: false},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			newTestProcess(tt.args.isOrig)
//			tearDown()
//		})
//	}
//}
//
//func TestProcess_GetContractName(t *testing.T) {
//
//	SetConfig()
//
//	process := newTestProcess(true)
//	defer tearDown()
//
//	type fields struct {
//		process *Process
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		{
//
//			name:   "TestProcess_GetContractName",
//			fields: fields{process: process},
//			want:   testContractName,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := tt.fields.process
//			if got := p.GetContractName(); got != tt.want {
//				t.Errorf("GetContractName() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestProcess_GetContractVersion(t *testing.T) {
//
//	SetConfig()
//
//	process := newTestProcess(true)
//	defer tearDown()
//
//	type fields struct {
//		process *Process
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		{
//
//			name:   "TestProcess_GetContractVersion",
//			fields: fields{process: process},
//			want:   testContractVersion,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := tt.fields.process
//			if got := p.GetContractVersion(); got != tt.want {
//				t.Errorf("GetContractVersion() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestProcess_GetProcessName(t *testing.T) {
//
//	SetConfig()
//
//	process := newTestProcess(true)
//	defer tearDown()
//
//	type fields struct {
//		process *Process
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		want   string
//	}{
//		{
//
//			name:   "TestProcess_GetProcessName",
//			fields: fields{process: process},
//			want:   testProcessName,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := tt.fields.process
//			if got := p.GetProcessName(); got != tt.want {
//				t.Errorf("GetProcessName() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestProcess_GetUser(t *testing.T) {
//
//	SetConfig()
//
//	process := newTestProcess(true)
//	defer tearDown()
//
//	type fields struct {
//		process *Process
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		want   int
//	}{
//		{
//
//			name:   "TestProcess_GetUser",
//			fields: fields{process: process},
//			want:   testUid,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := tt.fields.process
//			if got := p.GetUser(); got.GetUid() != tt.want {
//				t.Errorf("GetUser() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
////func TestProcess_PutMsg(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		msg interface{}
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
////				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
//func TestProcess_SetStream(t *testing.T) {
//
//	SetConfig()
//
//	process := newTestProcess(true)
//	defer tearDown()
//
//	type fields struct {
//		process *Process
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		want   ProcessState
//	}{
//		{
//
//			name:   "TestProcess_SetStream",
//			fields: fields{process: process},
//			want:   ready,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := tt.fields.process
//			p.SetStream(nil)
//			if p.processState != tt.want {
//				t.Errorf("SetStream() state = %v, want %v", p.processState, tt.want)
//			}
//		})
//	}
//}
//
////func TestProcess_Start(t *testing.T) {
////
////	SetConfig()
////
////	process := newTestProcess(true)
////	defer tearDown()
////
////	type fields struct {
////		process *Process
////	}
////
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		{
////
////			name:   "TestProcess_Start",
////			fields: fields{process: process},
////		},
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := tt.fields.process
////			p.Start()
////		})
////	}
////}
//
////
////func TestProcess_constructErrorResponse(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		txId   string
////		errMsg string
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		args   args
////		want   *protogo.DockerVMMessage
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if got := p.constructErrorResp(tt.args.txId, tt.args.errMsg); !reflect.DeepEqual(got, tt.want) {
////				t.Errorf("constructErrorResp() = %v, want %v", got, tt.want)
////			}
////		})
////	}
////}
////
////func TestProcess_handleChangeSandboxReq(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		msg *messages.ChangeSandboxReqMsg
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.handleChangeSandboxReq(tt.args.msg); (err != nil) != tt.wantErr {
////				t.Errorf("handleChangeSandboxReq() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_handleCloseSandboxReq(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.handleCloseSandboxReq(); (err != nil) != tt.wantErr {
////				t.Errorf("handleCloseSandboxReq() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_handleProcessExit(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		existErr *exitErr
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		args   args
////		want   bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if got := p.handleProcessExit(tt.args.existErr); got != tt.want {
////				t.Errorf("handleProcessExit() = %v, want %v", got, tt.want)
////			}
////		})
////	}
////}
////
////func TestProcess_handleTimeout(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.handleTimeout(); (err != nil) != tt.wantErr {
////				t.Errorf("handleTimeout() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_handleTxRequest(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		tx *protogo.DockerVMMessage
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.handleTxRequest(tt.args.tx); (err != nil) != tt.wantErr {
////				t.Errorf("handleTxRequest() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_handleTxResp(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		msg *protogo.DockerVMMessage
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.handleTxResp(tt.args.msg); (err != nil) != tt.wantErr {
////				t.Errorf("handleTxResp() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_killProcess(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.killProcess()
////		})
////	}
////}
////
////func TestProcess_launchProcess(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		want   *exitErr
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if got := p.launchProcess(); !reflect.DeepEqual(got, tt.want) {
////				t.Errorf("launchProcess() = %v, want %v", got, tt.want)
////			}
////		})
////	}
////}
////
////func TestProcess_listenProcess(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.listenProcess()
////		})
////	}
////}
////
////func TestProcess_printContractLog(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		contractPipe io.ReadCloser
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		args   args
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.printContractLog(tt.args.contractPipe)
////		})
////	}
////}
////
////func TestProcess_resetContext(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		msg *messages.ChangeSandboxReqMsg
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.resetContext(tt.args.msg); (err != nil) != tt.wantErr {
////				t.Errorf("resetContext() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_returnErrorResponse(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		txId   string
////		errMsg string
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		args   args
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.returnTxErrorResp(tt.args.txId, tt.args.errMsg)
////		})
////	}
////}
////
////func TestProcess_sendMsg(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		msg *protogo.DockerVMMessage
////	}
////	tests := []struct {
////		name    string
////		fields  fields
////		args    args
////		wantErr bool
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			if err := p.sendMsg(tt.args.msg); (err != nil) != tt.wantErr {
////				t.Errorf("sendMsg() error = %v, wantErr %v", err, tt.wantErr)
////			}
////		})
////	}
////}
////
////func TestProcess_startBusyTimer(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.startBusyTimer()
////		})
////	}
////}
////
////func TestProcess_startProcess(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.startProcess()
////		})
////	}
////}
////
////func TestProcess_startReadyTimer(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.startReadyTimer()
////		})
////	}
////}
////
////func TestProcess_stopTimer(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	tests := []struct {
////		name   string
////		fields fields
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.stopTimer()
////		})
////	}
////}
////
////func TestProcess_updateProcessState(t *testing.T) {
////	type fields struct {
////		processName      string
////		contractName     string
////		contractVersion  string
////		cGroupPath       string
////		user             interfaces.User
////		cmd              *exec.Cmd
////		ProcessState     ProcessState
////		isOrigProcess   bool
////		eventCh          chan interface{}
////		cmdReadyCh       chan bool
////		exitCh           chan *exitErr
////		txCh             chan *protogo.DockerVMMessage
////		respCh           chan *protogo.DockerVMMessage
////		timer            *time.Timer
////		Tx               *protogo.DockerVMMessage
////		logger           *zap.SugaredLogger
////		stream           protogo.DockerVMRpc_DockerVMCommunicateServer
////		processManager   interfaces.ProcessManager
////		requestGroup     interfaces.RequestGroup
////		requestScheduler interfaces.RequestScheduler
////		lock             sync.RWMutex
////	}
////	type args struct {
////		state ProcessState
////	}
////	tests := []struct {
////		name   string
////		fields fields
////		args   args
////	}{
////		// TODO: Add test cases.
////	}
////	for _, tt := range tests {
////		t.Run(tt.name, func(t *testing.T) {
////			p := &Process{
////				processName:      tt.fields.processName,
////				contractName:     tt.fields.contractName,
////				contractVersion:  tt.fields.contractVersion,
////				cGroupPath:       tt.fields.cGroupPath,
////				user:             tt.fields.user,
////				cmd:              tt.fields.cmd,
////				ProcessState:     tt.fields.ProcessState,
////				isOrigProcess:   tt.fields.isOrigProcess,
////				eventCh:          tt.fields.eventCh,
////				cmdReadyCh:       tt.fields.cmdReadyCh,
////				exitCh:           tt.fields.exitCh,
////				txCh:             tt.fields.txCh,
////				respCh:           tt.fields.respCh,
////				timer:            tt.fields.timer,
////				Tx:               tt.fields.Tx,
////				logger:           tt.fields.logger,
////				stream:           tt.fields.stream,
////				processManager:   tt.fields.processManager,
////				requestGroup:     tt.fields.requestGroup,
////				requestScheduler: tt.fields.requestScheduler,
////				lock:             tt.fields.lock,
////			}
////			p.updateProcessState(tt.args.state)
////		})
////	}
////}
//
//func newTestProcess(isOrig bool) *Process {
//	// new user controller
//	usersManager := NewUsersManager()
//	usersManager.generateNewUser(testUid)
//	user := NewUser(testUid)
//
//	// new original process manager
//	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
//	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
//	releaseRate := config.DockerVMConfig.GetReleaseRate()
//
//	origProcessManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
//	crossProcessManager := NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)
//
//	// start chain rpc server
//	chainRPCService := rpc.NewChainRPCService()
//
//	// new scheduler
//	scheduler := NewRequestScheduler(chainRPCService, origProcessManager, crossProcessManager,
//		&ContractManager{eventCh: make(chan *protogo.DockerVMMessage, contractManagerEventChSize)})
//	scheduler.requestGroups[groupKey] = &RequestGroup{
//		origTxController: &txController{
//			txCh:       make(chan *protogo.DockerVMMessage, origTxChSize),
//			processMgr: origProcessManager,
//		},
//		crossTxController: &txController{
//			txCh:       make(chan *protogo.DockerVMMessage, crossTxChSize),
//			processMgr: crossProcessManager,
//		},
//	}
//	origProcessManager.SetScheduler(scheduler)
//	crossProcessManager.SetScheduler(scheduler)
//	chainRPCService.SetScheduler(scheduler)
//
//	if isOrig {
//		return NewProcess(user, testContractName, testContractVersion, testProcessName, origProcessManager, scheduler, false)
//	}
//	return NewProcess(user, testContractName, testContractVersion, testProcessName, crossProcessManager, scheduler, true)
//
//}
//
//func tearDown() {
//	usersManager := NewUsersManager()
//	usersManager.releaseUser(testUid)
//}

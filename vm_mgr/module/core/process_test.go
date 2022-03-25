///*
//Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
//
package core

//
//import (
//	"io"
//	"os"
//	"os/exec"
//	"path/filepath"
//	"reflect"
//	"sync"
//	"testing"
//	"time"
//
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module/security"
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/pb/protogo"
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/protocol"
//	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/utils"
//	"github.com/golang/mock/gomock"
//	"go.uber.org/zap"
//)
//
//func TestNewCrossProcess(t *testing.T) {
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS)
//	timeTimer := time.NewTimer(processWaitingTime * time.Second)
//	type args struct {
//		user         *security.User
//		txRequest    *protogo.TxRequest
//		scheduler    protocol.Scheduler
//		processName  string
//		contractPath string
//		processPool  ProcessPoolInterface
//	}
//	tests := []struct {
//		name string
//		args args
//		want *Process
//	}{
//		{
//			name: "testNewCrossProces",
//			args: args{
//				user: nil,
//				txRequest: &protogo.TxRequest{
//					TxId:            "",
//					ContractName:    "txRequest.ContractName",
//					ContractVersion: "txRequest.ContractVersion",
//					Method:          "",
//					Parameters:      nil,
//					TxContext:       nil,
//				},
//				scheduler:    nil,
//				processName:  processName,
//				contractPath: "contractPath",
//				processPool:  nil,
//			},
//			want: &Process{
//				isCrossProcess:  true,
//				processName:     processName,
//				contractName:    "txRequest.ContractName",
//				contractVersion: "txRequest.ContractVersion",
//				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
//				TxWaitingQueue:  nil,
//				nextTxTrigger:       nil,
//				expireTimer:     timeTimer,
//				logger:          logger,
//
//				Handler:              nil,
//				user:                 nil,
//				contractPath:         "contractPath",
//				cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
//				processPoolInterface: nil,
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			tt.want.Handler = NewProcessHandler(tt.args.txRequest, tt.args.scheduler, tt.want)
//
//			got := NewCrossProcess(tt.args.user, tt.args.txRequest, tt.args.scheduler, tt.args.processName, tt.args.contractPath, tt.args.processPool)
//			got.Handler = tt.want.Handler
//			got.logger = logger
//			got.expireTimer = timeTimer
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewCrossProcess() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestNewProcess(t *testing.T) {
//	type args struct {
//		user         *security.User
//		txRequest    *protogo.TxRequest
//		scheduler    protocol.Scheduler
//		processName  string
//		contractPath string
//		processPool  ProcessPoolInterface
//	}
//	tests := []struct {
//		name string
//		args args
//		want *Process
//	}{
//		{
//			name: "testNewProcess",
//			args: args{
//				user: nil,
//				txRequest: &protogo.TxRequest{
//					TxId:            "",
//					ContractName:    "txRequest.ContractName",
//					ContractVersion: "txRequest.ContractVersion",
//					Method:          "",
//					Parameters:      nil,
//					TxContext:       nil,
//				},
//				scheduler:    nil,
//				processName:  processName,
//				contractPath: "contractPath",
//				processPool:  nil,
//			},
//
//			want: &Process{
//				isCrossProcess:  false,
//				processName:     processName,
//				contractName:    "txRequest.ContractName",
//				contractVersion: "txRequest.ContractVersion",
//				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
//				TxWaitingQueue:  nil,
//				nextTxTrigger:       nil,
//				expireTimer:     nil,
//				logger:          nil,
//
//				Handler:              nil,
//				user:                 nil,
//				contractPath:         "contractPath",
//				cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
//				processPoolInterface: nil,
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got := NewProcess(tt.args.user, tt.args.txRequest, tt.args.scheduler, tt.args.processName, tt.args.contractPath, tt.args.processPool)
//			tt.want.expireTimer = got.expireTimer
//			tt.want.logger = got.logger
//			tt.want.TxWaitingQueue = got.TxWaitingQueue
//			tt.want.nextTxTrigger = got.nextTxTrigger
//			tt.want.Handler = got.Handler
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewProcess() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestProcess_AddTxWaitingQueue(t *testing.T) {
//	basePath, _ := os.Getwd()
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS, basePath+testPath)
//	type fields struct {
//		processName          string
//		contractName         string
//		contractVersion      string
//		contractPath         string
//		cGroupPath           string
//		ProcessState         protogo.ProcessState
//		TxWaitingQueue       chan *protogo.TxRequest
//		nextTxTrigger            chan bool
//		expireTimer          *time.Timer
//		logger               *zap.SugaredLogger
//		Handler              *ProcessHandler
//		user                 *security.User
//		cmd                  *exec.Cmd
//		processPoolInterface ProcessPoolInterface
//		isCrossProcess       bool
//		done                 uint32
//		balanceRWMutex                sync.Mutex
//	}
//	type args struct {
//		tx *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		{
//			name: "testAddTxWaitingQueue",
//			fields: fields{
//				processName:          processName,
//				contractName:         "",
//				contractVersion:      "",
//				contractPath:         "",
//				cGroupPath:           "",
//				ProcessState:         protogo.ProcessState_PROCESS_STATE_CREATED,
//				TxWaitingQueue:       make(chan *protogo.TxRequest),
//				nextTxTrigger:            nil,
//				expireTimer:          nil,
//				logger:               logger,
//				Handler:              nil,
//				user:                 nil,
//				cmd:                  nil,
//				processPoolInterface: nil,
//				isCrossProcess:       false,
//				done:                 0,
//				balanceRWMutex:                sync.Mutex{},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				processName:          tt.fields.processName,
//				contractName:         tt.fields.contractName,
//				contractVersion:      tt.fields.contractVersion,
//				contractPath:         tt.fields.contractPath,
//				cGroupPath:           tt.fields.cGroupPath,
//				ProcessState:         tt.fields.ProcessState,
//				TxWaitingQueue:       tt.fields.TxWaitingQueue,
//				nextTxTrigger:            tt.fields.nextTxTrigger,
//				expireTimer:          tt.fields.expireTimer,
//				logger:               tt.fields.logger,
//				Handler:              tt.fields.Handler,
//				user:                 tt.fields.user,
//				cmd:                  tt.fields.cmd,
//				processPoolInterface: tt.fields.processPoolInterface,
//				isCrossProcess:       tt.fields.isCrossProcess,
//				done:                 tt.fields.done,
//				balanceRWMutex:                tt.fields.balanceRWMutex,
//			}
//
//			go func() {
//				for {
//					<-p.TxWaitingQueue
//				}
//			}()
//
//			p.AddTxWaitingQueue(&protogo.TxRequest{
//				TxId:            "",
//				ContractName:    "",
//				ContractVersion: "",
//				Method:          "",
//				Parameters:      nil,
//				TxContext: &protogo.TxContext{
//					CurrentHeight:       0,
//					WriteMap:            nil,
//					ReadMap:             nil,
//					OriginalProcessName: "",
//				},
//			})
//		})
//	}
//}
//
//func TestProcess_InvokeProcess(t *testing.T) {
//	logger := utils.GetLogHandler()
//	type fields struct {
//		processName          string
//		contractName         string
//		contractVersion      string
//		contractPath         string
//		cGroupPath           string
//		ProcessState         protogo.ProcessState
//		TxWaitingQueue       chan *protogo.TxRequest
//		nextTxTrigger            chan bool
//		expireTimer          *time.Timer
//		logger               *zap.SugaredLogger
//		Handler              *ProcessHandler
//		user                 *security.User
//		cmd                  *exec.Cmd
//		processPoolInterface ProcessPoolInterface
//		isCrossProcess       bool
//		done                 uint32
//		balanceRWMutex                sync.Mutex
//	}
//
//	requests := make(chan *protogo.TxRequest, 10)
//	go func() {
//		for {
//			requests <- &protogo.TxRequest{
//				TxId:            "0x8f0f3877af159da09bdbf3354e675495e29ee0193612e378bb43dabaa96c1cb8",
//				ContractName:    contractName,
//				ContractVersion: contractVersion,
//				Method:          "",
//				Parameters:      nil,
//				TxContext:       nil,
//			}
//		}
//	}()
//
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		{
//			name: "testInvokeProcessQueueEmpty",
//			fields: fields{
//				processName:     "",
//				contractName:    "",
//				contractVersion: "",
//				contractPath:    "",
//				cGroupPath:      "",
//				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
//				TxWaitingQueue:  make(chan *protogo.TxRequest),
//				nextTxTrigger:       nil,
//				expireTimer:     nil,
//				logger:          logger,
//				Handler: &ProcessHandler{
//					state:         "",
//					logger:        nil,
//					TxRequest:     nil,
//					stream:        nil,
//					scheduler:     nil,
//					process:       nil,
//					txExpireTimer: nil,
//				},
//				user:                 nil,
//				cmd:                  nil,
//				processPoolInterface: nil,
//				isCrossProcess:       false,
//				done:                 0,
//				balanceRWMutex:                sync.Mutex{},
//			},
//		},
//		{
//			name: "testInvokeProcess",
//			fields: fields{
//				processName:     "",
//				contractName:    "",
//				contractVersion: "",
//				contractPath:    "",
//				cGroupPath:      "",
//				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
//				TxWaitingQueue:  requests,
//				nextTxTrigger:       nil,
//				expireTimer:     nil,
//				logger:          logger,
//				Handler: &ProcessHandler{
//					state:         "",
//					logger:        nil,
//					TxRequest:     nil,
//					stream:        nil,
//					scheduler:     nil,
//					process:       nil,
//					txExpireTimer: time.NewTimer(time.Second),
//				},
//				user:                 nil,
//				cmd:                  nil,
//				processPoolInterface: nil,
//				isCrossProcess:       false,
//				done:                 0,
//				balanceRWMutex:                sync.Mutex{},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				processName:          tt.fields.processName,
//				contractName:         tt.fields.contractName,
//				contractVersion:      tt.fields.contractVersion,
//				contractPath:         tt.fields.contractPath,
//				cGroupPath:           tt.fields.cGroupPath,
//				ProcessState:         tt.fields.ProcessState,
//				TxWaitingQueue:       tt.fields.TxWaitingQueue,
//				nextTxTrigger:            tt.fields.nextTxTrigger,
//				expireTimer:          tt.fields.expireTimer,
//				logger:               tt.fields.logger,
//				Handler:              tt.fields.Handler,
//				user:                 tt.fields.user,
//				cmd:                  tt.fields.cmd,
//				processPoolInterface: tt.fields.processPoolInterface,
//				isCrossProcess:       tt.fields.isCrossProcess,
//				done:                 tt.fields.done,
//				balanceRWMutex:                tt.fields.balanceRWMutex,
//			}
//
//			p.handleNewTx()
//		})
//	}
//}
//
//func TestProcess_LaunchProcess(t *testing.T) {
//	basePath, _ := os.Getwd()
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS, basePath+testPath)
//	type fields struct {
//		processName          string
//		contractName         string
//		contractVersion      string
//		contractPath         string
//		cGroupPath           string
//		ProcessState         protogo.ProcessState
//		TxWaitingQueue       chan *protogo.TxRequest
//		nextTxTrigger            chan bool
//		expireTimer          *time.Timer
//		logger               *zap.SugaredLogger
//		Handler              *ProcessHandler
//		user                 *security.User
//		cmd                  *exec.Cmd
//		processPoolInterface ProcessPoolInterface
//		isCrossProcess       bool
//		done                 uint32
//		balanceRWMutex                sync.Mutex
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		wantErr bool
//	}{
//		{
//			name: "testLaunchProcessBad", //todo cmd
//			fields: fields{
//				processName:     "",
//				contractName:    "",
//				contractVersion: "",
//				contractPath:    "",
//				cGroupPath:      "",
//				ProcessState:    0,
//				TxWaitingQueue:  nil,
//				nextTxTrigger:       nil,
//				expireTimer:     nil,
//				logger:          logger,
//				Handler:         nil,
//				user: &security.User{
//					Uid:      0,
//					Gid:      0,
//					UserName: "",
//					SockPath: "",
//				},
//				cmd:                  nil,
//				processPoolInterface: nil,
//				isCrossProcess:       false,
//				done:                 0,
//				balanceRWMutex:                sync.Mutex{},
//			},
//			wantErr: true,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				processName:          tt.fields.processName,
//				contractName:         tt.fields.contractName,
//				contractVersion:      tt.fields.contractVersion,
//				contractPath:         tt.fields.contractPath,
//				cGroupPath:           tt.fields.cGroupPath,
//				ProcessState:         tt.fields.ProcessState,
//				TxWaitingQueue:       tt.fields.TxWaitingQueue,
//				nextTxTrigger:            tt.fields.nextTxTrigger,
//				expireTimer:          tt.fields.expireTimer,
//				logger:               tt.fields.logger,
//				Handler:              tt.fields.Handler,
//				user:                 tt.fields.user,
//				cmd:                  tt.fields.cmd,
//				processPoolInterface: tt.fields.processPoolInterface,
//				isCrossProcess:       tt.fields.isCrossProcess,
//				done:                 tt.fields.done,
//				balanceRWMutex:                tt.fields.balanceRWMutex,
//			}
//			if err := p.LaunchProcess(); (err != nil) != tt.wantErr {
//				t.Errorf("LaunchProcess() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}
//
//func TestProcess_StopProcess(t *testing.T) {
//	basePath, _ := os.Getwd()
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS, basePath+testPath)
//	poolInterface := NewMockProcessPoolInterface(gomock.NewController(t))
//	poolInterface.EXPECT().RetrieveProcessContext(processName).Return(&ProcessContext{
//		processList: [6]*Process{
//			{
//				processName: processName,
//				cmd: &exec.Cmd{
//					Process: &os.Process{},
//				},
//			},
//		},
//		size: 0,
//	}).AnyTimes()
//
//	type fields struct {
//		processName          string
//		contractName         string
//		contractVersion      string
//		contractPath         string
//		cGroupPath           string
//		ProcessState         protogo.ProcessState
//		TxWaitingQueue       chan *protogo.TxRequest
//		nextTxTrigger            chan bool
//		expireTimer          *time.Timer
//		logger               *zap.SugaredLogger
//		Handler              *ProcessHandler
//		user                 *security.User
//		cmd                  *exec.Cmd
//		processPoolInterface ProcessPoolInterface
//		isCrossProcess       bool
//		done                 uint32
//		balanceRWMutex                sync.Mutex
//	}
//	type args struct {
//		processTimeout bool
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		{
//			name: "testStopProcess",
//			fields: fields{
//				processName:          processName,
//				logger:               logger,
//				processPoolInterface: poolInterface,
//			},
//			args: args{processTimeout: true},
//		},
//		{
//			name: "testStopProcess",
//			fields: fields{
//				processName:          processName,
//				logger:               logger,
//				processPoolInterface: poolInterface,
//			},
//			args: args{processTimeout: false},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				processName:          tt.fields.processName,
//				processPoolInterface: tt.fields.processPoolInterface,
//				logger:               tt.fields.logger,
//			}
//			p.StopProcess(tt.args.processTimeout)
//		})
//	}
//}
//
//func TestProcess_killCrossProcess(t *testing.T) {
//	basePath, _ := os.Getwd()
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS, basePath+testPath)
//	type fields struct {
//		logger *zap.SugaredLogger
//		cmd    *exec.Cmd
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		{
//			name: "testKillCrossProcess",
//			fields: fields{
//				logger: logger,
//				cmd: &exec.Cmd{
//					Process: &os.Process{},
//				},
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				logger: tt.fields.logger,
//				cmd:    tt.fields.cmd,
//			}
//			p.killCrossProcess()
//		})
//	}
//}
//
//func TestProcess_printContractLog(t *testing.T) {
//	cmd := exec.Cmd{
//		Path: contractPath,
//		Args: []string{sockPath, processName, contractName, contractVersion, config.SandBoxLogLevel},
//	}
//
//	contractOut, _ := cmd.StdoutPipe()
//	type fields struct {
//		processName          string
//		contractName         string
//		contractVersion      string
//		contractPath         string
//		cGroupPath           string
//		ProcessState         protogo.ProcessState
//		TxWaitingQueue       chan *protogo.TxRequest
//		nextTxTrigger            chan bool
//		expireTimer          *time.Timer
//		logger               *zap.SugaredLogger
//		Handler              *ProcessHandler
//		user                 *security.User
//		cmd                  *exec.Cmd
//		processPoolInterface ProcessPoolInterface
//		isCrossProcess       bool
//		done                 uint32
//		balanceRWMutex                sync.Mutex
//	}
//	type args struct {
//		contractPipe io.ReadCloser
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		{
//			name: "printContractLog",
//			fields: fields{
//				processName:          "",
//				contractName:         "",
//				contractVersion:      "",
//				contractPath:         "",
//				cGroupPath:           "",
//				ProcessState:         0,
//				TxWaitingQueue:       nil,
//				nextTxTrigger:            nil,
//				expireTimer:          nil,
//				logger:               nil,
//				Handler:              nil,
//				user:                 nil,
//				cmd:                  nil,
//				processPoolInterface: nil,
//				isCrossProcess:       false,
//				done:                 0,
//				balanceRWMutex:                sync.Mutex{},
//			},
//			args: args{contractPipe: contractOut},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				processName:          tt.fields.processName,
//				contractName:         tt.fields.contractName,
//				contractVersion:      tt.fields.contractVersion,
//				contractPath:         tt.fields.contractPath,
//				cGroupPath:           tt.fields.cGroupPath,
//				ProcessState:         tt.fields.ProcessState,
//				TxWaitingQueue:       tt.fields.TxWaitingQueue,
//				nextTxTrigger:            tt.fields.nextTxTrigger,
//				expireTimer:          tt.fields.expireTimer,
//				logger:               tt.fields.logger,
//				Handler:              tt.fields.Handler,
//				user:                 tt.fields.user,
//				cmd:                  tt.fields.cmd,
//				processPoolInterface: tt.fields.processPoolInterface,
//				isCrossProcess:       tt.fields.isCrossProcess,
//				done:                 tt.fields.done,
//				balanceRWMutex:                tt.fields.balanceRWMutex,
//			}
//
//			go p.printContractLog(contractOut)
//		})
//	}
//}
//
//func TestProcess_resetProcessTimer(t *testing.T) {
//	type fields struct {
//		expireTimer *time.Timer
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		{
//			name: "testTesetProcessTimer",
//			fields: fields{
//				expireTimer: time.NewTimer(time.Second),
//			},
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				expireTimer: tt.fields.expireTimer,
//			}
//
//			go func() {
//				times := make(chan time.Time, 2)
//				for {
//					times <- time.Now()
//					p.expireTimer.C = times
//				}
//
//			}()
//
//			p.resetProcessTimer()
//		})
//	}
//}
//
//func TestProcess_triggerProcessState(t *testing.T) {
//	basePath, _ := os.Getwd()
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS, basePath+testPath)
//	type fields struct {
//		ProcessState protogo.ProcessState
//		nextTxTrigger    chan bool
//		logger       *zap.SugaredLogger
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		{
//			name: "testTriggerProcessState",
//			fields: fields{
//				ProcessState: protogo.ProcessState_PROCESS_STATE_CREATED,
//				logger:       logger,
//				nextTxTrigger:    make(chan bool, 0),
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				ProcessState: tt.fields.ProcessState,
//				nextTxTrigger:    tt.fields.nextTxTrigger,
//				logger:       tt.fields.logger,
//			}
//			go func() {
//				for {
//					<-p.nextTxTrigger
//				}
//			}()
//			p.triggerNextTx()
//		})
//	}
//}
//
//func TestProcess_updateProcessState(t *testing.T) {
//	logger := logger.NewDockerLogger(logger.MODULE_PROCESS)
//	type fields struct {
//		ProcessState protogo.ProcessState
//		logger       *zap.SugaredLogger
//	}
//
//	type args struct {
//		state protogo.ProcessState
//	}
//
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		{
//			name: "testUpdateProcessState",
//			fields: fields{
//				ProcessState: 0,
//				logger:       logger,
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			p := &Process{
//				ProcessState: tt.fields.ProcessState,
//				logger:       tt.fields.logger,
//			}
//			p.updateProcessState(protogo.ProcessState_PROCESS_STATE_CREATED)
//		})
//	}
//}

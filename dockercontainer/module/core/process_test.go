/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/protocol"
	"go.uber.org/zap"
	"io"
	"os/exec"
	"path/filepath"
	"reflect"
	"sync"
	"testing"
	"time"
)

func TestNewCrossProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir)
	timeTimer := time.NewTimer(processWaitingTime * time.Second)
	type args struct {
		user         *security.User
		txRequest    *protogo.TxRequest
		scheduler    protocol.Scheduler
		processName  string
		contractPath string
		processPool  ProcessPoolInterface
	}
	tests := []struct {
		name string
		args args
		want *Process
	}{
		{
			name: "testNewCrossProces",
			args: args{
				user: nil,
				txRequest: &protogo.TxRequest{
					TxId:            "",
					ContractName:    "txRequest.ContractName",
					ContractVersion: "txRequest.ContractVersion",
					Method:          "",
					Parameters:      nil,
					TxContext:       nil,
				},
				scheduler:    nil,
				processName:  processName,
				contractPath: "contractPath",
				processPool:  nil,
			},
			want: &Process{
				isCrossProcess:  true,
				processName:     processName,
				contractName:    "txRequest.ContractName",
				contractVersion: "txRequest.ContractVersion",
				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
				TxWaitingQueue:  nil,
				txTrigger:       nil,
				expireTimer:     timeTimer,
				logger:          log,

				Handler:              nil,
				user:                 nil,
				contractPath:         "contractPath",
				cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
				processPoolInterface: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.want.Handler = NewProcessHandler(tt.args.txRequest, tt.args.scheduler, tt.want)

			got := NewCrossProcess(tt.args.user, tt.args.txRequest, tt.args.scheduler, tt.args.processName, tt.args.contractPath, tt.args.processPool)
			got.Handler = tt.want.Handler
			got.logger = log
			got.expireTimer = timeTimer
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCrossProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewProcess(t *testing.T) {
	type args struct {
		user         *security.User
		txRequest    *protogo.TxRequest
		scheduler    protocol.Scheduler
		processName  string
		contractPath string
		processPool  ProcessPoolInterface
	}
	tests := []struct {
		name string
		args args
		want *Process
	}{
		{
			name: "testNewProcess",
			args: args{
				user: nil,
				txRequest: &protogo.TxRequest{
					TxId:            "",
					ContractName:    "txRequest.ContractName",
					ContractVersion: "txRequest.ContractVersion",
					Method:          "",
					Parameters:      nil,
					TxContext:       nil,
				},
				scheduler:    nil,
				processName:  processName,
				contractPath: "contractPath",
				processPool:  nil,
			},

			want: &Process{
				isCrossProcess:  false,
				processName:     processName,
				contractName:    "txRequest.ContractName",
				contractVersion: "txRequest.ContractVersion",
				ProcessState:    protogo.ProcessState_PROCESS_STATE_CREATED,
				TxWaitingQueue:  nil,
				txTrigger:       nil,
				expireTimer:     nil,
				logger:          nil,

				Handler:              nil,
				user:                 nil,
				contractPath:         "contractPath",
				cGroupPath:           filepath.Join(config.CGroupRoot, config.ProcsFile),
				processPoolInterface: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewProcess(tt.args.user, tt.args.txRequest, tt.args.scheduler, tt.args.processName, tt.args.contractPath, tt.args.processPool);
			tt.want.expireTimer = got.expireTimer
			tt.want.logger = got.logger
			tt.want.TxWaitingQueue = got.TxWaitingQueue
			tt.want.txTrigger = got.txTrigger
			tt.want.Handler = got.Handler
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcess_AddTxWaitingQueue(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS, config.DockerLogDir)
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	type args struct {
		tx *protogo.TxRequest
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testAddTxWaitingQueue",
			fields: fields{
				processName:          processName,
				contractName:         "",
				contractVersion:      "",
				contractPath:         "",
				cGroupPath:           "",
				ProcessState:         0,
				TxWaitingQueue:       make(chan *protogo.TxRequest),
				txTrigger:            nil,
				expireTimer:          nil,
				logger:               log,
				Handler:              nil,
				user:                 nil,
				cmd:                  nil,
				processPoolInterface: nil,
				isCrossProcess:       false,
				done:                 0,
				mutex:                sync.Mutex{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}

			go func() {
				for  {
					_ = <- p.TxWaitingQueue
				}
			}()

			p.AddTxWaitingQueue(&protogo.TxRequest{
				TxId:            "",
				ContractName:    "",
				ContractVersion: "",
				Method:          "",
				Parameters:      nil,
				TxContext:       &protogo.TxContext{
					CurrentHeight:       0,
					WriteMap:            nil,
					ReadMap:             nil,
					OriginalProcessName: "",
				},
			})
		})
	}
}

func TestProcess_InvokeProcess(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.InvokeProcess()
		})
	}
}

func TestProcess_LaunchProcess(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			if err := p.LaunchProcess(); (err != nil) != tt.wantErr {
				t.Errorf("LaunchProcess() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcess_StopProcess(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	type args struct {
		processTimeout bool
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.StopProcess(false)
		})
	}
}

func TestProcess_killCrossProcess(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.killCrossProcess()
		})
	}
}

func TestProcess_killProcess(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.killProcess()
		})
	}
}

func TestProcess_printContractLog(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	type args struct {
		contractPipe io.ReadCloser
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}

			cmd := exec.Cmd{
				Path: p.contractPath,
				Args: []string{p.user.SockPath, p.processName, p.contractName, p.contractVersion, config.SandBoxLogLevel},
			}

			contractOut, _ := cmd.StdoutPipe()
			p.printContractLog(contractOut)
		})
	}
}

func TestProcess_resetProcessTimer(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.resetProcessTimer()
		})
	}
}

func TestProcess_triggerProcessState(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.triggerProcessState()
		})
	}
}

func TestProcess_updateProcessState(t *testing.T) {
	type fields struct {
		processName          string
		contractName         string
		contractVersion      string
		contractPath         string
		cGroupPath           string
		ProcessState         protogo.ProcessState
		TxWaitingQueue       chan *protogo.TxRequest
		txTrigger            chan bool
		expireTimer          *time.Timer
		logger               *zap.SugaredLogger
		Handler              *ProcessHandler
		user                 *security.User
		cmd                  *exec.Cmd
		processPoolInterface ProcessPoolInterface
		isCrossProcess       bool
		done                 uint32
		mutex                sync.Mutex
	}
	type args struct {
		state protogo.ProcessState
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Process{
				processName:          tt.fields.processName,
				contractName:         tt.fields.contractName,
				contractVersion:      tt.fields.contractVersion,
				contractPath:         tt.fields.contractPath,
				cGroupPath:           tt.fields.cGroupPath,
				ProcessState:         tt.fields.ProcessState,
				TxWaitingQueue:       tt.fields.TxWaitingQueue,
				txTrigger:            tt.fields.txTrigger,
				expireTimer:          tt.fields.expireTimer,
				logger:               tt.fields.logger,
				Handler:              tt.fields.Handler,
				user:                 tt.fields.user,
				cmd:                  tt.fields.cmd,
				processPoolInterface: tt.fields.processPoolInterface,
				isCrossProcess:       tt.fields.isCrossProcess,
				done:                 tt.fields.done,
				mutex:                tt.fields.mutex,
			}
			p.updateProcessState(protogo.ProcessState_PROCESS_STATE_CREATED)
		})
	}
}

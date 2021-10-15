/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"reflect"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
	"go.uber.org/zap"
)

var (
	processName = "processName1"
)

func TestNewProcessPool(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	tests := []struct {
		name string
		want *ProcessPool
	}{
		{
			name: "testNewProcessPool",
			want: &ProcessPool{
				ProcessTable: sync.Map{},
				logger:       log,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewProcessPool()
			got.logger = log
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPool_CheckProcessExist(t *testing.T) {
	var processName string = "processName"
	var processNameFalse string = "processName1"
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)

	processTableValue := &ProcessContext{
		processList: [protocol.CallContractDepth + 1]*Process{},
	}

	processTable := sync.Map{}
	processTable.Store(processName, processTableValue)
	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}

	type args struct {
		processName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *Process
		want1  bool
	}{

		{
			name: "testCheckProcessExist",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				processName: processName,
			},
			want:  nil,
			want1: true,
		},

		{
			name: "testCheckProcessExistFalse",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				processName: processNameFalse,
			},
			want:  nil,
			want1: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			got, got1 := pp.CheckProcessExist(tt.args.processName)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("CheckProcessExist() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("CheckProcessExist() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestProcessPool_RegisterCrossProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	var processTableKey string = "processTableKey1"
	processTableValue := &ProcessContext{
		processList: [6]*Process{},
		size:        1,
	}
	processTable := sync.Map{}
	processTable.Store(processTableKey, processTableValue)

	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}
	type args struct {
		initialProcessName string
		calledProcess      *Process
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testRegisterCrossProcess",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				initialProcessName: processTableKey,
				calledProcess:      &Process{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			pp.RegisterCrossProcess(tt.args.initialProcessName, tt.args.calledProcess)
		})
	}
}

func TestProcessPool_RegisterNewProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}
	type args struct {
		processName string
		process     *Process
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testRegisterNewProcess",
			fields: fields{
				ProcessTable: sync.Map{},
				logger:       log,
			},
			args: args{
				processName: "processName1",
				process:     nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			pp.RegisterNewProcess(tt.args.processName, tt.args.process)
		})
	}
}

func TestProcessPool_ReleaseCrossProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)

	processTable := sync.Map{}
	processTableKey := "processTableKey1"
	processTableValue := &ProcessContext{
		processList: [6]*Process{},
		size:        1,
	}

	processTable.Store(processTableKey, processTableValue)
	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}

	type args struct {
		initialProcessName string
		currentHeight      uint32
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testReleaseCrossProcess",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				initialProcessName: processTableKey,
				currentHeight:      1,
			},
		},

		{
			name: "testReleaseCrossProcessFalse",
			fields: fields{
				ProcessTable: sync.Map{},
				logger:       log,
			},
			args: args{
				initialProcessName: processTableKey,
				currentHeight:      0,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			pp.ReleaseCrossProcess(tt.args.initialProcessName, tt.args.currentHeight)
		})
	}
}

func TestProcessPool_ReleaseProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	processTable := sync.Map{}
	processTableKey := processName
	processTable.Store(processTableKey, "")

	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}

	type args struct {
		processName string
	}

	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testReleaseProcess",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				processName: processName,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			pp.ReleaseProcess(tt.args.processName)
		})
	}
}

func TestProcessPool_RetrieveHandlerFromProcess(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	processTable := sync.Map{}
	processTableKey := processName
	processTableValue := &ProcessContext{
		processList: [6]*Process{
			{Handler: &ProcessHandler{}},
		},
		size: 0,
	}

	processTable.Store(processTableKey, processTableValue)

	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}
	type args struct {
		processName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ProcessHandler
	}{
		{
			name: "testRetrieveHandlerFromProcess",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				processName: processName,
			},

			want: &ProcessHandler{},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			if got := pp.RetrieveHandlerFromProcess(tt.args.processName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RetrieveHandlerFromProcess() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPool_RetrieveProcessContext(t *testing.T) {
	log := logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir)
	processTable := sync.Map{}
	processTableKey := processName
	processTableValue := &ProcessContext{
		processList: [6]*Process{
			{Handler: &ProcessHandler{}},
		},
		size: 0,
	}

	processTable.Store(processTableKey, processTableValue)

	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}
	type args struct {
		initialProcessName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *ProcessContext
	}{
		{
			name: "testRetrieveProcessContext",
			fields: fields{
				ProcessTable: processTable,
				logger:       log,
			},
			args: args{
				initialProcessName: processName,
			},
			want: processTableValue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			if got := pp.RetrieveProcessContext(tt.args.initialProcessName); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("RetrieveProcessContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPool_newProcessContext(t *testing.T) {
	type fields struct {
		ProcessTable sync.Map
		logger       *zap.SugaredLogger
	}
	tests := []struct {
		name   string
		fields fields
		want   *ProcessContext
	}{
		{
			name: "testNewProcessContext",
			fields: fields{
				ProcessTable: sync.Map{},
				logger:       nil,
			},

			want: &ProcessContext{
				processList: [protocol.CallContractDepth + 1]*Process{},
				size:        0,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pp := &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
			if got := pp.newProcessContext(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("newProcessContext() = %v, want %v", got, tt.want)
			}
		})
	}
}

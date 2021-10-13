/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"go.uber.org/zap"
	"reflect"
	"sync"
	"testing"
)

func TestNewProcessPool(t *testing.T) {
	tests := []struct {
		name string
		want *ProcessPool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewProcessPool(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessPool() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessPool_CheckProcessExist(t *testing.T) {
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
		})
	}
}

func TestProcessPool_RegisterNewProcess(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
		})
	}
}

func TestProcessPool_ReleaseCrossProcess(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
		})
	}
}

func TestProcessPool_ReleaseProcess(t *testing.T) {
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
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = &ProcessPool{
				ProcessTable: tt.fields.ProcessTable,
				logger:       tt.fields.logger,
			}
		})
	}
}

func TestProcessPool_RetrieveHandlerFromProcess(t *testing.T) {
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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
		// TODO: Add test cases.
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

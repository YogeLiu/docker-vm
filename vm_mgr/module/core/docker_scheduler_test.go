///*
//Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
//
//SPDX-License-Identifier: Apache-2.0
//*/
//
package core

//
//import (
//	"reflect"
//	"sync"
//	"testing"
//
//	"go.uber.org/zap"
//	"golang.org/x/sync/singleflight"
//)
//
//func TestDockerScheduler_GetByteCodeReqCh(t *testing.T) {
//	type fields struct {
//		getByteCodeReqCh chan *protogo.CDMMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   chan *protogo.CDMMessage
//	}{
//		{
//			name: "GetByteCodeReqCh",
//			fields: fields{
//				getByteCodeReqCh: make(chan *protogo.CDMMessage),
//			},
//		},
//	}
//
//	for _, tt := range tests {
//		go func() {
//			for {
//				tt.fields.getByteCodeReqCh <- &protogo.CDMMessage{
//					TxId:    "txId",
//					Type:    0,
//					Payload: []byte("payload"),
//				}
//			}
//		}()
//
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//			}
//
//			if got := s.GetByteCodeReqCh(); got == nil {
//				t.Errorf("GetByteCodeReqCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetCrossContractReqCh(t *testing.T) {
//	type fields struct {
//		crossContractsCh chan *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   chan *protogo.TxRequest
//	}{
//		{
//			name:   "testGetCrossContractReqCh",
//			fields: fields{crossContractsCh: make(chan *protogo.TxRequest)},
//		},
//	}
//	for _, tt := range tests {
//		go func() {
//			for {
//				tt.fields.crossContractsCh <- &protogo.TxRequest{
//					TxId:            "txId",
//					ContractName:    contractName,
//					ContractVersion: "contract",
//					Method:          "method",
//				}
//			}
//		}()
//
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//
//			if got := s.GetCrossContractReqCh(); got == nil {
//				t.Errorf("GetCrossContractReqCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetCrossContractResponseCh(t *testing.T) {
//	type fields struct {
//		responseChMap sync.Map
//	}
//	type args struct {
//		responseId string
//	}
//
//	responseValue := make(chan *SDKProtogo.DMSMessage)
//	responseChMap := sync.Map{}
//	responseChMap.Store("responseId", responseValue)
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   chan *SDKProtogo.DMSMessage
//	}{
//		{
//			name: "testGetCrossContractResponseCh",
//			fields: fields{
//				responseChMap: responseChMap,
//			},
//			args: args{responseId: "responseId"},
//		},
//	}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				responseChMap: tt.fields.responseChMap,
//			}
//			if got := s.GetCrossContractResponseCh(tt.args.responseId); got == nil {
//				t.Errorf("GetCrossContractResponseCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetGetStateReqCh(t *testing.T) {
//	type fields struct {
//		getStateReqCh chan *protogo.CDMMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   chan *protogo.CDMMessage
//	}{
//		{
//			name:   "testGetGetStateReqCh",
//			fields: fields{getStateReqCh: make(chan *protogo.CDMMessage)},
//		},
//	}
//	for _, tt := range tests {
//		go func() {
//			for {
//				tt.fields.getStateReqCh <- &protogo.CDMMessage{
//					TxId:    "txId",
//					Type:    0,
//					Payload: []byte("payload"),
//				}
//			}
//		}()
//
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				getStateReqCh: tt.fields.getStateReqCh,
//			}
//			if got := s.GetGetStateReqCh(); got == nil {
//				t.Errorf("GetGetStateReqCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetResponseChByTxId(t *testing.T) {
//	type fields struct {
//		responseChMap sync.Map
//	}
//	type args struct {
//		txId string
//	}
//
//	responseValue := make(chan *protogo.CDMMessage)
//	responseChMap := sync.Map{}
//	responseChMap.Store("txId", responseValue)
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   chan *protogo.CDMMessage
//	}{
//		{
//			name: "testGetResponseChByTxId",
//			fields: fields{
//				responseChMap: responseChMap,
//			},
//			args: args{
//				txId: "txId",
//			},
//		}}
//
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				responseChMap: tt.fields.responseChMap,
//			}
//			if got := s.GetResponseChByTxId(tt.args.txId); got == nil {
//				t.Errorf("GetResponseChByTxId() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetTxReqCh(t *testing.T) {
//	type fields struct {
//		txReqCh chan *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   chan *protogo.TxRequest
//	}{
//		{
//			name:   "testGetTxReqCh",
//			fields: fields{txReqCh: make(chan *protogo.TxRequest)},
//		},
//	}
//
//	for _, tt := range tests {
//		go func() {
//			for  {
//				tt.fields.txReqCh <- &protogo.TxRequest{
//					TxId:            "txId",
//					ContractName:    contractName,
//					ContractVersion: "contract",
//					Method:          "method",
//				}
//			}
//		}()
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				txReqCh: tt.fields.txReqCh,
//			}
//			if got := s.GetTxReqCh(); got == nil {
//				t.Errorf("GetTxReqCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_GetTxResponseCh(t *testing.T) {
//	type fields struct {
//		txResponseCh chan *protogo.TxResponse
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		want   chan *protogo.TxResponse
//	}{
//		{
//			name:   "testGetTxResponseCh",
//			fields: fields{txResponseCh: make(chan *protogo.TxResponse)},
//		},
//	}
//
//	for _, tt := range tests {
//		go func() {
//			for  {
//				tt.fields.txResponseCh <- &protogo.TxResponse{
//					TxId: "txId",
//				}
//			}
//		}()
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				txResponseCh: tt.fields.txResponseCh,
//			}
//			if got := s.GetTxResponseCh(); got == nil {
//				t.Errorf("GetTxResponseCh() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_RegisterCrossContractResponseCh(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		responseId string
//		responseCh chan *SDKProtogo.DMSMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_RegisterResponseCh(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		responseId string
//		responseCh chan *protogo.CDMMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_StartScheduler(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_StopScheduler(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_constructContractKey(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		contractName    string
//		contractVersion string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//			if got := s.constructContractKey(tt.args.contractName, tt.args.contractVersion); got != tt.want {
//				t.Errorf("constructContractKey() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_constructCrossContractProcessName(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		tx *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//			if got := s.constructCrossContractProcessName(tt.args.tx); got != tt.want {
//				t.Errorf("constructCrossContractProcessName() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_constructErrorResponse(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		txId   string
//		errMsg string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   *protogo.TxResponse
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//			if got := s.constructErrorResponse(tt.args.txId, tt.args.errMsg); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("constructErrorResponse() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_constructProcessName(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		tx *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//		want   string
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//			if got := s.constructProcessName(tt.args.tx); got != tt.want {
//				t.Errorf("constructProcessName() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_handleCallCrossContract(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		crossContractTx *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_handleTx(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		txRequest *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_initProcess(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		process *Process
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_listenIncomingTxRequest(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_listenProcessInvoke(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		process *Process
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_returnErrorCrossContractResponse(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		crossContractTx *protogo.TxRequest
//		errResponse     *SDKProtogo.DMSMessage
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestDockerScheduler_returnErrorTxResponse(t *testing.T) {
//	type fields struct {
//		lock             sync.Mutex
//		logger           *zap.SugaredLogger
//		singleFlight     singleflight.Group
//		usersManager   protocol.UserController
//		contractManager  *ContractManager
//		processPool      *ProcessManager
//		txReqCh          chan *protogo.TxRequest
//		txResponseCh     chan *protogo.TxResponse
//		getStateReqCh    chan *protogo.CDMMessage
//		getByteCodeReqCh chan *protogo.CDMMessage
//		responseChMap    sync.Map
//		crossContractsCh chan *protogo.TxRequest
//	}
//	type args struct {
//		txId   string
//		errMsg string
//	}
//	tests := []struct {
//		name   string
//		fields fields
//		args   args
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			_ = &DockerScheduler{
//				lock:             tt.fields.lock,
//				logger:           tt.fields.logger,
//				singleFlight:     tt.fields.singleFlight,
//				usersManager:   tt.fields.usersManager,
//				contractManager:  tt.fields.contractManager,
//				processPool:      tt.fields.processPool,
//				txReqCh:          tt.fields.txReqCh,
//				txResponseCh:     tt.fields.txResponseCh,
//				getStateReqCh:    tt.fields.getStateReqCh,
//				getByteCodeReqCh: tt.fields.getByteCodeReqCh,
//				responseChMap:    tt.fields.responseChMap,
//				crossContractsCh: tt.fields.crossContractsCh,
//			}
//		})
//	}
//}
//
//func TestNewDockerScheduler(t *testing.T) {
//	type args struct {
//		usersManager protocol.UserController
//		processPool    *ProcessManager
//	}
//	tests := []struct {
//		name string
//		args args
//		want *DockerScheduler
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			if got := NewDockerScheduler(tt.args.usersManager, tt.args.processPool); !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("NewDockerScheduler() = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

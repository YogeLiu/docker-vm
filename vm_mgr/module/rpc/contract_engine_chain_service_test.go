/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/mocks"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"fmt"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	"testing"
)

func TestChainRPCService_DockerVMCommunicate_Recv(t *testing.T) {

	SetConfig()

	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Send(nil).Return(nil).AnyTimes()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{Type: protogo.DockerVMType_TX_REQUEST}, nil).Times(1)
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).Times(1)
	stream.EXPECT().Recv().Return(nil, fmt.Errorf("err")).Times(1)

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler interfaces.RequestScheduler
		eventCh   chan *protogo.DockerVMMessage
	}
	type args struct {
		stream protogo.DockerVMRpc_DockerVMCommunicateServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCService_DockerVMCommunicate",
			fields: fields{
				logger:    logger.NewTestDockerLogger(),
				scheduler: &mocks.MockRequestScheduler{},
				eventCh:   make(chan *protogo.DockerVMMessage, _rpcEventChSize),
			},
			args:    args{stream: stream},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:  tt.fields.logger,
				eventCh: tt.fields.eventCh,
			}
			service.SetScheduler(tt.fields.scheduler)

			//stream.Send(&protogo.DockerVMMessage{})

			if err := service.DockerVMCommunicate(stream); (err != nil) != tt.wantErr {
				t.Errorf("DockerVMCommunicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//func TestChainRPCService_DockerVMCommunicate_Send(t *testing.T) {
//
//	SetConfig()
//
//	s := newMockStream(t)
//	defer s.finish()
//	stream := s.getStream()
//
//	errMsg := &protogo.DockerVMMessage{Type: protogo.DockerVMType_ERROR}
//	reqMsg := &protogo.DockerVMMessage{Type: protogo.DockerVMType_GET_BYTECODE_REQUEST}
//	stream.EXPECT().Send(reqMsg).Return(nil).Times(1)
//	stream.EXPECT().Send(errMsg).Return(nil).Times(1)
//	stream.EXPECT().Send(errMsg).Return(fmt.Errorf("err")).Times(1)
//
//	type fields struct {
//		logger    *zap.SugaredLogger
//		scheduler interfaces.RequestScheduler
//		eventCh   chan *protogo.DockerVMMessage
//	}
//	type args struct {
//		stream protogo.DockerVMRpc_DockerVMCommunicateServer
//	}
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "TestChainRPCService_DockerVMCommunicate",
//			fields: fields{
//				logger:    logger.NewTestDockerLogger(),
//				scheduler: &mocks.MockRequestScheduler{},
//				eventCh:   make(chan *protogo.DockerVMMessage, _rpcEventChSize),
//			},
//			args:    args{stream: stream},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			service := &ChainRPCService{
//				logger:  tt.fields.logger,
//				eventCh: tt.fields.eventCh,
//			}
//			service.PutMsg(reqMsg)
//			service.PutMsg(errMsg)
//			service.PutMsg(&protogo.DockerVMMessage{})
//			service.PutMsg(errMsg)
//			service.SetScheduler(tt.fields.scheduler)
//
//			//stream.Send(&protogo.DockerVMMessage{})
//
//			if err := service.DockerVMCommunicate(stream); (err != nil) != tt.wantErr {
//				t.Errorf("DockerVMCommunicate() error = %v, wantErr %v", err, tt.wantErr)
//			}
//		})
//	}
//}

func TestChainRPCService_PutMsg(t *testing.T) {

	SetConfig()

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler interfaces.RequestScheduler
		eventCh   chan *protogo.DockerVMMessage
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
			name: "TestChainRPCService_PutMsg",
			fields: fields{
				logger:    logger.NewTestDockerLogger(),
				scheduler: &mocks.MockRequestScheduler{},
				eventCh:   make(chan *protogo.DockerVMMessage, _rpcEventChSize),
			},
			args:    args{msg: &protogo.DockerVMMessage{}},
			wantErr: false,
		},
		{
			name: "TestChainRPCService_PutMsg",
			fields: fields{
				logger:    logger.NewTestDockerLogger(),
				scheduler: &mocks.MockRequestScheduler{},
				eventCh:   make(chan *protogo.DockerVMMessage, _rpcEventChSize),
			},
			args:    args{msg: "string"},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:  tt.fields.logger,
				eventCh: tt.fields.eventCh,
			}
			service.SetScheduler(tt.fields.scheduler)

			if err := service.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//func TestChainRPCService_sendMsgRoutine(t *testing.T) {
//	s := newMockStream(t)
//	defer s.finish()
//	stream := s.getStream()
//	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
//	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()
//
//	SetConfig()
//
//	type fields struct {
//		logger     *zap.SugaredLogger
//		scheduler  interfaces.RequestScheduler
//		eventCh    chan *protogo.DockerVMMessage
//	}
//
//	type args struct {
//		scheduler interfaces.RequestScheduler
//	}
//
//	tests := []struct {
//		name    string
//		fields  fields
//		args    args
//		wantErr bool
//	}{
//		{
//			name: "TestChainRPCService_sendMsgRoutine",
//			fields: fields{
//				logger:     logger.NewTestDockerLogger(),
//				eventCh:    make(chan *protogo.DockerVMMessage, _rpcEventChSize),
//			},
//			args:    args{scheduler: nil},
//			wantErr: false,
//		},
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			service := &ChainRPCService{
//				logger:     tt.fields.logger,
//				eventCh:    tt.fields.eventCh,
//			}
//
//			service.wg.Add(1)
//
//			go func() {
//				for {
//					service.stopSendCh <- struct{}{}
//				}
//			}()
//
//			service.sendMsgRoutine()
//
//		})
//	}
//}

func TestNewChainRPCService(t *testing.T) {
	SetConfig()

	tests := []struct {
		name    string
		wantErr bool
	}{
		{
			name:    "TestNewChainRPCService",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewChainRPCService()
			if got == nil {
				t.Errorf("NewChainRPCService() got = %v", got)
			}
		})
	}
}

type mockStream struct {
	c *gomock.Controller
}

func newMockStream(t *testing.T) *mockStream {
	return &mockStream{c: gomock.NewController(t)}
}

func (m *mockStream) getStream() *MockDockerVMRpc_DockerVMCommunicateServer {
	return NewMockDockerVMRpc_DockerVMCommunicateServer(m.c)
}

func (m *mockStream) finish() {
	m.c.Finish()
}

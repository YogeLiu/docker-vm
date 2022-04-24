/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	"sync"
	"testing"
)

func TestChainRPCService_DockerVMCommunicate(t *testing.T) {

	SetConfig()

	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
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
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
			},
			args:    args{stream: stream},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
				stream:     tt.args.stream,
			}
			sendWait := &sync.WaitGroup{}
			sendWait.Add(1)
			go func(group *sync.WaitGroup) {
				service.stream.Send(&protogo.DockerVMMessage{})
				sendWait.Done()
			}(sendWait)
			sendWait.Wait()

			go func() {
				for {
					service.stopSendCh <- struct{}{}
					service.stopRecvCh <- struct{}{}
				}
			}()

			if err := service.DockerVMCommunicate(service.stream); (err != nil) != tt.wantErr {
				t.Errorf("DockerVMCommunicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainRPCService_PutMsg(t *testing.T) {

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
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
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
			},
			args:    args{msg: &protogo.DockerVMMessage{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
			}

			if err := service.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainRPCService_SetScheduler(t *testing.T) {

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
	}

	type args struct {
		scheduler interfaces.RequestScheduler
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCService_SetScheduler",
			fields: fields{
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
			},
			args:    args{scheduler: nil},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
			}

			service.SetScheduler(tt.args.scheduler)
		})
	}
}

func TestChainRPCService_recvMsgRoutine(t *testing.T) {

	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
		stream     protogo.DockerVMRpc_DockerVMCommunicateServer
	}

	type args struct {
		scheduler interfaces.RequestScheduler
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCService_recvMsgRoutine",
			fields: fields{
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
				stream:     stream,
			},
			args:    args{scheduler: nil},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
				stream:     tt.fields.stream,
			}

			service.wg.Add(1)

			go func() {
				for {
					service.stopRecvCh <- struct{}{}
				}
			}()

			service.recvMsgRoutine()

		})
	}
}

func TestChainRPCService_recvMsg(t *testing.T) {

	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
		stream     protogo.DockerVMRpc_DockerVMCommunicateServer
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "TestChainRPCService_recvMsg",
			fields: fields{
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
				stream:     stream,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
				stream:     tt.fields.stream,
			}

			if _, err := service.recvMsg(); (err != nil) != tt.wantErr {
				t.Errorf("recvMsg() error = %v, wantErr %v", err, tt.wantErr)
			}

		})
	}
}

func TestChainRPCService_sendMsg(t *testing.T) {
	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
		stream     protogo.DockerVMRpc_DockerVMCommunicateServer
	}

	type args struct {
		msg *protogo.DockerVMMessage
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCService_sendMsg",
			fields: fields{
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
				stream:     stream,
			},
			args:    args{msg: &protogo.DockerVMMessage{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
				stream:     tt.fields.stream,
			}

			if err := service.sendMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("sendMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainRPCService_sendMsgRoutine(t *testing.T) {
	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.DockerVMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	SetConfig()

	type fields struct {
		logger     *zap.SugaredLogger
		scheduler  interfaces.RequestScheduler
		eventCh    chan *protogo.DockerVMMessage
		stopSendCh chan struct{}
		stopRecvCh chan struct{}
		wg         *sync.WaitGroup
		stream     protogo.DockerVMRpc_DockerVMCommunicateServer
	}

	type args struct {
		scheduler interfaces.RequestScheduler
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestChainRPCService_sendMsgRoutine",
			fields: fields{
				logger:     logger.NewTestDockerLogger(),
				eventCh:    make(chan *protogo.DockerVMMessage, rpcEventChSize),
				stopSendCh: make(chan struct{}),
				stopRecvCh: make(chan struct{}),
				wg:         new(sync.WaitGroup),
				stream:     stream,
			},
			args:    args{scheduler: nil},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := &ChainRPCService{
				logger:     tt.fields.logger,
				eventCh:    tt.fields.eventCh,
				stopSendCh: tt.fields.stopSendCh,
				stopRecvCh: tt.fields.stopRecvCh,
				wg:         tt.fields.wg,
				stream:     tt.fields.stream,
			}

			service.wg.Add(1)

			go func() {
				for {
					service.stopSendCh <- struct{}{}
				}
			}()


			service.sendMsgRoutine()

		})
	}
}

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

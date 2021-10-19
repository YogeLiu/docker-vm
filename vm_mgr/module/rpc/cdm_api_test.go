/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import (
	"reflect"
	"sync"
	"testing"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/protocol"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/utils"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

func TestCDMApi_CDMCommunicate(t *testing.T) {
	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Recv().Return(&protogo.CDMMessage{}, nil).AnyTimes()
	stream.EXPECT().Send(&protogo.CDMMessage{}).Return(nil).AnyTimes()

	txResponse := make(chan *protogo.TxResponse)
	cmdMessage := make(chan *protogo.CDMMessage)

	ms := newMockScheduler(t)
	defer ms.finish()
	scheduler := ms.getScheduler()
	scheduler.EXPECT().GetTxResponseCh().Return(txResponse).AnyTimes()
	scheduler.EXPECT().GetGetStateReqCh().Return(cmdMessage).AnyTimes()
	scheduler.EXPECT().GetByteCodeReqCh().Return(cmdMessage).AnyTimes()

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	type args struct {
		stream protogo.CDMRpc_CDMCommunicateServer
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testCDMCommunicate",
			fields: fields{
				logger:    utils.GetLogHandler(),
				wg:        &sync.WaitGroup{},
				scheduler: scheduler,
			},
			args: args{
				stream: stream,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}

			sendWait := &sync.WaitGroup{}
			go func(group *sync.WaitGroup) {
				cdm.stream.Send(&protogo.CDMMessage{})
				<-cdm.scheduler.GetTxResponseCh()
				<-cdm.scheduler.GetGetStateReqCh()
				<-cdm.scheduler.GetByteCodeReqCh()
				sendWait.Done()
			}(sendWait)
			sendWait.Wait()

			go func() {
				for {
					cdm.stop <- struct{}{}
				}
			}()

			if err := cdm.CDMCommunicate(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("CDMCommunicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMApi_closeConnection(t *testing.T) {
	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "testCloseConnection",
			fields: fields{
				stream: nil,
				stop:   make(chan struct{}),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}
			cdm.closeConnection()
		})
	}
}

func TestCDMApi_constructCDMMessage(t *testing.T) {
	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	type args struct {
		txResponseMsg *protogo.TxResponse
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *protogo.CDMMessage
	}{
		{
			name:   "testConstructCDMMessage",
			fields: fields{},
			args: args{
				txResponseMsg: &protogo.TxResponse{
					TxId: "txId",
				},
			},
			want: &protogo.CDMMessage{
				TxId:    "txId",
				Type:    protogo.CDMType_CDM_TYPE_TX_RESPONSE,
				Payload: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}

			payload, _ := tt.args.txResponseMsg.Marshal()
			tt.want.Payload = payload
			if got := cdm.constructCDMMessage(tt.args.txResponseMsg); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructCDMMessage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCDMApi_handleGetByteCodeResponse(t *testing.T) {
	responseCh := make(chan *protogo.CDMMessage)
	s := newMockScheduler(t)
	defer s.finish()

	scheduler := s.getScheduler()
	scheduler.EXPECT().GetResponseChByTxId("txId").Return(responseCh)
	go func() {
		for {
			<-responseCh
		}
	}()

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
	}
	type args struct {
		cdmMessage *protogo.CDMMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testHandleGetByteCodeResponse",
			fields: fields{
				logger:    utils.GetLogHandler(),
				scheduler: scheduler,
			},
			args: args{cdmMessage: &protogo.CDMMessage{
				TxId: "txId",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
			}
			if err := cdm.handleGetByteCodeResponse(tt.args.cdmMessage); (err != nil) != tt.wantErr {
				t.Errorf("handleGetByteCodeResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMApi_handleGetStateResponse(t *testing.T) {
	responseCh := make(chan *protogo.CDMMessage)
	s := newMockScheduler(t)
	defer s.finish()

	scheduler := s.getScheduler()
	scheduler.EXPECT().GetResponseChByTxId("txId").Return(responseCh)
	go func() {
		for {
			<-responseCh
		}
	}()

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	type args struct {
		cdmMessage *protogo.CDMMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testHandleGetStateResponse",
			fields: fields{
				logger:    utils.GetLogHandler(),
				scheduler: scheduler,
			},
			args: args{cdmMessage: &protogo.CDMMessage{
				TxId: "txId",
			}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}
			if err := cdm.handleGetStateResponse(tt.args.cdmMessage); (err != nil) != tt.wantErr {
				t.Errorf("handleGetStateResponse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMApi_handleTxRequest(t *testing.T) {
	requests := make(chan *protogo.TxRequest)
	s := newMockScheduler(t)
	defer s.finish()
	scheduler := s.getScheduler()
	scheduler.EXPECT().GetTxReqCh().Return(requests).AnyTimes()

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	type args struct {
		cdmMessage *protogo.CDMMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testHandleTxRequest",
			fields: fields{
				logger:    utils.GetLogHandler(),
				scheduler: scheduler,
			},
			args: args{
				cdmMessage: &protogo.CDMMessage{
					TxId: "txId",
					Type: protogo.CDMType_CDM_TYPE_UNDEFINED,
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}

			go func() {
				for {
					<-cdm.scheduler.GetTxReqCh()
				}
			}()

			if err := cdm.handleTxRequest(tt.args.cdmMessage); (err != nil) != tt.wantErr {
				t.Errorf("handleTxRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMApi_recvMsgRoutine(t *testing.T) {
	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}
			cdm.recvMsgRoutine()
		})
	}
}

func TestCDMApi_sendMessage(t *testing.T) {
	s := newMockStream(t)
	defer s.finish()
	stream := s.getStream()
	stream.EXPECT().Send(&protogo.CDMMessage{}).Return(nil)

	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	type args struct {
		msg *protogo.CDMMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "sendMessage",
			fields: fields{
				logger: utils.GetLogHandler(),
				stream: stream,
			},
			args: args{
				msg: &protogo.CDMMessage{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}
			if err := cdm.sendMessage(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("sendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestCDMApi_sendMsgRoutine(t *testing.T) {
	type fields struct {
		logger    *zap.SugaredLogger
		scheduler protocol.Scheduler
		stream    protogo.CDMRpc_CDMCommunicateServer
		stop      chan struct{}
		wg        *sync.WaitGroup
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cdm := &CDMApi{
				logger:    tt.fields.logger,
				scheduler: tt.fields.scheduler,
				stream:    tt.fields.stream,
				stop:      tt.fields.stop,
				wg:        tt.fields.wg,
			}
			cdm.sendMsgRoutine()
		})
	}
}

func TestNewCDMApi(t *testing.T) {
	s := newMockScheduler(t)
	defer s.finish()
	scheduler := s.getScheduler()

	type args struct {
		scheduler protocol.Scheduler
	}

	tests := []struct {
		name string
		args args
		want *CDMApi
	}{
		{
			name: "testNewCDMAp",
			args: args{
				scheduler: scheduler,
			},
			want: &CDMApi{
				scheduler: scheduler,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewCDMApi(tt.args.scheduler)
			tt.want.logger = got.logger
			tt.want.wg = got.wg

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewCDMApi() = %v, want %v", got, tt.want)
			}
		})
	}
}

type mockScheduler struct {
	c *gomock.Controller
}

func newMockScheduler(t *testing.T) *mockScheduler {
	return &mockScheduler{c: gomock.NewController(t)}
}

func (s *mockScheduler) getScheduler() *protocol.MockScheduler {
	return protocol.NewMockScheduler(s.c)
}

func (s *mockScheduler) finish() {
	s.c.Finish()
}

type mockStream struct {
	c *gomock.Controller
}

func newMockStream(t *testing.T) *mockStream {
	return &mockStream{c: gomock.NewController(t)}
}

func (m *mockStream) getStream() *MockCDMRpc_CDMCommunicateServer {
	return NewMockCDMRpc_CDMCommunicateServer(m.c)
}

func (m *mockStream) finish() {
	m.c.Finish()
}

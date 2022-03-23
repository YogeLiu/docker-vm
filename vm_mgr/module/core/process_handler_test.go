/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"reflect"
	"testing"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	SDKProtogo "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb_sdk/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/protocol"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
)

func TestNewProcessHandler(t *testing.T) {
	type args struct {
		txRequest *protogo.TxRequest
		scheduler interfaces.Scheduler
		process   ProcessInterface
	}
	tests := []struct {
		name string
		args args
		want *ProcessHandler
	}{
		{
			name: "testNewProcessHandler",
			args: args{},

			want: &ProcessHandler{
				state: created,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewProcessHandler(tt.args.txRequest, tt.args.scheduler, tt.args.process)
			tt.want.logger = got.logger
			tt.want.txExpireTimer = got.txExpireTimer
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessHandler() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessHandler_HandleContract(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "testHandleContractInitContract",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          initContract,
					Parameters:      nil,
					TxContext:       nil,
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: time.NewTimer(time.Second),
			},
			wantErr: false,
		},
		{
			name: "testHandleContractInvokeContract",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          invokeContract,
					Parameters:      nil,
					TxContext: &protogo.TxContext{
						CurrentHeight: 10,
					},
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: time.NewTimer(time.Second),
			},
			wantErr: false,
		},

		{
			name: "testHandleContractUpgradeContract",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          upgradeContract,
					Parameters:      nil,
					TxContext:       nil,
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: time.NewTimer(time.Second),
			},
			wantErr: false,
		},
		{
			name: "testHandleContractDefault",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          method,
					Parameters:      nil,
					TxContext:       nil,
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: time.NewTimer(time.Second),
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.HandleContract(); (err != nil) != tt.wantErr {
				t.Errorf("HandleContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_HandleMessage(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		msg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testHandleMessageCreated",
			fields: fields{
				state:         created,
				logger:        utils.GetLogHandler(),
				TxRequest:     nil,
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
			args: args{
				msg: &SDKProtogo.DMSMessage{
					TxId:          txId,
					Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTER,
					CurrentHeight: 10,
					Payload:       []byte(payload),
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.HandleMessage(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("HandleMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_SetStream(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		stream SDKProtogo.DMSRpc_DMSCommunicateServer
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name: "testSetStream",
			fields: fields{
				state: created,
			},
			args: args{
				stream: &MockDMSRpc_DMSCommunicateServer{
					ctrl: gomock.NewController(t),
					recorder: &MockDMSRpc_DMSCommunicateServerMockRecorder{
						mock: NewMockDMSRpc_DMSCommunicateServer(gomock.NewController(t)),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}

			h.SetStream(tt.args.stream)
		})
	}
}

func TestProcessHandler_handleCallContract(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		callContractMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.handleCallContract(tt.args.callContractMsg); (err != nil) != tt.wantErr {
				t.Errorf("handleCallContract() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_handleCompleted(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		completedMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.handleCompleted(tt.args.completedMsg); (err != nil) != tt.wantErr {
				t.Errorf("handleCompleted() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_handleCreated(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		registerMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testHandleCreated",
			fields: fields{
				state:         created,
				logger:        utils.GetLogHandler(),
				TxRequest:     nil,
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
			args: args{
				registerMsg: &SDKProtogo.DMSMessage{
					TxId:          txId,
					Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_REGISTER,
					CurrentHeight: 0,
					Payload:       nil,
				},
			},
			wantErr: false,
		},

		{
			name: "testHandleCreated",
			fields: fields{
				state:         created,
				logger:        utils.GetLogHandler(),
				TxRequest:     nil,
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
			args: args{
				registerMsg: &SDKProtogo.DMSMessage{
					TxId:          txId,
					Type:          SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_UNDEFINED,
					CurrentHeight: 0,
					Payload:       nil,
				},
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.handleCreated(tt.args.registerMsg); (err != nil) != tt.wantErr {
				t.Errorf("handleCreated() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_handleGetState(t *testing.T) {
	responseCh := make(chan *protogo.CDMMessage, ReqChanSize*8)
	go func() {
		for {
			responseCh <- &protogo.CDMMessage{
				TxId:    txId,
				Type:    protogo.CDMType_CDM_TYPE_UNDEFINED,
				Payload: []byte(payload),
			}
		}
	}()

	c := gomock.NewController(t)
	defer c.Finish()
	messages := make(chan *protogo.TxResponse, 10)
	messages <- &protogo.TxResponse{
		TxId: txId,
	}

	scheduler := interfaces.NewMockScheduler(c)
	scheduler.EXPECT().RegisterResponseCh(txId, gomock.Any()).Return().AnyTimes()
	scheduler.EXPECT().GetGetStateReqCh().Return(responseCh).AnyTimes()
	//scheduler.EXPECT().GetResponseChByTxId(txId).Return(messages).AnyTimes()
	scheduler.EXPECT().GetTxResponseCh().Return(messages).AnyTimes()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		getStateMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		//{
		//	name: "testHandleGetState",
		//	fields: fields{
		//		state:  created,
		//		logger: utils.GetLogHandler(),
		//		TxRequest: &protogo.TxRequest{
		//			TxId:            txId,
		//			ContractName:    contractName,
		//			ContractVersion: contractVersion,
		//			Method:          "method1",
		//		},
		//		stream:        server,
		//		scheduler:     scheduler,
		//		process:       nil,
		//		txExpireTimer: nil,
		//	},
		//
		//	args: args{
		//		getStateMsg: &SDKProtogo.DMSMessage{
		//			TxId:    txId,
		//			Type:    SDKProtogo.DMSMessageType_DMS_MESSAGE_TYPE_GET_STATE_RESPONSE,
		//			Payload: []byte(payload),
		//		},
		//	},
		//},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}

			go func() {
				getResponseCh := h.scheduler.GetTxResponseCh()
				for {
					getResponseCh <- &protogo.TxResponse{
						TxId: txId,
					}
				}
			}()

			if err := h.handleGetState(tt.args.getStateMsg); (err != nil) != tt.wantErr {
				t.Errorf("handleGetState() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_handlePrepare(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		readyMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.handlePrepare(tt.args.readyMsg); (err != nil) != tt.wantErr {
				t.Errorf("handlePrepare() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_handleReady(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		readyMsg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.handleReady(tt.args.readyMsg); (err != nil) != tt.wantErr {
				t.Errorf("handleReady() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_resetHandler(t *testing.T) {
	type fields struct {
		state state
	}
	tests := []struct {
		name   string
		fields fields
	}{
		{
			name: "testResetHandler",
			fields: fields{
				state: "state1",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state: tt.fields.state,
			}
			h.resetState()
		})
	}
}

func TestProcessHandler_sendInit(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()

	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "testSendInit",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          method,
					Parameters:      map[string][]byte{},
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.sendInit(); (err != nil) != tt.wantErr {
				t.Errorf("sendInit() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_sendInvoke(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}

	tests := []struct {
		name    string
		fields  fields
		wantErr bool
	}{
		{
			name: "testSendInvoke",
			fields: fields{
				state:  "",
				logger: utils.GetLogHandler(),
				TxRequest: &protogo.TxRequest{
					TxId:            txId,
					ContractName:    contractName,
					ContractVersion: contractVersion,
					Method:          method,
					Parameters:      map[string][]byte{},
					TxContext: &protogo.TxContext{
						CurrentHeight: 10,
					},
				},
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.sendInvoke(); (err != nil) != tt.wantErr {
				t.Errorf("sendInvoke() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_sendMessage(t *testing.T) {
	c := gomock.NewController(t)
	defer c.Finish()

	server := NewMockDMSRpc_DMSCommunicateServer(c)
	server.EXPECT().Send(gomock.Any()).Return(nil).AnyTimes()
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	type args struct {
		msg *SDKProtogo.DMSMessage
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "testSendMessage",
			fields: fields{
				state:         created,
				logger:        utils.GetLogHandler(),
				TxRequest:     nil,
				stream:        server,
				scheduler:     nil,
				process:       nil,
				txExpireTimer: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			if err := h.sendMessage(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("sendMessage() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessHandler_startTimer(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			h.startTimer()
		})
	}
}

func TestProcessHandler_stopTimer(t *testing.T) {
	type fields struct {
		state         state
		logger        *zap.SugaredLogger
		TxRequest     *protogo.TxRequest
		stream    SDKProtogo.DMSRpc_DMSCommunicateServer
		scheduler interfaces.Scheduler
		process   ProcessInterface
		txExpireTimer *time.Timer
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &ProcessHandler{
				state:         tt.fields.state,
				logger:        tt.fields.logger,
				TxRequest:     tt.fields.TxRequest,
				stream:        tt.fields.stream,
				scheduler:     tt.fields.scheduler,
				process:       tt.fields.process,
				txExpireTimer: tt.fields.txExpireTimer,
			}
			h.stopTimer()
		})
	}
}

func Test_constructCallContractErrorResponse(t *testing.T) {
	type args struct {
		errMsg        string
		txId          string
		currentHeight uint32
	}
	tests := []struct {
		name string
		args args
		want *SDKProtogo.DMSMessage
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := constructCallContractErrorResponse(tt.args.errMsg, tt.args.txId, tt.args.currentHeight); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("constructCallContractErrorResponse() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_crossContractChKey(t *testing.T) {
	type args struct {
		txId          string
		currentHeight uint32
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := crossContractChKey(tt.args.txId, tt.args.currentHeight); got != tt.want {
				t.Errorf("crossContractChKey() = %v, want %v", got, tt.want)
			}
		})
	}
}

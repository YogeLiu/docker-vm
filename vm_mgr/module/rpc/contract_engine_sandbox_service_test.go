package rpc

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/mocks"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
	"fmt"
	"go.uber.org/zap"
	"reflect"
	"testing"
)

func TestNewSandboxRPCService(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()

	type args struct {
		origProcessMgr  interfaces.ProcessManager
		crossProcessMgr interfaces.ProcessManager
	}
	tests := []struct {
		name string
		args args
		want *SandboxRPCService
	}{
		{
			name: "TestNewSandboxRPCService",
			args: args{
				origProcessMgr:  nil,
				crossProcessMgr: nil,
			},
			want: &SandboxRPCService{
				logger:          log,
				origProcessMgr:  nil,
				crossProcessMgr: nil,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewSandboxRPCService(tt.args.origProcessMgr, tt.args.crossProcessMgr)
			got.logger = log
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewSandboxRPCService() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSandboxRPCService_DockerVMCommunicate(t *testing.T) {

	SetConfig()

	log := logger.NewTestDockerLogger()
	s := newMockStream(t)
	defer s.finish()

	stream := s.getStream()
	stream.EXPECT().Send(&protogo.DockerVMMessage{}).Return(nil).AnyTimes()

	origProcessMgr := &mocks.MockProcessManager{}
	crossProcessMgr := &mocks.MockProcessManager{}

	type fields struct {
		logger          *zap.SugaredLogger
		origProcessMgr  interfaces.ProcessManager
		crossProcessMgr interfaces.ProcessManager
	}
	type args struct {
		stream  protogo.DockerVMRpc_DockerVMCommunicateServer
		payload *protogo.DockerVMMessage
		err     error
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name: "TestSandboxRPCService_DockerVMCommunicate",
			fields: fields{
				logger:          log,
				origProcessMgr:  origProcessMgr,
				crossProcessMgr: crossProcessMgr,
			},
			args: args{
				stream:  stream,
				payload: nil,
				err:     fmt.Errorf("test error"),
			},
			wantErr: true,
		},
		{
			name: "TestSandboxRPCService_DockerVMCommunicate",
			fields: fields{
				logger:          log,
				origProcessMgr:  origProcessMgr,
				crossProcessMgr: crossProcessMgr,
			},
			args: args{
				stream:  stream,
				payload: nil,
				err:     nil,
			},
			wantErr: true,
		},
		{
			name: "TestSandboxRPCService_DockerVMCommunicate",
			fields: fields{
				logger:          log,
				origProcessMgr:  origProcessMgr,
				crossProcessMgr: crossProcessMgr,
			},
			args: args{
				stream: stream,
				payload: &protogo.DockerVMMessage{
					CrossContext: &protogo.CrossContext{
						ProcessName: "wrong",
					},
				},
				err: nil,
			},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &SandboxRPCService{
				logger:          tt.fields.logger,
				origProcessMgr:  tt.fields.origProcessMgr,
				crossProcessMgr: tt.fields.crossProcessMgr,
			}
			stream.Send(&protogo.DockerVMMessage{})
			stream.EXPECT().Recv().Return(tt.args.payload, tt.args.err).Times(1)

			if err := s.DockerVMCommunicate(tt.args.stream); (err != nil) != tt.wantErr {
				t.Errorf("DockerVMCommunicate() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

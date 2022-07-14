package rpc

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"testing"
)

const (
	testSockDir = "../../../test/sandbox/"
)

//func TestCreateUnixListener(t *testing.T) {
//	type args struct {
//		listenAddress *net.UnixAddr
//		sockPath      string
//	}
//	tests := []struct {
//		name    string
//		args    args
//		want    *net.UnixListener
//		wantErr bool
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			got, err := CreateUnixListener(tt.args.listenAddress, tt.args.sockPath)
//			if (err != nil) != tt.wantErr {
//				t.Errorf("CreateUnixListener() error = %v, wantErr %v", err, tt.wantErr)
//				return
//			}
//			if !reflect.DeepEqual(got, tt.want) {
//				t.Errorf("CreateUnixListener() got = %v, want %v", got, tt.want)
//			}
//		})
//	}
//}

func TestNewSandboxRPCServer(t *testing.T) {

	SetConfig()

	type args struct {
		sockDir string
	}
	tests := []struct {
		name    string
		args    args
		wantErr bool
	}{
		{
			name:    "TestNewSandboxRPCServer",
			args:    args{sockDir: testSockDir},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewSandboxRPCServer(tt.args.sockDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("NewSandboxRPCServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
		})
	}
}

func TestSandboxRPCServer_StartSandboxRPCServer(t *testing.T) {

	SetConfig()

	sandboxRPCServer, _ := NewSandboxRPCServer(testSockDir)
	sandboxRPCServer.logger = logger.NewTestDockerLogger()

	type fields struct {
		sandboxRPCServer *SandboxRPCServer
	}

	type args struct {
		service *SandboxRPCService
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestSandboxRPCServer_StartSandboxRPCServer",
			fields:  fields{sandboxRPCServer: sandboxRPCServer},
			args:    args{service: &SandboxRPCService{}},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := tt.fields.sandboxRPCServer
			if err := s.StartSandboxRPCServer(tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("StartSandboxRPCServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

//
//func TestSandboxRPCServer_StopSandboxRPCServer(t *testing.T) {
//	type fields struct {
//		Listener net.Listener
//		Server   *grpc.Server
//		logger   *zap.SugaredLogger
//	}
//	tests := []struct {
//		name   string
//		fields fields
//	}{
//		// TODO: Add test cases.
//	}
//	for _, tt := range tests {
//		t.Run(tt.name, func(t *testing.T) {
//			s := &SandboxRPCServer{
//				Listener: tt.fields.Listener,
//				Server:   tt.fields.Server,
//				logger:   tt.fields.logger,
//			}
//			s.StopSandboxRPCServer()
//		})
//	}
//}

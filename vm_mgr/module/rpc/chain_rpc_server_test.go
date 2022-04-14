package rpc

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
	"net"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"testing"
	"time"
)

const (
	configFileName = "vm"
	dockerMountDir = "test"
)

func TestChainRPCServer_StartChainRPCServer(t *testing.T) {
	type fields struct {
		Listener net.Listener
		Server   *grpc.Server
		logger   *zap.SugaredLogger
	}
	type args struct {
		service *ChainRPCService
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
			s := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}
			if err := s.StartChainRPCServer(tt.args.service); (err != nil) != tt.wantErr {
				t.Errorf("StartChainRPCServer() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestChainRPCServer_StopChainRPCServer(t *testing.T) {
	type fields struct {
		Listener net.Listener
		Server   *grpc.Server
		logger   *zap.SugaredLogger
	}
	tests := []struct {
		name   string
		fields fields
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &ChainRPCServer{
				Listener: tt.fields.Listener,
				Server:   tt.fields.Server,
				logger:   tt.fields.logger,
			}
		})
	}
}

func TestNewChainRPCServer(t *testing.T) {
	config.InitConfig()
	tests := []struct {
		name    string
		want    *ChainRPCServer
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := NewChainRPCServer()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewChainRPCServer() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewChainRPCServer() got = %v, want %v", got, tt.want)
			}
		})
	}
}

// newChainRPCServer build new chainmaker to docker manager rpc server
func newChainRPCServer(defaultPort string) (*ChainRPCServer, error) {
	enableUnixDomainSocket, _ := strconv.ParseBool(os.Getenv("UdsOpen"))

	var listener net.Listener
	var err error

	if !enableUnixDomainSocket {
		port := os.Getenv("Port")

		if port == "" {
			port = defaultPort
		}

		listener, err = net.Listen("tcp", ":"+port)
		if err != nil {
			return nil, err
		}
	} else {

		absCdmUDSPath := filepath.Join(config.SockBaseDir, config.SockName)

		listenAddress, err := net.ResolveUnixAddr("unix", absCdmUDSPath)
		if err != nil {
			return nil, err
		}

		listener, err = CreateUnixListener(listenAddress, absCdmUDSPath)
		if err != nil {
			return nil, err
		}

	}

	//set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	//set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime: ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	//set default connection timeout
	maxSendSizeConfig := os.Getenv("MaxSendMessageSize")
	maxRecvSizeConfig := os.Getenv("MaxRecvMessageSize")

	maxSendSize, _ := strconv.Atoi(maxSendSizeConfig)
	maxRecvSize, _ := strconv.Atoi(maxRecvSizeConfig)

	serverOpts = append(serverOpts, grpc.ConnectionTimeout(ConnectionTimeout))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(maxSendSize*1024*1024))
	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(maxRecvSize*1024*1024))

	server := grpc.NewServer(serverOpts...)

	return &CDMServer{
		Listener: listener,
		Server:   server,
		logger:   utils.GetLogHandler(),
	}, nil
}

package rpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

type RuntimeServer struct {
	Listener  net.Listener
	rpcServer *grpc.Server
	config    *config.DockerVMConfig
	log       *logger.CMLogger
}

func NewRuntimeServer(chainId string, vmConfig *config.DockerVMConfig) (*RuntimeServer, error) {
	if vmConfig == nil {
		return nil, errors.New("invalid parameter, config is nil")
	}

	listener, err := createListener(chainId, vmConfig)
	if err != nil {
		return nil, err
	}

	// set up server options for keepalive and TLS
	var serverOpts []grpc.ServerOption

	// add keepalive
	serverKeepAliveParameters := keepalive.ServerParameters{
		Time:    1 * time.Minute,
		Timeout: 20 * time.Second,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(serverKeepAliveParameters))

	// set enforcement policy
	kep := keepalive.EnforcementPolicy{
		MinTime: config.ServerMinInterval,
		// allow keepalive w/o rpc
		PermitWithoutStream: true,
	}

	serverOpts = append(serverOpts, grpc.KeepaliveEnforcementPolicy(kep))

	serverOpts = append(serverOpts, grpc.ConnectionTimeout(config.ConnectionTimeout))

	server := grpc.NewServer(serverOpts...)

	return &RuntimeServer{
		Listener:  listener,
		rpcServer: server,
		log:       logger.GetLoggerByChain("[Runtime Server]", chainId),
		config:    vmConfig,
	}, nil
}

func (s *RuntimeServer) StartRuntimeServer(runtimeService *RuntimeService) error {
	if s.Listener == nil {
		return errors.New("nil listener")
	}

	if s.rpcServer == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.rpcServer, runtimeService)

	s.log.Debug("start runtime server")
	go func() {
		err := s.rpcServer.Serve(s.Listener)
		if err != nil {
			s.log.Errorf("runtime server fail to start: %s", err)
		}
	}()

	return nil
}

func (s *RuntimeServer) StopRuntimeServer() {
	s.log.Info("stop runtime server")

	if s.rpcServer != nil {
		s.rpcServer.Stop()
	}
}

func createListener(chainId string, vmConfig *config.DockerVMConfig) (net.Listener, error) {
	if vmConfig.DockerVMUDSOpen {
		runtimeServerSockPath := filepath.Join(vmConfig.DockerVMMountPath, chainId)
		return createUnixListener(runtimeServerSockPath)
	}

	// TODO: TestPort 2 config
	return createTCPListener(config.TestPort)
}

func createUnixListener(sockPath string) (*net.UnixListener, error) {
	listenAddress, err := net.ResolveUnixAddr("unix", sockPath)
	if err != nil {
		return nil, fmt.Errorf("Failed to listen: %v", err)
	}

start:
	listener, err := net.ListenUnix("unix", listenAddress)
	if err != nil {
		err = os.Remove(sockPath)
		if err != nil {
			return nil, err
		}
		goto start
	}

	if err = os.Chmod(sockPath, 0777); err != nil {
		return nil, err
	}

	return listener, nil
}

func createTCPListener(port string) (*net.TCPListener, error) {
	listenAddress, err := net.ResolveTCPAddr("tcp", port)
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", listenAddress)
}

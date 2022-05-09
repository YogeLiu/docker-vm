package rpc

import (
	"errors"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/pb/protogo"
	"google.golang.org/grpc"
	"google.golang.org/grpc/keepalive"
)

var runtimeServerOnce sync.Once

type RuntimeServer struct {
	listener  net.Listener
	rpcServer *grpc.Server
	config    *config.DockerVMConfig
	logger    protocol.Logger
}

func NewRuntimeServer(chainId string, vmConfig *config.DockerVMConfig) (*RuntimeServer, error) {
	errCh := make(chan error, 1)
	instanceCh := make(chan *RuntimeServer, 1)

	runtimeServerOnce.Do(func() {
		if vmConfig == nil {
			errCh <- errors.New("invalid parameter, config is nil")
			return
		}

		listener, err := createListener(chainId, vmConfig)
		if err != nil {
			errCh <- err
			return
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

		instanceCh <- &RuntimeServer{
			listener:  listener,
			rpcServer: server,
			logger:    logger.GetLoggerByChain("[Runtime Server]", chainId),
			config:    vmConfig,
		}
	})

	select {
	case err := <-errCh:
		return nil, err
	case instance := <-instanceCh:
		return instance, nil
	}
}

func (s *RuntimeServer) StartRuntimeServer(runtimeService *RuntimeService) error {
	if s.listener == nil {
		return errors.New("nil listener")
	}

	if s.rpcServer == nil {
		return errors.New("nil server")
	}

	protogo.RegisterDockerVMRpcServer(s.rpcServer, runtimeService)

	s.logger.Debug("start runtime server")
	go func() {
		err := s.rpcServer.Serve(s.listener)
		if err != nil {
			s.logger.Errorf("runtime server fail to start: %s", err)
		}
	}()

	return nil
}

func (s *RuntimeServer) StopRuntimeServer() {
	s.logger.Info("stop runtime server")

	if s.rpcServer != nil {
		s.rpcServer.Stop()
	}
}

func createListener(chainId string, vmConfig *config.DockerVMConfig) (net.Listener, error) {
	if vmConfig.DockerVMUDSOpen {
		runtimeServerSockPath := filepath.Join(vmConfig.DockerVMMountPath, chainId)
		return createUnixListener(runtimeServerSockPath)
	}

	// TODO: TLS
	return createTCPListener(strconv.Itoa(vmConfig.RuntimeServer.Port))
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
	listenAddress, err := net.ResolveTCPAddr("tcp", ":"+port)
	if err != nil {
		return nil, err
	}

	return net.ListenTCP("tcp", listenAddress)
}

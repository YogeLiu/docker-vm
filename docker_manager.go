/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/mitchellh/mapstructure"

	"chainmaker.org/chainmaker/vm-docker-go/v2/rpc"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	//"github.com/docker/docker/client"
)

const (
	dockerMountDir     = "/mount"
	dockerContainerDir = "../module/vm/docker-go/vm_mgr"
)

type DockerManager struct {
	chainId   string
	mgrLogger *logger.CMLogger

	ctx context.Context

	cdmClient      *rpc.CDMClient // grpc client
	clientInitOnce sync.Once

	dockerVMConfig        *config.DockerVMConfig        // original config from local config
	dockerContainerConfig *config.DockerContainerConfig // container setting
}

// NewDockerManager return docker manager and running a default container
func NewDockerManager(chainId string, vmConfig map[string]interface{}) *DockerManager {

	dockerVMConfig := &config.DockerVMConfig{}
	_ = mapstructure.Decode(vmConfig, dockerVMConfig)

	// if enable docker vm is false, docker manager is nil
	startDockerVm := dockerVMConfig.EnableDockerVM
	if !startDockerVm {
		return nil
	}

	// init docker manager logger
	dockerManagerLogger := logger.GetLoggerByChain("[Docker Manager]", chainId)
	dockerManagerLogger.Debugf("init docker manager")

	// validate and init settings
	dockerContainerConfig := newDockerContainerConfig()
	err := validateVMSettings(dockerVMConfig, dockerContainerConfig, chainId)
	if err != nil {
		dockerManagerLogger.Errorf("fail to init docker manager, please check the docker config, %s", err)
		return nil
	}

	// init docker manager
	newDockerManager := &DockerManager{
		chainId:               chainId,
		mgrLogger:             dockerManagerLogger,
		ctx:                   context.Background(),
		cdmClient:             rpc.NewCDMClient(chainId, dockerVMConfig),
		clientInitOnce:        sync.Once{},
		dockerVMConfig:        dockerVMConfig,
		dockerContainerConfig: dockerContainerConfig,
	}

	// init mount directory and subdirectory
	err = newDockerManager.initMountDirectory()
	if err != nil {
		dockerManagerLogger.Errorf("fail to init mount directory: %s", err)
		return nil
	}

	return newDockerManager
}

// StartVM Start Docker VM
func (m *DockerManager) StartVM() error {
	if m == nil {
		return nil
	}
	m.mgrLogger.Info("start docker vm...")

	// todo verify vm contract service info

	return nil
}

// StopVM stop docker vm and remove container, image
func (m *DockerManager) StopVM() error {
	if m == nil {
		return nil
	}

	m.clientInitOnce = sync.Once{}

	return nil
}

func (m *DockerManager) NewRuntimeInstance(txSimContext protocol.TxSimContext, chainId, method,
	codePath string, contract *common.Contract,
	byteCode []byte, logger protocol.Logger) (protocol.RuntimeInstance, error) {

	// add client state logic
	if !m.getCDMState() {
		m.startCDMClient()
	}

	return &RuntimeInstance{
		ChainId: chainId,
		Client:  m.cdmClient,
		Log:     logger,
	}, nil
}

// StartCDMClient start CDM grpc rpc
func (m *DockerManager) startCDMClient() {

	for !m.getCDMState() {
		m.mgrLogger.Debugf("cdm rpc state is: %v, try reconnecting...", m.getCDMState())
		m.cdmClient.ConnStatus = m.cdmClient.StartClient()
		time.Sleep(2 * time.Second)
	}

	// add reconnect when send or receive msg failed
	go func() {
		for {
			select {
			case <-m.cdmClient.ReconnectChan:
				m.cdmClient.ConnStatus = false
				//	reconnect
				m.cdmClient.ReconnectChan = make(chan bool)
				for !m.getCDMState() {
					m.mgrLogger.Debugf("cdm rpc state is: %v, try reconnecting...", m.getCDMState())
					m.cdmClient.ConnStatus = m.cdmClient.StartClient()
					time.Sleep(2 * time.Second)
				}
			}
		}
	}()

}

func (m *DockerManager) getCDMState() bool {
	return m.cdmClient.ConnStatus
}

// InitMountDirectory init mount directory and subdirectories
func (m *DockerManager) initMountDirectory() error {

	var err error

	// create mount directory
	mountDir := m.dockerContainerConfig.HostMountDir
	err = m.createDir(mountDir)
	if err != nil {
		return nil
	}
	m.mgrLogger.Debug("set mount dir: ", mountDir)

	// create sub directory: contracts, share, sock
	contractDir := filepath.Join(mountDir, config.ContractsDir)
	err = m.createDir(contractDir)
	if err != nil {
		m.mgrLogger.Errorf("fail to build image, err: [%s]", err)
		return err
	}
	m.mgrLogger.Debug("set contract dir: ", contractDir)

	sockDir := filepath.Join(mountDir, config.SockDir)
	err = m.createDir(sockDir)
	if err != nil {
		return err
	}
	m.mgrLogger.Debug("set sock dir: ", sockDir)

	return nil

}

// ------------------ utility functions --------------

func (m *DockerManager) createDir(directory string) error {
	exist, err := m.exists(directory)
	if err != nil {
		m.mgrLogger.Errorf("fail to get container, err: [%s]", err)
		return err
	}

	if !exist {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			m.mgrLogger.Errorf("fail to remove image, err: [%s]", err)
			return err
		}
	}

	return nil
}

// exists returns whether the given file or directory exists
func (m *DockerManager) exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func validateVMSettings(config *config.DockerVMConfig,
	dockerContainerConfig *config.DockerContainerConfig, chainId string) error {

	var hostMountDir string
	if len(config.DockerVMMountPath) == 0 {
		return errors.New("doesn't set host mount directory path correctly")
	}

	// set host mount directory path
	if !filepath.IsAbs(config.DockerVMMountPath) {
		hostMountDir, _ = filepath.Abs(config.DockerVMMountPath)
		hostMountDir = filepath.Join(hostMountDir, chainId)
	} else {
		hostMountDir = filepath.Join(config.DockerVMMountPath, chainId)
	}

	dockerContainerConfig.HostMountDir = hostMountDir

	return nil
}

func newDockerContainerConfig() *config.DockerContainerConfig {

	containerConfig := &config.DockerContainerConfig{
		AttachStdOut: true,
		AttachStderr: true,
		ShowStdout:   true,
		ShowStderr:   true,

		VMMgrDir: dockerContainerDir,

		HostMountDir: "",
	}

	return containerConfig
}

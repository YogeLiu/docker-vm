/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-engine/v2/config"
	"chainmaker.org/chainmaker/vm-engine/v2/interfaces"
	"chainmaker.org/chainmaker/vm-engine/v2/pb/protogo"
	"chainmaker.org/chainmaker/vm-engine/v2/rpc"
	"github.com/mitchellh/mapstructure"
)

// InstancesManager manager all sandbox instances
type InstancesManager struct {
	chainId               string
	mgrLogger             protocol.Logger
	clientMgr             interfaces.ContractEngineClientMgr // grpc client
	runtimeService        *rpc.RuntimeService                //
	runtimeServer         *rpc.RuntimeServer                 // grpc server
	dockerVMConfig        *config.DockerVMConfig             // original config from local config
	dockerContainerConfig *config.DockerContainerConfig      // container setting
}

// NewInstancesManager return docker vm instance manager
func NewInstancesManager(chainId string, logger protocol.Logger, vmConfig map[string]interface{}) *InstancesManager {

	dockerVMConfig := &config.DockerVMConfig{}
	if err := mapstructure.Decode(vmConfig, dockerVMConfig); err != nil {
		logger.Warnf("failed to decode vm config")
	}

	// if enable docker vm is false, docker manager is nil
	startDockerVm := dockerVMConfig.EnableDockerVM
	if !startDockerVm {
		return nil
	}

	// validate and init settings
	dockerContainerConfig := newDockerContainerConfig()
	err := validateVMSettings(dockerVMConfig, dockerContainerConfig, chainId)
	if err != nil {
		logger.Errorf("fail to init docker manager, please check the docker config, %s", err)
		return nil
	}

	// init docker manager
	newDockerManager := &InstancesManager{
		chainId:               chainId,
		mgrLogger:             logger,
		clientMgr:             rpc.NewClientManager(chainId, logger, dockerVMConfig),
		dockerVMConfig:        dockerVMConfig,
		dockerContainerConfig: dockerContainerConfig,
	}

	// init mount directory and subdirectory
	err = newDockerManager.initMountDirectory()
	if err != nil {
		logger.Errorf("fail to init mount directory: %s", err)
		return nil
	}

	// runtime server
	server, err := rpc.NewRuntimeServer(chainId, logger, dockerVMConfig)
	if err != nil {
		logger.Errorf("fail to init docker manager, %s", err)
		return nil
	}
	newDockerManager.runtimeServer = server
	newDockerManager.runtimeService = rpc.NewRuntimeService(chainId, logger)

	return newDockerManager
}

// NewRuntimeInstance returns new runtime instance
func (m *InstancesManager) NewRuntimeInstance(txSimContext protocol.TxSimContext, chainId, method,
	codePath string, contract *commonPb.Contract,
	byteCode []byte, logger protocol.Logger) (protocol.RuntimeInstance, error) {

	return &RuntimeInstance{
		chainId:             chainId,
		clientMgr:           m.clientMgr,
		runtimeService:      m.runtimeService,
		logger:              logger,
		event:               make([]*commonPb.ContractEvent, 0),
		sandboxMsgCh:        make(chan *protogo.DockerVMMessage, 1),
		contractEngineMsgCh: make(chan *protogo.DockerVMMessage, 1),
	}, nil
}

// StartVM Start Docker VM
// TODO: 多链时不共用同一个实例，chainId的作用
func (m *InstancesManager) StartVM() error {
	if m == nil {
		return nil
	}
	if !m.dockerVMConfig.EnableDockerVM {
		m.mgrLogger.Infof("vm engine disabled, no need to start")
		return nil
	}
	m.mgrLogger.Info("start docker vm...")
	var err error

	// start runtime server
	if err = m.runtimeServer.StartRuntimeServer(m.runtimeService); err != nil {
		return err
	}

	m.mgrLogger.Debugf("chain[%s] docker vm start success :)", m.chainId)

	// start ContractEngine RPC client
	err = m.clientMgr.Start()
	if err != nil {
		return err
	}

	return nil
}

// StopVM stop docker
func (m *InstancesManager) StopVM() error {
	if m == nil {
		return nil
	}

	err := m.clientMgr.Stop()
	if err != nil {
		return err
	}

	m.runtimeServer.StopRuntimeServer()

	m.mgrLogger.Info("stop chain [%s] docker vm", m.chainId)
	return nil
}

// InitMountDirectory init mount directory and subdirectories
func (m *InstancesManager) initMountDirectory() error {

	// create mount directory
	mountDir := m.dockerContainerConfig.HostMountDir
	err := m.createDir(mountDir)
	if err != nil {
		return nil
	}
	m.mgrLogger.Debug("set mount dir: ", mountDir)

	// create subDirectory: contracts
	contractDir := filepath.Join(mountDir, config.ContractsDir)
	err = m.createDir(contractDir)
	if err != nil {
		m.mgrLogger.Errorf("fail to build image, err: [%s]", err)
		return err
	}
	m.mgrLogger.Debug("set contract dir: ", contractDir)

	// create RuntimeServer sock directory
	sockDir := filepath.Join(mountDir, config.SockDir)
	err = m.createDir(sockDir)
	if err != nil {
		return err
	}
	m.mgrLogger.Debug("set sock dir: ", sockDir)

	// create config directory
	//configDir := filepath.Join(mountDir, config.DockerConfigDir)
	//err = m.createDir(configDir)
	//if err != nil {
	//	return err
	//}
	//m.mgrLogger.Debug("set config dir: ", configDir)
	//_, err = fileutils.CopyFile("../vm_mgr/config/vm.yml", filepath.Join(configDir, "vm.yml"))
	//if err != nil {
	//	return err
	//}

	// create log directory
	logDir := m.dockerContainerConfig.HostLogDir
	err = m.createDir(logDir)
	if err != nil {
		return nil
	}
	m.mgrLogger.Debug("set log dir: ", logDir)

	return nil
}

// ------------------ utility functions --------------

func (m *InstancesManager) createDir(directory string) error {
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
func (m *InstancesManager) exists(path string) (bool, error) {
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
	var hostLogDir string
	var err error
	if len(config.DockerVMMountPath) == 0 {
		return errors.New("doesn't set host mount directory path correctly")
	}

	if len(config.DockerVMLogPath) == 0 {
		return errors.New("doesn't set host log directory path correctly")
	}

	// set host mount directory path
	if !filepath.IsAbs(config.DockerVMMountPath) {
		hostMountDir, err = filepath.Abs(config.DockerVMMountPath)
		if err != nil {
			return fmt.Errorf("failed to abs DockerVMMountPath filepath, %s", config.DockerVMMountPath)
		}
		//hostMountDir = filepath.Join(hostMountDir, chainId)
	}
	//else {
	//	//hostMountDir = filepath.Join(config.DockerVMMountPath, chainId)
	//}

	// set host log directory
	if !filepath.IsAbs(config.DockerVMLogPath) {
		hostLogDir, err = filepath.Abs(config.DockerVMLogPath)
		if err != nil {
			return fmt.Errorf("failed to abs DockerVMLogPath filepath, %s", config.DockerVMLogPath)
		}
		//hostLogDir = filepath.Join(hostLogDir, chainId)
	}
	//else {
	//	//hostLogDir = filepath.Join(config.DockerVMLogPath, chainId)
	//}

	dockerContainerConfig.HostMountDir = hostMountDir
	dockerContainerConfig.HostLogDir = hostLogDir

	return nil
}

func newDockerContainerConfig() *config.DockerContainerConfig {

	containerConfig := &config.DockerContainerConfig{
		HostMountDir: "",
		HostLogDir:   "",
	}

	return containerConfig
}

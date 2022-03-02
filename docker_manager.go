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

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/mitchellh/mapstructure"

	"chainmaker.org/chainmaker/vm-docker-go/v2/rpc"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	//"github.com/docker/docker/client"
)

const (
	dockerMountDir       = "/mount"
	dockerLogDir         = "/log"
	dockerContainerDir   = "../module/vm/docker-go/vm_mgr"
	defaultContainerName = "chainmaker-vm-docker-go-container"
	imageVersion         = "v2.2.0_alpha_qc"

	enablePProf = false // switch for enable pprof, just for testing

)

type DockerManager struct {
	chainId   string
	mgrLogger *logger.CMLogger

	ctx context.Context

	cdmClient      *rpc.CDMClient // grpc client
	clientInitOnce sync.Once
	cdmState       bool

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
		cdmState:              false,
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

	// 以另外一种方式 判断容器服务有没有启动，并在日志 打印版本等信息

	// todo 校验信息：链id，等其他信息

	// check container is running or not
	// if running, stop it,

	// pprof 在容器里设置（也换成环境变量的方式）
	m.mgrLogger.Debugf("docker vm start success :)")

	// display container info in the console
	//go func() {
	//	err = m.displayInConsole(m.dockerContainerConfig.ContainerName)
	//	if err != nil {
	//		m.mgrLogger.Errorf("docker vm fail: %s", err)
	//		return
	//	}
	//}()
	return nil
}

// StopVM stop docker vm and remove container, image
func (m *DockerManager) StopVM() error {
	if m == nil {
		return nil
	}
	//var err error

	// todo 是否需要停掉 合约管理容器： 如何停掉，停掉是否要删掉容器，那是否需要开启任务处理
	//err := m.stopContainer()
	//if err != nil {
	//	return err
	//}

	//err = m.removeImage()
	//if err != nil {
	//	return err
	//}

	m.cdmState = false
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

	m.clientInitOnce.Do(func() {
		state := m.cdmClient.StartClient()
		m.cdmState = state
		m.mgrLogger.Debugf("cdm rpc state is: %v", state)
	})
}

func (m *DockerManager) getCDMState() bool {
	return m.cdmState
}

// InitMountDirectory init mount directory and subdirectories
// todo 创建路径的操作需不需要在容器里再做一遍
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

// create container with pprof feature
// which is open network and open an ip port in docker container
//func (m *DockerManager) createPProfContainer() error {
//	hostPort := config.PProfPort
//	openPort := nat.Port(hostPort + "/tcp")
//
//	sdkHostPort := config.SDKPort
//	sdkOpenPort := nat.Port(sdkHostPort + "/tcp")
//
//	envs := m.constructEnvs(true)
//	envs = append(envs, fmt.Sprintf("%s=%v", config.EnvPprofPort, config.PProfPort))
//	envs = append(envs, fmt.Sprintf("%s=%v", config.EnvEnablePprof, true))
//
//	_, err := m.dockerAPIClient.ContainerCreate(m.ctx, &container.Config{
//		Cmd:          nil,
//		Image:        m.dockerContainerConfig.ImageName,
//		Env:          envs,
//		AttachStdout: m.dockerContainerConfig.AttachStdOut,
//		AttachStderr: m.dockerContainerConfig.AttachStderr,
//		ExposedPorts: nat.PortSet{
//			openPort:    struct{}{},
//			sdkOpenPort: struct{}{},
//		},
//	}, &container.HostConfig{
//		Privileged: true,
//		Mounts: []mount.Mount{
//			{
//				Type:        mount.TypeBind,
//				Source:      m.dockerContainerConfig.HostMountDir,
//				Target:      m.dockerContainerConfig.DockerMountDir,
//				ReadOnly:    false,
//				Consistency: mount.ConsistencyFull,
//				BindOptions: &mount.BindOptions{
//					Propagation:  mount.PropagationRPrivate,
//					NonRecursive: false,
//				},
//				VolumeOptions: nil,
//				TmpfsOptions:  nil,
//			},
//			{
//				Type:        mount.TypeBind,
//				Source:      m.dockerContainerConfig.HostLogDir,
//				Target:      m.dockerContainerConfig.DockerLogDir,
//				ReadOnly:    false,
//				Consistency: mount.ConsistencyFull,
//				BindOptions: &mount.BindOptions{
//					Propagation:  mount.PropagationRPrivate,
//					NonRecursive: false,
//				},
//				VolumeOptions: nil,
//				TmpfsOptions:  nil,
//			},
//		},
//		PortBindings: nat.PortMap{
//			openPort: []nat.PortBinding{
//				{
//					HostIP:   "0.0.0.0",
//					HostPort: hostPort,
//				},
//			},
//			sdkOpenPort: []nat.PortBinding{
//				{
//					HostIP:   "0.0.0.0",
//					HostPort: sdkHostPort,
//				},
//			},
//		},
//	}, nil, nil, m.dockerContainerConfig.ContainerName)
//
//	if err != nil {
//		m.mgrLogger.Errorf("create container [%s] failed", m.dockerContainerConfig.ContainerName)
//		return err
//	}
//
//	m.mgrLogger.Infof("create container [%s] success :)", m.dockerContainerConfig.ContainerName)
//	return nil
//}

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

		DockerMountDir: dockerMountDir,
		HostMountDir:   "",
	}

	return containerConfig
}

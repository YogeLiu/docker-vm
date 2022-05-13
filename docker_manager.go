/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	commonPb "chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/rpc"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/fileutils"
	"github.com/docker/go-connections/nat"
	"github.com/mitchellh/mapstructure"
)

const (
	dockerMountDir       = "/mount"
	dockerLogDir         = "/log"
	dockerContainerDir   = "../module/vm/docker-go/vm_mgr"
	defaultContainerName = "chainmaker-vm-docker-go-container"
	imageVersion         = "refactor"
)

var (
	// TODO: 独立部署之后移除
	imageName = fmt.Sprintf("chainmakerofficial/chainmaker-vm-docker-go:%s", imageVersion)
)

// TODO: rename -> VMInstanceManager
type DockerManager struct {
	chainId   string
	mgrLogger protocol.Logger
	// TODO: 不需要docker manager拉起后可以移除
	ctx context.Context
	// TODO: contractEngine 容器手动启动，不需要docker manager拉起
	dockerAPIClient       *client.Client                     // docker client
	clientMgr             interfaces.ContractEngineClientMgr // grpc client
	runtimeService        *rpc.RuntimeService                //
	runtimeServer         *rpc.RuntimeServer                 // grpc server
	dockerVMConfig        *config.DockerVMConfig             // original config from local config
	dockerContainerConfig *config.DockerContainerConfig      // container setting
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
	// TODO: 移除logger库引用, 统一使用VM传递过来的logger
	dockerManagerLogger := logger.GetLoggerByChain("[VM]", chainId)
	dockerManagerLogger.Debugf("init docker manager")

	// validate and init settings
	dockerContainerConfig := newDockerContainerConfig()
	err := validateVMSettings(dockerVMConfig, dockerContainerConfig, chainId)
	if err != nil {
		dockerManagerLogger.Errorf("fail to init docker manager, please check the docker config, %s", err)
		return nil
	}

	// init docker api client
	dockerAPIClient, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	// init docker manager
	newDockerManager := &DockerManager{
		chainId:               chainId,
		mgrLogger:             dockerManagerLogger,
		ctx:                   context.Background(),
		dockerAPIClient:       dockerAPIClient,
		clientMgr:             rpc.NewClientManager(chainId, dockerVMConfig),
		dockerVMConfig:        dockerVMConfig,
		dockerContainerConfig: dockerContainerConfig,
	}

	// init mount directory and subdirectory
	err = newDockerManager.initMountDirectory()
	if err != nil {
		dockerManagerLogger.Errorf("fail to init mount directory: %s", err)
		return nil
	}

	// runtime server
	server, err := rpc.NewRuntimeServer(chainId, dockerVMConfig)
	if err != nil {
		dockerManagerLogger.Errorf("fail to init docker manager, %s", err)
		return nil
	}
	newDockerManager.runtimeServer = server
	newDockerManager.runtimeService = rpc.NewRuntimeService(chainId, logger.GetLoggerByChain("[Runtime Server]", chainId))

	return newDockerManager
}

// StartVM Start Docker VM
func (m *DockerManager) StartVM() error {
	if m == nil {
		return nil
	}
	m.mgrLogger.Info("start docker vm...")
	var err error

	// check container is running or not
	// if running, stop it,
	isRunning, err := m.getContainer(false)
	if err != nil {
		return err
	}
	if isRunning {
		m.mgrLogger.Debugf("stop running container [%s]", m.dockerContainerConfig.ContainerName)
		err = m.stopContainer()
		if err != nil {
			return err
		}
	}

	// check container exist or not, if not exist, create new container
	containerExist, err := m.getContainer(true)
	if err != nil {
		return err
	}

	if containerExist {
		m.mgrLogger.Debugf("remove container [%s]", m.dockerContainerConfig.ContainerName)
		err = m.removeContainer()
		if err != nil {
			return err
		}
	}

	// check image exist or not, if not exist, create new image
	imageExisted, err := m.imageExist()
	if err != nil {
		return err
	}

	if !imageExisted {
		m.mgrLogger.Errorf("cannot find docker vm image, please pull the image")
		return fmt.Errorf("cannot find docker vm image")
	}

	m.mgrLogger.Debugf("create container [%s]", m.dockerContainerConfig.ContainerName)

	if m.dockerVMConfig.DockerVMUDSOpen {
		if m.dockerVMConfig.EnablePprof {
			err = m.createPProfContainer()
		} else {
			err = m.createContainer()
		}
		if err != nil {
			return err
		}
	} else {
		err = m.createContainerWithTCP(strconv.Itoa(m.dockerVMConfig.ContractEngine.Port))
		if err != nil {
			return err
		}
	}

	// start runtime server
	if err = m.runtimeServer.StartRuntimeServer(m.runtimeService); err != nil {
		return err
	}

	// running container
	m.mgrLogger.Infof("start running container [%s]", m.dockerContainerConfig.ContainerName)
	if err = m.dockerAPIClient.ContainerStart(m.ctx, m.dockerContainerConfig.ContainerName,
		types.ContainerStartOptions{}); err != nil {
		return err
	}

	m.mgrLogger.Debugf("docker vm start success :)")

	// display container info in the console
	go func() {
		err = m.displayInConsole(m.dockerContainerConfig.ContainerName)
		if err != nil {
			m.mgrLogger.Errorf("docker vm fail: %s", err)
			return
		}
	}()

	if err := m.clientMgr.Start(); err != nil {
		return err
	}

	return nil
}

func (m *DockerManager) NewRuntimeInstance(txSimContext protocol.TxSimContext, chainId, method,
	codePath string, contract *common.Contract,
	byteCode []byte, logger protocol.Logger) (protocol.RuntimeInstance, error) {

	return &RuntimeInstance{
		chainId:        chainId,
		clientMgr:      m.clientMgr,
		runtimeService: m.runtimeService,
		logger:         logger,
		event:          make([]*commonPb.ContractEvent, 0),
	}, nil
}

// StopVM stop docker vm and remove container, image
func (m *DockerManager) StopVM() error {
	if m == nil {
		return nil
	}
	var err error

	err = m.clientMgr.Stop()
	if err != nil {
		return err
	}

	err = m.stopContainer()
	if err != nil {
		return err
	}

	err = m.removeContainer()
	if err != nil {
		return err
	}

	//err = m.removeImage()
	//if err != nil {
	//	return err
	//}

	m.runtimeServer.StopRuntimeServer()

	m.mgrLogger.Info("stop and remove docker vm [%s]", m.dockerContainerConfig.ContainerName)
	return nil
}

// ------------------ image functions --------------

// imageExist check docker m image exist or not
func (m *DockerManager) imageExist() (bool, error) {
	imageList, err := m.dockerAPIClient.ImageList(m.ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false, err
	}

	for _, v1 := range imageList {
		for _, v2 := range v1.RepoTags {
			if v2 == m.dockerContainerConfig.ImageName {
				return true, nil
			}
		}
	}
	return false, nil
}

// ------------------ container functions --------------

// createContainer create container based on image
func (m *DockerManager) createContainer() error {

	_, err := m.dockerAPIClient.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        m.dockerContainerConfig.ImageName,
		AttachStdout: m.dockerContainerConfig.AttachStdOut,
		AttachStderr: m.dockerContainerConfig.AttachStderr,
	}, &container.HostConfig{
		NetworkMode: "none",
		Privileged:  true,
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostMountDir,
				Target:      m.dockerContainerConfig.DockerMountDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostLogDir,
				Target:      m.dockerContainerConfig.DockerLogDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
		},
	}, nil, nil, m.dockerContainerConfig.ContainerName)

	if err != nil {
		m.mgrLogger.Errorf("create container [%s] failed", m.dockerContainerConfig.ContainerName)
		return err
	}

	m.mgrLogger.Infof("create container [%s] success :)", m.dockerContainerConfig.ContainerName)
	return nil
}

// getContainer check container status: exist, not exist, running, or not running
func (m *DockerManager) getContainer(all bool) (bool, error) {
	containerList, err := m.dockerAPIClient.ContainerList(m.ctx, types.ContainerListOptions{All: all})
	if err != nil {
		return false, err
	}

	indexName := "/" + m.dockerContainerConfig.ContainerName
	for _, v1 := range containerList {
		for _, v2 := range v1.Names {
			if v2 == indexName {
				return true, nil
			}
		}
	}
	return false, nil
}

// removeContainer remove container
func (m *DockerManager) removeContainer() error {
	m.mgrLogger.Infof("Removing container [%s] ...", m.dockerContainerConfig.ContainerName)
	return m.dockerAPIClient.ContainerRemove(m.ctx, m.dockerContainerConfig.ContainerName, types.ContainerRemoveOptions{})
}

// stopContainer stop container
func (m *DockerManager) stopContainer() error {
	return m.dockerAPIClient.ContainerStop(m.ctx, m.dockerContainerConfig.ContainerName, nil)
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

	configDir := filepath.Join(mountDir, config.DockerConfigDir)
	err = m.createDir(configDir)
	if err != nil {
		return err
	}
	m.mgrLogger.Debug("set config dir: ", configDir)
	_, err = fileutils.CopyFile("../vm_mgr/config/vm.yml", filepath.Join(configDir, "vm.yml"))
	if err != nil {
		return err
	}

	// create log directory
	logDir := m.dockerContainerConfig.HostLogDir
	err = m.createDir(logDir)
	if err != nil {
		return nil
	}
	m.mgrLogger.Debug("set log dir: ", logDir)

	return nil

}

// displayInConsole display container std out in host std out -- need finish loop accept
func (m *DockerManager) displayInConsole(containerID string) error {
	//display container std out
	out, err := m.dockerAPIClient.ContainerLogs(m.ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: m.dockerContainerConfig.ShowStdout,
		ShowStderr: m.dockerContainerConfig.ShowStderr,
		Follow:     true,
		Timestamps: false,
	})
	if err != nil {
		return err
	}
	defer func(out io.ReadCloser) {
		err = out.Close()
		if err != nil {
			return
		}
	}(out)

	hdr := make([]byte, 8)
	for {
		_, err = out.Read(hdr)
		if err != nil {
			if err == io.EOF {
				break
			}
		}
		var w io.Writer
		switch hdr[0] {
		case 1:
			w = os.Stdout
		default:
			w = os.Stderr
		}
		count := binary.BigEndian.Uint32(hdr[4:])
		dat := make([]byte, count)
		_, _ = out.Read(dat)
		_, err = fmt.Fprint(w, string(dat))
		if err != nil {
			return err
		}
	}

	return nil
}

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

// create container based on image, just for testing
func (m *DockerManager) createContainerWithTCP(port string) error {

	openPort := nat.Port(port + "/tcp")

	_, err := m.dockerAPIClient.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        m.dockerContainerConfig.ImageName,
		AttachStdout: m.dockerContainerConfig.AttachStdOut,
		AttachStderr: m.dockerContainerConfig.AttachStderr,
		ExposedPorts: nat.PortSet{
			openPort: struct{}{},
		},
	}, &container.HostConfig{
		Privileged: true,
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostMountDir,
				Target:      m.dockerContainerConfig.DockerMountDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostLogDir,
				Target:      m.dockerContainerConfig.DockerLogDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
		},
		PortBindings: nat.PortMap{
			openPort: []nat.PortBinding{
				{
					HostIP:   "127.0.0.1",
					HostPort: port,
				},
			},
		},
	}, nil, nil, m.dockerContainerConfig.ContainerName)

	if err != nil {
		m.mgrLogger.Errorf("create container [%s] failed", m.dockerContainerConfig.ContainerName)
		return err
	}

	m.mgrLogger.Infof("create container [%s] success :)", m.dockerContainerConfig.ContainerName)
	return nil
}

// create container with pprof feature
// which is open network and open an ip port in docker container
func (m *DockerManager) createPProfContainer() error {
	hostPort := strconv.Itoa(int(m.dockerVMConfig.DockerVMPprofPort))
	openPort := nat.Port(hostPort + "/tcp")

	sdkHostPort := strconv.Itoa(int(m.dockerVMConfig.SandBoxPprofPort))
	sdkOpenPort := nat.Port(sdkHostPort + "/tcp")

	_, err := m.dockerAPIClient.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        m.dockerContainerConfig.ImageName,
		AttachStdout: m.dockerContainerConfig.AttachStdOut,
		AttachStderr: m.dockerContainerConfig.AttachStderr,
		ExposedPorts: nat.PortSet{
			openPort:    struct{}{},
			sdkOpenPort: struct{}{},
		},
	}, &container.HostConfig{
		Privileged: true,
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostMountDir,
				Target:      m.dockerContainerConfig.DockerMountDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
			{
				Type:        mount.TypeBind,
				Source:      m.dockerContainerConfig.HostLogDir,
				Target:      m.dockerContainerConfig.DockerLogDir,
				ReadOnly:    false,
				Consistency: mount.ConsistencyFull,
				BindOptions: &mount.BindOptions{
					Propagation:  mount.PropagationRPrivate,
					NonRecursive: false,
				},
				VolumeOptions: nil,
				TmpfsOptions:  nil,
			},
		},
		PortBindings: nat.PortMap{
			openPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: hostPort,
				},
			},
			sdkOpenPort: []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: sdkHostPort,
				},
			},
		},
	}, nil, nil, m.dockerContainerConfig.ContainerName)

	if err != nil {
		m.mgrLogger.Errorf("create container [%s] failed", m.dockerContainerConfig.ContainerName)
		return err
	}

	m.mgrLogger.Infof("create container [%s] success :)", m.dockerContainerConfig.ContainerName)
	return nil
}

func validateVMSettings(config *config.DockerVMConfig,
	dockerContainerConfig *config.DockerContainerConfig, chainId string) error {

	var hostMountDir string
	var hostLogDir string
	var containerName string
	if len(config.DockerVMMountPath) == 0 {
		return errors.New("doesn't set host mount directory path correctly")
	}

	if len(config.DockerVMLogPath) == 0 {
		return errors.New("doesn't set host log directory path correctly")
	}

	// set host mount directory path
	if !filepath.IsAbs(config.DockerVMMountPath) {
		hostMountDir, _ = filepath.Abs(config.DockerVMMountPath)
		hostMountDir = filepath.Join(hostMountDir, chainId)
	} else {
		hostMountDir = filepath.Join(config.DockerVMMountPath, chainId)
	}

	// set host log directory
	if !filepath.IsAbs(config.DockerVMLogPath) {
		hostLogDir, _ = filepath.Abs(config.DockerVMLogPath)
		hostLogDir = filepath.Join(hostLogDir, chainId)
	} else {
		hostLogDir = filepath.Join(config.DockerVMLogPath, chainId)
	}

	// set docker container name
	if len(config.DockerVMContainerName) == 0 {
		containerName = fmt.Sprintf("%s-%s", chainId, defaultContainerName)
	} else {
		containerName = fmt.Sprintf("%s-%s", chainId, config.DockerVMContainerName)
	}

	if config.EnablePprof {
		if config.DockerVMPprofPort == 0 {
			return errors.New("docker vm pprof port cannot be 0")
		}

		if config.SandBoxPprofPort == 0 {
			return errors.New("sandbox pprof port cannot be 0")
		}
	}

	dockerContainerConfig.ContainerName = containerName
	dockerContainerConfig.HostMountDir = hostMountDir
	dockerContainerConfig.HostLogDir = hostLogDir

	return nil
}

func newDockerContainerConfig() *config.DockerContainerConfig {

	containerConfig := &config.DockerContainerConfig{
		AttachStdOut: true,
		AttachStderr: true,
		ShowStdout:   true,
		ShowStderr:   true,

		ImageName:     imageName,
		ContainerName: "",
		VMMgrDir:      dockerContainerDir,

		DockerMountDir: dockerMountDir,
		DockerLogDir:   dockerLogDir,
		HostMountDir:   "",
		HostLogDir:     "",
	}

	return containerConfig
}

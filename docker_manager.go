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
	"sync"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/protocol/v2"

	"github.com/mitchellh/mapstructure"

	"chainmaker.org/chainmaker/vm-docker-go/rpc"

	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
)

var (
	dockerContainerDir = "../module/vm/docker-go/vm_mgr"
	containerName      string
)

const (
	targetDir            = "/mount"
	dockerLogDir         = "/log"
	imageName            = "chainmakerofficial/chainmaker-vm-docker-go:develop"
	defaultContainerName = "chainmaker-vm-docker-go-container"
)

type DockerManager struct {
	chainId            string
	AttachStdOut       bool
	AttachStderr       bool
	ShowStdout         bool
	ShowStderr         bool
	imageName          string
	containerName      string
	hostMountPointPath string
	targetDir          string
	dockerDir          string

	hostLogDir   string
	dockerLogDir string

	lock   sync.Mutex
	ctx    context.Context
	client *client.Client

	Log            *logger.CMLogger
	CDMClient      *rpc.CDMClient
	CDMState       bool
	clientInitOnce sync.Once

	dockerVMConfig *config.DockerVMConfig
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

	dockerManagerLogger := logger.GetLoggerByChain("[Docker Manager]", chainId)
	dockerManagerLogger.Debugf("init docker manager")

	// init docker manager
	newDockerManager := &DockerManager{
		chainId:        chainId,
		AttachStdOut:   true,
		AttachStderr:   true,
		ShowStdout:     true,
		ShowStderr:     true,
		Log:            dockerManagerLogger,
		CDMState:       false,
		dockerVMConfig: dockerVMConfig,
		clientInitOnce: sync.Once{},
	}

	// validate settings
	err := newDockerManager.validateVMSettings(dockerVMConfig)
	if err != nil {
		dockerManagerLogger.Errorf("fail to init docker manager, please check the docker config, %s", err)
		return nil
	}

	// init docker manager
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	newDockerManager.ctx = context.Background()
	newDockerManager.client = cli
	newDockerManager.CDMClient = rpc.NewCDMClient(chainId, dockerVMConfig)
	newDockerManager.imageName = imageName
	newDockerManager.containerName = containerName
	newDockerManager.targetDir = targetDir
	newDockerManager.dockerDir = dockerContainerDir
	newDockerManager.dockerLogDir = dockerLogDir

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
	m.Log.Info("start docker vm...")
	var err error

	// check container is running or not
	// if running, stop it,
	isRunning, err := m.getContainer(false)
	if err != nil {
		return err
	}
	if isRunning {
		m.Log.Debugf("stop running container [%s]", m.containerName)
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
		m.Log.Debugf("remove container [%s]", m.containerName)
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
		m.Log.Errorf("cannot find docker vm image, please pull the image")
		return fmt.Errorf("cannot find docker vm image")
	}

	m.Log.Debugf("create container [%s]", m.containerName)
	err = m.createContainer()
	if err != nil {
		return err
	}

	// running container
	m.Log.Infof("start running container [%s]", m.containerName)
	if err = m.client.ContainerStart(m.ctx, m.containerName, types.ContainerStartOptions{}); err != nil {
		return err
	}

	m.Log.Debugf("docker vm start success :)")

	// display container info in the console
	go func() {
		err = m.displayInConsole(m.containerName)
		if err != nil {
			return
		}
	}()
	return nil
}

// StopVM stop docker vm and remove container, image
func (m *DockerManager) StopVM() error {
	if m == nil {
		return nil
	}
	var err error

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

	m.CDMState = false
	m.clientInitOnce = sync.Once{}

	m.Log.Info("stop and remove docker vm [%s]", m.containerName)
	return nil
}

func (m *DockerManager) NewRuntimeInstance(txSimContext protocol.TxSimContext, chainId, method, codePath string, contract *common.Contract,
	byteCode []byte, logger protocol.Logger) (protocol.RuntimeInstance, error) {

	// add client state logic
	if !m.getCDMState() {
		m.startCDMClient()
	}

	return &RuntimeInstance{
		ChainId: chainId,
		Client:  m.CDMClient,
		Log:     logger,
	}, nil
}

// StartCDMClient start CDM grpc rpc
func (m *DockerManager) startCDMClient() {

	m.clientInitOnce.Do(func() {
		state := m.CDMClient.StartClient()
		m.CDMState = state
		m.Log.Debugf("cdm rpc state is: %v", state)
	})
}

func (m *DockerManager) getCDMState() bool {
	return m.CDMState
}

// ------------------ image functions --------------

//// BuildImage build image based on Dockerfile
//func (m *DockerManager) buildImage(dockerFolderRelPath string) error {
//
//	// get absolute path for docker folder
//	dockerFolderPath, err := filepath.Abs(dockerFolderRelPath)
//	if err != nil {
//		return err
//	}
//
//	// tar whole directory
//	buildCtx, err := archive.TarWithOptions(dockerFolderPath, &archive.TarOptions{})
//	if err != nil {
//		return err
//	}
//
//	buildOpts := types.ImageBuildOptions{
//		Dockerfile: "Dockerfile",
//		Tags:       []string{m.imageName},
//		Remove:     true,
//	}
//
//	// build image
//	resp, err := m.client.ImageBuild(m.ctx, buildCtx, buildOpts)
//	if err != nil {
//		return err
//	}
//	defer func(body io.ReadCloser) {
//		err = body.Close()
//		if err != nil {
//			return
//		}
//	}(resp.Body)
//
//	err = m.displayBuildProcess(resp.Body)
//	if err != nil {
//		return err
//	}
//
//	m.Log.Infof("build image [%s] success :)", m.imageName)
//
//	return nil
//}

// check docker m image exist or not
func (m *DockerManager) imageExist() (bool, error) {
	imageList, err := m.client.ImageList(m.ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false, err
	}

	for _, v1 := range imageList {
		for _, v2 := range v1.RepoTags {
			if v2 == m.imageName {
				return true, nil
			}
			//currentImageName := strings.Split(v2, ":")
			//if currentImageName[0] == m.imageName {
			//	return true, nil
			//}
		}
	}
	return false, nil
}

//
//// remove image
//func (m *DockerManager) removeImage() error {
//	imageExist, err := m.imageExist()
//	if !imageExist || err != nil {
//		return nil
//	}
//
//	m.Log.Infof("Removing image [%s] ...", m.imageName)
//	if _, err = m.client.ImageRemove(m.ctx, m.imageName,
//		types.ImageRemoveOptions{PruneChildren: true, Force: true}); err != nil {
//		return err
//	}
//
//	_, err = m.client.ImagesPrune(m.ctx, filters.Args{})
//	if err != nil {
//		return err
//	}
//	return nil
//}

// ------------------ container functions --------------

// create container based on image
func (m *DockerManager) createContainer() error {

	// test mount dir is exit or not

	envs := m.constructEnvs()

	_, err := m.client.ContainerCreate(m.ctx, &container.Config{
		Cmd:          nil,
		Image:        m.imageName,
		Env:          envs,
		AttachStdout: m.AttachStdOut,
		AttachStderr: m.AttachStderr,
	}, &container.HostConfig{
		NetworkMode: "none",
		Privileged:  true,
		Mounts: []mount.Mount{
			{
				Type:        mount.TypeBind,
				Source:      m.hostMountPointPath,
				Target:      m.targetDir,
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
				Source:      m.hostLogDir,
				Target:      m.dockerLogDir,
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
	}, nil, nil, m.containerName)

	if err != nil {
		m.Log.Errorf("create container [%s] failed", m.containerName)
		return err
	}

	m.Log.Infof("create container [%s] success :)", m.containerName)
	return nil
}

// check container status: exist, not exist, running, or not running
// all true: docker ps -a
// all false: docker ps
func (m *DockerManager) getContainer(all bool) (bool, error) {
	containerList, err := m.client.ContainerList(m.ctx, types.ContainerListOptions{All: all})
	if err != nil {
		return false, err
	}

	indexName := "/" + m.containerName
	for _, v1 := range containerList {
		for _, v2 := range v1.Names {
			if v2 == indexName {
				return true, nil
			}
		}
	}
	return false, nil
}

// remove container
func (m *DockerManager) removeContainer() error {
	m.Log.Infof("Removing container [%s] ...", m.containerName)
	return m.client.ContainerRemove(m.ctx, m.containerName, types.ContainerRemoveOptions{})
}

// stop container
func (m *DockerManager) stopContainer() error {
	return m.client.ContainerStop(m.ctx, m.containerName, nil)
}

// InitMountDirectory init mount directory and sub directories
// if not exist, create directories
func (m *DockerManager) initMountDirectory() error {

	var err error

	// create mount directory
	mountDir := m.hostMountPointPath
	err = m.createDir(mountDir)
	if err != nil {
		return nil
	}
	m.Log.Debug("set mount dir: ", mountDir)

	// create sub directory: contracts, share, sock
	contractDir := filepath.Join(mountDir, config.ContractsDir)
	err = m.createDir(contractDir)
	if err != nil {
		m.Log.Errorf("fail to build image, err: [%s]", err)
		return err
	}
	m.Log.Debug("set contract dir: ", contractDir)

	sockDir := filepath.Join(mountDir, config.SockDir)
	err = m.createDir(sockDir)
	if err != nil {
		return err
	}
	m.Log.Debug("set sock dir: ", sockDir)

	// create log directory
	logDir := m.hostLogDir
	err = m.createDir(logDir)
	if err != nil {
		return nil
	}
	m.Log.Debug("set log dir: ", logDir)

	return nil

}

// ------------------ utility functions --------------

func (m *DockerManager) constructEnvs() []string {

	configsMap := make(map[string]string)

	configsMap[""] = fmt.Sprintf("udsopen=%v", m.dockerVMConfig.DockerVMUDSOpen)

	var containerConfig []string
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%v", config.ENV_ENABLE_UDS, m.dockerVMConfig.EnableDockerVM))
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%d", config.ENV_TX_SIZE, m.dockerVMConfig.TxSize))
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%d", config.ENV_USER_NUM, m.dockerVMConfig.UserNum))
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%d", config.ENV_TX_TIME_LIMIT, m.dockerVMConfig.TxTimeLimit))
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%s", config.ENV_LOG_LEVEL, m.dockerVMConfig.LogLevel))
	containerConfig = append(containerConfig, fmt.Sprintf("%s=%v", config.ENV_LOG_IN_CONSOLE, m.dockerVMConfig.LogInConsole))

	return containerConfig

}

//func (m *DockerManager) convertConfigToMap(config interface{}, configsMap map[string]string) {
//	v := reflect.ValueOf(config)
//	typeOfS := v.Type()
//
//	for i := 0; i < v.NumField(); i++ {
//		fieldName := typeOfS.Field(i).Name
//		value := v.Field(i).Interface()
//
//		env := fmt.Sprintf("%s=%v", fieldName, value)
//		configsMap[fieldName] = env
//	}
//}

//type ErrorLine struct {
//	Error       string      `json:"error"`
//	ErrorDetail ErrorDetail `json:"errorDetail"`
//}
//
//type ErrorDetail struct {
//	Message string `json:"message"`
//}

// display container std out in host std out -- need finish loop accept
func (m *DockerManager) displayInConsole(containerID string) error {
	//display container std out
	out, err := m.client.ContainerLogs(m.ctx, containerID, types.ContainerLogsOptions{
		ShowStdout: m.ShowStdout,
		ShowStderr: m.ShowStderr,
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
		_, err := out.Read(hdr)
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

//// display build process
//func (m *DockerManager) displayBuildProcess(rd io.Reader) error {
//	var lastLine string
//
//	scanner := bufio.NewScanner(rd)
//	for scanner.Scan() {
//
//		lastLine = scanner.Text()
//		lastLine = strings.TrimLeft(lastLine, "{stream\":")
//		lastLine = strings.TrimRight(lastLine, "\"}")
//		lastLine = strings.TrimRight(lastLine, "\\n")
//		if len(lastLine) == 0 {
//			continue
//		}
//		if strings.Contains(lastLine, "---\\u003e") {
//			continue
//		}
//
//		if strings.Contains(lastLine, "Removing intermediate") {
//			continue
//		}
//
//		if strings.Contains(lastLine, "ux\":{\"ID\":\"sha256") {
//			continue
//		}
//		//fmt.Println(lastLine)
//	}
//
//	errLine := &ErrorLine{}
//	_ = json.Unmarshal([]byte(lastLine), errLine)
//	if errLine.Error != "" {
//		return errors.New(errLine.Error)
//	}
//
//	return scanner.Err()
//}

func (m *DockerManager) createDir(directory string) error {
	exist, err := m.exists(directory)
	if err != nil {
		m.Log.Errorf("fail to get container, err: [%s]", err)
		return err
	}

	if !exist {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			m.Log.Errorf("fail to remove image, err: [%s]", err)
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

func (m *DockerManager) validateVMSettings(config *config.DockerVMConfig) error {

	if len(config.DockerVMMountPath) == 0 {
		return errors.New("doesn't set host mount directory path correctly")
	}

	if len(config.DockerVMLogPath) == 0 {
		return errors.New("doesn't set host log directory path correctly")
	}

	// host mount directory path
	if !filepath.IsAbs(config.DockerVMMountPath) {
		hostMountPointPath, _ := filepath.Abs(config.DockerVMMountPath)
		m.hostMountPointPath = filepath.Join(hostMountPointPath, m.chainId)
	}

	// host log directory
	if !filepath.IsAbs(config.DockerVMLogPath) {
		hostLogPointPath, _ := filepath.Abs(config.DockerVMLogPath)
		m.hostLogDir = filepath.Join(hostLogPointPath, m.chainId)
	}

	if len(config.DockerVMContainerName) == 0 {
		m.Log.Infof("container name doesn't set, set as default: chainmaker-docker-go-vm-container")
		containerName = fmt.Sprintf("%s-%s", m.chainId, defaultContainerName)
	} else {
		containerName = fmt.Sprintf("%s-%s", m.chainId, config.DockerVMContainerName)
	}

	return nil
}

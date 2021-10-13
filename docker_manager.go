/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package docker_go

import (
	"bufio"
	"context"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"sync"

	"chainmaker.org/chainmaker/protocol/v2"

	client2 "chainmaker.org/chainmaker/vm-docker-go/client"

	"chainmaker.org/chainmaker/localconf/v2"
	"chainmaker.org/chainmaker/logger/v2"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/config"
	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/archive"
)

var (
	dockerContainerDir string
	sourceDir          string
	imageName          string
	containerName      string
	hostLogDir         string
)

const (
	targetDir    = "/mount"
	dockerLogDir = "/log"
)

type DockerManager struct {
	AttachStdOut  bool
	AttachStderr  bool
	ShowStdout    bool
	ShowStderr    bool
	imageName     string
	containerName string
	sourceDir     string
	targetDir     string
	dockerDir     string

	hostLogDir   string
	dockerLogDir string

	lock   sync.Mutex
	ctx    context.Context
	client *client.Client

	Log       *logger.CMLogger
	CDMClient *client2.CDMClient
	CDMState  bool

	config *localconf.CMConfig
}

// NewDockerManager return docker manager and running a default container
func NewDockerManager(chainId string, config *localconf.CMConfig) *DockerManager {

	// test enter point
	var chainmakerConfig *localconf.CMConfig
	if config != nil {
		chainmakerConfig = config
	} else {
		chainmakerConfig = localconf.ChainMakerConfig
	}

	dockerManagerLogger := logger.GetLoggerByChain("[Docker Manager]", chainId)
	dockerManagerLogger.Debugf("init docker manager")
	// init docker manager
	newDockerManager := &DockerManager{
		AttachStdOut: true,
		AttachStderr: true,
		ShowStdout:   true,
		ShowStderr:   true,
		Log:          dockerManagerLogger,
		CDMState:     false,
	}

	// validate settings
	valid, err := newDockerManager.validateConfig(chainmakerConfig)
	if err != nil || !valid {
		dockerManagerLogger.Errorf("fail to init docker manager, please check the docker config, %s", err)
		return nil
	}

	// if enable docker vm is false, docker manager is nil
	startDockerVm := chainmakerConfig.DockerConfig.EnableDockerVM
	if !startDockerVm {
		return nil
	}

	// init docker manager
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil
	}

	newDockerManager.ctx = context.Background()
	newDockerManager.client = cli
	newDockerManager.CDMClient = client2.NewCDMClient(chainId, chainmakerConfig)
	newDockerManager.imageName = imageName
	newDockerManager.containerName = containerName
	newDockerManager.sourceDir = sourceDir
	newDockerManager.targetDir = targetDir
	newDockerManager.dockerDir = dockerContainerDir

	newDockerManager.hostLogDir = hostLogDir
	newDockerManager.dockerLogDir = dockerLogDir
	newDockerManager.config = chainmakerConfig

	// init mount directory and subdirectory
	err = newDockerManager.InitMountDirectory()
	if err != nil {
		dockerManagerLogger.Errorf("fail to init mount directory: %s", err)
		return nil
	}

	return newDockerManager
}

func (m *DockerManager) NewRuntimeInstance(chainId string, logger protocol.Logger) (protocol.RuntimeInstance, error) {
	return &RuntimeInstance{
		ChainId: chainId,
		Client:  m.CDMClient,
		Log:     logger,
	}, nil
}

// StartDockerVM Start Docker VM
func (m *DockerManager) StartDockerVM() error {
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
		m.Log.Debugf("Starting building image --- %s", m.imageName)
		err = m.buildImage(m.dockerDir)
		if err != nil {
			return err
		}
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

	m.Log.Info("docker vm start success :)")

	// display container info in the console
	go func() {
		err = m.displayInConsole(m.containerName)
		if err != nil {
			return
		}
	}()
	return nil
}

// StopAndRemoveDockerVM stop docker vm and remove container, image
func (m *DockerManager) StopAndRemoveDockerVM() error {
	var err error

	err = m.stopContainer()
	if err != nil {
		return err
	}

	err = m.removeContainer()
	if err != nil {
		return err
	}

	err = m.removeImage()
	if err != nil {
		return err
	}

	m.Log.Info("stop and remove docker vm [%s]", m.containerName)
	return nil
}

// StartCDMClient start CDM grpc client
func (m *DockerManager) StartCDMClient() {
	m.lock.Lock()
	defer m.lock.Unlock()
	state := m.CDMClient.StartClient()

	m.CDMState = state

	m.Log.Debugf("cdm client state is: %v", state)
}

func (m *DockerManager) GetCDMState() bool {
	return m.CDMState
}

// ------------------ image functions --------------

// BuildImage build image based on Dockerfile
func (m *DockerManager) buildImage(dockerFolderRelPath string) error {

	// get absolute path for docker folder
	dockerFolderPath, err := filepath.Abs(dockerFolderRelPath)
	if err != nil {
		return err
	}

	// tar whole directory
	buildCtx, err := archive.TarWithOptions(dockerFolderPath, &archive.TarOptions{})
	if err != nil {
		return err
	}

	buildOpts := types.ImageBuildOptions{
		Dockerfile: "Dockerfile",
		Tags:       []string{m.imageName},
		Remove:     true,
	}

	// build image
	resp, err := m.client.ImageBuild(m.ctx, buildCtx, buildOpts)
	if err != nil {
		return err
	}
	defer func(body io.ReadCloser) {
		err = body.Close()
		if err != nil {
			return
		}
	}(resp.Body)

	err = m.displayBuildProcess(resp.Body)
	if err != nil {
		return err
	}

	m.Log.Infof("build image [%s] success :)", m.imageName)

	return nil
}

// check docker m image exist or not
func (m *DockerManager) imageExist() (bool, error) {
	imageList, err := m.client.ImageList(m.ctx, types.ImageListOptions{All: true})
	if err != nil {
		return false, err
	}

	for _, v1 := range imageList {
		for _, v2 := range v1.RepoTags {
			currentImageName := strings.Split(v2, ":")
			if currentImageName[0] == m.imageName {
				return true, nil
			}
		}
	}
	return false, nil
}

// remove image
func (m *DockerManager) removeImage() error {
	imageExist, err := m.imageExist()
	if !imageExist || err != nil {
		return nil
	}

	m.Log.Infof("Removing image [%s] ...", m.imageName)
	if _, err = m.client.ImageRemove(m.ctx, m.imageName,
		types.ImageRemoveOptions{PruneChildren: true, Force: true}); err != nil {
		return err
	}

	_, err = m.client.ImagesPrune(m.ctx, filters.Args{})
	if err != nil {
		return err
	}
	return nil
}

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
				Source:      m.sourceDir,
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
func (m *DockerManager) InitMountDirectory() error {

	var err error

	// create mount directory
	mountDir := m.sourceDir
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

	shareDir := filepath.Join(mountDir, config.ShareDir)
	err = m.createDir(shareDir)
	if err != nil {
		m.Log.Errorf("fail to display build process, err: [%s]", err)
		return err
	}
	m.Log.Debug("set share dir: ", shareDir)

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
	dockerConfig := m.config.DockerConfig

	configsMap := make(map[string]string)

	m.convertConfigToMap(dockerConfig.DockerVmConfig, configsMap)

	m.convertConfigToMap(dockerConfig.DockerRpcConfig, configsMap)

	m.convertConfigToMap(dockerConfig.DockerPprofConfig, configsMap)

	// set log level
	configsMap["Log_Level"] = fmt.Sprintf("Log_Level=%s", m.config.LogConfig.SystemLog.LogLevels["vm"])

	// set log in console
	logInConsole := strconv.FormatBool(m.config.LogConfig.SystemLog.LogInConsole)
	configsMap["Log_In_Console"] = fmt.Sprintf("Log_In_Console=%s", logInConsole)

	// assembly envs
	configs := make([]string, len(configsMap))
	index := 0
	for _, value := range configsMap {
		configs[index] = value
		index++
	}

	return configs
}

func (m *DockerManager) convertConfigToMap(config interface{}, configsMap map[string]string) {
	v := reflect.ValueOf(config)
	typeOfS := v.Type()

	for i := 0; i < v.NumField(); i++ {
		fieldName := typeOfS.Field(i).Name
		value := v.Field(i).Interface()

		env := fmt.Sprintf("%s=%v", fieldName, value)
		configsMap[fieldName] = env
	}
}

type ErrorLine struct {
	Error       string      `json:"error"`
	ErrorDetail ErrorDetail `json:"errorDetail"`
}

type ErrorDetail struct {
	Message string `json:"message"`
}

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

// display build process
func (m *DockerManager) displayBuildProcess(rd io.Reader) error {
	var lastLine string

	scanner := bufio.NewScanner(rd)
	for scanner.Scan() {

		lastLine = scanner.Text()
		lastLine = strings.TrimLeft(lastLine, "{stream\":")
		lastLine = strings.TrimRight(lastLine, "\"}")
		lastLine = strings.TrimRight(lastLine, "\\n")
		if len(lastLine) == 0 {
			continue
		}
		if strings.Contains(lastLine, "---\\u003e") {
			continue
		}

		if strings.Contains(lastLine, "Removing intermediate") {
			continue
		}

		if strings.Contains(lastLine, "ux\":{\"ID\":\"sha256") {
			continue
		}
		fmt.Println(lastLine)
	}

	errLine := &ErrorLine{}
	_ = json.Unmarshal([]byte(lastLine), errLine)
	if errLine.Error != "" {
		return errors.New(errLine.Error)
	}

	return scanner.Err()
}

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

func (m *DockerManager) validateConfig(config *localconf.CMConfig) (bool, error) {
	dockerConfig := config.DockerConfig
	var err error

	if dockerConfig.DockerContainerDir == "" {
		m.Log.Errorf("doesn't set docker container path")
		return false, nil
	}
	dockerContainerDir = dockerConfig.DockerContainerDir

	// host mount directory
	sourceDir = dockerConfig.MountPath
	if !filepath.IsAbs(sourceDir) {
		sourceDir, err = filepath.Abs(sourceDir)
		if err != nil {
			m.Log.Errorf("doesn't set host mount directory path correctly")
			return false, err
		}
	}

	// host log directory
	hostLogDir = config.LogConfig.SystemLog.DockerFilePath
	if !filepath.IsAbs(hostLogDir) {
		hostLogDir, err = filepath.Abs(hostLogDir)
		if err != nil {
			m.Log.Errorf("doesn't set host log directory path correctly")
			return false, err
		}
	}

	if dockerConfig.ImageName == "" {
		m.Log.Infof("image name doesn't set, set as default: chainmaker-docker-go-image")
		imageName = "chainmaker-docker-go-image"
	} else {
		imageName = dockerConfig.ImageName
	}

	if dockerConfig.ContainerName == "" {
		m.Log.Infof("container name doesn't set, set as default: chainmaker-docker-go-container")
		containerName = "chainmaker-docker-go-container"
	} else {
		containerName = dockerConfig.ContainerName
	}

	return true, nil
}

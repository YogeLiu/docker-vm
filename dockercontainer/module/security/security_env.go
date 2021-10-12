/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"os"
	"path/filepath"
	"strconv"

	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/config"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/utils"
	"go.uber.org/zap"
)

const (
	ipcPath = "/proc/sys/kernel"
)

var (
	ipcFiles   = []string{"shmmax", "shmall", "msgmax", "msgmnb", "msgmni"}
	ipcSemFile = "sem"
)

type SecurityEnv struct {
	logger *zap.SugaredLogger
}

func NewSecurityEnv() *SecurityEnv {
	return &SecurityEnv{
		logger: logger.NewDockerLogger(logger.MODULE_SECURITY_ENV),
	}
}

func (s *SecurityEnv) InitSecurityEnv() error {
	if err := s.setTmpMod(); err != nil {
		return err
	}

	if err := SetCGroup(); err != nil {
		s.logger.Errorf("fail to setCGroup, err : [%s]", err)
		return err
	}

	if err := s.setIPC(); err != nil {
		s.logger.Errorf("fail to set ipc err: [%s]", err)
		return err
	}

	s.logger.Infof("init security env completed")

	return nil
}

func (s *SecurityEnv) InitConfig() error {

	var err error

	// set mount dir mod
	mountDir := config.DockerMountDir

	// set mount sub directory: contracts, share, sock
	contractDir := filepath.Join(mountDir, config.ContractsDir)

	config.ContractBaseDir = contractDir

	shareDir := filepath.Join(mountDir, config.ShareDir)
	config.ShareBaseDir = shareDir

	sockDir := filepath.Join(mountDir, config.SockDir)
	config.SockBaseDir = sockDir

	// set timeout
	timeLimitConfig := os.Getenv("TimeLimit")
	timeLimit, err := strconv.Atoi(timeLimitConfig)
	if err != nil {
		s.logger.Errorf("fail to convert timeLimitConfig: [%s], err: [%s]", timeLimitConfig, err)
		timeLimit = 2
	}
	config.SandBoxTimeout = timeLimit

	// set dms directory
	if err = s.setDMSDir(); err != nil {
		s.logger.Errorf("fail to set dms directory, err: [%s]", err)
		return err
	}
	s.logger.Debug("set dms dir: ", config.DMSDir)

	return nil

}

func (s *SecurityEnv) setDMSDir() error {
	return os.Mkdir(config.DMSDir, 0755)
}

func (s *SecurityEnv) setTmpMod() error {
	return os.Chmod("/tmp/", 0755)
}

func (s *SecurityEnv) setIPC() error {
	for _, file := range ipcFiles {
		currentFile := filepath.Join(ipcPath, file)
		err := utils.WriteToFile(currentFile, 0)
		if err != nil {
			return err
		}
	}

	ipcSemPath := filepath.Join(ipcPath, ipcSemFile)
	err := utils.WriteToFIle(ipcSemPath, "0 0 0 0")
	if err != nil {
		return err
	}
	return nil
}

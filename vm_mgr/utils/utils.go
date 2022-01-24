/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"go.uber.org/zap"
)

// WriteToFile WriteFile write value to file
func WriteToFile(path string, value int) error {
	if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", value)), 0755); err != nil {
		return err
	}
	return nil
}

func WriteToFIle(path, info string) error {
	if err := ioutil.WriteFile(path, []byte(info), 0755); err != nil {
		return err
	}
	return nil
}

// RunCmd exec cmd
func RunCmd(command string) error {
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)

	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func ProcessLoggerName(processName string) string {
	loggerName := "[ " + processName + " ]"
	return loggerName
}

func GetTestLogPath() string {
	basePath, _ := os.Getwd()
	return basePath + config.TestPath
}

func GetLogHandler() *zap.SugaredLogger {
	return logger.NewDockerLogger(logger.MODULE_PROCESS, GetTestLogPath())
}

// ConstructContractKey contractName:contractVersion
func ConstructContractKey(contractName, contractVersion string) string {
	return contractName + ":" + contractVersion
}

// ConstructProcessName contractName:contractVersion:index
func ConstructProcessName(contractName, contractVersion string, index uint64) string {
	return contractName + ":" + contractVersion + "#" + strconv.FormatInt(time.Now().UnixMicro(), 10) +
		":" + strconv.FormatUint(index, 10)
}

// cross processName: txId:height
func ConstructCrossContractProcessName(txId string, txDepth uint64) string {
	return txId + ":" + strconv.FormatUint(txDepth, 10)
}

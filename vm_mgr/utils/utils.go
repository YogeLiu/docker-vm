/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

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
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

// ConstructProcessName contractName:contractVersion#timestamp:index
func ConstructProcessName(contractName, contractVersion string, index uint64) string {
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	sb.WriteString(":")
	sb.WriteString(strconv.FormatUint(index, 10))
	return sb.String()
}

// ConstructOriginalProcessName contractName:contractVersion#timestamp:index#txCount
func ConstructOriginalProcessName(processName string, txCount uint64) string {
	var sb strings.Builder
	sb.WriteString(processName)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(txCount, 10))
	return sb.String()
}

// ConstructConcatOriginalAndCrossProcessName contractName:contractVersion#timestamp:index#txCount&txId:timestamp:depth
func ConstructConcatOriginalAndCrossProcessName(originalProcessName, crossProcessName string) string {
	var sb strings.Builder
	sb.WriteString(originalProcessName)
	sb.WriteString("&")
	sb.WriteString(crossProcessName)
	return sb.String()
}

// ConstructCrossContractProcessName txId:timestamp:depth
func ConstructCrossContractProcessName(txId string, txDepth uint64) string {
	var sb strings.Builder
	sb.WriteString(txId)
	sb.WriteString(":")
	sb.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	sb.WriteString(":")
	sb.WriteString(strconv.FormatUint(txDepth, 10))
	return sb.String()
}

// TrySplitCrossProcessNames if processName is a crossProcessName, return original process name and cross process name.
// if processName is a balance process name, return contract key and processName itself.
func TrySplitCrossProcessNames(processName string) (bool, string, string) {
	nameList := strings.Split(processName, "&")
	if len(nameList) == 2 {
		return true, nameList[0], nameList[1]
	}
	nameList = strings.Split(processName, "#")
	return false, ConstructContractKey(nameList[0], nameList[1]), processName
}

func GetContractKeyFromProcessName(processName string) string {
	nameList := strings.Split(processName, "#")
	return ConstructContractKey(nameList[0], nameList[1])
}

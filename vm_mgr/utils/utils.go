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
)

// WriteToFile WriteFile write value to file
func WriteToFile(path string, value string) error {
	if err := ioutil.WriteFile(path, []byte(value), 0755); err != nil {
		return err
	}
	return nil
}

// Mkdir make directory
func Mkdir(path string) error {
	// if dir existed, remove first
	_, err := os.Stat(path)
	if err == nil {
		err = RemoveDir(path)
		return err
	}

	// make dir
	if err = os.Mkdir(path, 0755); err != nil {
		return fmt.Errorf("fail to create directory, [%s]", err)
	}
	return nil
}

// RemoveDir remove dir from disk
func RemoveDir(path string) error {
	// chmod contract file
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("fail to set mod of %s, %v", path, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("fail to remove file %s, %v", path, err)
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

//func GetTestLogPath() string {
//	basePath, _ := os.Getwd()
//	return basePath + config.TestPath
//}

//func GetLogHandler() *zap.SugaredLogger {
//	return logger.NewDockerLogger(logger.MODULE_PROCESS, GetTestLogPath())
//}

// ConstructContractKey contractName#contractVersion
func ConstructContractKey(contractName, contractVersion string) string {
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

// ConstructProcessName contractName#contractVersion#timestamp:index
func ConstructProcessName(contractName, contractVersion string, index int) string {
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatInt(time.Now().UnixNano(), 10))
	sb.WriteString(":")
	sb.WriteString(strconv.Itoa(index))
	return sb.String()
}

// ConstructRequestGroupKey contractName#contractVersion#timestamp:index
func ConstructRequestGroupKey(contractName string, contractVersion string) string {
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

// ConstructOriginalProcessName contractName#contractVersion#timestamp:index#txCount
func ConstructOriginalProcessName(processName string, txCount uint64) string {
	var sb strings.Builder
	sb.WriteString(processName)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(txCount, 10))
	return sb.String()
}

// ConstructConcatOriginalAndCrossProcessName contractName#contractVersion#timestamp:index#txCount&txId:timestamp:depth
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
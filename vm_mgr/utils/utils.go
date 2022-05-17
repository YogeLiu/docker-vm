/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
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

// ConstructContractKey contractName#contractVersion
func ConstructContractKey(contractName, contractVersion string) string {
	var sb strings.Builder
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

// ConstructProcessName contractName#contractVersion#timestamp:index
func ConstructProcessName(contractName, contractVersion string, index uint64, isOrig bool) string {
	var sb strings.Builder
	typeStr := "o"
	if !isOrig {
		typeStr = "c"
	}
	sb.WriteString(typeStr)
	sb.WriteString("#")
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(index, 10))
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

// HasUsed judge whether a vm has been used
func HasUsed(ctxBitmap uint64) bool {
	typeBit := uint64(1 << (59 - common.RuntimeType_DOCKER_GO))
	return typeBit&ctxBitmap > 0
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
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
		return fmt.Errorf("failed to create directory, [%s]", err)
	}
	return nil
}

// RemoveDir remove dir from disk
func RemoveDir(path string) error {
	// chmod contract file
	if err := os.Chmod(path, 0755); err != nil {
		return fmt.Errorf("failed to set mod of %s, %v", path, err)
	}
	if err := os.Remove(path); err != nil {
		return fmt.Errorf("failed to remove file %s, %v", path, err)
	}
	return nil
}

// RunCmd exec cmd
func RunCmd(command string) error {
	var stderr bytes.Buffer
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)
	cmd.Stderr = &stderr
	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("%v, %v", err, stderr.String())
	}

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("%v, %v", err, stderr.String())
	}

	return nil
}

// ConstructContractKey contractName#contractVersion
func ConstructContractKey(chainID, contractName, contractVersion string) string {
	var sb strings.Builder
	sb.WriteString(chainID)
	sb.WriteString("#")
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	return sb.String()
}

// ConstructProcessName contractName#contractVersion#timestamp:index
func ConstructProcessName(chainID, contractName, contractVersion string,
	localIndex int, overallIndex uint64, isOrig bool) string {
	var sb strings.Builder
	typeStr := "o"
	if !isOrig {
		typeStr = "c"
	}
	sb.WriteString(chainID)
	sb.WriteString("#")
	sb.WriteString(typeStr)
	sb.WriteString("#")
	sb.WriteString(contractName)
	sb.WriteString("#")
	sb.WriteString(contractVersion)
	sb.WriteString("#")
	sb.WriteString(strconv.Itoa(localIndex))
	sb.WriteString("#")
	sb.WriteString(strconv.FormatUint(overallIndex, 10))
	return sb.String()
}

// IsOrig judge whether the tx is original or cross
func IsOrig(tx *protogo.DockerVMMessage) bool {
	isOrig := tx.CrossContext.CurrentDepth == 0 || !hasUsed(tx.CrossContext.CrossInfo)
	return isOrig
}

// hasUsed judge whether a vm has been used
func hasUsed(ctxBitmap uint64) bool {
	typeBit := uint64(1 << (59 - common.RuntimeType_DOCKER_GO))
	return typeBit&ctxBitmap > 0
}

// Min returns min value
func Min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func CreateDir(directory string) error {
	exist, err := exists(directory)
	if err != nil {
		return err
	}

	if !exist {
		err = os.MkdirAll(directory, 0755)
		if err != nil {
			return err
		}
	}

	return nil
}

func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

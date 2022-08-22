/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/pb/protogo"
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
	typeBit := uint64(1 << (59 - common.RuntimeType_GO))
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

func EnterNextStep(msg *protogo.DockerVMMessage, stepType protogo.StepType, log string) {

	if stepType != protogo.StepType_RUNTIME_PREPARE_TX_REQUEST {
		endTxStep(msg, log)
	}
	addTxStep(msg, stepType)
	if stepType == protogo.StepType_RUNTIME_HANDLE_TX_RESPONSE {
		endTxStep(msg, log)
	}
}

func addTxStep(msg *protogo.DockerVMMessage, stepType protogo.StepType) {
	stepDur := &protogo.StepDuration{
		Type:      stepType,
		StartTime: time.Now().UnixNano(),
	}
	msg.StepDurations = append(msg.StepDurations, stepDur)
}

func endTxStep(msg *protogo.DockerVMMessage, log string) {
	if len(msg.StepDurations) == 0 {
		return
	}
	stepLen := len(msg.StepDurations)
	currStep := msg.StepDurations[stepLen-1]
	currStep.Msg = log
	firstStep := msg.StepDurations[0]
	currStep.UntilDuration = time.Since(time.Unix(0, firstStep.StartTime)).Nanoseconds()
	currStep.StepDuration = time.Since(time.Unix(0, currStep.StartTime)).Nanoseconds()
}

func PrintTxSteps(msg *protogo.DockerVMMessage) string {
	var sb strings.Builder
	for _, step := range msg.StepDurations {
		sb.WriteString(fmt.Sprintf("<step: %q, start time: %v, step cost: %vms, until cost: %vms, msg: %s> ",
			step.Type, time.Unix(0, step.StartTime),
			time.Duration(step.StepDuration).Seconds()*1000,
			time.Duration(step.UntilDuration).Seconds()*1000,
			step.Msg))
	}
	return sb.String()
}

func PrintTxStepsWithTime(msg *protogo.DockerVMMessage, untilDuration time.Duration) (string, bool) {
	if len(msg.StepDurations) == 0 {
		return "", false
	}
	lastStep := msg.StepDurations[len(msg.StepDurations)-1]
	var sb strings.Builder
	if lastStep.UntilDuration > untilDuration.Nanoseconds() {
		sb.WriteString(fmt.Sprintf("slow tx overall: "))
		sb.WriteString(PrintTxSteps(msg))
		return sb.String(), true
	}
	for _, step := range msg.StepDurations {
		if step.StepDuration > time.Millisecond.Nanoseconds()*500 {
			sb.WriteString(fmt.Sprintf("slow tx at step %q, step cost: %vms: ",
				step.Type, time.Duration(step.StepDuration).Seconds()*1000))
			sb.WriteString(PrintTxSteps(msg))
			return sb.String(), true
		}
	}
	return "", false
}

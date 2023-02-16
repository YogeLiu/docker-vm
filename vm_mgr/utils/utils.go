/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package utils

import (
	"bytes"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/pb-go/v2/common"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
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

// ConstructContractKey chainID#contractName#contractVersion
func ConstructContractKey(names ...string) string {
	return strings.Join(names, "#")
}

// ConstructProcessName chainID#contractName#contractVersion#timestamp#localIndex#overallIndex
func ConstructProcessName(chainID, contractName, contractVersion string,
	localIndex int, overallIndex uint64, isOrig bool) string {
	typeStr := "o"
	if !isOrig {
		typeStr = "c"
	}
	return constructProcessName(chainID, typeStr, contractName, contractVersion,
		strconv.Itoa(localIndex), strconv.FormatUint(overallIndex, 10))
}

// constructProcessName chainID#contractName#contractVersion#timestamp#localIndex#overallIndex
func constructProcessName(names ...string) string {
	return strings.Join(names, "#")
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

func CreateDir(directory string, perm fs.FileMode) error {
	exist, err := exists(directory)
	if err != nil {
		return err
	}

	if !exist {
		err = os.MkdirAll(directory, perm)
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

// EnterNextStep enter next duration tx step
func EnterNextStep(msg *protogo.DockerVMMessage, stepType protogo.StepType, getStr func() string) {
	if config.DockerVMConfig.Slow.Disable {
		return
	}

	if stepType != protogo.StepType_RUNTIME_PREPARE_TX_REQUEST {
		endTxStep(msg)
	}
	addTxStep(msg, stepType, getStr())
	if stepType == protogo.StepType_RUNTIME_HANDLE_TX_RESPONSE ||
		stepType == protogo.StepType_ENGINE_PROCESS_RECEIVE_TX_RESPONSE {
		endTxStep(msg)
	}
}

func addTxStep(msg *protogo.DockerVMMessage, stepType protogo.StepType, log string) {
	stepDur := &protogo.StepDuration{
		Type:      stepType,
		StartTime: time.Now().UnixNano(),
		Msg:       log,
	}
	msg.StepDurations = append(msg.StepDurations, stepDur)
}

func endTxStep(msg *protogo.DockerVMMessage) {
	if len(msg.StepDurations) == 0 {
		return
	}
	stepLen := len(msg.StepDurations)
	currStep := msg.StepDurations[stepLen-1]
	firstStep := msg.StepDurations[0]
	currStep.UntilDuration = time.Since(time.Unix(0, firstStep.StartTime)).Nanoseconds()
	currStep.StepDuration = time.Since(time.Unix(0, currStep.StartTime)).Nanoseconds()
}

// PrintTxSteps print all duration tx steps
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

// PrintTxStepsWithTime print all duration tx steps with time limt
func PrintTxStepsWithTime(msg *protogo.DockerVMMessage) (string, bool) {
	if len(msg.StepDurations) == 0 {
		return "", false
	}
	lastStep := msg.StepDurations[len(msg.StepDurations)-1]
	var sb strings.Builder
	if lastStep.UntilDuration > config.DockerVMConfig.Slow.TxTime.Nanoseconds() {
		sb.WriteString("slow tx overall: ")
		sb.WriteString(PrintTxSteps(msg))
		return sb.String(), true
	}
	for _, step := range msg.StepDurations {
		if step.StepDuration > config.DockerVMConfig.Slow.StepTime.Nanoseconds() {
			sb.WriteString(fmt.Sprintf("slow tx at step %q, step cost: %vms: ",
				step.Type, time.Duration(step.StepDuration).Seconds()*1000))
			sb.WriteString(PrintTxSteps(msg))
			return sb.String(), true
		}
	}
	return "", false
}

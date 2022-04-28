/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"os"
	"strconv"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
)

// ProcessBalance control load balance of process which related to same contract
// key of map is processGroup: contractName:contractVersion
// value of map is peerBalance: including a list of process, each name is contractName:contractVersion:index
type ProcessBalance struct {
	processes map[string]*Process
	// maxProcess for same contract
	maxProcess int64
	index      uint64
}

func NewProcessBalance() *ProcessBalance {
	return &ProcessBalance{
		processes:  make(map[string]*Process),
		index:      0,
		maxProcess: getMaxPeer(),
	}
}

func (pb *ProcessBalance) AddProcess(process *Process, processName string) {
	pb.processes[processName] = process
}

func (pb *ProcessBalance) GetProcess(processName string) *Process {
	return pb.processes[processName]
}

func (pb *ProcessBalance) RemoveProcess(processName string) {
	delete(pb.processes, processName)
}

func (pb *ProcessBalance) GetNextProcessIndex() uint64 {
	val := pb.index
	pb.index++
	return val
}

func (pb *ProcessBalance) GetAvailableProcess() (*Process, error) {
	var resultProcessName string
	minSize := processWaitingQueueSize
	for processName, process := range pb.processes {
		processSize := process.Size()
		if processSize < minSize {
			minSize = processSize
			resultProcessName = processName
		}
	}

	// found
	if len(resultProcessName) != 0 {
		if pb.needCreateNewProccess(int64(minSize)) {
			return nil, nil
		}
		return pb.GetProcess(resultProcessName), nil
	}

	// not found: all processes is full and pb is full or pb is empty
	if pb.Size() < pb.maxProcess {
		return nil, nil
	}
	return nil, fmt.Errorf("faild to get process, the balance is full")
}

func (pb *ProcessBalance) needCreateNewProccess(processSize int64) bool {
	if pb.Size() >= pb.maxProcess {
		return false
	}
	if processSize > triggerNewProcessSize {
		return true
	}
	return false
}

func (pb *ProcessBalance) Size() int64 {
	return int64(len(pb.processes))
}

func getMaxPeer() int64 {
	mc := os.Getenv(config.ENV_MAX_CONCURRENCY)
	maxConcurrency, err := strconv.Atoi(mc)
	if err != nil {
		maxConcurrency = defaultMaxProcess
	}
	return int64(maxConcurrency)
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
)

const (
	processIncreaseDelta = 1
	txQueueSize          = 100000
)

// ProcessBalance control load balance of process which related to same contract
// key of map is processGroup: contractName:contractVersion
// value of map is peerBalance: including a list of process, each name is contractName:contractVersion:index
type ProcessBalance struct {
	processes map[string]*Process
	// maxProcess for same contract
	maxProcess int64
	index      uint64

	txQueue chan *protogo.TxRequest
}

func NewProcessBalance() *ProcessBalance {
	return &ProcessBalance{
		processes:  make(map[string]*Process),
		index:      0,
		maxProcess: utils.GetMaxConcurrencyFromEnv(),

		txQueue: make(chan *protogo.TxRequest, txQueueSize),
	}
}

func (pb *ProcessBalance) GetTxQueue() chan *protogo.TxRequest {
	return pb.txQueue
}

func (pb *ProcessBalance) AddTx(txRequest *protogo.TxRequest) {
	pb.txQueue <- txRequest
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

func (pb *ProcessBalance) needCreateNewProcess() bool {
	if pb.Size() >= pb.maxProcess {
		return false
	}
	if len(pb.txQueue) < processIncreaseDelta {
		return false
	}
	return true
}

func (pb *ProcessBalance) Size() int64 {
	return int64(len(pb.processes))
}

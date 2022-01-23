/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"fmt"
	"os"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"go.uber.org/zap"
)

// balancer strategy
const (
	// StrategyRoundRobin every time, launch max num backend,
	// and then using round-robin to feed txs
	StrategyRoundRobin = iota

	// StrategyLeastSize first launch one backend, when reaching the 2/3 capacity, launch the second one
	// using the least connections' strategy to feed txs
	StrategyLeastSize
)

const (
	defaultMaxProcess          = 10
	defaultMaxWaitingQueueSize = 2
)

var (
	// max process num for same contract
	maxProcess int
	// increase process condition: when reach limit, add new process
	waitingQueueLimit int
)

// ProcessBalance control load balance of process which related to same contract
// key of map is processGroup: contractName:contractVersion
// value of map is peerBalance: including a list of process, each name is contractName:contractVersion:index
type ProcessBalance struct {
	processes sync.Map
	// curIdx: round-robin index
	curIdx   uint64
	strategy int
	// size of the processes
	size int
}

//// PeerDepth control cross contract
//// key of map is processName: contractName:contractVersion:index
//// value of map is peerDepth: including a list of process, the first one is original process
//// the reset are cross contract process
//type PeerDepth struct {
//	processes [protocol2.CallContractDepth + 1]*Process
//	size  int
//}

type ProcessManager struct {
	logger *zap.SugaredLogger
	mutex  sync.Mutex

	// map[string]*ProcessBalance:
	// string: processNamePrefix: contractName:contractVersion
	// peerBalance:
	// sync.Map: map[index]*Process
	// int: index
	// processName: contractName:contractVersion#index (original process)
	contractTable sync.Map

	// map[string1]map[int]*CrossProcess  sync.Map[map]
	// string1: originalProcessName: contractName:contractVersion#index#txCount - related to tx
	// int: height
	// processName: txId:height#txCount (cross process)
	crossTable sync.Map
}

func NewProcessManager() *ProcessManager {
	pmLogger := logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER, config.DockerLogDir)

	maxProcess = getMaxPeer()
	waitingQueueLimit = getProcessLimitSize()
	pmLogger.Infof("init process manager with max concurrency [%d], limit size [%d]", maxProcess, waitingQueueLimit)

	return &ProcessManager{
		logger:        pmLogger,
		contractTable: sync.Map{},
		crossTable:    sync.Map{},
		//crossTable:   sync.Map{},
	}
}

// SetStrategy
func (pm *ProcessManager) setStrategy(key string, _strategy int) {

	pm.logger.Infof("set process manager strategy [%d]", _strategy)

	balance, ok := pm.contractTable.Load(key)
	if ok {
		balance.(*ProcessBalance).strategy = _strategy
	}

}

// RegisterNewProcess register new original process
// @param: processNamePrefix: contract:version
// after register, processName:contract:version#index
func (pm *ProcessManager) RegisterNewProcess(processNamePrefix string, process *Process) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.logger.Infof("register new process: [%s]", processNamePrefix)

	success := pm.addProcessIntoBalance(processNamePrefix, process)
	if success {
		process.Handler.processName = process.processName
	}

	return success
}

// ReleaseProcess release original process
// @param: processName: contract:version#index
func (pm *ProcessManager) ReleaseProcess(processName string) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.logger.Infof("release process: [%s]", processName)

	nameList := strings.Split(processName, "#")
	key := nameList[0]
	idx, _ := strconv.Atoi(nameList[1])

	pm.removeProcessFromProcessBalance(key, idx)
}

func (pm *ProcessManager) RegisterCrossProcess(originalProcessName string, height uint32, calledProcess *Process) {
	pm.logger.Debugf("register cross process [%s], original process name [%s]", calledProcess.processName, originalProcessName)
	//pm.crossTable.Store(originalProcessName, calledProcess)
	pm.addCrossProcessIntoDepth(originalProcessName, height, calledProcess)
	calledProcess.Handler.processName = calledProcess.processName
}

func (pm *ProcessManager) ReleaseCrossProcess(crossProcessName string, originalProcessName string, currentHeight uint32) {
	pm.logger.Debugf("release cross process [%s], original process name [%s]", crossProcessName, originalProcessName)
	//pm.crossTable.Delete(originalProcessName)
	pm.removeCrosseProcessFromDepth(originalProcessName, currentHeight)
}

// GetProcess retrieve peer from process manager, could be original process or cross process:
// contractName:contractVersion#index#txCount
// original process: contractName:contractVersion#index
func (pm *ProcessManager) GetProcess(processName string) *Process {
	pm.logger.Debugf("get process [%s]", processName)
	nameList := strings.Split(processName, "#")

	if len(nameList) == 4 {
		crossKey := strings.Join(nameList[:3], "#")
		height, _ := strconv.Atoi(nameList[3])
		cd, _ := pm.crossTable.Load(crossKey)
		crossDepth := cd.(map[uint32]*Process)
		return crossDepth[uint32(height)]
	}

	key := nameList[0]
	idx, _ := strconv.Atoi(nameList[1])

	processBalance := pm.getProcessBalance(key)
	process, _ := processBalance.processes.Load(idx)
	return process.(*Process)
}

// GetAvailableProcess return one process from peer balance based on current strategy
// based on different strategy, using different method to get next
// if peer list just have one peer, always return it
// when this peer reach limit, scheduler will generate new peer, limit May 2/3 of capacity or 4/5 of capacity
// then function will return this new peer always, because we should feed this new process -- using lease size function
// when return peer reach limit, then generate new peer, do above process
// eventually, peer list reach limit, and return peer also reach limit, using round-robin algorithm to return peer
// get one peer from a group of processes
func (pm *ProcessManager) GetAvailableProcess(processKey string) *Process {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// peer list just have one peer, always return it
	// until returned peer reach limit, will generate new peer into balancer
	pb, ok := pm.contractTable.Load(processKey)
	if !ok {
		return nil
	}
	processBalance := pb.(*ProcessBalance)
	//
	//if processBalance.size == 1 {
	//	curPb, _ := processBalance.processes.Load(0)
	//	return curPb.(*Process)
	//}

	// peer list contains 1 ~ maxProcess processes, return lease size peer
	// when returned peer reach limit, will generate new peer into balancer
	// until returned peer also reach limit, set strategy as StrategyRoundRobin
	if processBalance.size <= maxProcess && processBalance.strategy == StrategyLeastSize {
		return pm.getProcessWithLeastSize(processBalance)
	}

	// peer list is full, using Round-Robin to return peer
	// at this time, strategy == StrategyRoundRobin
	// when one peer timeout, release this peer, reset strategy as StrategyLeastSize
	// so using lease size to return peer
	return pm.getProcessWithRoundRobin(processBalance)
}

// ========================= Peer Balance functions ===============================

func (pm *ProcessManager) addProcessIntoBalance(key string, process *Process) bool {

	processBalance, ok := pm.contractTable.Load(key)

	if !ok {
		newProcessBalance := &ProcessBalance{
			processes: sync.Map{},
			curIdx:    0,
			strategy:  StrategyLeastSize,
			size:      0,
		}
		process.processName = fmt.Sprintf("%s#%d", process.processName, 0)
		newProcessBalance.processes.Store(0, process)
		newProcessBalance.size++
		pm.contractTable.Store(key, newProcessBalance)
		pm.logger.Debugf("add process into newProcessBalance [%s]", process.processName)
		return true
	}

	pb := processBalance.(*ProcessBalance)
	curSize := pb.size

	if curSize < maxProcess {
		process.processName = fmt.Sprintf("%s#%d", process.processName, curSize)
		pb.processes.Store(curSize, process)
		pb.size++

		if pb.size == maxProcess {
			pm.setStrategy(key, StrategyRoundRobin)
		}

		pm.logger.Debugf("add process into balance [%s], processBalance size:%d", process.processName, pb.size)

		return true
	}

	return false

}

func (pm *ProcessManager) getProcessBalance(key string) *ProcessBalance {
	processBalance, ok := pm.contractTable.Load(key)
	if ok {
		return processBalance.(*ProcessBalance)
	}
	return nil
}

func (pm *ProcessManager) removeProcessFromProcessBalance(key string, idx int) {
	pb, ok := pm.contractTable.Load(key)
	if !ok {
		return
	}

	processBalance := pb.(*ProcessBalance)

	processBalance.processes.Delete(idx)
	processBalance.size--
	processBalance.strategy = StrategyLeastSize

	if processBalance.size == 0 {
		//delete(pm.contractTable, key)
		pm.contractTable.Delete(key)
	}
}

// ============================== Peer Depth functions ================================

//func (pm *ProcessManager) addFirstPeerIntoDepth(processName string, process *Process) {
//	pm.logger.Debugf("add first peer into depth [%s]", process.processName)
//	newPeerDepth := &PeerDepth{
//		processes: [protocol2.CallContractDepth + 1]*Process{},
//		size:  0,
//	}
//	newPeerDepth.processes[0] = process
//	newPeerDepth.size++
//	pm.crossTable[processName] = newPeerDepth
//}
//
//func (pm *ProcessManager) removeFirstPeerFromDepth(processName string) {
//	pm.logger.Debugf("remove first peer from depth [%s]", processName)
//	delete(pm.crossTable, processName)
//}

func (pm *ProcessManager) addCrossProcessIntoDepth(originalProcessName string, height uint32, calledProcess *Process) {

	pm.logger.Debugf("add cross process %s with initial process name %s",
		calledProcess.processName, originalProcessName)

	processDepth, ok := pm.crossTable.Load(originalProcessName)
	if !ok {
		newProcessDepth := make(map[uint32]*Process)
		newProcessDepth[height] = calledProcess
		pm.crossTable.Store(originalProcessName, newProcessDepth)
		processDepth = newProcessDepth
	}

	//pm.mutex.Lock()
	//defer pm.mutex.Unlock()

	pd := processDepth.(map[uint32]*Process)
	pd[height] = calledProcess

}

func (pm *ProcessManager) removeCrosseProcessFromDepth(originalProcessName string, currentHeight uint32) {
	if currentHeight == 0 {
		return
	}

	pm.logger.Debugf("release cross process with position: %v", currentHeight)

	//pm.mutex.Lock()
	//defer pm.mutex.Unlock()

	peerDepth, ok := pm.crossTable.Load(originalProcessName)
	if ok {
		pd := peerDepth.(map[uint32]*Process)
		delete(pd, currentHeight)
		//peerDepth.processes[currentHeight-1] = nil
	}
}

func (pm *ProcessManager) getProcessDepth(originalProcessName string) map[uint32]*Process {

	depthTable, ok := pm.crossTable.Load(originalProcessName)
	if ok {
		return depthTable.(map[uint32]*Process)
	}
	return nil
}

// =============================== utils functions =============================

// getProcessWithRoundRobin get peer with Round Robin algorithm, which is equally get next peer
func (pm *ProcessManager) getProcessWithRoundRobin(processBalance *ProcessBalance) *Process {
	// loop entire backends to find out an Alive backend
	next := pm.nextIndex(processBalance)
	process, ok := processBalance.processes.Load(next)
	if ok {
		return process.(*Process)
	}

	return nil
}

// getProcessWithLeastSize get a process with the smallest size
func (pm *ProcessManager) getProcessWithLeastSize(processBalance *ProcessBalance) *Process {

	nextIdx := 0
	minSize := processWaitingQueueSize + 1
	f := func(k, v interface{}) bool {
		if v.(*Process).Size() < minSize {
			minSize = v.(*Process).Size()
			nextIdx = k.(int)
		}

		return true
	}

	processBalance.processes.Range(f)

	atomic.StoreUint64(&processBalance.curIdx, uint64(nextIdx)) // mark the current one
	process, _ := processBalance.processes.Load(nextIdx)
	return process.(*Process)
}

func (pm *ProcessManager) nextIndex(processBalance *ProcessBalance) int {
	return int(atomic.AddUint64(&processBalance.curIdx, uint64(1)) % uint64(processBalance.size))
}

func getMaxPeer() int {
	mc := os.Getenv(config.ENV_MAX_CONCURRENCY)
	maxConcurrency, err := strconv.Atoi(mc)
	if err != nil {
		maxConcurrency = defaultMaxProcess
	}
	return maxConcurrency
}

func getProcessLimitSize() int {
	batchSize := runtime.NumCPU() * 4
	processLimitSize := batchSize / maxProcess
	if processLimitSize < defaultMaxWaitingQueueSize {
		return defaultMaxWaitingQueueSize
	}
	return processLimitSize
}

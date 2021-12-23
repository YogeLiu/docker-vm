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

	protocol2 "chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
	"go.uber.org/zap"
)

// balancer strategy
const (
	// SRoundRobin every time, launch max num backend,
	// and then using round-robin to feed txs
	SRoundRobin = iota

	// SLeast first launch one backend, when reaching the 2/3 capacity, launch the second one
	// using the least connections' strategy to feed txs
	SLeast
)

const (
	defaultMaxPeer = 10
	minPeerLimit   = 5
)

var (
	// max process num for same contract
	maxPeer int
	// increase process condition: when reach limit, add new process
	peerLimit int
)

// PeerBalance control load balance of process which related to same contract
// key of map is processGroup: contractName:contractVersion
// value of map is peerBalance: including a list of process, each name is contractName:contractVersion:index
type PeerBalance struct {
	peers    []*Process
	curIdx   uint64
	size     int
	strategy int
}

// PeerDepth control cross contract
// key of map is processName: contractName:contractVersion:index
// value of map is peerDepth: including a list of process, the first one is original process
// the reset are cross contract process
type PeerDepth struct {
	peers [protocol2.CallContractDepth + 1]*Process
	size  int
}

type ProcessManager struct {
	logger       *zap.SugaredLogger
	balanceTable map[string]*PeerBalance
	depthTable   map[string]*PeerDepth
	crossTable   sync.Map
	mutex        sync.Mutex
}

func NewProcessManager() *ProcessManager {
	pmLogger := logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER, config.DockerLogDir)

	maxPeer = getMaxPeer()
	peerLimit = getProcessLimitSize()
	pmLogger.Infof("init process manager with max concurrency [%d], limit size [%d]", maxPeer, peerLimit)

	return &ProcessManager{
		logger:       pmLogger,
		balanceTable: make(map[string]*PeerBalance),
		depthTable:   make(map[string]*PeerDepth),
		crossTable:   sync.Map{},
	}
}

// SetStrategy
func (pm *ProcessManager) setStrategy(key string, _strategy int) {

	pm.logger.Infof("set process manager strategy [%d]", _strategy)

	balance, ok := pm.balanceTable[key]
	if ok {
		balance.strategy = _strategy
	}
}

// RegisterNewProcess register new original process
// @param: processNamePrefix: contract:version
// after register, processName:contract:version#prefix
func (pm *ProcessManager) RegisterNewProcess(processNamePrefix string, process *Process) bool {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	pm.logger.Infof("register new process: [%s]", processNamePrefix)

	success := pm.addPeerIntoBalance(processNamePrefix, process)
	if success {
		pm.addFirstPeerIntoDepth(process.processName, process)
	}
	process.Handler.processName = process.processName

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

	pm.removePeerFromBalance(key, idx)
	pm.removeFirstPeerFromDepth(processName)
}

func (pm *ProcessManager) RegisterCrossProcess(initialProcessName string, calledProcess *Process) {
	pm.logger.Debugf("register cross process [%s]", calledProcess.processName)
	pm.crossTable.Store(calledProcess.processName, calledProcess)
	pm.addPeerIntoDepth(initialProcessName, calledProcess)
}

func (pm *ProcessManager) ReleaseCrossProcess(crossProcessName string, initialProcessName string, currentHeight uint32) {
	pm.logger.Debugf("release cross process [%s]", crossProcessName)
	pm.crossTable.Delete(crossProcessName)
	pm.removePeerFromDepth(initialProcessName, currentHeight)
}

func (pm *ProcessManager) GetPeer(processName string) *Process {
	pm.logger.Debugf("get process [%s]", processName)
	nameList := strings.Split(processName, "#")

	if len(nameList) == 1 {
		cp, _ := pm.crossTable.Load(processName)
		crossProcess := cp.(*Process)
		return crossProcess
	}

	key := nameList[0]
	idx, _ := strconv.Atoi(nameList[1])

	peerBalance := pm.getPeerBalance(key)
	return peerBalance.peers[idx]
}

// GetAvailableProcess return one process from peer balance based on current strategy
func (pm *ProcessManager) GetAvailableProcess(processKey string) *Process {
	return pm.getPeerFromBalance(processKey)
}

// ========================= Peer Balance functions ===============================

func (pm *ProcessManager) addPeerIntoBalance(key string, peer *Process) bool {

	peerBalance, ok := pm.balanceTable[key]

	if !ok {
		balance := &PeerBalance{
			peers:    make([]*Process, maxPeer),
			curIdx:   0,
			size:     0,
			strategy: SLeast,
		}
		peer.processName = fmt.Sprintf("%s#%d", peer.processName, 0)
		balance.peers[0] = peer
		balance.size++

		pm.balanceTable[key] = balance
		pm.logger.Infof("add process into balance [%s]", peer.processName)
		return true
	}

	curSize := peerBalance.size

	if curSize < maxPeer {
		peer.processName = fmt.Sprintf("%s#%d", peer.processName, curSize)
		peerBalance.peers[curSize] = peer
		peerBalance.size++

		if peerBalance.size == maxPeer {
			pm.setStrategy(key, SRoundRobin)
		}

		pm.logger.Infof("add process into balance [%s]", peer.processName)

		return true
	}

	return false

}

// GetPeerFromBalance get next peer based on key
// based on different strategy, using different method to get next
// if peer list just have one peer, always return it
// when this peer reach limit, scheduler will generate new peer, limit May 2/3 of capacity or 4/5 of capacity
// then function will return this new peer always, because we should feed this new process -- using lease size function
// when return peer reach limit, then generate new peer, do above process
// eventually, peer list reach limit, and return peer also reach limit, using round-robin algorithm to return peer
// get one peer from a group of peers
func (pm *ProcessManager) getPeerFromBalance(key string) *Process {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// peer list just have one peer, always return it
	// until returned peer reach limit, will generate new peer into balancer
	peerBalance, ok := pm.balanceTable[key]
	if !ok {
		return nil
	}

	if peerBalance.size == 1 {
		return peerBalance.peers[0]
	}

	// peer list contains 1 ~ maxPeer peers, return lease size peer
	// when returned peer reach limit, will generate new peer into balancer
	// until returned peer also reach limit, set strategy as SRoundRobin
	if peerBalance.size <= maxPeer && peerBalance.strategy == SLeast {
		return pm.getNextPeerLeastSize(peerBalance)
	}

	// peer list is full, using Round-Robin to return peer
	// at this time, strategy == SRoundRobin
	// when one peer timeout, release this peer, reset strategy as SLeast
	// so using lease size to return peer
	return pm.getNextPeerRoundRobin(peerBalance)

}

func (pm *ProcessManager) getPeerBalance(key string) *PeerBalance {
	peerBalance, ok := pm.balanceTable[key]
	if ok {
		return peerBalance
	}
	return nil
}

func (pm *ProcessManager) removePeerFromBalance(key string, idx int) {
	peerBalance, ok := pm.balanceTable[key]
	if !ok {
		return
	}

	peerBalance.peers[idx] = nil
	peerBalance.size--
	peerBalance.strategy = SLeast

	if peerBalance.size == 0 {
		delete(pm.balanceTable, key)
	}
}

// ============================== Peer Depth functions ================================

func (pm *ProcessManager) addFirstPeerIntoDepth(processName string, process *Process) {
	pm.logger.Debugf("add first peer into depth [%s]", process.processName)
	newPeerDepth := &PeerDepth{
		peers: [protocol2.CallContractDepth + 1]*Process{},
		size:  0,
	}
	newPeerDepth.peers[0] = process
	newPeerDepth.size++
	pm.depthTable[processName] = newPeerDepth
}

func (pm *ProcessManager) removeFirstPeerFromDepth(processName string) {
	pm.logger.Debugf("remove first peer from depth [%s]", processName)
	delete(pm.depthTable, processName)
}

func (pm *ProcessManager) addPeerIntoDepth(initialProcessName string, calledProcess *Process) {

	pm.logger.Debugf("add cross process %s with initial process name %s",
		calledProcess.processName, initialProcessName)

	peerDepth, ok := pm.depthTable[initialProcessName]
	if ok {
		peerDepth.peers[peerDepth.size] = calledProcess
		peerDepth.size += 1
	}

}

func (pm *ProcessManager) removePeerFromDepth(initialProcessName string, currentHeight uint32) {
	if currentHeight == 0 {
		return
	}
	pm.logger.Debugf("release cross process with position: %v", currentHeight)
	peerDepth, ok := pm.depthTable[initialProcessName]
	if ok {
		peerDepth.peers[currentHeight] = nil
		peerDepth.size -= 1
	}
}

func (pm *ProcessManager) getPeerDepth(initialProcessName string) *PeerDepth {
	pd, ok := pm.depthTable[initialProcessName]
	if ok {
		return pd
	}
	return nil
}

// =============================== utils functions =============================

// getNextPeerRoundRobin get peer with Round Robin algorithm, which is equally get next peer
func (pm *ProcessManager) getNextPeerRoundRobin(group *PeerBalance) *Process {

	if group.size == 0 {
		return nil
	}

	if group.size == 1 {
		return group.peers[0]
	}

	// loop entire backends to find out an Alive backend
	next := pm.nextIndex(group)
	l := len(group.peers) + next // start from next and move a full cycle
	for i := next; i < l; i++ {
		idx := i % len(group.peers) // take an index by modding with length
		// if we have an alive backend, use it and store if its not the original one
		if group.peers[idx] != nil {
			atomic.StoreUint64(&group.curIdx, uint64(idx)) // mark the current one
			return group.peers[idx]
		}
	}
	return nil
}

// getNextPeerLeastSize get next peer with the smallest size
func (pm *ProcessManager) getNextPeerLeastSize(group *PeerBalance) *Process {

	nextIdx := 0
	minSize := processWaitingQueueSize + 1

	for i := 0; i < len(group.peers); i++ {
		curPeer := group.peers[i]
		if group.peers[i] != nil && curPeer.Size() < minSize {
			minSize = curPeer.Size()
			nextIdx = i
		}
	}

	atomic.StoreUint64(&group.curIdx, uint64(nextIdx)) // mark the current one
	return group.peers[nextIdx]
}

func (pm *ProcessManager) nextIndex(group *PeerBalance) int {
	return int(atomic.AddUint64(&group.curIdx, uint64(1)) % uint64(group.size))
}

func getMaxPeer() int {
	mc := os.Getenv(config.ENV_MAX_CONCURRENCY)
	maxConcurrency, err := strconv.Atoi(mc)
	if err != nil {
		maxConcurrency = defaultMaxPeer
	}
	return maxConcurrency
}

func getProcessLimitSize() int {
	batchSize := runtime.NumCPU() * 4
	processLimitSize := batchSize / maxPeer
	if processLimitSize < minPeerLimit {
		return minPeerLimit
	}
	return processLimitSize
}

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
	// SRoundRobin every time, launch max num backend,
	// and then using round-robin to feed txs
	SRoundRobin = iota

	// SLeast first launch one backend, when reaching the 2/3 capacity, launch the second one
	// using the least connections' strategy to feed txs
	SLeast
)

const (
	defaultMaxPeer = 10
	minPeerLimit   = 2
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
	peers    sync.Map
	curIdx   uint64
	strategy int
	size     int
}

//// PeerDepth control cross contract
//// key of map is processName: contractName:contractVersion:index
//// value of map is peerDepth: including a list of process, the first one is original process
//// the reset are cross contract process
//type PeerDepth struct {
//	peers [protocol2.CallContractDepth + 1]*Process
//	size  int
//}

type ProcessManager struct {
	logger *zap.SugaredLogger
	mutex  sync.Mutex

	// map[string]*PeerBalance:
	// string: processNamePrefix: contractName:contractVersion
	// peerBalance:
	// sync.Map: map[index]*Process
	// int: index
	// process: original process: contractName:contractVersion#index
	balanceTable sync.Map

	// map[string1]map[string]*CrossProcess  sync.Map[map]
	// string1: originalProcessName: contractName:contractVersion#index#count - related to tx
	// string2: height
	// process: cross process: txId:height#count
	depthTable sync.Map

	// map[string]*CrossProcess
	// string: contractName:contractVersion#index#count -- related to tx
	// *CrossProcess: txId:height#count
	crossTable sync.Map

	//balanceTable map[string]*PeerBalance
	//depthTable map[string]*PeerDepth

}

func NewProcessManager() *ProcessManager {
	pmLogger := logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER, config.DockerLogDir)

	maxPeer = getMaxPeer()
	peerLimit = getProcessLimitSize()
	pmLogger.Infof("init process manager with max concurrency [%d], limit size [%d]", maxPeer, peerLimit)

	return &ProcessManager{
		logger:       pmLogger,
		balanceTable: sync.Map{},
		depthTable:   sync.Map{},
		crossTable:   sync.Map{},
	}
}

// SetStrategy
func (pm *ProcessManager) setStrategy(key string, _strategy int) {

	pm.logger.Infof("set process manager strategy [%d]", _strategy)

	balance, ok := pm.balanceTable.Load(key)
	if ok {
		balance.(*PeerBalance).strategy = _strategy
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

	pm.removePeerFromBalance(key, idx)
}

func (pm *ProcessManager) RegisterCrossProcess(originalProcessName string, height uint32, calledProcess *Process) {
	pm.logger.Debugf("register cross process [%s], original process name [%s]", calledProcess.processName, originalProcessName)
	pm.crossTable.Store(originalProcessName, calledProcess)
	pm.addPeerIntoDepth(originalProcessName, height, calledProcess)
	calledProcess.Handler.processName = calledProcess.processName
}

func (pm *ProcessManager) ReleaseCrossProcess(crossProcessName string, originalProcessName string, currentHeight uint32) {
	pm.logger.Debugf("release cross process [%s], original process name [%s]", crossProcessName, originalProcessName)
	pm.crossTable.Delete(originalProcessName)
	pm.removePeerFromDepth(originalProcessName, currentHeight)
}

// GetPeer retrieve peer from process manager, could be original process or cross process:
// contractName:contractVersion#index#count
// original process: contractName:contractVersion#index
func (pm *ProcessManager) GetPeer(processName string) *Process {
	pm.logger.Debugf("get process [%s]", processName)
	nameList := strings.Split(processName, "#")

	if len(nameList) == 3 {
		cp, _ := pm.crossTable.Load(processName)
		crossProcess := cp.(*Process)
		return crossProcess
	}

	key := nameList[0]
	idx, _ := strconv.Atoi(nameList[1])

	peerBalance := pm.getPeerBalance(key)
	peer, _ := peerBalance.peers.Load(idx)
	return peer.(*Process)
}

// GetAvailableProcess return one process from peer balance based on current strategy
func (pm *ProcessManager) GetAvailableProcess(processKey string) *Process {
	return pm.getPeerFromBalance(processKey)
}

// ========================= Peer Balance functions ===============================

func (pm *ProcessManager) addPeerIntoBalance(key string, peer *Process) bool {

	peerBalance, ok := pm.balanceTable.Load(key)

	if !ok {
		balance := &PeerBalance{
			peers:    sync.Map{},
			curIdx:   0,
			strategy: SLeast,
			size:     0,
		}
		peer.processName = fmt.Sprintf("%s#%d", peer.processName, 0)
		balance.peers.Store(0, peer)

		pm.balanceTable.Store(key, balance)
		pm.logger.Debugf("add process into balance [%s]", peer.processName)
		return true
	}

	pb := peerBalance.(*PeerBalance)
	curSize := pb.size

	if curSize < maxPeer {
		peer.processName = fmt.Sprintf("%s#%d", peer.processName, curSize)
		pb.peers.Store(curSize, peer)
		pb.size++

		if pb.size == maxPeer {
			pm.setStrategy(key, SRoundRobin)
		}

		pm.logger.Debugf("add process into balance [%s]", peer.processName)

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
	pb, ok := pm.balanceTable.Load(key)
	if !ok {
		return nil
	}
	peerBalance := pb.(*PeerBalance)

	if peerBalance.size == 1 {
		curPb, _ := peerBalance.peers.Load(0)
		return curPb.(*Process)
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
	peerBalance, ok := pm.balanceTable.Load(key)
	if ok {
		return peerBalance.(*PeerBalance)
	}
	return nil
}

func (pm *ProcessManager) removePeerFromBalance(key string, idx int) {
	pb, ok := pm.balanceTable.Load(key)
	if !ok {
		return
	}

	peerBalance := pb.(*PeerBalance)

	peerBalance.peers.Delete(idx)
	peerBalance.size--
	peerBalance.strategy = SLeast

	if peerBalance.size == 0 {
		//delete(pm.balanceTable, key)
		pm.balanceTable.Delete(key)
	}
}

// ============================== Peer Depth functions ================================

//func (pm *ProcessManager) addFirstPeerIntoDepth(processName string, process *Process) {
//	pm.logger.Debugf("add first peer into depth [%s]", process.processName)
//	newPeerDepth := &PeerDepth{
//		peers: [protocol2.CallContractDepth + 1]*Process{},
//		size:  0,
//	}
//	newPeerDepth.peers[0] = process
//	newPeerDepth.size++
//	pm.depthTable[processName] = newPeerDepth
//}
//
//func (pm *ProcessManager) removeFirstPeerFromDepth(processName string) {
//	pm.logger.Debugf("remove first peer from depth [%s]", processName)
//	delete(pm.depthTable, processName)
//}

func (pm *ProcessManager) addPeerIntoDepth(originalProcessName string, height uint32, calledProcess *Process) {

	pm.logger.Debugf("add cross process %s with initial process name %s",
		calledProcess.processName, originalProcessName)

	peerDepth, ok := pm.depthTable.Load(originalProcessName)
	if !ok {
		//newPeerDepth := &PeerDepth{
		//	peers: [protocol2.CallContractDepth + 1]*Process{},
		//	size:  0,
		//}

		newPeerDepth := make(map[uint32]*Process)
		newPeerDepth[height] = calledProcess
		pm.depthTable.Store(originalProcessName, newPeerDepth)

		peerDepth = newPeerDepth
	}

	//pm.mutex.Lock()
	//defer pm.mutex.Unlock()

	pd := peerDepth.(map[uint32]*Process)
	pd[height] = calledProcess

}

func (pm *ProcessManager) removePeerFromDepth(originalProcessName string, currentHeight uint32) {
	if currentHeight == 0 {
		return
	}

	pm.logger.Debugf("release cross process with position: %v", currentHeight)

	//pm.mutex.Lock()
	//defer pm.mutex.Unlock()

	peerDepth, ok := pm.depthTable.Load(originalProcessName)
	if ok {
		pd := peerDepth.(map[uint32]*Process)
		delete(pd, currentHeight)
		//peerDepth.peers[currentHeight-1] = nil
	}
}

func (pm *ProcessManager) getPeerDepth(originalProcessName string) map[uint32]*Process {

	depthTable, ok := pm.depthTable.Load(originalProcessName)
	if ok {
		return depthTable.(map[uint32]*Process)
	}

	//pd, ok := pm.depthTable[initialProcessName]
	//if ok {
	//	return pd
	//}
	return nil
}

// =============================== utils functions =============================

// getNextPeerRoundRobin get peer with Round Robin algorithm, which is equally get next peer
func (pm *ProcessManager) getNextPeerRoundRobin(group *PeerBalance) *Process {

	if group.size == 0 {
		return nil
	}

	if group.size == 1 {
		peer, _ := group.peers.Load(0)
		return peer.(*Process)
	}

	// loop entire backends to find out an Alive backend
	next := pm.nextIndex(group)
	process, ok := group.peers.Load(next)
	if ok {
		return process.(*Process)
	}
	//l := group.size + next // start from next and move a full cycle
	//for i := next; i < l; i++ {
	//	idx := i % group.size // take an index by modding with length
	//	// if we have an alive backend, use it and store if its not the original one
	//	if group.peers[idx] != nil {
	//		atomic.StoreUint64(&group.curIdx, uint64(idx)) // mark the current one
	//		return group.peers[idx]
	//	}
	//}
	return nil
}

// getNextPeerLeastSize get next peer with the smallest size
func (pm *ProcessManager) getNextPeerLeastSize(group *PeerBalance) *Process {

	nextIdx := 0
	minSize := processWaitingQueueSize + 1

	curIdx := 0
	f := func(k, v interface{}) bool {
		curIdx++
		if v.(*Process).Size() < minSize {
			minSize = v.(*Process).Size()
			nextIdx = k.(int)
		}

		return true
	}

	group.peers.Range(f)

	//for i := 0; i < group.size; i++ {
	//	curPeer := group.peers.Load(i)
	//	if group.peers[i] != nil && curPeer.Size() < minSize {
	//		minSize = curPeer.Size()
	//		nextIdx = i
	//	}
	//}

	atomic.StoreUint64(&group.curIdx, uint64(nextIdx)) // mark the current one
	peer, _ := group.peers.Load(nextIdx)
	return peer.(*Process)
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

package core

//
//import (
//	"sync"
//	"testing"
//
//	"github.com/stretchr/testify/assert"
//)
//
//func TestProcessManager_SetStrategy(t *testing.T) {
//
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	b0, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, SLeast, b0.strategy)
//
//	processManager.setStrategy(processNamePrefix, SRoundRobin)
//
//	b1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, SRoundRobin, b1.strategy)
//
//	processManager.setStrategy(processNamePrefix, SLeast)
//
//	b2, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, SLeast, b2.strategy)
//}
//
//func TestProcessManager_RegisterNewProcess(t *testing.T) {
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	balance, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance.size)
//	assert.Equal(t, "contract:1.0#0", balance.peers[0].processName)
//
//	peerDepth, _ := processManager.depthTable["contract:1.0#0"]
//	assert.Equal(t, 1, peerDepth.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth.peers[0].processName)
//
//	// test1
//	process1 := newProcess(processNamePrefix, false)
//
//	processManager.RegisterNewProcess(processNamePrefix, process1)
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 2, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.peers[0].processName)
//	assert.Equal(t, "contract:1.0#1", balance1.peers[1].processName)
//
//	peerDepth1, _ := processManager.depthTable["contract:1.0#1"]
//	assert.Equal(t, 1, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#1", peerDepth1.peers[0].processName)
//}
//
//func TestProcessManager_ReleaseProcess(t *testing.T) {
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	process1 := newProcess(processNamePrefix, false)
//	processManager.RegisterNewProcess(processNamePrefix, process1)
//
//	processManager.ReleaseProcess("contract:1.0#0")
//
//	balance0, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance0.size)
//	assert.Equal(t, "contract:1.0#1", balance0.peers[1].processName)
//	assert.Nil(t, balance0.peers[0])
//	_, ok := processManager.depthTable["contract:1.0#0"]
//	assert.False(t, ok)
//	peerDepth0, ok := processManager.depthTable["contract:1.0#1"]
//	assert.True(t, ok)
//	assert.Equal(t, 1, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#1", peerDepth0.peers[0].processName)
//
//	processManager.ReleaseProcess("contract:1.0#1")
//	_, ok = processManager.balanceTable[processNamePrefix]
//	assert.False(t, ok)
//	_, ok = processManager.depthTable["contract:1.0#1"]
//	assert.False(t, ok)
//}
//
//func TestProcessManager_RegisterCrossProcess(t *testing.T) {
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	crossProcess0 := newProcess("cross0", true)
//	processManager.RegisterCrossProcess("contract:1.0#0", crossProcess0)
//
//	balance0, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance0.size)
//	assert.Equal(t, "contract:1.0#0", balance0.peers[0].processName)
//
//	peerDepth0, ok := processManager.depthTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 2, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth0.peers[0].processName)
//	assert.Equal(t, "cross0", peerDepth0.peers[1].processName)
//
//	// test1
//	crossProcess1 := newProcess("cross1", true)
//	processManager.RegisterCrossProcess("contract:1.0#0", crossProcess1)
//
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.peers[0].processName)
//
//	peerDepth1, ok := processManager.depthTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 3, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth1.peers[0].processName)
//	assert.Equal(t, "cross0", peerDepth1.peers[1].processName)
//	assert.Equal(t, "cross1", peerDepth1.peers[2].processName)
//
//}
//
//func TestProcessManager_ReleaseCrossProcess(t *testing.T) {
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	crossProcess0 := newProcess("cross0", true)
//	processManager.RegisterCrossProcess("contract:1.0#0", crossProcess0)
//
//	crossProcess1 := newProcess("cross1", true)
//	processManager.RegisterCrossProcess("contract:1.0#0", crossProcess1)
//
//	processManager.ReleaseCrossProcess("cross1", "contract:1.0#0", 2)
//
//	balance0, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance0.size)
//	assert.Equal(t, "contract:1.0#0", balance0.peers[0].processName)
//
//	peerDepth0, ok := processManager.depthTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 2, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth0.peers[0].processName)
//	assert.Equal(t, "cross0", peerDepth0.peers[1].processName)
//
//	processManager.ReleaseCrossProcess("cross0", "contract:1.0#0", 1)
//
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.peers[0].processName)
//
//	peerDepth1, ok := processManager.depthTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 1, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth1.peers[0].processName)
//
//}
//
//func TestProcessManager_GetAvailableProcess(t *testing.T) {
//	maxPeer = 3
//	processManager := NewProcessManager()
//
//	processNamePrefix := "contract:1.0"
//
//	// test0
//	process0 := newProcess(processNamePrefix, false)
//	process1 := newProcess(processNamePrefix, false)
//	process2 := newProcess(processNamePrefix, false)
//
//	processManager.RegisterNewProcess(processNamePrefix, process0)
//
//	result := processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//
//	processManager.RegisterNewProcess(processNamePrefix, process1)
//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//	balance0, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(0), balance0.curIdx)
//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(0), balance0.curIdx)
//	//
//	process0.waitingQueueSize = 100
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#1", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(1), balance0.curIdx)
//	//
//	process1.waitingQueueSize = 200
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(0), balance0.curIdx)
//	//
//	process0.waitingQueueSize = 200
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(0), balance0.curIdx)
//	//
//	processManager.RegisterNewProcess(processNamePrefix, process2)
//	//processManager.SetStrategy(processNamePrefix, SRoundRobin)
//	//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#1", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, SRoundRobin, balance0.strategy)
//	assert.Equal(t, uint64(1), balance0.curIdx)
//	//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#2", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(2), balance0.curIdx)
//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#0", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, uint64(0), balance0.curIdx)
//
//}
//
//func newProcess(processName string, isCross bool) *Process {
//	return &Process{
//		processName:      processName,
//		isCrossProcess:   isCross,
//		contractName:     "",
//		contractVersion:  "",
//		contractPath:     "",
//		cGroupPath:       "",
//		user:             nil,
//		cmd:              nil,
//		ProcessState:     0,
//		TxWaitingQueue:   nil,
//		waitingQueueSize: 0,
//		txTrigger:        nil,
//		expireTimer:      nil,
//		Handler:          nil,
//		logger:           nil,
//		processMgr:       nil,
//		done:             0,
//		mutex:            sync.Mutex{},
//	}
//}

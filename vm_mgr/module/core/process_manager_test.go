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
//	assert.Equal(t, StrategyLeastSize, b0.strategy)
//
//	processManager.setStrategy(processNamePrefix, StrategyRoundRobin)
//
//	b1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, StrategyRoundRobin, b1.strategy)
//
//	processManager.setStrategy(processNamePrefix, StrategyLeastSize)
//
//	b2, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, StrategyLeastSize, b2.strategy)
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
//	assert.Equal(t, "contract:1.0#0", balance.processes[0].processName)
//
//	peerDepth, _ := processManager.crossTable["contract:1.0#0"]
//	assert.Equal(t, 1, peerDepth.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth.processes[0].processName)
//
//	// test1
//	process1 := newProcess(processNamePrefix, false)
//
//	processManager.RegisterNewProcess(processNamePrefix, process1)
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 2, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.processes[0].processName)
//	assert.Equal(t, "contract:1.0#1", balance1.processes[1].processName)
//
//	peerDepth1, _ := processManager.crossTable["contract:1.0#1"]
//	assert.Equal(t, 1, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#1", peerDepth1.processes[0].processName)
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
//	assert.Equal(t, "contract:1.0#1", balance0.processes[1].processName)
//	assert.Nil(t, balance0.processes[0])
//	_, ok := processManager.crossTable["contract:1.0#0"]
//	assert.False(t, ok)
//	peerDepth0, ok := processManager.crossTable["contract:1.0#1"]
//	assert.True(t, ok)
//	assert.Equal(t, 1, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#1", peerDepth0.processes[0].processName)
//
//	processManager.ReleaseProcess("contract:1.0#1")
//	_, ok = processManager.balanceTable[processNamePrefix]
//	assert.False(t, ok)
//	_, ok = processManager.crossTable["contract:1.0#1"]
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
//	assert.Equal(t, "contract:1.0#0", balance0.processes[0].processName)
//
//	peerDepth0, ok := processManager.crossTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 2, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth0.processes[0].processName)
//	assert.Equal(t, "cross0", peerDepth0.processes[1].processName)
//
//	// test1
//	crossProcess1 := newProcess("cross1", true)
//	processManager.RegisterCrossProcess("contract:1.0#0", crossProcess1)
//
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.processes[0].processName)
//
//	peerDepth1, ok := processManager.crossTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 3, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth1.processes[0].processName)
//	assert.Equal(t, "cross0", peerDepth1.processes[1].processName)
//	assert.Equal(t, "cross1", peerDepth1.processes[2].processName)
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
//	assert.Equal(t, "contract:1.0#0", balance0.processes[0].processName)
//
//	peerDepth0, ok := processManager.crossTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 2, peerDepth0.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth0.processes[0].processName)
//	assert.Equal(t, "cross0", peerDepth0.processes[1].processName)
//
//	processManager.ReleaseCrossProcess("cross0", "contract:1.0#0", 1)
//
//	balance1, _ := processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, 1, balance1.size)
//	assert.Equal(t, "contract:1.0#0", balance1.processes[0].processName)
//
//	peerDepth1, ok := processManager.crossTable["contract:1.0#0"]
//	assert.True(t, ok)
//	assert.Equal(t, 1, peerDepth1.size)
//	assert.Equal(t, "contract:1.0#0", peerDepth1.processes[0].processName)
//
//}
//
//func TestProcessManager_GetAvailableProcess(t *testing.T) {
//	maxProcess = 3
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
//	//processManager.SetStrategy(processNamePrefix, StrategyRoundRobin)
//	//
//	result = processManager.GetAvailableProcess(processNamePrefix)
//	assert.Equal(t, "contract:1.0#1", result.processName)
//	balance0, _ = processManager.balanceTable[processNamePrefix]
//	assert.Equal(t, StrategyRoundRobin, balance0.strategy)
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
//		nextTxTrigger:        nil,
//		expireTimer:      nil,
//		Handler:          nil,
//		log:           nil,
//		processMgr:       nil,
//		done:             0,
//		balanceRWMutex:            sync.Mutex{},
//	}
//}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"sync"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"github.com/emirpasic/gods/maps/linkedhashmap"

	"go.uber.org/zap"
)

const (
	processManagerEventChSize = 15000
)

type ProcessManager struct {
	logger *zap.SugaredLogger
	lock   sync.RWMutex

	maxProcessNum  int
	releaseRate    float64
	isCrossManager bool

	idleProcesses        *linkedhashmap.Map            // process name -> idle Process
	busyProcesses        map[string]interfaces.Process // process name -> busy Process
	processGroups        map[string]map[string]bool    // group key -> Process name set
	waitingRequestGroups *linkedhashmap.Map            // group key -> bool

	eventCh    chan interface{}
	cleanTimer *time.Timer

	userManager interfaces.UserManager
	requestScheduler interfaces.RequestScheduler
}

type requestGroupKey struct {
	contractName    string
	contractVersion string
}

func NewProcessManager(maxProcessNum int, rate float64, isCrossManager bool, userManager interfaces.UserManager) *ProcessManager {
	return &ProcessManager{
		logger: logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER),
		lock:   sync.RWMutex{},

		maxProcessNum:  maxProcessNum,
		releaseRate:    rate,
		isCrossManager: isCrossManager,

		idleProcesses:        linkedhashmap.New(),
		busyProcesses:        make(map[string]interfaces.Process),
		processGroups:        make(map[string]map[string]bool),
		waitingRequestGroups: linkedhashmap.New(),

		eventCh:    make(chan interface{}, processManagerEventChSize),
		cleanTimer: time.NewTimer(config.DockerVMConfig.GetReleasePeriod()),

		userManager: userManager,
	}
}

func (pm *ProcessManager) Start() {
	go func() {
		select {
		case msg := <-pm.eventCh:
			switch msg.(type) {

			case *messages.GetProcessReqMsg:
				m, _ := msg.(*messages.GetProcessReqMsg)
				groupKey := utils.ConstructRequestGroupKey(m.ContractName, m.ContractVersion)
				pm.logger.Debugf("%s requests to get %d process(es)", groupKey, m.ProcessNum)
				pm.handleGetProcessReq(m)

			//case *messages.LaunchSandboxRespMsg:
			//	m, _ := msg.(*messages.LaunchSandboxRespMsg)
			//	pm.logger.Debugf("recieved launch sandbox resp from %s", m.ProcessName)
			//	if err := pm.handleLaunchSandboxResp(m); err != nil {
			//		pm.logger.Errorf("failed to handle launch sandbox resp, %v", err)
			//	}
			//
			//case *messages.ChangeSandboxRespMsg:
			//	m, _ := msg.(*messages.ChangeSandboxRespMsg)
			//	pm.logger.Debugf("recieved change sandbox resp from %s", m.ProcessName)
			//	if err := pm.handleChangeSandboxResp(m); err != nil {
			//		pm.logger.Errorf("failed to handle launch sandbox resp, %v", err)
			//	}
			//
			//case *messages.CloseSandboxRespMsg:
			//	m, _ := msg.(*messages.CloseSandboxRespMsg)
			//	pm.handleCloseSandboxResp(m)

			case *messages.SandboxExitRespMsg:
				m, _ := msg.(*messages.SandboxExitRespMsg)
				pm.handleSandboxExitResp(m)

			default:
				pm.logger.Errorf("unknown msg type")

			}
		case <-pm.cleanTimer.C:
			pm.handleCleanIdleProcesses()

		}
	}()
}

func (pm *ProcessManager) SetScheduler(scheduler interfaces.RequestScheduler) {
	pm.requestScheduler = scheduler
}

// PutMsg put invoking requests into chan, waiting for process manager to handle request
//  @param msg types include GetProcessReqMsg, LaunchSandboxRespMsg, ChangeSandboxRespMsg and CloseSandboxRespMsg
func (pm *ProcessManager) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage, messages.GetProcessReqMsg, messages.LaunchSandboxRespMsg, messages.ChangeSandboxReqMsg, messages.CloseSandboxRespMsg:
		pm.eventCh <- msg
	default:
		pm.logger.Errorf("unknown msg type")
	}
	return nil
}

// handleGetProcessReq handle get process request from request group
func (pm *ProcessManager) handleGetProcessReq(msg *messages.GetProcessReqMsg) {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	needProcessNum := msg.ProcessNum
	availableProcessNum := pm.getAvailableProcessNum()

	// firstly, allocate processes that can be launch
	if availableProcessNum > 0 {
		newProcessNum := needProcessNum
		// available process num not enough
		if availableProcessNum < needProcessNum {
			newProcessNum = availableProcessNum
		}
		// create new process concurrently
		var wg sync.WaitGroup
		for i := 0; i < newProcessNum; i++ {
			wg.Add(1)
			i := i
			go func() {
				processName := utils.ConstructProcessName(msg.ContractName, msg.ContractVersion, i)
				process, err := pm.createNewProcess(msg.ContractName, msg.ContractVersion, processName)
				if err != nil {
					pm.logger.Errorf("failed to create new process, %v", err)
					wg.Done()
					return
				}
				// add process to busy list and total map
				pm.addProcessToCache(msg.ContractName, msg.ContractVersion, processName, process, true)
				wg.Done()
			}()
		}
		wg.Wait()
		// update the process num still need
		needProcessNum -= newProcessNum
	}

	// secondly, allocate processes from idle processes
	if needProcessNum > 0 {
		newProcessNum := needProcessNum

		// idle process num not enough
		if pm.idleProcesses.Size() < needProcessNum {
			newProcessNum = pm.idleProcesses.Size()
		}

		// idle processes to remove
		removedIdleProcesses := pm.batchPopIdleProcesses(needProcessNum)

		// change processes context concurrently
		var wg sync.WaitGroup
		for i := 0; i < newProcessNum; i++ {
			wg.Add(1)
			i := i
			go func() {
				// generate new process name
				processName := utils.ConstructProcessName(msg.ContractName, msg.ContractVersion, i)
				err := removedIdleProcesses[i].PutMsg(&messages.ChangeSandboxReqMsg{
					ContractName:    msg.ContractName,
					ContractVersion: msg.ContractVersion,
					ProcessName:     processName,
				})
				if err != nil {
					pm.logger.Errorf("failed to change process context, %v", err)
					wg.Done()
					return
				}
				// add process to busy list and total map
				pm.addProcessToCache(msg.ContractName, msg.ContractVersion, processName, removedIdleProcesses[i], true)
				wg.Done()
			}()
		}
		wg.Wait()
		// update the process num still need
		needProcessNum -= newProcessNum
	}

	// no available process, put to waiting request group
	if needProcessNum > 0 {
		groupKey := requestGroupKey{
			contractName:    msg.ContractName,
			contractVersion: msg.ContractVersion,
		}
		if _, ok := pm.waitingRequestGroups.Get(groupKey); !ok {
			pm.waitingRequestGroups.Put(groupKey, true)
			pm.logger.Debugf("put request group %s into waiting request group.",
				utils.ConstructRequestGroupKey(msg.ContractName, msg.ContractVersion))
			return
		}
		pm.logger.Debugf("request group %s existed",
			utils.ConstructRequestGroupKey(msg.ContractName, msg.ContractVersion))
	}
}

//// handleLaunchSandboxResp handle sandbox launch response
//func (pm *ProcessManager) handleLaunchSandboxResp(msg *messages.LaunchSandboxRespMsg) error {
//
//	pm.lock.Lock()
//	defer pm.lock.Unlock()
//
//	if msg.Err != nil {
//		pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
//		return fmt.Errorf("failed to launch sandbox, %v", msg.Err)
//	}
//	return nil
//}
//
//// handleChangeSandboxResp handle sandbox change response
//func (pm *ProcessManager) handleChangeSandboxResp(msg *messages.ChangeSandboxRespMsg) error {
//
//	pm.lock.Lock()
//	defer pm.lock.Unlock()
//
//	if msg.Err != nil {
//		pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
//		return fmt.Errorf("failed to change sandbox, %v", msg.Err)
//	}
//	return nil
//}
//
//// handleCloseSandboxResp handle close sandbox response
//func (pm *ProcessManager) handleCloseSandboxResp(msg *messages.CloseSandboxRespMsg) {
//
//	pm.lock.Lock()
//	defer pm.lock.Unlock()
//
//	pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
//}

// handleCleanIdleProcesses handle sandbox exit response, release user and remove process from cache
func (pm *ProcessManager) handleSandboxExitResp(msg *messages.SandboxExitRespMsg) {
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
	pm.logger.Errorf("sandbox exited, %v", msg.Err)
}

// handleCleanIdleProcesses handle clean idle processes
func (pm *ProcessManager) handleCleanIdleProcesses() {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	// calculate the process num to release
	availableProcessNum := pm.getAvailableProcessNum()
	releaseNum := int(pm.releaseRate * float64(pm.maxProcessNum))
	releaseNum = releaseNum - availableProcessNum
	if releaseNum <= 0 {
		return
	}
	if pm.idleProcesses.Size() < releaseNum {
		releaseNum = pm.idleProcesses.Size()
	}

	// pop the idle processes to be released
	removedIdleProcesses := pm.batchPopIdleProcesses(releaseNum)

	// put close sandbox msg to process
	var wg sync.WaitGroup
	for i := 0; i < releaseNum; i++ {
		wg.Add(1)
		i := i
		go func() {
			// generate new process name
			err := removedIdleProcesses[i].PutMsg(&messages.CloseSandboxReqMsg{})
			if err != nil {
				pm.logger.Errorf("failed to kill process, %v", err)
				wg.Done()
				return
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// start timer
	pm.startTimer()

	pm.logger.Debugf("removed %d idle processes", releaseNum)
	return
}

// closeSandbox release user and process
func (pm *ProcessManager) closeSandbox(contractName, contractVersion, processName string) {
	pm.releaseUser(processName)
	pm.removeProcessFromCache(contractName, contractVersion, processName)
}

// CreateNewProcess create a new process
func (pm *ProcessManager) createNewProcess(contractName, contractVersion, processName string) (interfaces.Process, error) {
	user, err := pm.userManager.GetAvailableUser()
	if err != nil {
		pm.logger.Errorf("fail to get available user, %v", err)
		return nil, err
	}
	var process interfaces.Process
	process = NewProcess(user, contractName, contractVersion, processName, pm, pm.requestScheduler, pm.isCrossManager)
	process.Start()

	return process, nil
}

// release linux user
func (pm *ProcessManager) releaseUser(processName string) {

	pm.logger.Debugf("release process %s", processName)

	process, _ := pm.getProcessByName(processName)
	_ = pm.userManager.AddAvailableUser(process.GetUser())
}

// GetProcessByName returns process by process name
func (pm *ProcessManager) GetProcessByName(processName string) (interfaces.Process, bool) {

	pm.lock.RLock()
	defer pm.lock.RUnlock()

	return pm.getProcessByName(processName)
}

func (pm *ProcessManager) getProcessByName(processName string) (interfaces.Process, bool) {

	if val, ok := pm.idleProcesses.Get(processName); ok {
		return val.(interfaces.Process), true
	}
	if val, ok := pm.busyProcesses[processName]; ok {
		return val, true
	}
	return nil, false
}

// GetProcessNumByContractKey returns process by contractName#contractVersion
func (pm *ProcessManager) GetProcessNumByContractKey(contractName, contractVersion string) int {

	pm.lock.RLock()
	defer pm.lock.RUnlock()

	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)
	if val, ok := pm.processGroups[groupKey]; ok {
		return len(val)
	}
	return 0
}

// ChangeProcessState changes the process state
func (pm *ProcessManager) ChangeProcessState(processName string, toBusy bool) error {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	if toBusy {
		process, ok := pm.idleProcesses.Get(processName)
		if !ok {
			return fmt.Errorf("process not exist in idle processes")
		}
		pm.busyProcesses[processName] = process.(interfaces.Process)
		pm.idleProcesses.Remove(processName)

	} else {
		process, ok := pm.busyProcesses[processName]
		if !ok {
			return fmt.Errorf("process not exist in busy processes")
		}
		pm.idleProcesses.Put(processName, process)
		if err := pm.allocateIdleProcess(); err != nil {
			pm.logger.Warnf("allocate idle process failed, %v", err)
		}
		delete(pm.busyProcesses, processName)
	}
	return nil
}

// batchPopIdleProcesses batch pop idle processes
func (pm *ProcessManager) batchPopIdleProcesses(num int) []*Process {
	// idle processes to remove
	removedIdleProcesses := make([]*Process, num)
	cnt := 0
	pm.idleProcesses.Each(func(key interface{}, value interface{}) {
		if cnt >= num {
			return
		}
		process := value.(*Process)
		removedIdleProcesses = append(removedIdleProcesses, process)
		pm.idleProcesses.Remove(process.GetProcessName())
		pm.removeFromProcessGroup(process.GetContractName(), process.GetContractVersion(), process.GetProcessName())
		cnt++
	})
	return removedIdleProcesses
}

func (pm *ProcessManager) addProcessToCache(contractName, contractVersion, processName string, process interfaces.Process, isBusy bool) {

	if isBusy {
		pm.busyProcesses[processName] = process
	} else {
		pm.idleProcesses.Put(processName, process)
		if err := pm.allocateIdleProcess(); err != nil {
			pm.logger.Warnf("allocate idle process failed, %v", err)
		}
	}

	pm.addToProcessGroup(contractName, contractVersion, processName)
}

// removeProcessFromCache remove process from busyProcesses, idleProcesses and processGroup
func (pm *ProcessManager) removeProcessFromCache(contractName, contractVersion, processName string) {

	delete(pm.busyProcesses, processName)
	pm.idleProcesses.Remove(processName)
	pm.removeFromProcessGroup(contractName, contractVersion, processName)
}

func (pm *ProcessManager) getAvailableProcessNum() int {

	return pm.maxProcessNum - pm.idleProcesses.Size() - len(pm.busyProcesses)
}

// allocateIdleProcess allocate idle process to waiting request groups
func (pm *ProcessManager) allocateIdleProcess() error {

	// calculate allocate num
	allocateNum := pm.waitingRequestGroups.Size()
	if allocateNum == 0 {
		return nil
	}

	if pm.idleProcesses.Size() < allocateNum {
		allocateNum = pm.idleProcesses.Size()
	}

	for i := 0; i < allocateNum; i++ {
		// match the process and request group
		groupIt := pm.waitingRequestGroups.Iterator()
		processIt := pm.idleProcesses.Iterator()

		// get old process, contract name, contract version and process name
		process := processIt.Value().(interfaces.Process)
		contractName := process.GetContractName()
		contractVersion := process.GetContractVersion()
		processName := process.GetProcessName()

		// get new contract name and contract version
		newGroupKey := groupIt.Key().(requestGroupKey)

		// remove request group from queue
		pm.waitingRequestGroups.Remove(newGroupKey)
		// remove idle processes from queue
		pm.idleProcesses.Remove(processName)

		// remove old process from process group
		pm.removeFromProcessGroup(contractName, contractVersion, processName)

		// notify process change sandbox context
		newProcessName := utils.ConstructProcessName(newGroupKey.contractName, newGroupKey.contractVersion, i)
		err := process.PutMsg(&messages.ChangeSandboxReqMsg{
			ContractName:    newGroupKey.contractName,
			ContractVersion: newGroupKey.contractVersion,
			ProcessName:     newProcessName,
		})
		if err != nil {
			return err
		}

		// add process to busy process list
		pm.busyProcesses[newProcessName] = process
		// add process to process group
		pm.addToProcessGroup(newGroupKey.contractName, newGroupKey.contractVersion, newProcessName)
	}
	return nil
}

func (pm *ProcessManager) addToProcessGroup(contractName, contractVersion, processName string) {
	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)

	if _, ok := pm.processGroups[groupKey]; !ok {
		pm.processGroups[groupKey] = make(map[string]bool)
	}
	pm.processGroups[groupKey][processName] = true
}

func (pm *ProcessManager) removeFromProcessGroup(contractName, contractVersion, processName string) {
	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)

	if _, ok := pm.processGroups[groupKey]; !ok {
		return
	}
	delete(pm.processGroups[groupKey], processName)
}

func (pm *ProcessManager) startTimer() {
	pm.logger.Debugf("start clean timer")
	if !pm.cleanTimer.Stop() && len(pm.cleanTimer.C) > 0 {
		<-pm.cleanTimer.C
	}
	pm.cleanTimer.Reset(config.DockerVMConfig.GetReleasePeriod())
}

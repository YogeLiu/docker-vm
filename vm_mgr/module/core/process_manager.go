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
	// processManagerEventChSize is process manager event chan size
	processManagerEventChSize = 15000
)

// ProcessManager manager the life cycle of processes
// there are 2 ProcessManager, one for original process, the other for cross process
type ProcessManager struct {
	logger *zap.SugaredLogger // process scheduler logger
	lock   sync.RWMutex       // process scheduler lock

	maxProcessNum  int     // max process num
	releaseRate    float64 // the minimum rate of available process
	isCrossManager bool    // cross process manager or original process manager

	idleProcesses        *linkedhashmap.Map            // idle processes linked hashmap (process name -> idle Process)
	busyProcesses        map[string]interfaces.Process // busy process map (process name -> busy Process)
	processGroups        map[string]map[string]bool    // process group by contract key (contract key -> Process name set)
	waitingRequestGroups *linkedhashmap.Map            // waiting request groups linked hashmap (group key -> bool)

	eventCh    chan interface{} // process manager event channel
	cleanTimer *time.Timer      // clean timer for release idle processes

	userManager      interfaces.UserManager      // user manager
	requestScheduler interfaces.RequestScheduler // request scheduler
}

// NewProcessManager returns new process manager
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

// Start process manager, listen event chan and clean timer,
// types: messages.GetProcessReqMsg, messages.SandboxExitRespMsg and cleanIdleProcesses timer
func (pm *ProcessManager) Start() {
	go func() {
		for {
			select {
			case msg := <-pm.eventCh:
				switch msg.(type) {

				case *messages.GetProcessReqMsg:
					m, _ := msg.(*messages.GetProcessReqMsg)
					pm.handleGetProcessReq(m)

				case *messages.SandboxExitRespMsg:
					m, _ := msg.(*messages.SandboxExitRespMsg)
					pm.handleSandboxExitResp(m)

				default:
					pm.logger.Errorf("unknown msg type, msg: %+v", msg)

				}
			case <-pm.cleanTimer.C:
				pm.handleCleanIdleProcesses()

			}
		}
	}()
}

// SetScheduler set request scheduler
func (pm *ProcessManager) SetScheduler(scheduler interfaces.RequestScheduler) {
	pm.requestScheduler = scheduler
}

// PutMsg put invoking requests into chan, waiting for process manager to handle request
//  @param req types include GetProcessReqMsg, LaunchSandboxRespMsg, ChangeSandboxRespMsg and CloseSandboxRespMsg
func (pm *ProcessManager) PutMsg(msg interface{}) error {
	switch msg.(type) {
	case *protogo.DockerVMMessage,
		*messages.GetProcessReqMsg,
		*messages.LaunchSandboxRespMsg,
		*messages.ChangeSandboxReqMsg,
		*messages.CloseSandboxRespMsg:
		pm.eventCh <- msg
	default:
		return fmt.Errorf("unknown msg type, msg: %+v", msg)
	}
	return nil
}

// GetProcessByName returns process by process name
func (pm *ProcessManager) GetProcessByName(processName string) (interfaces.Process, bool) {

	pm.lock.RLock()
	defer pm.lock.RUnlock()

	return pm.getProcessByName(processName)
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

// handleGetProcessReq handle get process request from request group
func (pm *ProcessManager) handleGetProcessReq(msg *messages.GetProcessReqMsg) {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	groupKey := utils.ConstructContractKey(msg.ContractName, msg.ContractVersion)
	pm.logger.Debugf("request group %s request to get %d process(es)", groupKey, msg.ProcessNum)

	// do not need any process
	if msg.ProcessNum == 0 {
		//if err := pm.sendProcessReadyResp(0, false, msg.ContractName, msg.ContractVersion); err != nil {
		//	pm.logger.Errorf("failed to send process ready resp, %v", err)
		//}
		pm.waitingRequestGroups.Remove(groupKey)
		pm.logger.Debugf("request group %s does not need more processes, removed from waiting request group", groupKey)

		return
	}

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
			index := i
			go func() {
				// create new process
				processName := utils.ConstructProcessName(msg.ContractName, msg.ContractVersion, index)
				process, err := pm.createNewProcess(msg.ContractName, msg.ContractVersion, processName)
				if err != nil {
					pm.logger.Errorf("failed to create new process, %v", err)
					wg.Done()
					return
				}

				// add process to cache
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
		removedIdleProcesses, err := pm.batchPopIdleProcesses(newProcessNum)
		if err != nil {
			pm.logger.Errorf("failed to batch pop idle processes, %v", err)
		}

		// change processes context concurrently
		var wg sync.WaitGroup
		lock := sync.Mutex{}
		for i := 0; i < newProcessNum; i++ {
			wg.Add(1)
			index := i
			go func() {
				// generate new process name
				processName := utils.ConstructProcessName(msg.ContractName, msg.ContractVersion, index)

				// notify process to change context
				err = removedIdleProcesses[index].PutMsg(&messages.ChangeSandboxReqMsg{
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
				lock.Lock()
				pm.addProcessToCache(msg.ContractName, msg.ContractVersion, processName, removedIdleProcesses[index], true)
				lock.Unlock()
				wg.Done()
			}()
		}
		wg.Wait()
		// update the process num still need
		needProcessNum -= newProcessNum
	}

	// no available process, put to waiting request group
	if needProcessNum > 0 {
		group := &messages.RequestGroupKey{
			ContractName:    msg.ContractName,
			ContractVersion: msg.ContractVersion,
		}
		if _, ok := pm.waitingRequestGroups.Get(group); !ok {
			pm.waitingRequestGroups.Put(group, true)
			pm.logger.Debugf("put request group %s into waiting request group.", groupKey)
		}
	}

	if err := pm.sendProcessReadyResp(msg.ProcessNum-needProcessNum, needProcessNum > 0, msg.ContractName, msg.ContractVersion); err != nil {
		pm.logger.Errorf("failed to send process ready resp, %v", err)
	}
}

// handleSandboxExitResp handle sandbox exit response, release user and remove process from cache
func (pm *ProcessManager) handleSandboxExitResp(msg *messages.SandboxExitRespMsg) {
	// TODO： allocate to waiting group
	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
	pm.logger.Debugf("sandbox exited, %v", msg.Err)

	if err := pm.allocateNewProcess(); err != nil {
		pm.logger.Errorf("failed to allocate new process, %v", err)
	}
}

// handleCleanIdleProcesses handle clean idle processes
func (pm *ProcessManager) handleCleanIdleProcesses() {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	// calculate the process num to release
	availableProcessNum := pm.getAvailableProcessNum()
	releaseNum := int(pm.releaseRate * float64(pm.maxProcessNum))
	releaseNum = releaseNum - availableProcessNum

	// available process num > release num, no need to release
	if releaseNum <= 0 {
		pm.logger.Debugf("there are enough idle processes")
		return
	}

	// idle process num < release num, release all idle processes
	if pm.idleProcesses.Size() < releaseNum {
		releaseNum = pm.idleProcesses.Size()
	}

	// pop the idle processes
	removedIdleProcesses, err := pm.batchPopIdleProcesses(releaseNum)
	if err != nil {
		pm.logger.Errorf("failed to batch pop idle processes, %v", err)
	}

	// put close sandbox req to process
	var wg sync.WaitGroup
	for i := 0; i < releaseNum; i++ {
		wg.Add(1)
		index := i
		go func() {
			// send close sandbox request
			err = removedIdleProcesses[index].PutMsg(&messages.CloseSandboxReqMsg{})
			if err != nil {
				pm.logger.Errorf("failed to kill process, %v", err)
				wg.Done()
				return
			}
			wg.Done()
		}()
	}
	wg.Wait()

	// start timer for next clean
	pm.startTimer()

	pm.logger.Debugf("removed %d idle processes", releaseNum)

	return
}

// CreateNewProcess create a new process
func (pm *ProcessManager) createNewProcess(contractName, contractVersion, processName string) (interfaces.Process, error) {

	user, err := pm.userManager.GetAvailableUser()
	if err != nil {
		pm.logger.Errorf("fail to get available user, %v", err)
		return nil, err
	}

	// check whether request scheduler was initialized
	if pm.requestScheduler == nil {
		return nil, fmt.Errorf("request scheduler has not been initialized")
	}

	// new process and start
	var process interfaces.Process
	process = NewProcess(user, contractName, contractVersion, processName, pm, pm.requestScheduler, pm.isCrossManager)
	go process.Start()

	return process, nil
}

// closeSandbox releases user and process
func (pm *ProcessManager) closeSandbox(contractName, contractVersion, processName string) {

	pm.releaseUser(processName)
	pm.removeProcessFromCache(contractName, contractVersion, processName)
}

// releaseUser releases linux user
func (pm *ProcessManager) releaseUser(processName string) {

	pm.logger.Debugf("release process %s", processName)

	process, _ := pm.getProcessByName(processName)
	_ = pm.userManager.FreeUser(process.GetUser())
}

// getProcessByName returns process by name
func (pm *ProcessManager) getProcessByName(processName string) (interfaces.Process, bool) {

	if val, ok := pm.idleProcesses.Get(processName); ok {
		return val.(interfaces.Process), true
	}
	if val, ok := pm.busyProcesses[processName]; ok {
		return val, true
	}
	return nil, false
}

// batchPopIdleProcesses batch pop idle processes
func (pm *ProcessManager) batchPopIdleProcesses(num int) ([]interfaces.Process, error) {

	if num > pm.idleProcesses.Size() {
		return nil, fmt.Errorf("num > current size")
	}

	// idle processes to remove
	var removedIdleProcesses []interfaces.Process

	for i := 0; i < num; i++ {
		processIt := pm.idleProcesses.Iterator()
		processIt.Next()

		process := processIt.Value().(interfaces.Process)
		removedIdleProcesses = append(removedIdleProcesses, process)
		pm.removeProcessFromCache(process.GetContractName(), process.GetContractVersion(), process.GetProcessName())
	}
	return removedIdleProcesses, nil
}

// addProcessToCache add process to busy / idle process cache and process group
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

// getAvailableProcessNum returns available process num
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

		groupIt.Next()
		processIt.Next()

		// get the oldest process, contract name, contract version and process name
		process := processIt.Value().(interfaces.Process)
		contractName := process.GetContractName()
		contractVersion := process.GetContractVersion()
		processName := process.GetProcessName()

		// get new contract name and contract version
		newGroupKey := groupIt.Key().(*messages.RequestGroupKey)

		// send process ready resp to request group
		if err := pm.sendProcessReadyResp(1, false, newGroupKey.ContractName, newGroupKey.ContractVersion); err != nil {
			pm.logger.Errorf("failed to send process ready resp, %v", err)
		}

		// remove request group from queue
		pm.waitingRequestGroups.Remove(newGroupKey)

		// remove idle processes from queue
		pm.idleProcesses.Remove(processName)
		// remove old process from process group
		pm.removeFromProcessGroup(contractName, contractVersion, processName)

		// notify process change sandbox context
		newProcessName := utils.ConstructProcessName(newGroupKey.ContractName, newGroupKey.ContractVersion, i)
		err := process.PutMsg(&messages.ChangeSandboxReqMsg{
			ContractName:    newGroupKey.ContractName,
			ContractVersion: newGroupKey.ContractVersion,
			ProcessName:     newProcessName,
		})
		if err != nil {
			return err
		}

		// add process to busy process list
		pm.busyProcesses[newProcessName] = process
		// add process to process group
		pm.addToProcessGroup(newGroupKey.ContractName, newGroupKey.ContractVersion, newProcessName)
	}
	return nil
}

// allocateNewProcess allocate new process to waiting request groups
func (pm *ProcessManager) allocateNewProcess() error {

	// calculate allocate num
	allocateNum := pm.waitingRequestGroups.Size()
	if allocateNum == 0 {
		return nil
	}

	availableProcessNum := pm.getAvailableProcessNum()
	if availableProcessNum < allocateNum {
		allocateNum = availableProcessNum
	}

	for i := 0; i < allocateNum; i++ {
		// match the process and request group
		groupIt := pm.waitingRequestGroups.Iterator()

		groupIt.Next()

		// get new contract name and contract version
		newGroupKey := groupIt.Key().(*messages.RequestGroupKey)

		// send process ready resp to request group
		if err := pm.sendProcessReadyResp(1, false, newGroupKey.ContractName, newGroupKey.ContractVersion); err != nil {
			pm.logger.Errorf("failed to send process ready resp, %v", err)
		}

		// remove request group from queue
		pm.waitingRequestGroups.Remove(newGroupKey)

		// create new process
		processName := utils.ConstructProcessName(newGroupKey.ContractName, newGroupKey.ContractVersion, i)
		process, err := pm.createNewProcess(newGroupKey.ContractName, newGroupKey.ContractVersion, processName)
		if err != nil {
			return fmt.Errorf("failed to create new process, %v", err)
		}

		// add process to cache
		pm.addProcessToCache(newGroupKey.ContractName, newGroupKey.ContractVersion, processName, process, true)
	}
	return nil
}

// addToProcessGroup add process to process map group by contract key
func (pm *ProcessManager) addToProcessGroup(contractName, contractVersion, processName string) {

	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)

	if _, ok := pm.processGroups[groupKey]; !ok {
		pm.processGroups[groupKey] = make(map[string]bool)
	}
	pm.processGroups[groupKey][processName] = true
}

// removeFromProcessGroup remove process from process group
func (pm *ProcessManager) removeFromProcessGroup(contractName, contractVersion, processName string) {

	groupKey := utils.ConstructRequestGroupKey(contractName, contractVersion)

	// remove process from process group
	if _, ok := pm.processGroups[groupKey]; !ok {
		return
	}
	delete(pm.processGroups[groupKey], processName)

	// remove a group in process groups
	if len(pm.processGroups[groupKey]) == 0 {
		delete(pm.processGroups, groupKey)
		if err := pm.closeRequestGroup(contractName, contractVersion); err != nil {
			pm.logger.Warnf("failed to close request group, %v", err)
		}
	}
}

// closeRequestGroup closes a request group
func (pm *ProcessManager) closeRequestGroup(contractName, contractVersion string) error {
	return pm.requestScheduler.PutMsg(
		&messages.RequestGroupKey{
			ContractName:    contractName,
			ContractVersion: contractVersion,
		},
	)
}

func (pm *ProcessManager) sendProcessReadyResp(processNum int, toWaiting bool, contractName, contractVersion string) error {
	group, ok := pm.requestScheduler.GetRequestGroup(contractName, contractVersion)
	if !ok {
		return fmt.Errorf("failed to get request group, contract name: %s, contract version: %s", contractName, contractVersion)
	}
	if err := group.PutMsg(&messages.GetProcessRespMsg{
		IsCross:    pm.isCrossManager,
		ToWaiting:  toWaiting,
		ProcessNum: processNum,
	}); err != nil {
		return fmt.Errorf("failed to put msg into request group eventCh, %v", err)
	}
	return nil
}

// startTimer start request group clean timer
func (pm *ProcessManager) startTimer() {
	pm.logger.Debugf("start clean timer")
	if !pm.cleanTimer.Stop() && len(pm.cleanTimer.C) > 0 {
		<-pm.cleanTimer.C
	}
	pm.cleanTimer.Reset(config.DockerVMConfig.GetReleasePeriod())
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
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

	maxProcessNum int     // max process num
	releaseRate   float64 // the minimum rate of available process
	isOrigManager bool    // cross process manager or original process manager

	idleProcesses        *linkedhashmap.Map            // idle processes linked hashmap (process name -> idle Process)
	busyProcesses        map[string]interfaces.Process // busy process map (process name -> busy Process)
	processGroups        map[string]map[string]bool    // process group by contract key (contract key -> Process name set)
	waitingRequestGroups *linkedhashmap.Map            // waiting request groups linked hashmap (group key -> bool)

	eventCh        chan interface{} // process manager event channel
	allocateIdleCh chan struct{}
	allocateNewCh  chan struct{}
	cleanTimer     *time.Timer // clean timer for release idle processes

	userManager      interfaces.UserManager      // user manager
	requestScheduler interfaces.RequestScheduler // request scheduler
	processCnt       uint64
}

// NewProcessManager returns new process manager
func NewProcessManager(maxProcessNum int, rate float64, isOrigManager bool, userManager interfaces.UserManager) *ProcessManager {
	return &ProcessManager{
		logger: logger.NewDockerLogger(logger.MODULE_PROCESS_MANAGER),
		lock:   sync.RWMutex{},

		maxProcessNum: maxProcessNum,
		releaseRate:   rate,
		isOrigManager: isOrigManager,

		idleProcesses:        linkedhashmap.New(),
		busyProcesses:        make(map[string]interfaces.Process),
		processGroups:        make(map[string]map[string]bool),
		waitingRequestGroups: linkedhashmap.New(),

		eventCh:        make(chan interface{}, processManagerEventChSize),
		allocateIdleCh: make(chan struct{}, processManagerEventChSize),
		allocateNewCh:  make(chan struct{}, processManagerEventChSize),

		cleanTimer: time.NewTimer(config.DockerVMConfig.Process.ReleasePeriod),

		userManager: userManager,
	}
}

// Start process manager, listen event chan and clean timer,
// types: messages.GetProcessReqMsg, messages.SandboxExitMsg and cleanIdleProcesses timer
func (pm *ProcessManager) Start() {

	pm.logger.Debugf("start process manager routine")

	go func() {
		for {
			select {
			case msg := <-pm.eventCh:
				switch msg.(type) {

				case *messages.GetProcessReqMsg:
					m, _ := msg.(*messages.GetProcessReqMsg)
					if err := pm.handleGetProcessReq(m); err != nil {
						pm.logger.Errorf("failed to handle get process req, %v", err)
					}

				case *messages.SandboxExitMsg:
					m, _ := msg.(*messages.SandboxExitMsg)
					pm.handleSandboxExitMsg(m)

				default:
					pm.logger.Errorf("unknown msg type, msg: %+v", msg)

				}

			case <-pm.cleanTimer.C:
				pm.handleCleanIdleProcesses()

			case <-pm.allocateIdleCh:
				if err := pm.handleAllocateIdleProcesses(); err != nil {
					pm.logger.Errorf("failed to allocate idle processes, %v", err)
				}

			case <-pm.allocateNewCh:
				if err := pm.handleAllocateNewProcesses(); err != nil {
					pm.logger.Errorf("failed to allocate idle processes, %v", err)
				}

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
	case *messages.GetProcessReqMsg, *messages.SandboxExitMsg:
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

	groupKey := utils.ConstructContractKey(contractName, contractVersion)
	if val, ok := pm.processGroups[groupKey]; ok {
		return len(val)
	}
	return 0
}

// ChangeProcessState changes the process state
func (pm *ProcessManager) ChangeProcessState(processName string, toBusy bool) error {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.logger.Debugf("start change process %s state", processName)

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
		pm.addProcessToIdle(processName, process)
		delete(pm.busyProcesses, processName)
	}

	pm.logger.Debugf("end change process %s state", processName)

	return nil
}

// handleGetProcessReq handle get process request from request group
func (pm *ProcessManager) handleGetProcessReq(msg *messages.GetProcessReqMsg) error {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	groupKey := utils.ConstructContractKey(msg.ContractName, msg.ContractVersion)

	pm.logger.Debugf("request group %s request to get %d process(es)", groupKey, msg.ProcessNum)

	// do not need any process
	if msg.ProcessNum == 0 {
		pm.removeFromWaitingGroup(msg.ContractName, msg.ContractVersion)
		pm.logger.Debugf("request group %s does not need more processes, removed from waiting request group", groupKey)

		return nil
	}

	needProcessNum := msg.ProcessNum
	availableProcessNum := pm.getAvailableProcessNum()

	// 1. allocate processes that can be launched
	if availableProcessNum > 0 {

		newProcessNum := utils.Min(needProcessNum, availableProcessNum)
		// create new process concurrently
		var wg sync.WaitGroup
		lock := sync.Mutex{}
		wg.Add(newProcessNum)
		for i := 0; i < newProcessNum; i++ {
			go func() {
				defer wg.Done()
				// create new process
				lock.Lock()
				processName := pm.generateProcessName(msg.ContractName, msg.ContractVersion)
				lock.Unlock()

				process, err := pm.createNewProcess(msg.ContractName, msg.ContractVersion, processName)
				if err != nil {
					pm.logger.Errorf("failed to create new process, %v", err)
					return
				}

				// add process to cache
				lock.Lock()
				defer lock.Unlock()

				needProcessNum--
				pm.addProcessToCache(msg.ContractName, msg.ContractVersion, processName, process, true)
			}()
		}
		wg.Wait()
	}

	allocatedAvailableProcessNum := msg.ProcessNum - needProcessNum

	idleProcessesSize := pm.idleProcesses.Size()
	// 2. allocate processes from idle processes
	if needProcessNum > 0 && idleProcessesSize > 0 {
		newProcessNum := utils.Min(needProcessNum, idleProcessesSize)
		// idle processes to remove
		idleProcesses, err := pm.peekIdleProcesses(newProcessNum)
		if err != nil {
			return fmt.Errorf("failed to peek idle processes, %v", err)
		}

		var processlist string
		for _, v := range idleProcesses {
			processlist += v.GetProcessName() + " "
		}
		pm.logger.Debugf("allocate idle process list, %s", processlist)

		// change processes context concurrently
		var wg sync.WaitGroup
		lock := sync.Mutex{}
		wg.Add(newProcessNum)
		for i := 0; i < newProcessNum; i++ {
			go func(process interfaces.Process) {
				defer wg.Done()

				// generate new process name
				oldContractName := process.GetContractName()
				oldContractVersion := process.GetContractVersion()
				oldProcessName := process.GetProcessName()

				// meet the same idle process
				if msg.ContractName == oldContractName && msg.ContractVersion == oldContractVersion {
					lock.Lock()
					defer lock.Unlock()
					needProcessNum--
					return
				}

				lock.Lock()
				newProcessName := pm.generateProcessName(msg.ContractName, msg.ContractVersion)
				lock.Unlock()

				// waiting for kill completed
				if err := process.ChangeSandbox(msg.ContractName, msg.ContractVersion, newProcessName); err != nil {
					pm.logger.Warnf("failed to change process %s, %v", oldProcessName, err)
					return
				}

				lock.Lock()
				defer lock.Unlock()
				// remove process from idle process list, add to busy process list
				needProcessNum--
				pm.removeProcessFromCache(oldContractName, oldContractVersion, oldProcessName)
				pm.addProcessToCache(msg.ContractName, msg.ContractVersion, newProcessName, process, true)

			}(idleProcesses[i])
		}
		wg.Wait()
	}

	allocatedIdleProcessNum := msg.ProcessNum - allocatedAvailableProcessNum - needProcessNum

	pm.logger.Debugf("request group %s request to get %d process(es), "+
		"allocated %d available process(es), %d idle process(es)", groupKey,
		msg.ProcessNum, allocatedAvailableProcessNum, allocatedIdleProcessNum)

	// no available process, put to waiting request group
	if msg.ProcessNum == needProcessNum {
		group := messages.RequestGroupKey{
			ContractName:    msg.ContractName,
			ContractVersion: msg.ContractVersion,
		}
		if _, ok := pm.waitingRequestGroups.Get(group); !ok {
			pm.waitingRequestGroups.Put(group, true)
			pm.logger.Debugf("put request group %s into waiting request group", groupKey)
		}
	} else {
		if err := pm.sendProcessReadyResp(msg.ProcessNum-needProcessNum, msg.ContractName, msg.ContractVersion); err != nil {
			return fmt.Errorf("failed to send process ready resp, %v", err)
		}
	}

	return nil
}

// handleSandboxExitMsg handle sandbox exit response, release user and remove process from cache
func (pm *ProcessManager) handleSandboxExitMsg(msg *messages.SandboxExitMsg) {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.logger.Debugf("handle sandbox exit msg")

	pm.closeSandbox(msg.ContractName, msg.ContractVersion, msg.ProcessName)
	pm.logger.Debugf("sandbox exited, %v", msg.Err)

	pm.allocateNewCh <- struct{}{}
}

// handleCleanIdleProcesses handle clean idle processes
func (pm *ProcessManager) handleCleanIdleProcesses() {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.logger.Debugf("handle periodic clean idle processes")

	// calculate the process num to release
	availableProcessNum := pm.getAvailableProcessNum()
	releaseNum := int(pm.releaseRate * float64(pm.maxProcessNum))
	releaseNum = releaseNum - availableProcessNum

	releaseNum = utils.Min(releaseNum, pm.idleProcesses.Size())
	// available process num > release num, no need to release
	if releaseNum <= 0 {
		pm.logger.Debugf("there are enough idle processes")
		return
	}

	// peek the idle processes
	processes, err := pm.peekIdleProcesses(releaseNum)
	if err != nil {
		pm.logger.Errorf("failed to peek idle processes, %v", err)
	}

	pm.logger.Debugf("try to remove %d idle processes", releaseNum)

	// put close sandbox req to process
	var actualNum int
	var wg sync.WaitGroup
	var lock sync.Mutex
	wg.Add(releaseNum)
	for i := 0; i < releaseNum; i++ {
		process := processes[i]
		go func() {
			defer wg.Done()
			// send close sandbox request
			err := process.CloseSandbox()
			if err != nil {
				pm.logger.Errorf("failed to kill process, %v", err)
				return
			}
			lock.Lock()
			defer lock.Unlock()

			actualNum++
			pm.closeSandbox(process.GetContractName(), process.GetContractVersion(), process.GetProcessName())
		}()
	}
	wg.Wait()

	if actualNum > 0 {
		pm.allocateNewCh <- struct{}{}
	}
	// start timer for next clean
	pm.startTimer()

	return
}

// handleAllocateIdleProcesses allocate idle process to waiting request groups
func (pm *ProcessManager) handleAllocateIdleProcesses() error {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.logger.Debugf("handle allocate idle processes")

	// calculate allocate num
	allocateNum := utils.Min(pm.waitingRequestGroups.Size(), pm.idleProcesses.Size())
	if allocateNum == 0 {
		return nil
	}

	// idle processes to remove
	idleProcesses, err := pm.peekIdleProcesses(allocateNum)
	var processlist string
	for _, v := range idleProcesses {
		processlist += v.GetProcessName() + " "
	}
	pm.logger.Debugf("allocate idle process list, %s", processlist)
	if err != nil {
		pm.logger.Errorf("failed to peek idle processes, %v", err)
	}

	waitingRequestGroups, err := pm.peekWaitingRequestGroups(allocateNum)
	if err != nil {
		pm.logger.Errorf("failed to peek waiting groups, %v", err)
	}

	// change processes context concurrently
	var wg sync.WaitGroup
	lock := sync.Mutex{}
	wg.Add(allocateNum)
	for i := 0; i < allocateNum; i++ {
		process := idleProcesses[i]
		group := waitingRequestGroups[i]
		go func() {
			defer wg.Done()

			// generate new process name
			oldContractName := process.GetContractName()
			oldContractVersion := process.GetContractVersion()
			oldProcessName := process.GetProcessName()

			// meet the same idle process
			if group.ContractName == oldContractName && group.ContractVersion == oldContractVersion {
				lock.Lock()
				pm.removeFromWaitingGroup(group.ContractName, group.ContractVersion)
				lock.Unlock()
				// send process ready resp to request group
				if err := pm.sendProcessReadyResp(0, group.ContractName, group.ContractVersion); err != nil {
					pm.logger.Errorf("failed to send process ready resp, %v", err)
					return
				}

				return
			}

			lock.Lock()
			newProcessName := pm.generateProcessName(group.ContractName, group.ContractVersion)
			lock.Unlock()

			if err := process.ChangeSandbox(group.ContractName, group.ContractVersion, newProcessName); err != nil {
				pm.logger.Warnf("failed to change sandbox, %v", err)
				return
			}

			// send process ready resp to request group
			if err := pm.sendProcessReadyResp(1, group.ContractName, group.ContractVersion); err != nil {
				pm.logger.Errorf("failed to send process ready resp, %v", err)
				return
			}

			// remove process from idle process list, add to busy process list
			lock.Lock()
			defer lock.Unlock()

			pm.removeFromWaitingGroup(group.ContractName, group.ContractVersion)
			pm.removeProcessFromCache(oldContractName, oldContractVersion, oldProcessName)
			pm.addProcessToCache(group.ContractName, group.ContractVersion, newProcessName, process, true)
		}()
	}
	wg.Wait()

	return nil
}

// handleAllocateNewProcesses allocate new process to waiting request groups
func (pm *ProcessManager) handleAllocateNewProcesses() error {

	pm.lock.Lock()
	defer pm.lock.Unlock()

	pm.logger.Debugf("handle allocate new processes")

	// calculate allocate num
	allocateNum := utils.Min(pm.waitingRequestGroups.Size(), pm.getAvailableProcessNum())
	if allocateNum == 0 {
		return nil
	}

	waitingRequestGroups, err := pm.peekWaitingRequestGroups(allocateNum)
	if err != nil {
		pm.logger.Errorf("failed to peek waiting groups, %v", err)
	}

	var wg sync.WaitGroup
	lock := sync.Mutex{}
	wg.Add(allocateNum)
	for i := 0; i < allocateNum; i++ {
		group := waitingRequestGroups[i]
		go func() {
			defer wg.Done()

			// send process ready resp to request group
			if err := pm.sendProcessReadyResp(1, group.ContractName, group.ContractVersion); err != nil {
				pm.logger.Errorf("failed to send process ready resp, %v", err)
				return
			}

			// create new process
			lock.Lock()
			processName := pm.generateProcessName(group.ContractName, group.ContractVersion)
			lock.Unlock()

			process, err := pm.createNewProcess(group.ContractName, group.ContractVersion, processName)
			if err != nil {
				pm.logger.Errorf("failed to create new process, %v", err)
				return
			}

			// remove process from idle process list, add to busy process list
			lock.Lock()
			defer lock.Unlock()

			pm.removeFromWaitingGroup(group.ContractName, group.ContractVersion)
			pm.addProcessToCache(group.ContractName, group.ContractVersion, processName, process, true)
		}()
	}
	wg.Wait()

	return nil
}

// CreateNewProcess create a new process
func (pm *ProcessManager) createNewProcess(contractName, contractVersion, processName string) (interfaces.Process, error) {

	// check whether request scheduler was initialized
	if pm.requestScheduler == nil {
		return nil, fmt.Errorf("request scheduler has not been initialized")
	}

	user, err := pm.userManager.GetAvailableUser()
	if err != nil {
		pm.logger.Errorf("failed to get available user, %v", err)
		return nil, err
	}

	// new process and start
	var process interfaces.Process
	process = NewProcess(user, contractName, contractVersion, processName, pm, pm.requestScheduler, pm.isOrigManager)
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

// peekIdleProcesses returns idle processes from head
func (pm *ProcessManager) peekIdleProcesses(num int) ([]interfaces.Process, error) {

	if num > pm.idleProcesses.Size() {
		return nil, fmt.Errorf("num > current size")
	}

	// idle processes to remove
	var processes []interfaces.Process

	var key string
	processIt := pm.idleProcesses.Iterator()

	for i := 0; i < num; i++ {
		processIt.Next()
		process := processIt.Value().(interfaces.Process)
		processes = append(processes, process)
		key += processIt.Key().(string) + " "
	}
	pm.logger.Debugf("peekIdleProcesses keys: %s", key)
	return processes, nil
}

// peekWaitingRequestGroups returns waiting request groups from head
func (pm *ProcessManager) peekWaitingRequestGroups(num int) ([]messages.RequestGroupKey, error) {

	if num > pm.waitingRequestGroups.Size() {
		return nil, fmt.Errorf("num > current size")
	}

	// idle processes to remove
	var groups []messages.RequestGroupKey

	groupIt := pm.waitingRequestGroups.Iterator()
	for i := 0; i < num; i++ {
		groupIt.Next()

		group := groupIt.Key().(messages.RequestGroupKey)
		groups = append(groups, group)
	}
	return groups, nil
}

// addProcessToCache add process to busy / idle process cache and process group
func (pm *ProcessManager) addProcessToCache(contractName, contractVersion, processName string, process interfaces.Process, isBusy bool) {

	if isBusy {
		pm.busyProcesses[processName] = process
	} else {
		pm.addProcessToIdle(processName, process)
	}

	pm.addToProcessGroup(contractName, contractVersion, processName)
}

// removeProcessFromCache remove process from busyProcesses, idleProcesses and processGroup
func (pm *ProcessManager) removeProcessFromCache(contractName, contractVersion, processName string) {

	delete(pm.busyProcesses, processName)
	pm.idleProcesses.Remove(processName)
	pm.removeFromProcessGroup(contractName, contractVersion, processName)
}

// addProcessToIdle add process to idle list
func (pm *ProcessManager) addProcessToIdle(processName string, process interfaces.Process) {

	// add process to idle list
	pm.idleProcesses.Put(processName, process)

	// construct group key
	groupKey := messages.RequestGroupKey{
		ContractName:    process.GetContractName(),
		ContractVersion: process.GetContractVersion(),
	}

	// remove waiting request group if meet the same contract
	if _, ok := pm.waitingRequestGroups.Get(groupKey); ok {
		pm.removeFromWaitingGroup(group.ContractName, group.ContractVersion)
	}

	// allocate idle process to waiting request group
	if pm.waitingRequestGroups.Size() > 0 {
		pm.allocateIdleCh <- struct{}{}
	}
}

// getAvailableProcessNum returns available process num
func (pm *ProcessManager) getAvailableProcessNum() int {

	return pm.maxProcessNum - pm.idleProcesses.Size() - len(pm.busyProcesses)
}

// addToProcessGroup add process to process map group by contract key
func (pm *ProcessManager) addToProcessGroup(contractName, contractVersion, processName string) {

	groupKey := utils.ConstructContractKey(contractName, contractVersion)

	if _, ok := pm.processGroups[groupKey]; !ok {
		pm.processGroups[groupKey] = make(map[string]bool)
	}
	pm.processGroups[groupKey][processName] = true

	pm.logger.Debugf("add %s - %s to process group, total num [%d]", groupKey, processName, len(pm.processGroups[groupKey]))
}

// removeFromProcessGroup remove process from process group
func (pm *ProcessManager) removeFromProcessGroup(contractName, contractVersion, processName string) {

	groupKey := utils.ConstructContractKey(contractName, contractVersion)

	// remove process from process group
	if _, ok := pm.processGroups[groupKey]; !ok {
		return
	}
	delete(pm.processGroups[groupKey], processName)

	pm.logger.Debugf("delete %s - %s from process group, total num [%d]", groupKey, processName, len(pm.processGroups[groupKey]))

	// remove group in process groups and waiting groups
	if len(pm.processGroups[groupKey]) == 0 {
		delete(pm.processGroups, groupKey)
		pm.removeFromWaitingGroup(contractName, contractVersion)
		if err := pm.closeRequestGroup(contractName, contractVersion); err != nil {
			pm.logger.Warnf("failed to close request group, %v", err)
		}
	}
}

func (pm *ProcessManager) removeFromWaitingGroup(contractName, contractVersion string) {
	pm.waitingRequestGroups.Remove(messages.RequestGroupKey{
		ContractName:    contractName,
		ContractVersion: contractVersion,
	})
	if err := pm.sendProcessReadyResp(0, contractName, contractVersion); err != nil {
		pm.logger.Errorf("failed to send process ready resp, %v", err)
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

// sendProcessReadyResp sends process ready resp to request group
func (pm *ProcessManager) sendProcessReadyResp(processNum int, contractName, contractVersion string) error {

	// GetRequestGroup is safe because waiting group exists -> request group exists
	group, ok := pm.requestScheduler.GetRequestGroup(contractName, contractVersion)
	if !ok {
		return fmt.Errorf("failed to get request group, contract name: %s, contract version: %s", contractName, contractVersion)
	}

	respMsg := &messages.GetProcessRespMsg{
		IsOrig:     pm.isOrigManager,
		ProcessNum: processNum,
	}

	pm.logger.Debugf("send process ready resp msg %v to %s", respMsg, utils.ConstructContractKey(contractName, contractVersion))

	if err := group.PutMsg(respMsg); err != nil {
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
	pm.cleanTimer.Reset(config.DockerVMConfig.Process.WaitingTxTime)
}

// generateProcessName generate new process name
func (pm *ProcessManager) generateProcessName(contractName, contractVersion string) string {
	groupKey := utils.ConstructContractKey(contractName, contractVersion)
	localIndex := len(pm.processGroups[groupKey])
	atomic.AddUint64(&pm.processCnt, 1)

	return utils.ConstructProcessName(contractName, contractVersion, localIndex, pm.processCnt, pm.isOrigManager)
}

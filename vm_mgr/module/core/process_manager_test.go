/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/interfaces"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/messages"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/pb/protogo"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"reflect"
	"strconv"
	"testing"
)

func TestNewProcessManager(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()

	idleProcesses := linkedhashmap.New()
	busyProcesses := make(map[string]interfaces.Process)
	processGroups := make(map[string]map[string]bool)
	waitingRequestGroups := linkedhashmap.New()
	eventCh := make(chan interface{}, processManagerEventChSize)

	log := logger.NewTestDockerLogger()

	type args struct {
		maxProcessNum int
		rate          float64
		isOrigManager bool
		userManager   interfaces.UserManager
	}
	tests := []struct {
		name string
		args args
		want *ProcessManager
	}{
		{
			name: "TestNewProcessManager",
			args: args{
				maxProcessNum: maxOriginalProcessNum,
				rate:          releaseRate,
				isOrigManager: true,
				userManager:   userManager,
			},
			want: &ProcessManager{
				maxProcessNum:        maxOriginalProcessNum,
				releaseRate:          releaseRate,
				isOrigManager:        true,
				userManager:          userManager,
				logger:               log,
				idleProcesses:        idleProcesses,
				busyProcesses:        busyProcesses,
				processGroups:        processGroups,
				waitingRequestGroups: waitingRequestGroups,
				eventCh:              eventCh,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NewProcessManager(tt.args.maxProcessNum, tt.args.rate, tt.args.isOrigManager, tt.args.userManager)
			got.logger = log
			got.idleProcesses = idleProcesses
			got.busyProcesses = busyProcesses
			got.processGroups = processGroups
			got.waitingRequestGroups = waitingRequestGroups
			got.eventCh = eventCh
			tt.want.cleanTimer = got.cleanTimer

			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewProcessManager() = %v, wantInCache %v", got, tt.want)
			}
		})
	}
}

func TestProcessManager_ChangeProcessState(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	processManager.idleProcesses.Put("testIdleProcess", &Process{})
	processManager.busyProcesses["testBusyProcess"] = &Process{}

	type fields struct {
		processManager *ProcessManager
	}
	type args struct {
		processName string
		toBusy      bool
	}
	tests := []struct {
		name       string
		fields     fields
		args       args
		wantErr    bool
		wantInIdle bool
		wantInBusy bool
	}{
		{
			name: "TestProcessManager_ChangeProcessState_ToIdle_NotExist",
			fields: fields{
				processManager: processManager,
			},
			args: args{
				processName: "testIdleProcess",
				toBusy:      false,
			},
			wantErr:    true,
			wantInIdle: true,
			wantInBusy: false,
		},
		{
			name: "TestProcessManager_ChangeProcessState_ToBusy_Exist",
			fields: fields{
				processManager: processManager,
			},
			args: args{
				processName: "testIdleProcess",
				toBusy:      true,
			},
			wantErr:    false,
			wantInIdle: false,
			wantInBusy: true,
		},
		{
			name: "TestProcessManager_ChangeProcessState_ToBusy_NotExist",
			fields: fields{
				processManager: processManager,
			},
			args: args{
				processName: "testBusyProcess",
				toBusy:      true,
			},
			wantErr:    true,
			wantInIdle: false,
			wantInBusy: true,
		},
		{
			name: "TestProcessManager_ChangeProcessState_ToIdle_Exist",
			fields: fields{
				processManager: processManager,
			},
			args: args{
				processName: "testBusyProcess",
				toBusy:      false,
			},
			wantErr:    false,
			wantInIdle: true,
			wantInBusy: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			if err := pm.ChangeProcessState(tt.args.processName, tt.args.toBusy); (err != nil) != tt.wantErr {
				t.Errorf("ChangeProcessState() error = %v, wantErr %v", err, tt.wantErr)
			}
			if _, ok := pm.idleProcesses.Get(tt.args.processName); ok != tt.wantInIdle {
				t.Errorf("ChangeProcessState() inIdle = %v, wantInIdle %v", ok, tt.wantInIdle)
			}
			if _, ok := pm.busyProcesses[tt.args.processName]; ok != tt.wantInBusy {
				t.Errorf("ChangeProcessState() inBusy = %v, wantInBusy %v", ok, tt.wantInBusy)
			}
		})
	}
}

func TestProcessManager_GetProcessByName(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	idleProcess := &Process{}
	busyProcess := &Process{}
	processManager.idleProcesses.Put("testIdleProcess", idleProcess)
	processManager.busyProcesses["testBusyProcess"] = busyProcess

	type fields struct {
		processManager *ProcessManager
	}
	type args struct {
		processName string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interfaces.Process
		want1  bool
	}{
		{
			name:   "TestProcessManager_GetProcessByName_Idle_Exist",
			fields: fields{processManager: processManager},
			args:   args{processName: "testIdleProcess"},
			want:   idleProcess,
			want1:  true,
		},
		{
			name:   "TestProcessManager_GetProcessByName_Idle_NotExist",
			fields: fields{processManager: processManager},
			args:   args{processName: "testIdleProcess2"},
			want:   nil,
			want1:  false,
		},
		{
			name:   "TestProcessManager_GetProcessByName_Busy_Exist",
			fields: fields{processManager: processManager},
			args:   args{processName: "testBusyProcess"},
			want:   busyProcess,
			want1:  true,
		},
		{
			name:   "TestProcessManager_GetProcessByName_Busy_NotExist",
			fields: fields{processManager: processManager},
			args:   args{processName: "testBusyProcess2"},
			want:   nil,
			want1:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			got, got1 := pm.GetProcessByName(tt.args.processName)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetProcessByName() got = %v, wantInCache %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("GetProcessByName() got1 = %v, wantInCache %v", got1, tt.want1)
			}
		})
	}
}

func TestProcessManager_GetProcessNumByContractKey(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	processManager.processGroups["testContractName#1.0.0"] = make(map[string]bool)
	processManager.processGroups["testContractName#1.0.0"]["process1"] = true
	processManager.processGroups["testContractName#1.0.0"]["process2"] = true

	processManager.processGroups["testContractName#1.0.1"] = make(map[string]bool)
	processManager.processGroups["testContractName#1.0.1"]["process1"] = true

	processManager.processGroups["testContractName2#1.0.0"] = make(map[string]bool)
	processManager.processGroups["testContractName2#1.0.0"]["process1"] = true

	type args struct {
		contractName    string
		contractVersion string
	}

	type fields struct {
		processManager *ProcessManager
	}

	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{
			name:   "TestProcessManager_GetProcessNumByContractKey1",
			fields: fields{processManager: processManager},
			args:   args{contractName: "testContractName", contractVersion: "1.0.0"},
			want:   2,
		},
		{
			name:   "TestProcessManager_GetProcessNumByContractKey2",
			fields: fields{processManager: processManager},
			args:   args{contractName: "testContractName", contractVersion: "1.0.1"},
			want:   1,
		},
		{
			name:   "TestProcessManager_GetProcessNumByContractKey3",
			fields: fields{processManager: processManager},
			args:   args{contractName: "testContractName2", contractVersion: "1.0.0"},
			want:   1,
		},
		{
			name:   "TestProcessManager_GetProcessNumByContractKey4",
			fields: fields{processManager: processManager},
			args:   args{contractName: "testContractName2", contractVersion: "1.0.1"},
			want:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			if got := pm.GetProcessNumByContractKey(tt.args.contractName, tt.args.contractVersion); got != tt.want {
				t.Errorf("GetProcessNumByContractKey() = %v, wantInCache %v", got, tt.want)
			}
		})
	}
}

func TestProcessManager_PutMsg(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	type fields struct {
		processManager *ProcessManager
	}
	type args struct {
		msg interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		{
			name:    "TestProcessManager_PutMsg_DockerVMMessage",
			fields:  fields{processManager: processManager},
			args:    args{msg: &protogo.DockerVMMessage{}},
			wantErr: false,
		},
		{
			name:    "TestProcessManager_PutMsg_GetProcessReqMsg",
			fields:  fields{processManager: processManager},
			args:    args{msg: &messages.GetProcessReqMsg{}},
			wantErr: false,
		},
		{
			name:    "TestProcessManager_PutMsg_LaunchSandboxRespMsg",
			fields:  fields{processManager: processManager},
			args:    args{msg: &messages.LaunchSandboxRespMsg{}},
			wantErr: false,
		},
		{
			name:    "TestProcessManager_PutMsg_ChangeSandboxReqMsg",
			fields:  fields{processManager: processManager},
			args:    args{msg: &messages.ChangeSandboxReqMsg{}},
			wantErr: false,
		},
		{
			name:    "TestProcessManager_PutMsg_CloseSandboxRespMsg",
			fields:  fields{processManager: processManager},
			args:    args{msg: &messages.CloseSandboxRespMsg{}},
			wantErr: false,
		},
		{
			name:    "TestProcessManager_PutMsg_String",
			fields:  fields{processManager: processManager},
			args:    args{msg: "test"},
			wantErr: true,
		},
		{
			name:    "TestProcessManager_PutMsg_Int",
			fields:  fields{processManager: processManager},
			args:    args{msg: 0},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			if err := pm.PutMsg(tt.args.msg); (err != nil) != tt.wantErr {
				t.Errorf("PutMsg() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestProcessManager_SetScheduler(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		scheduler interfaces.RequestScheduler
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{
			name:   "TestProcessManager_SetScheduler",
			fields: fields{processManager: processManager},
			args:   args{scheduler: &RequestScheduler{}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			pm.SetScheduler(tt.args.scheduler)
		})
	}
}

func TestProcessManager_Start(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	type fields struct {
		processManager *ProcessManager
	}

	tests := []struct {
		name   string
		fields fields
	}{
		{
			name:   "TestProcessManager_Start",
			fields: fields{processManager: processManager},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			pm.Start()
		})
	}
}

func TestProcessManager_addProcessToCache(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	idleProcess := &Process{}
	busyProcess := &Process{}

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		contractName    string
		contractVersion string
		processName     string
		process         interfaces.Process
		isBusy          bool
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantInIdle  bool
		wantInBusy  bool
		wantInCache bool
		wantNum     int
	}{
		{
			name:   "TestProcessManager_addProcessToCache_Idle",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    "testContractName1",
				contractVersion: "1.0.0",
				processName:     "testProcessName1",
				process:         idleProcess,
				isBusy:          false,
			},
			wantInIdle:  true,
			wantInBusy:  false,
			wantInCache: true,
			wantNum:     1,
		},
		{
			name:   "TestProcessManager_addProcessToCache_Busy",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    "testContractName2",
				contractVersion: "1.0.0",
				processName:     "testProcessName2",
				process:         busyProcess,
				isBusy:          true,
			},
			wantInIdle:  false,
			wantInBusy:  true,
			wantInCache: true,
			wantNum:     1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.addProcessToCache(tt.args.contractName, tt.args.contractVersion, tt.args.processName, tt.args.process, tt.args.isBusy)
			if _, ok := pm.idleProcesses.Get(tt.args.processName); ok != tt.wantInIdle {
				t.Errorf("addProcessToCache() inIdle = %v, wantInIdle %v", ok, tt.wantInIdle)
			}
			if _, ok := pm.busyProcesses[tt.args.processName]; ok != tt.wantInBusy {
				t.Errorf("addProcessToCache() inBusy = %v, wantInBusy %v", ok, tt.wantInBusy)
			}
			groupKey := utils.ConstructContractKey(tt.args.contractName, tt.args.contractVersion)
			_, ok := pm.processGroups[groupKey][tt.args.processName]
			if ok != tt.wantInCache {
				t.Errorf("addProcessToCache() inCache = %v, wantInCache %v", ok, tt.wantInBusy)
			}
			if len(pm.processGroups[groupKey]) != tt.wantNum {
				t.Errorf("addProcessToCache() num = %v, wantNum %v", len(pm.processGroups[groupKey]), tt.wantNum)
			}
		})
	}
}

func TestProcessManager_addToProcessGroup(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		contractName    string
		contractVersion string
		processName     string
	}
	tests := []struct {
		name        string
		fields      fields
		args        args
		wantInIdle  bool
		wantInBusy  bool
		wantInCache bool
		wantNum     int
	}{
		{
			name:   "TestProcessManager_addToProcessGroup",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    "testContractName1",
				contractVersion: "1.0.0",
				processName:     "testProcessName1",
			},
			wantInCache: true,
			wantNum:     1,
		},
		{
			name:   "TestProcessManager_addProcessToCache_Busy",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    "testContractName2",
				contractVersion: "1.0.0",
				processName:     "testProcessName2",
			},
			wantInCache: true,
			wantNum:     1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.addToProcessGroup(tt.args.contractName, tt.args.contractVersion, tt.args.processName)
			groupKey := utils.ConstructContractKey(tt.args.contractName, tt.args.contractVersion)
			_, ok := pm.processGroups[groupKey][tt.args.processName]
			if ok != tt.wantInCache {
				t.Errorf("addProcessToCache() inCache = %v, wantInCache %v", ok, tt.wantInBusy)
			}
			if len(pm.processGroups[groupKey]) != tt.wantNum {
				t.Errorf("addProcessToCache() num = %v, wantNum %v", len(pm.processGroups[groupKey]), tt.wantNum)
			}
		})
	}
}

func TestProcessManager_allocateIdleProcess(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log
	processManager.requestScheduler = &RequestScheduler{
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),
	}

	testContractName1 := "testContractName1"
	testContractName2 := "testContractName2"
	testProcessName1 := "testProcessName1"
	testProcessName2 := "testProcessName2"
	testContractVersion := "1.0.0"

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		&Process{
			contractName:    testContractName1,
			contractVersion: testContractVersion,
			processName:     testProcessName1,
			eventCh:         make(chan interface{}, processEventChSize),
		},
		false,
	)
	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName2,
		&Process{
			contractName:    testContractName1,
			contractVersion: testContractVersion,
			processName:     testProcessName2,
			eventCh:         make(chan interface{}, processEventChSize),
		},
		false,
	)
	group := &messages.RequestGroupKey{
		ContractName:    testContractName2,
		ContractVersion: testContractVersion,
	}
	processManager.waitingRequestGroups.Put(group, true)

	type fields struct {
		processManager *ProcessManager
	}

	tests := []struct {
		name                        string
		fields                      fields
		wantWaitingRequestGroupsNum int
		wantIdleProcessNum          int
		wantBusyProcessNum          int
		wantOrigContractProcessNum  int
		wantNewContractProcessNum   int
		wantErr                     bool
	}{
		{
			name:                        "TestProcessManager_allocateIdleProcess",
			fields:                      fields{processManager: processManager},
			wantWaitingRequestGroupsNum: 0,
			wantIdleProcessNum:          1,
			wantBusyProcessNum:          1,
			wantOrigContractProcessNum:  1,
			wantNewContractProcessNum:   1,
			wantErr:                     false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			if err := pm.handleAllocateIdleProcesses(); (err != nil) != tt.wantErr {
				t.Errorf("handleAllocateIdleProcesses() error = %v, wantErr %v", err, tt.wantErr)
			}
			requestGroupNum := pm.waitingRequestGroups.Size()
			if requestGroupNum != tt.wantWaitingRequestGroupsNum {
				t.Errorf("handleAllocateIdleProcesses() waiting request group size = %v, "+
					"wantWaitingRequestGroupsNum %v", requestGroupNum, tt.wantWaitingRequestGroupsNum)
			}
			idleProcessNum := pm.idleProcesses.Size()
			if idleProcessNum != tt.wantIdleProcessNum {
				t.Errorf("handleAllocateIdleProcesses() idle process size = %v, "+
					"wantIdleProcessNum %v", idleProcessNum, tt.wantIdleProcessNum)
			}
			busyProcessNum := len(pm.busyProcesses)
			if busyProcessNum != tt.wantBusyProcessNum {
				t.Errorf("handleAllocateIdleProcesses() busy process size = %v, "+
					"wantBusyProcessNum %v", busyProcessNum, tt.wantBusyProcessNum)
			}
			origContractProcessNum := len(pm.processGroups["testContractName1#1.0.0"])
			if origContractProcessNum != tt.wantOrigContractProcessNum {
				t.Errorf("handleAllocateIdleProcesses() original contract process size = %v, "+
					"wantOrigContractProcessNum %v", origContractProcessNum, tt.wantOrigContractProcessNum)
			}
			newContractProcessNum := len(pm.processGroups["testContractName2#1.0.0"])
			if newContractProcessNum != tt.wantNewContractProcessNum {
				t.Errorf("handleAllocateIdleProcesses() new contract process size = %v, "+
					"wantNewContractProcessNum %v", newContractProcessNum, tt.wantNewContractProcessNum)
			}
		})
	}
}

func TestProcessManager_batchPopIdleProcesses(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log
	processManager.requestScheduler = &RequestScheduler{
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),
	}

	testContractName1 := "testContractName1"
	testContractName2 := "testContractName2"
	testProcessName1 := "testProcessName1"
	testProcessName2 := "testProcessName2"
	testContractVersion := "1.0.0"

	process1 := &Process{
		contractName:    testContractName1,
		contractVersion: testContractVersion,
		processName:     testProcessName1,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	process2 := &Process{
		contractName:    testContractName2,
		contractVersion: testContractVersion,
		processName:     testProcessName2,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		process1,
		false,
	)
	processManager.addProcessToCache(
		testContractName2,
		testContractVersion,
		testProcessName2,
		process2,
		false,
	)

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		num int
	}
	tests := []struct {
		name               string
		fields             fields
		args               args
		wantProcessNum     int
		wantIdleProcessNum int
		wantErr            bool
	}{
		{
			name:               "TestProcessManager_batchPopIdleProcesses_Overflow",
			fields:             fields{processManager: processManager},
			args:               args{num: 3},
			wantProcessNum:     0,
			wantIdleProcessNum: 2,
			wantErr:            true,
		},
		{
			name:               "TestProcessManager_batchPopIdleProcesses_GoodCase",
			fields:             fields{processManager: processManager},
			args:               args{num: 2},
			wantProcessNum:     2,
			wantIdleProcessNum: 0,
			wantErr:            false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			got, err := pm.peekIdleProcesses(tt.args.num)
			if len(got) != tt.wantProcessNum {
				t.Errorf("peekIdleProcesses() = %v, want %v", len(got), tt.wantProcessNum)
			}
			if (err != nil) != tt.wantErr {
				t.Errorf("peekIdleProcesses() err = %v, wantErr %v", got, tt.wantErr)
			}
			idleProcessNum := pm.idleProcesses.Size()
			if idleProcessNum != tt.wantIdleProcessNum {
				t.Errorf("peekIdleProcesses() idleProcessNum = %v, wantIdleProcessNum %v", idleProcessNum, tt.wantIdleProcessNum)
			}
		})
	}
}

func TestProcessManager_getAvailableProcessNum(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	process1 := &Process{
		contractName:    testContractName1,
		contractVersion: testContractVersion,
		processName:     testProcessName1,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		process1,
		false,
	)

	type fields struct {
		processManager *ProcessManager
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{
			name:   "TestProcessManager_getAvailableProcessNum",
			fields: fields{processManager: processManager},
			want:   maxOriginalProcessNum - 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			if got := pm.getAvailableProcessNum(); got != tt.want {
				t.Errorf("getAvailableProcessNum() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestProcessManager_handleCleanIdleProcesses(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	for i := 0; i < maxOriginalProcessNum; i++ {
		processManager.addProcessToCache(
			testContractName1,
			testContractVersion,
			testProcessName1+strconv.Itoa(i),
			&Process{
				contractName:    testContractName1,
				contractVersion: testContractVersion,
				processName:     testProcessName1 + strconv.Itoa(i),
				eventCh:         make(chan interface{}, processEventChSize),
			},
			false,
		)
	}

	type fields struct {
		processManager *ProcessManager
	}

	tests := []struct {
		name               string
		fields             fields
		wantIdleProcessNum int
	}{
		{
			name:               "TestProcessManager_handleCleanIdleProcesses",
			fields:             fields{processManager: processManager},
			wantIdleProcessNum: maxOriginalProcessNum - int(float64(maxOriginalProcessNum)*releaseRate),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.handleCleanIdleProcesses()
			if tt.wantIdleProcessNum != pm.idleProcesses.Size() {
				t.Errorf("handleCleanIdleProcesses() idle process num= %v, want %v", pm.idleProcesses.Size(), tt.wantIdleProcessNum)
			}
		})
	}
}

func TestProcessManager_handleGetProcessReq(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	testContractName2 := "testContractName2"

	for i := 0; i < maxOriginalProcessNum; i++ {
		processManager.addProcessToCache(
			testContractName1,
			testContractVersion,
			testProcessName1+strconv.Itoa(i),
			&Process{
				contractName:    testContractName1,
				contractVersion: testContractVersion,
				processName:     testProcessName1 + strconv.Itoa(i),
				eventCh:         make(chan interface{}, processEventChSize),
			},
			false,
		)
	}

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		msg *messages.GetProcessReqMsg
	}

	tests := []struct {
		name                    string
		fields                  fields
		args                    args
		wantBusyProcessNum      int
		wantIdleProcessNum      int
		wantContract1ProcessNum int
		wantContract2ProcessNum int
	}{
		{
			name:   "TestProcessManager_handleGetProcessReq",
			fields: fields{processManager: processManager},
			args: args{msg: &messages.GetProcessReqMsg{
				ContractName:    testContractName2,
				ContractVersion: testContractVersion,
				ProcessNum:      4,
			}},
			wantBusyProcessNum:      4,
			wantIdleProcessNum:      maxOriginalProcessNum - 4,
			wantContract1ProcessNum: maxOriginalProcessNum - 4,
			wantContract2ProcessNum: 4,
		},
	}
	contractKey1 := utils.ConstructContractKey(testContractName1, testContractVersion)
	contractKey2 := utils.ConstructContractKey(testContractName2, testContractVersion)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := processManager
			pm.handleGetProcessReq(tt.args.msg)
			if tt.wantBusyProcessNum != len(pm.busyProcesses) {
				t.Errorf("TestProcessManager_handleGetProcessReq() busy process num= %v, want %v", len(pm.busyProcesses), tt.wantBusyProcessNum)
			}
			if tt.wantIdleProcessNum != pm.idleProcesses.Size() {
				t.Errorf("TestProcessManager_handleGetProcessReq() idle process num= %v, want %v", pm.idleProcesses.Size(), tt.wantIdleProcessNum)
			}
			if tt.wantContract1ProcessNum != len(pm.processGroups[contractKey1]) {
				t.Errorf("TestProcessManager_handleGetProcessReq() contract1 process num= %v, want %v", len(pm.processGroups[contractKey1]), tt.wantContract1ProcessNum)
			}
			if tt.wantContract2ProcessNum != len(pm.processGroups[contractKey2]) {
				t.Errorf("TestProcessManager_handleGetProcessReq() contract2 process num= %v, want %v", len(pm.processGroups[contractKey2]), tt.wantContract2ProcessNum)
			}
		})
	}
}

func TestProcessManager_handleSandboxExitResp(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log
	processManager.requestScheduler = &RequestScheduler{
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),
	}

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	process1 := &Process{
		contractName:    testContractName1,
		contractVersion: testContractVersion,
		processName:     testProcessName1,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		process1,
		false,
	)

	//processManager.waitingRequestGroups.Put(&messages.RequestGroupKey{
	//	ContractName:    testContractName1,
	//	ContractVersion: testContractVersion,
	//}, true)

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		msg *messages.SandboxExitMsg
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantProcessNum int
	}{
		{
			name:   "TestProcessManager_handleSandboxExitResp",
			fields: fields{processManager: processManager},
			args: args{msg: &messages.SandboxExitMsg{
				ContractName:    testContractName1,
				ContractVersion: testContractVersion,
				ProcessName:     testProcessName1,
				Err:             nil,
			}},
			wantProcessNum: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.handleSandboxExitMsg(tt.args.msg)
			if pm.idleProcesses.Size() != tt.wantProcessNum {
				t.Errorf("TestProcessManager_handleSandboxExitResp() process num= %v, want %v", pm.idleProcesses.Size(), tt.wantProcessNum)
			}
		})
	}
}

func TestProcessManager_removeFromProcessGroup(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log
	processManager.requestScheduler = &RequestScheduler{
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),
	}

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	process1 := &Process{
		contractName:    testContractName1,
		contractVersion: testContractVersion,
		processName:     testProcessName1,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		process1,
		false,
	)

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		contractName    string
		contractVersion string
		processName     string
	}

	tests := []struct {
		name           string
		fields         fields
		args           args
		wantProcessNum int
	}{
		{
			name:   "TestProcessManager_removeFromProcessGroup",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    testContractName1,
				contractVersion: testContractVersion,
				processName:     testProcessName1,
			},
			wantProcessNum: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.removeFromProcessGroup(tt.args.contractName, tt.args.contractVersion, tt.args.processName)
			groupKey := utils.ConstructContractKey(tt.args.contractName, tt.args.contractVersion)
			if len(pm.processGroups[groupKey]) != tt.wantProcessNum {
				t.Errorf("TestProcessManager_removeFromProcessGroup() process num= %v, want %v", pm.idleProcesses.Size(), tt.wantProcessNum)
			}
		})
	}
}

func TestProcessManager_removeProcessFromCache(t *testing.T) {

	SetConfig()

	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	releaseRate := config.DockerVMConfig.GetReleaseRate()
	userManager := NewUsersManager()
	log := logger.NewTestDockerLogger()

	processManager := NewProcessManager(maxOriginalProcessNum, releaseRate, false, userManager)
	processManager.logger = log
	processManager.requestScheduler = &RequestScheduler{
		closeCh: make(chan *messages.RequestGroupKey, closeChSize),
	}

	testContractName1 := "testContractName1"
	testProcessName1 := "testProcessName1"
	testContractVersion := "1.0.0"

	process1 := &Process{
		contractName:    testContractName1,
		contractVersion: testContractVersion,
		processName:     testProcessName1,
		eventCh:         make(chan interface{}, processEventChSize),
	}

	processManager.addProcessToCache(
		testContractName1,
		testContractVersion,
		testProcessName1,
		process1,
		false,
	)

	type fields struct {
		processManager *ProcessManager
	}

	type args struct {
		contractName    string
		contractVersion string
		processName     string
	}
	tests := []struct {
		name           string
		fields         fields
		args           args
		wantProcessNum int
	}{
		{
			name:   "TestProcessManager_removeProcessFromCache",
			fields: fields{processManager: processManager},
			args: args{
				contractName:    testContractName1,
				contractVersion: testContractVersion,
				processName:     testProcessName1,
			},
			wantProcessNum: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pm := tt.fields.processManager
			pm.removeProcessFromCache(tt.args.contractName, tt.args.contractVersion, tt.args.processName)
			if pm.idleProcesses.Size() != tt.wantProcessNum {
				t.Errorf("TestProcessManager_removeProcessFromCache() process num= %v, want %v", pm.idleProcesses.Size(), tt.wantProcessNum)
			}
		})
	}
}

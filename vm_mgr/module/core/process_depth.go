/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import "strings"

type ProcessDepth struct {
	processes map[string]*Process
}

func NewProcessDepth() *ProcessDepth {
	return &ProcessDepth{
		processes: make(map[string]*Process),
	}
}

func (pd *ProcessDepth) AddProcess(crossProcessName string, crossProcess *Process) {
	pd.processes[crossProcessName] = crossProcess
}

func (pd *ProcessDepth) GetProcess(crossProcessName string) *Process {
	return pd.processes[crossProcessName]
}

func (pd *ProcessDepth) RemoveProcess(crossProcessName string) {
	delete(pd.processes, crossProcessName)
}

func (pd *ProcessDepth) Size() int {
	return len(pd.processes)
}

func (pd *ProcessDepth) GetConcatProcessName() string {
	var resultName strings.Builder
	for processName, _ := range pd.processes {
		resultName.WriteString(processName)
		resultName.WriteString(", ")
	}
	return resultName.String()
}

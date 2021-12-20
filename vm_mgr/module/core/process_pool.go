/*
	Copyright (C) BABEC. All rights reserved.
	SPDX-License-Identifier: Apache-2.0
*/
package core

import (
	"sync"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"

	"chainmaker.org/chainmaker/protocol/v2"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"

	"go.uber.org/zap"
)

type ProcessContext struct {
	processList [protocol.CallContractDepth + 1]*Process
	size        uint32
}

type ProcessPool struct {
	ProcessTable sync.Map
	logger       *zap.SugaredLogger
}

func NewProcessPool() *ProcessPool {

	return &ProcessPool{
		ProcessTable: sync.Map{},
		logger:       logger.NewDockerLogger(logger.MODULE_PROCESS_POOL, config.DockerLogDir),
	}
}

func (pp *ProcessPool) newProcessContext() *ProcessContext {
	return &ProcessContext{
		processList: [protocol.CallContractDepth + 1]*Process{},
		size:        0,
	}
}

func (pp *ProcessPool) CheckProcessExist(processName string) (*Process, bool) {
	processContext, ok := pp.ProcessTable.Load(processName)
	if ok {
		return processContext.(*ProcessContext).processList[0], true
	}
	return nil, false
}

func (pp *ProcessPool) RetrieveProcessContext(initialProcessName string) *ProcessContext {
	pc, _ := pp.ProcessTable.Load(initialProcessName)
	processContext := pc.(*ProcessContext)
	return processContext
}

func (pp *ProcessPool) RetrieveHandlerFromProcess(processName string) *ProcessHandler {
	processContext, _ := pp.ProcessTable.Load(processName)
	return processContext.(*ProcessContext).processList[0].Handler
}

func (pp *ProcessPool) RegisterNewProcess(processName string, process *Process) {
	pp.logger.Debugf("register new process: [%s]", processName)
	newProcessContext := pp.newProcessContext()
	newProcessContext.processList[0] = process
	newProcessContext.size += 1
	pp.ProcessTable.Store(processName, newProcessContext)
}

func (pp *ProcessPool) ReleaseProcess(processName string) {
	pp.logger.Debugf("release process [%s]", processName)
	pp.ProcessTable.Delete(processName)
}

func (pp *ProcessPool) RegisterCrossProcess(initialProcessName string, calledProcess *Process) {

	pp.logger.Debugf("register cross process %s with initial process name %s",
		calledProcess.processName, initialProcessName)

	pc, _ := pp.ProcessTable.Load(initialProcessName)
	processContext := pc.(*ProcessContext)

	processContext.processList[processContext.size] = calledProcess
	processContext.size += 1

}

func (pp *ProcessPool) ReleaseCrossProcess(initialProcessName string, currentHeight uint32) {
	if currentHeight == 0 {
		return
	}
	pp.logger.Debugf("release cross process with position: %v", currentHeight)
	pc, _ := pp.ProcessTable.Load(initialProcessName)
	processContext := pc.(*ProcessContext)
	processContext.processList[currentHeight] = nil
	processContext.size -= 1

}

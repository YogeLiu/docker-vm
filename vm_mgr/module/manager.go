/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package module

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/rpc"
	security2 "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"go.uber.org/zap"
)

type ManagerImpl struct {
	cdmRpcServer   *rpc.CDMServer
	dmsRpcServer   *rpc.DMSServer
	scheduler      *core.DockerScheduler
	userController *core.UsersManager
	securityEnv    *security2.SecurityEnv
	processManager *core.ProcessManager
	logger         *zap.SugaredLogger
}

func NewManager(managerLogger *zap.SugaredLogger) (*ManagerImpl, error) {

	processManager := core.NewProcessManager(nil, nil)

	// new scheduler
	scheduler := core.NewDockerScheduler(processManager)
	processManager.SetScheduler(scheduler)

	// new chain maker to docker manager server
	cdmRpcServer, err := rpc.NewCDMServer()
	if err != nil {
		managerLogger.Errorf("fail to init new CDMServer, err: [%s]", err)
		return nil, err
	}

	manager := &ManagerImpl{
		cdmRpcServer:   cdmRpcServer,
		dmsRpcServer:   nil,
		scheduler:      scheduler,
		userController: nil,
		securityEnv:    nil,
		processManager: processManager,
		logger:         managerLogger,
	}

	return manager, nil
}

// InitContainer init all servers
func (m *ManagerImpl) InitContainer() {

	errorC := make(chan error, 1)

	var err error

	// start cdm server
	cdmApiInstance := rpc.NewCDMApi(m.scheduler)
	if err = m.cdmRpcServer.StartCDMServer(cdmApiInstance); err != nil {
		errorC <- err
	}

	// start scheduler
	m.scheduler.StartScheduler()

	m.logger.Infof("docker vm start successfully")

	// listen error signal
	err = <-errorC
	if err != nil {
		m.logger.Error("docker vm encounters error ", err)
	}
	m.StopManager()
	close(errorC)

}

// StopManager stop all servers
func (m *ManagerImpl) StopManager() {
	m.cdmRpcServer.StopCDMServer()
	m.dmsRpcServer.StopDMSServer()
	m.logger.Info("All is stopped!")
}

/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package module

import (
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/module/rpc"
	security2 "chainmaker.org/chainmaker/vm-docker-go/dockercontainer/module/security"
	"go.uber.org/zap"
)

type ManagerImpl struct {
	cdmRpcServer   *rpc.CDMServer
	dmsRpcServer   *rpc.DMSServer
	scheduler      *core.DockerScheduler
	userController *core.UsersManager
	securityEnv    *security2.SecurityEnv
	processPool    *core.ProcessPool
	logger         *zap.SugaredLogger
}

func NewManager(managerLogger *zap.SugaredLogger) (*ManagerImpl, error) {

	// set config
	securityEnv := security2.NewSecurityEnv()
	err := securityEnv.InitConfig()
	if err != nil {
		managerLogger.Errorf("fail to init directory: %s", err)
		return nil, err
	}

	// new users controller
	userController := core.NewUsersManager()

	// new process pool
	processPool := core.NewProcessPool()

	// new scheduler
	scheduler := core.NewDockerScheduler(userController, processPool)

	// new docker manager to sandbox server
	dmsRpcServer, err := rpc.NewDMSServer()
	if err != nil {
		managerLogger.Errorf("fail to init new DMSServer, err: [%s]", err)
		return nil, err
	}

	// new chain maker to docker manager server
	cdmRpcServer, err := rpc.NewCDMServer()
	if err != nil {
		managerLogger.Errorf("fail to init new DMSServer, err: [%s]", err)
		return nil, err
	}

	manager := &ManagerImpl{
		cdmRpcServer:   cdmRpcServer,
		dmsRpcServer:   dmsRpcServer,
		scheduler:      scheduler,
		userController: userController,
		securityEnv:    securityEnv,
		processPool:    processPool,
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

	// start dms server
	dmsApiInstance := rpc.NewDMSApi(m.processPool)
	if err = m.dmsRpcServer.StartDMSServer(dmsApiInstance); err != nil {
		errorC <- err
	}

	// init sandBox
	if err = m.securityEnv.InitSecurityEnv(); err != nil {
		errorC <- err
	}

	// create new users
	go func() {
		err = m.userController.CreateNewUsers()
		if err != nil {
			errorC <- err
		}
	}()

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
	m.scheduler.StopScheduler()
	m.logger.Info("All is stopped!")
}

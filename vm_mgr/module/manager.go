/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package module

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/rpc"
	security2 "chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"go.uber.org/zap"
)

type ManagerImpl struct {
	cdmRpcServer   *rpc.ChainRPCServer
	dmsRpcServer   *rpc.SandboxRPCServer
	scheduler      *core.DockerScheduler
	userController *core.UsersManager
	securityEnv    *security2.SecurityEnv
	processManager *core.ProcessManager
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
	usersManager := core.NewUsersManager()

	contractManager := core.NewContractManager()
	// new process pool
	processManager := core.NewProcessManager(usersManager, contractManager)

	// new scheduler
	scheduler := core.NewDockerScheduler(processManager)
	processManager.SetScheduler(scheduler)
	contractManager.SetScheduler(scheduler)

	managerLogger.Debugf("init grpc server, max send size [%dM], max recv size[%dM]",
		utils.GetMaxSendMsgSizeFromEnv(), utils.GetMaxRecvMsgSizeFromEnv())

	// new docker manager to sandbox server
	dmsRpcServer, err := rpc.NewSandboxRPCServer()
	if err != nil {
		managerLogger.Errorf("fail to init new SandboxRPCServer, err: [%s]", err)
		return nil, err
	}

	// new chain maker to docker manager server
	cdmRpcServer, err := rpc.NewChainRPCServer()
	if err != nil {
		managerLogger.Errorf("fail to init new ChainRPCServer, err: [%s]", err)
		return nil, err
	}

	manager := &ManagerImpl{
		cdmRpcServer:   cdmRpcServer,
		dmsRpcServer:   dmsRpcServer,
		scheduler:      scheduler,
		userController: usersManager,
		securityEnv:    securityEnv,
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
	cdmApiInstance := rpc.NewChainRPCService(m.scheduler)
	if err = m.cdmRpcServer.StartChainRPCServer(cdmApiInstance); err != nil {
		errorC <- err
	}

	// start dms server
	dmsApiInstance := rpc.NewSandboxRPCService(m.processManager)
	if err = m.dmsRpcServer.StartSandboxRPCServer(dmsApiInstance); err != nil {
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
	m.cdmRpcServer.StopChainRPCServer()
	m.dmsRpcServer.StopSandboxRPCServer()
	m.logger.Info("All is stopped!")
}

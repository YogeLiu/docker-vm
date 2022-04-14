/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package module

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/core"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/rpc"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module/security"
	"fmt"
	"go.uber.org/zap"
	"path/filepath"
)

type ManagerImpl struct {
	chainRPCServer   *rpc.ChainRPCServer
	sandboxRPCServer *rpc.SandboxRPCServer
	scheduler        *core.RequestScheduler
	userController *core.UserManager
	securityEnv    *security.SecurityCenter
	processManager *core.ProcessManager
	logger         *zap.SugaredLogger
}

func NewManager(managerLogger *zap.SugaredLogger) (*ManagerImpl, error) {

	// set config
	if err := config.InitConfig(filepath.Join(config.DockerMountDir, config.ConfigFileName)); err != nil {
		managerLogger.Fatalf("failed to init config, %v", err)
	}

	securityEnv := security.NewSecurityCenter()

	// new user controller
	usersManager := core.NewUsersManager()

	// new contract manager
	contractManager, err := core.NewContractManager()
	if err != nil {
		return nil, fmt.Errorf("failed to new contract manager, %v", err)
	}

	// new original process manager
	maxOriginalProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum
	maxCrossProcessNum := config.DockerVMConfig.Process.MaxOriginalProcessNum * protocol.CallContractDepth
	releaseRate := config.DockerVMConfig.GetReleaseRate()

	origProcessManager := core.NewProcessManager(maxOriginalProcessNum, releaseRate, false, usersManager)
	crossProcessManager := core.NewProcessManager(maxCrossProcessNum, releaseRate, true, usersManager)


	managerLogger.Debugf("init grpc server, max send size [%dM], max recv size[%dM]",
		config.DockerVMConfig.RPC.MaxSendMsgSize, config.DockerVMConfig.RPC.MaxRecvMsgSize)

	// new chain maker to docker manager server
	chainRPCServer, err := rpc.NewChainRPCServer()
	if err != nil {
		return nil, fmt.Errorf("fail to init new ChainRPCServer, %v", err)
	}

	// new docker manager to sandbox server
	sandboxRPCServer, err := rpc.NewSandboxRPCServer()
	if err != nil {
		return nil, fmt.Errorf("fail to init new SandboxRPCServer, %v", err)
	}

	// start cdm server
	chainRPCService := rpc.NewChainRPCService()
	if err = chainRPCServer.StartChainRPCServer(chainRPCService); err != nil {
		return nil, fmt.Errorf("failed to start ChainRPCService, %v", err)
	}

	// start dms server
	sandboxRPCService := rpc.NewSandboxRPCService(origProcessManager, crossProcessManager)
	if err = sandboxRPCServer.StartSandboxRPCServer(sandboxRPCService); err != nil {
		return nil, fmt.Errorf("failed to start SandboxRPCService, %v", err)
	}

	// new scheduler
	scheduler := core.NewRequestScheduler(chainRPCService, origProcessManager, crossProcessManager, contractManager)
	origProcessManager.SetScheduler(scheduler)
	crossProcessManager.SetScheduler(scheduler)
	contractManager.SetScheduler(scheduler)
	chainRPCService.SetScheduler(scheduler)



	manager := &ManagerImpl{
		chainRPCServer:   chainRPCServer,
		sandboxRPCServer: sandboxRPCServer,
		scheduler:        scheduler,
		userController:   usersManager,
		securityEnv:      securityEnv,
		processManager:   origProcessManager,
		logger:           managerLogger,
	}

	return manager, nil
}

// InitContainer init all servers
func (m *ManagerImpl) InitContainer() {

	errorC := make(chan error, 1)

	var err error

	// init sandBox
	if err = m.securityEnv.InitSecurityCenter(); err != nil {
		errorC <- err
	}

	// create new users
	go func() {
		err = m.userController.BatchCreateUsers()
		if err != nil {
			errorC <- err
		}
	}()

	// start scheduler
	m.scheduler.Start()

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
	m.chainRPCServer.StopChainRPCServer()
	m.sandboxRPCServer.StopSandboxRPCServer()
	m.logger.Info("All is stopped!")
}

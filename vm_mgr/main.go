package main

import (
	"fmt"
	"net/http"
	"path/filepath"
	"time"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"

	_ "net/http/pprof"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	// set config
	err := config.InitConfig(filepath.Join(config.DockerMountDir, config.ConfigFileName))

	// init docker container logger
	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER)
	if err != nil {
		managerLogger.Warnf("failed to init config, %v, use default config", err)
	}

	//err = initSockPath(managerLogger)
	//if err != nil {
	//	managerLogger.Errorf("failed to init sock path, %v", err)
	//	return
	//}
	// start pprof
	startPProf(managerLogger)

	// new docker manager
	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("failed to create docker manager: %s", err)
		return
	}

	managerLogger.Debugf("docker manager created")

	// init docker manager
	go manager.InitContainer()

	managerLogger.Debugf("docker manager initialized")

	// infinite loop, finished when call stopVM outside
	for i := 0; ; i++ {
		time.Sleep(time.Hour)
	}
}

// start pprof when open enable pprof switch
func startPProf(managerLogger *zap.SugaredLogger) {

	enablePProf := config.DockerVMConfig.Pprof.ContractEnginePprof.Enable

	if enablePProf {
		managerLogger.Infof("start pprof")
		go func() {
			pprofPort := config.DockerVMConfig.Pprof.ContractEnginePprof.Port
			addr := fmt.Sprintf(":%d", pprofPort)
			err := http.ListenAndServe(addr, nil)
			if err != nil {
				managerLogger.Errorf("failed to start pprof, %v", err)
			}
		}()
	}
}

//func initSockPath(managerLogger *zap.SugaredLogger) error {
//	// mkdir paths
//	sockDir := filepath.Join(config.DockerMountDir, config.SandboxRPCDir)
//	err := utils.CreateDir(sockDir)
//	if err != nil {
//		return err
//	}
//	managerLogger.Debug("set sock dir: ", sockDir)
//
//	return nil
//}

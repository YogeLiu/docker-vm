package main

import (
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER, config.DockerLogDir)

	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("Err in creating docker manager: %s", err)
		return
	}

	managerLogger.Debugf("docker manager created")

	go manager.InitContainer()

	managerLogger.Debugf("docker manager init...")

	// infinite loop
	for i := 0; ; i++ {
		time.Sleep(time.Hour)
	}
}

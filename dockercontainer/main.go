package main

import (
	"time"

	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/logger"
	"chainmaker.org/chainmaker-go/docker-go/dockercontainer/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER)

	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("Err in creating docker manager: %s", err)
		return
	}

	managerLogger.Infof("docker manager created")

	go manager.InitContainer()

	managerLogger.Infof("docker manager init...")

	// infinite loop
	for i := 0; ; i++ {
		time.Sleep(time.Hour)
	}
}

package main

import (
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "net/http/pprof"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/vm_mgr/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER, config.DockerLogDir)

	enablePProf, _ := strconv.ParseBool(os.Getenv(config.EnvEnablePprof))
	if enablePProf {
		startPProf(managerLogger)
	}

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

func startPProf(managerLogger *zap.SugaredLogger) {
	managerLogger.Infof("start pprof")

	go func() {
		pprofPort, _ := strconv.Atoi(os.Getenv(config.EnvPprofPort))
		addr := fmt.Sprintf(":%d", pprofPort)
		err := http.ListenAndServe(addr, nil)
		if err != nil {
			fmt.Println(err)
		}
	}()
}

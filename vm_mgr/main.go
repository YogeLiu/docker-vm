package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/module"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	// init docker container logger
	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER, config.DockerLogDir)

	// new docker manager
	manager, err := module.NewManager(managerLogger)
	if err != nil {
		managerLogger.Errorf("Err in creating docker manager: %s", err)
		return
	}

	managerLogger.Debugf("docker manager created")

	// init docker manager
	go manager.InitContainer()

	managerLogger.Debugf("docker manager init...")

	// infinite loop, finished when call stopVM outside
	for i := 0; ; i++ {
		time.Sleep(time.Hour)
	}
}

// start pprof when open enable pprof switch
func startPProf(managerLogger *zap.SugaredLogger) {

	envEnablePProf := os.Getenv(config.EnvEnablePprof)

	if len(envEnablePProf) != 0 {
		enablePProf, _ := strconv.ParseBool(envEnablePProf)
		if enablePProf {
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
	}
}

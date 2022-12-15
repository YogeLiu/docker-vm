/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package main

import (
	"fmt"
	"net/http"
	_ "net/http/pprof"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/logger"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/module"
	"chainmaker.org/chainmaker/vm-docker-go/v3/vm_mgr/utils"
	"go.uber.org/zap"
)

var managerLogger *zap.SugaredLogger

func main() {

	// init docker container logger
	managerLogger = logger.NewDockerLogger(logger.MODULE_MANAGER, config.DockerLogDir)

	// start pprof
	startPProf(managerLogger)

	// init files paths
	err := initFilesPath(managerLogger)
	if err != nil {
		managerLogger.Errorf("fail to init paths, err: [%s]", err)
		return
	}

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

func initFilesPath(logger *zap.SugaredLogger) error {
	var err error
	// mkdir socket paths
	sockDir := filepath.Join(config.DockerMountDir, config.SockDir)
	err = utils.CreateDir(sockDir)
	if err != nil {
		return err
	}
	logger.Debug("set sock dir: ", sockDir)

	return nil
}

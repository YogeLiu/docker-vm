package utils

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"go.uber.org/zap"
)

// WriteToFile WriteFile write value to file
func WriteToFile(path string, value int) error {
	if err := ioutil.WriteFile(path, []byte(fmt.Sprintf("%d", value)), 0755); err != nil {
		return err
	}
	return nil
}

func WriteToFIle(path, info string) error {
	if err := ioutil.WriteFile(path, []byte(info), 0755); err != nil {
		return err
	}
	return nil
}

// RunCmd exec cmd
func RunCmd(command string) error {
	commands := strings.Split(command, " ")
	cmd := exec.Command(commands[0], commands[1:]...)

	cmd.Stdout = os.Stdout

	if err := cmd.Start(); err != nil {
		return err
	}

	if err := cmd.Wait(); err != nil {
		return err
	}

	return nil
}

func ProcessLoggerName(processName string) string {
	loggerName := "[ " + processName + " ]"
	return loggerName
}

func GetTestLogPath() string {
	basePath, _ := os.Getwd()
	return basePath + config.TestPath
}

func GetLogHandler() *zap.SugaredLogger {
	return logger.NewDockerLogger(logger.MODULE_PROCESS, GetTestLogPath())
}

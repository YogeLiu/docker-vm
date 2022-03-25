package security

import (
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
	"path/filepath"
	"strings"
)

const (
	ipcPath = "/proc/sys/kernel"
	ipcFiles   = "shmmax,shmall,msgmax,msgmnb,msgmni"
	ipcSemFile = "sem"
)

func (s *SecurityCenter) disableIPC() error {
	fileList := strings.Split(ipcFiles, ",")
	for _, f := range fileList {
		currentFile := filepath.Join(ipcPath, f)
		err := utils.WriteToFile(currentFile, "0")
		if err != nil {
			return err
		}
	}

	ipcSemPath := filepath.Join(ipcPath, ipcSemFile)
	err := utils.WriteToFile(ipcSemPath, "0 0 0 0")
	if err != nil {
		return err
	}
	return nil
}

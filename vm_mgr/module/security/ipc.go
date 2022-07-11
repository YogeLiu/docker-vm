/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
	"path/filepath"
	"strings"
)

const (
	_ipcPath    = "/proc/sys/kernel"
	_ipcFiles   = "shmmax,shmall,msgmax,msgmnb,msgmni"
	_ipcSemFile = "sem"
)

func (s *SecurityCenter) disableIPC() error {
	fileList := strings.Split(_ipcFiles, ",")
	for _, f := range fileList {
		currentFile := filepath.Join(_ipcPath, f)
		err := utils.WriteToFile(currentFile, "0")
		if err != nil {
			return err
		}
	}

	ipcSemPath := filepath.Join(_ipcPath, _ipcSemFile)
	err := utils.WriteToFile(ipcSemPath, "0 0 0 0")
	if err != nil {
		return err
	}
	return nil
}

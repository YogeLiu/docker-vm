/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"os"
	"path/filepath"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/utils"
)

type CGroup struct {
}

func SetCGroup() error {
	if _, err := os.Stat(config.CGroupRoot); os.IsNotExist(err) {
		err = os.MkdirAll(config.CGroupRoot, 0755)
		if err != nil {
			return err
		}
	}

	err := setMemoryList()
	if err != nil {
		return err
	}
	return nil
}

func setMemoryList() error {
	// set memroy limit
	mPath := filepath.Join(config.CGroupRoot, config.MemoryLimitFile)
	err := utils.WriteToFile(mPath, config.RssLimit*1024*1024)
	if err != nil {
		return err
	}

	// set swap memory limit to zero
	sPath := filepath.Join(config.CGroupRoot, config.SwapLimitFile)
	err = utils.WriteToFile(sPath, 0)
	if err != nil {
		return err
	}

	return nil
}

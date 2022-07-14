/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"os"
	"path/filepath"
	"strconv"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/utils"
)

const (
	CGroupRoot      = "/sys/fs/cgroup/memory/chainmaker" // cgroup location
	ProcsFile       = "cgroup.procs"                     // process file
	MemoryLimitFile = "memory.limit_in_bytes"            // memory limit file
	SwapLimitFile   = "memory.swappiness"                // swap setting file
	RssLimit        = 50000                              // rss limit file, 10MB
)

type CGroup struct {
}

// setCGroup sets cgroup in order to limit memory and swap
func setCGroup() error {
	if _, err := os.Stat(CGroupRoot); os.IsNotExist(err) {
		err = os.Mkdir(CGroupRoot, 0755)
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
	// set memory limit
	mPath := filepath.Join(CGroupRoot, MemoryLimitFile)
	err := utils.WriteToFile(mPath, strconv.FormatInt(RssLimit*1024*1024, 10))
	if err != nil {
		return err
	}

	// set swap memory limit to zero
	sPath := filepath.Join(CGroupRoot, SwapLimitFile)
	err = utils.WriteToFile(sPath, "0")
	if err != nil {
		return err
	}

	return nil
}

/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/logger"
	"go.uber.org/zap"
	"os"
)

// SecurityCenter handles all security configs
type SecurityCenter struct {
	logger *zap.SugaredLogger
}

// NewSecurityCenter returns security env
func NewSecurityCenter() *SecurityCenter {
	return &SecurityCenter{
		logger: logger.NewDockerLogger(logger.MODULE_SECURITY_ENV),
	}
}

// InitSecurityCenter set security env, includes:
// 1. chmod "/tmp" 0755
// 2. set cgroup
// 3. disable IPC: disable inter-process communication to ensure isolation
func (s *SecurityCenter) InitSecurityCenter() error {
	if err := os.Chmod("/tmp/", 0755); err != nil {
		return err
	}

	if err := setCGroup(); err != nil {
		s.logger.Errorf("failed to setCGroup, err : [%s]", err)
		return err
	}

	if err := s.disableIPC(); err != nil {
		s.logger.Errorf("failed to set ipc err: [%s]", err)
		return err
	}

	s.logger.Debugf("init security env completed")

	return nil
}

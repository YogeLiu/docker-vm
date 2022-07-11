/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package core

import (
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/interfaces"
	"fmt"
	"path/filepath"
)

// User is linux user
type User struct {
	Uid      int    // user id
	Gid      int    // user group id
	UserName string // username
	SockPath string // sandbox rpc sock path
}

// check interface implement
var _ interfaces.User = (*User)(nil)

// NewUser returns new user
func NewUser(id int) *User {
	userName := fmt.Sprintf("u-%d", id)
	sockPath := filepath.Join(config.SandboxRPCDir, config.SandboxRPCSockName)

	return &User{
		Uid:      id,
		Gid:      id,
		UserName: userName,
		SockPath: sockPath,
	}
}

// GetUid returns user id
func (u *User) GetUid() int {
	return u.Uid
}

// GetGid returns group id
func (u *User) GetGid() int {
	return u.Gid
}

// GetSockPath returns sock path
func (u *User) GetSockPath() string {
	return u.SockPath
}

// GetUserName returns user name
func (u *User) GetUserName() string {
	return u.UserName
}

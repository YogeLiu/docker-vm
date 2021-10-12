/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package security

type User struct {
	Uid      int
	Gid      int
	UserName string
	SockPath string
}

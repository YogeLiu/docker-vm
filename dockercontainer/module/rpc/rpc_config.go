/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package rpc

import "time"

const (
	// ServerMinInterval server min interval
	ServerMinInterval = time.Duration(1) * time.Minute
	// ConnectionTimeout connection timeout time
	ConnectionTimeout = 5 * time.Second
)

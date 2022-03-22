/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import "time"

const (
	// ServerMinInterval is the server min interval
	ServerMinInterval = time.Duration(1) * time.Minute

	// ConnectionTimeout is the connection timeout time
	ConnectionTimeout = 5 * time.Second

	// ServerKeepAliveTime is the idle duration before server ping
	ServerKeepAliveTime = 1 * time.Minute

	// ServerKeepAliveTimeout is the ping timeout
	ServerKeepAliveTimeout = 20 * time.Second
)

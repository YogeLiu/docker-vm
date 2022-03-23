/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import "time"

const (
	// CGroupRoot cgroup location is not allow user to change
	CGroupRoot = "/sys/fs/cgroup/memory/chainmaker"
	// ProcsFile process file
	ProcsFile = "cgroup.procs"
	// MemoryLimitFile memory limit file
	MemoryLimitFile = "memory.limit_in_bytes"
	// SwapLimitFile swap setting file
	SwapLimitFile = "memory.swappiness"
	// RssLimit rss limit file
	RssLimit = 50000 // 10 MB

	// SandboxRPCDir docker manager sandbox dir
	SandboxRPCDir = "/dms"
	// SandboxRPCSockPath docker manager sandbox domain socket path
	SandboxRPCSockPath = "dms.sock"

	// DockerMountDir mount directory in docker
	DockerMountDir = "/mount"
	// DockerLogDir mount directory for log
	DockerLogDir = "/log"
	// LogFileName log name
	LogFileName = "docker-go.log"

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// SockDir dir save domain socket file
	SockDir = "sock"
	// ChainRPCSockName domain socket file name
	ChainRPCSockName = "cdm.sock"

	// TestPath docker log dir for test
	TestPath = "/"

	// ServerMinInterval is the server min interval
	ServerMinInterval = time.Duration(1) * time.Minute
	// ConnectionTimeout is the connection timeout time
	ConnectionTimeout = 5 * time.Second
	// ServerKeepAliveTime is the idle duration before server ping
	ServerKeepAliveTime = 1 * time.Minute
	// ServerKeepAliveTimeout is the ping timeout
	ServerKeepAliveTimeout = 20 * time.Second

)

var (
	// ContractBaseDir contract base directory, save here for easy use
	ContractBaseDir string
	// ShareBaseDir share base directory
	ShareBaseDir string
	// SockBaseDir domain socket directory
	SockBaseDir string
	// SandBoxTimeout sandbox timeout
	SandBoxTimeout = 2
	// SandBoxLogLevel sand box log level defaut is INFO
	SandBoxLogLevel string
)

const (
	ENV_ENABLE_UDS        = "ENV_ENABLE_UDS"
	ENV_USER_NUM          = "ENV_USER_NUM"
	ENV_TX_TIME_LIMIT     = "ENV_TX_TIME_LIMIT"
	ENV_LOG_LEVEL         = "ENV_LOG_LEVEL"
	ENV_LOG_IN_CONSOLE    = "ENV_LOG_IN_CONSOLE"
	ENV_MAX_CONCURRENCY   = "ENV_MAX_CONCURRENCY"
	ENV_MAX_SEND_MSG_SIZE = "ENV_MAX_SEND_MSG_SIZE"
	ENV_MAX_RECV_MSG_SIZE = "ENV_MAX_RECV_MSG_SIZE"

	EnvEnablePprof = "ENV_ENABLE_PPROF"
	EnvPprofPort   = "ENV_PPROF_PORT"

	ENV_MAX_LOCAL_CONTRACT_NUM = "ENV_MAX_LOCAL_CONTRACT_NUM"
)

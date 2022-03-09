/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

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

	// DMSDir docker manager sandbox dir
	DMSDir = "/dms"
	// DMSSockPath docker manager sandbox domain socket path
	DMSSockPath = "dms.sock"

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
	// SockName domain socket file name
	SockName = "cdm.sock"

	MaxSendSize = 20
	MaxRecvSize = 20

	// DefaultUserNum user num
	DefaultUserNum = 100
	// DefaultListenPort default listening port in container
	DefaultListenPort     = 22359
	DefaultTxTimeLimit    = 2
	DefaultLogLevel       = "INFO"
	DefaultLogInConsole   = false
	DefaultMaxConcurrency = 50

	// TestPath docker log dir for test
	TestPath = "/"
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
	ENV_USER_NUM          = "ENV_USER_NUM"
	ENV_TX_TIME_LIMIT     = "ENV_TX_TIME_LIMIT"
	ENV_LOG_LEVEL         = "ENV_LOG_LEVEL"
	ENV_LOG_IN_CONSOLE    = "ENV_LOG_IN_CONSOLE"
	ENV_MAX_CONCURRENCY   = "ENV_MAX_CONCURRENCY"
	ENV_MAX_SEND_MSG_SIZE = "ENV_MAX_SEND_MSG_SIZE"
	ENV_MAX_RECV_MSG_SIZE = "ENV_MAX_RECV_MSG_SIZE"
	ENV_VM_SERVICE_PORT   = "ENV_VM_SERVICE_PORT"

	EnvEnablePprof = "ENV_ENABLE_PPROF"
	EnvPprofPort   = "ENV_PPROF_PORT"
)

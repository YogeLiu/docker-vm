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
	RssLimit = 500 // 10 MB

	// DMSDir docker manager sandbox dir
	DMSDir = "/dms"
	// DMSSockPath docker manager sandbox domain socket path
	DMSSockPath = "dms.sock"

	// ContractsDir dir save executable contract
	ContractsDir = "contracts"
	// ShareDir dir save log
	ShareDir = "share"
	// SockDir dir save domain socket file
	SockDir = "sock"
	// SockName domain socket file name
	SockName = "cdm.sock"
	// DockerMountDir mount directory in docker
	DockerMountDir = "/mount"
	// DockerLogDir mount directory for log
	DockerLogDir = "/log"
	// LogFileName log name
	LogFileName = "docker-go.log"
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

/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package config

import (
	"chainmaker.org/chainmaker/protocol/v2"
	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/logger"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"sync"
	"time"
)

const (
	// DockerMountDir mount directory in docker
	DockerMountDir = "/mount"

	// ConfigFileName is the docker vm config file path
	ConfigFileName = "vm"

	// ConfigFileType is the docker vm config file type
	ConfigFileType = "yml"
)

var DockerVMConfig *conf
var once sync.Once

type conf struct {
	logger   *zap.SugaredLogger
	RPC      rpcConf      `mapstructure:"rpc"`
	Process  processConf  `mapstructure:"process"`
	Log      logConf      `mapstructure:"log"`
	Pprof    pprofConf    `mapstructure:"pprof"`
	Contract contractConf `mapstructure:"contract"`
}

type ChainRPCProtocolType int

const (
	UDS ChainRPCProtocolType = iota
	TCP
)

type rpcConf struct {
	ChainRPCProtocol       ChainRPCProtocolType `mapstructure:"chain_rpc_protocol"`
	ChainRPCPort           int                  `mapstructure:"chain_rpc_port"`
	MaxSendMsgSize         int                  `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize         int                  `mapstructure:"max_recv_msg_size"`
	ServerMinInterval      int                  `mapstructure:"provider"`
	ConnectionTimeout      int                  `mapstructure:"provider"`
	ServerKeepAliveTime    int                  `mapstructure:"provider"`
	ServerKeepAliveTimeout int                  `mapstructure:"provider"`
}

type processConf struct {
	MaxOriginalProcessNum int `mapstructure:"max_original_process_num"`
	BusyTimeout           int `mapstructure:"busy_timeout"`
	ReadyTimeout          int `mapstructure:"ready_timeout"`
	ReleaseRate           int `mapstructure:"release_rate"`
	ReleasePeriod         int `mapstructure:"release_period"`
}

type logConf struct {
	ContractEngineLog logInstanceConf `mapstructure:"contract_engine"`
	SandboxLog        logInstanceConf `mapstructure:"sandbox"`
}

type logInstanceConf struct {
	Level   string `mapstructure:"level"`
	Console bool   `mapstructure:"console"`
}

type pprofConf struct {
	ContractEnginePprof pprofInstanceConf `mapstructure:"contract_engine"`
	SandboxPprof        pprofInstanceConf `mapstructure:"sandbox"`
}

type pprofInstanceConf struct {
	Enable bool `mapstructure:"enable"`
	Port   int  `mapstructure:"port"`
}

type contractConf struct {
	MaxFileNum int `mapstructure:"max_file_num"`
}

func InitConfig() {
	once.Do(func() {
		// init viper
		viper.SetConfigName(ConfigFileName)
		viper.AddConfigPath(DockerMountDir)
		viper.SetConfigType(ConfigFileType)

		// read config from file
		DockerVMConfig = &conf{
			logger: logger.NewDockerLogger(logger.MODULE_CONFIG),
		}
		if err := viper.ReadInConfig(); err != nil {
			DockerVMConfig.logger.Fatalf("failed to read conf, %v", err)
		}

		DockerVMConfig.setDefaultConfigs()

		// unmarshal config
		err := viper.Unmarshal(&DockerVMConfig)
		if err != nil {
			DockerVMConfig.logger.Fatalf("failed to unmarshal conf file, %v", err)
		}

		DockerVMConfig.logger.Infof("vm conf loaded: %+v", DockerVMConfig)
	})
}

func (c *conf) setDefaultConfigs() {

	// set rpc default configs
	const rpcPrefix = "rpc"
	viper.SetDefault(rpcPrefix+".chain_rpc_protocol", 1)
	viper.SetDefault(rpcPrefix+".chain_rpc_port", 8015)
	viper.SetDefault(rpcPrefix+".max_send_msg_size", 4)
	viper.SetDefault(rpcPrefix+".max_recv_msg_size", 4)
	viper.SetDefault(rpcPrefix+".server_min_interval", 60)
	viper.SetDefault(rpcPrefix+".connection_timeout", 5)
	viper.SetDefault(rpcPrefix+".server_keep_alive_time", 60)
	viper.SetDefault(rpcPrefix+".server_keep_alive_timeout", 20)

	// set process default configs
	const processPrefix = "process"
	viper.SetDefault(processPrefix+".max_original_process_num", 50)
	viper.SetDefault(processPrefix+".busy_timeout", 2000)
	viper.SetDefault(processPrefix+".idle_timeout", 200)
	viper.SetDefault(processPrefix+".release_rate", 30)
	viper.SetDefault(processPrefix+".release_period", 10)

	// set log default configs
	const logPrefix = "log"
	viper.SetDefault(logPrefix+".contract_engine.level", "info")
	viper.SetDefault(logPrefix+".sandbox.level", "info")

	// set pprof default configs
	const pprofPrefix = "pprof"
	viper.SetDefault(pprofPrefix+".contract_engine.port", 21521)
	viper.SetDefault(pprofPrefix+".sandbox.port", 21522)

	// set contract default configs
	const contractPrefix = "contract"
	viper.SetDefault(contractPrefix+".max_file_num", 1024)
}

func (c *conf) restrainConfig() {
	if c.Process.ReleaseRate < 0 {
		c.Process.ReleaseRate = 0
	} else if c.Process.ReleaseRate > 100 {
		c.Process.ReleaseRate = 100
	}
}

func (c *conf) GetServerMinInterval() time.Duration {
	return time.Duration(c.RPC.ServerMinInterval) * time.Second
}

func (c *conf) GetConnectionTimeout() time.Duration {
	return time.Duration(c.RPC.ConnectionTimeout) * time.Second
}

func (c *conf) GetServerKeepAliveTime() time.Duration {
	return time.Duration(c.RPC.ServerKeepAliveTime) * time.Second
}

func (c *conf) GetServerKeepAliveTimeout() time.Duration {
	return time.Duration(c.RPC.ServerKeepAliveTimeout) * time.Second
}

func (c *conf) GetBusyTimeout() time.Duration {
	return time.Duration(c.Process.BusyTimeout) * time.Millisecond
}

func (c *conf) GetReadyTimeout() time.Duration {
	return time.Duration(c.Process.ReadyTimeout) * time.Millisecond
}

func (c *conf) GetReleasePeriod() time.Duration {
	return time.Duration(c.Process.ReleasePeriod) * time.Second
}

func (c *conf) GetReleaseRate() float64 {
	return float64(c.Process.ReleaseRate) / 100.0
}

func (c *conf) GetMaxUserNum() int {
	return c.Process.MaxOriginalProcessNum * (protocol.CallContractDepth + 1)
}

/*
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/
package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"chainmaker.org/chainmaker/protocol/v2"
	"github.com/spf13/viper"
)

const (
	// DockerMountDir mount directory in docker
	DockerMountDir = "/mount"

	// ConfigFileName is the docker vm config file path
	ConfigFileName = "config/vm.yml"

	// SandboxRPCDir docker manager sandbox dir
	SandboxRPCDir = "/sandbox"

	// SandboxRPCSockName docker manager sandbox domain socket path
	SandboxRPCSockName = "sandbox.sock"
)

// DockerVMConfig is the docker vm config
var DockerVMConfig *conf

type conf struct {
	RPC      rpcConf      `mapstructure:"rpc"`
	Process  processConf  `mapstructure:"process"`
	Log      logConf      `mapstructure:"log"`
	Pprof    pprofConf    `mapstructure:"pprof"`
	Contract contractConf `mapstructure:"contract"`
	Slow     slowConf     `mapstructure:"slow"`
}

type ChainRPCProtocolType int

const (
	UDS ChainRPCProtocolType = iota
	TCP
)

type rpcConf struct {
	ChainRPCProtocol       ChainRPCProtocolType `mapstructure:"chain_rpc_protocol"`
	ChainHost              string               `mapstructure:"chain_host"`
	ChainRPCPort           int                  `mapstructure:"chain_rpc_port"`
	SandboxRPCPort         int                  `mapstructure:"sandbox_rpc_port"`
	MaxSendMsgSize         int                  `mapstructure:"max_send_msg_size"`
	MaxRecvMsgSize         int                  `mapstructure:"max_recv_msg_size"`
	ServerMinInterval      time.Duration        `mapstructure:"server_min_interval"`
	ConnectionTimeout      time.Duration        `mapstructure:"connection_timeout"`
	ServerKeepAliveTime    time.Duration        `mapstructure:"server_keep_alive_time"`
	ServerKeepAliveTimeout time.Duration        `mapstructure:"server_keep_alive_timeout"`
}

type processConf struct {
	MaxOriginalProcessNum int           `mapstructure:"max_original_process_num"`
	ExecTxTimeout         time.Duration `mapstructure:"exec_tx_timeout"`
	WaitingTxTime         time.Duration `mapstructure:"waiting_tx_time"`
	ReleaseRate           int           `mapstructure:"release_rate"`
	ReleasePeriod         time.Duration `mapstructure:"release_period"`
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
	MaxFileSize int `mapstructure:"max_file_size"`
}

type slowConf struct {
	Disable  bool          `mapstructure:"disable"`
	StepTime time.Duration `mapstructure:"step_time"`
	TxTime   time.Duration `mapstructure:"tx_time"`
}

func InitConfig(configFileName string) error {
	// init viper
	viper.SetConfigFile(configFileName)

	var err error

	DockerVMConfig.setDefaultConfigs()

	// read config from file
	if err = viper.ReadInConfig(); err != nil {
		err = fmt.Errorf("failed to read conf %v, use default configs", err)
	}

	// unmarshal config
	if marsErr := viper.Unmarshal(&DockerVMConfig); marsErr != nil {
		err = fmt.Errorf("%v, failed to unmarshal conf file, %v", err, marsErr)
	}

	if envErr := DockerVMConfig.setEnv(); err != nil {
		err = fmt.Errorf("%v, failed to set config env, %v", err, envErr)
	}

	return err
}

func (c *conf) setDefaultConfigs() {

	// set rpc default configs
	const rpcPrefix = "rpc"
	viper.SetDefault(rpcPrefix+".chain_rpc_protocol", 1)
	viper.SetDefault(rpcPrefix+".chain_host", "127.0.0.1")
	viper.SetDefault(rpcPrefix+".chain_rpc_port", 22351)
	viper.SetDefault(rpcPrefix+".sandbox_rpc_port", 32351)
	viper.SetDefault(rpcPrefix+".max_send_msg_size", 100)
	viper.SetDefault(rpcPrefix+".max_recv_msg_size", 100)
	viper.SetDefault(rpcPrefix+".server_min_interval", 60*time.Second)
	viper.SetDefault(rpcPrefix+".connection_timeout", 5*time.Second)
	viper.SetDefault(rpcPrefix+".server_keep_alive_time", 60*time.Second)
	viper.SetDefault(rpcPrefix+".server_keep_alive_timeout", 20*time.Second)

	// set process default configs
	const processPrefix = "process"
	viper.SetDefault(processPrefix+".max_original_process_num", 20)
	viper.SetDefault(processPrefix+".exec_tx_timeout", 8*time.Second)
	viper.SetDefault(processPrefix+".waiting_tx_time", 200*time.Millisecond)
	viper.SetDefault(processPrefix+".release_rate", 0)
	viper.SetDefault(processPrefix+".release_period", 10*time.Minute)

	// set log default configs
	const logPrefix = "log"
	viper.SetDefault(logPrefix+".contract_engine.level", "info")
	viper.SetDefault(logPrefix+".contract_engine.console", false)
	viper.SetDefault(logPrefix+".sandbox.level", "info")
	viper.SetDefault(logPrefix+".sandbox.console", false)

	// set pprof default configs
	const pprofPrefix = "pprof"
	viper.SetDefault(pprofPrefix+".contract_engine.port", 21215)
	viper.SetDefault(pprofPrefix+".sandbox.port", 21522)

	// set contract default configs
	const contractPrefix = "contract"
	viper.SetDefault(contractPrefix+".max_file_size", 20480)

	// set slow default configs
	const slowPrefix = "slow"
	viper.SetDefault(slowPrefix+".step_time", 3*time.Second)
	viper.SetDefault(slowPrefix+".tx_time", 6*time.Second)
	viper.SetDefault(slowPrefix+".disable", false)
}

func (c *conf) setEnv() error {

	var errs []string

	if chainRPCProtocol, ok := os.LookupEnv("CHAIN_RPC_PROTOCOL"); ok {
		p, err := strconv.Atoi(chainRPCProtocol)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to Atoi chainRPCProtocol: %v", err))
		} else {
			c.RPC.ChainRPCProtocol = ChainRPCProtocolType(p)
		}
	}

	if chainHost, ok := os.LookupEnv("CHAIN_HOST"); ok {
		var err error
		if len(chainHost) == 0 {
			errs = append(errs, fmt.Sprintf("chainHost is empty: %v", err))
		} else {
			c.RPC.ChainHost = chainHost
		}
	}

	if chainRPCPort, ok := os.LookupEnv("CHAIN_RPC_PORT"); ok {
		var err error
		if c.RPC.ChainRPCPort, err = strconv.Atoi(chainRPCPort); err != nil {
			errs = append(errs, fmt.Sprintf("failed to Atoi chainRPCPort: %v", err))
		}
	}

	if sandboxRPCPort, ok := os.LookupEnv("SANDBOX_RPC_PORT"); ok {
		var err error
		if c.RPC.SandboxRPCPort, err = strconv.Atoi(sandboxRPCPort); err != nil {
			errs = append(errs, fmt.Sprintf("failed to Atoi sandboxRPCPort: %v", err))
		}
	}

	if maxSendMsgSize, ok := os.LookupEnv("MAX_SEND_MSG_SIZE"); ok {
		var err error
		if c.RPC.MaxSendMsgSize, err = strconv.Atoi(maxSendMsgSize); err != nil {
			errs = append(errs, fmt.Sprintf("failed to Atoi maxSendMsgSize: %v", err))
		}
	}

	if maxRecvMsgSize, ok := os.LookupEnv("MAX_RECV_MSG_SIZE"); ok {
		var err error
		if c.RPC.MaxRecvMsgSize, err = strconv.Atoi(maxRecvMsgSize); err != nil {
			errs = append(errs, fmt.Sprintf("failed to Atoi maxRecvMsgSize: %v", err))
		}
	}

	if connectionTimeout, ok := os.LookupEnv("MAX_CONN_TIMEOUT"); ok {
		timeout, err := strconv.ParseInt(connectionTimeout, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to ParseInt connectionTimeout: %v", err))
		} else {
			c.RPC.ConnectionTimeout = time.Duration(timeout) * time.Second
		}
	}

	if processNum, ok := os.LookupEnv("MAX_ORIGINAL_PROCESS_NUM"); ok {
		var err error
		if c.Process.MaxOriginalProcessNum, err = strconv.Atoi(processNum); err != nil {
			errs = append(errs, fmt.Sprintf("failed to ParseInt processNum: %v", err))
		}
	}

	if contractEngineLogLevel, ok := os.LookupEnv("DOCKERVM_CONTRACT_ENGINE_LOG_LEVEL"); ok {
		c.Log.ContractEngineLog.Level = contractEngineLogLevel
	}

	if sandboxLogLevel, ok := os.LookupEnv("DOCKERVM_SANDBOX_LOG_LEVEL"); ok {
		c.Log.SandboxLog.Level = sandboxLogLevel
	}

	if logInConsole, ok := os.LookupEnv("DOCKERVM_LOG_IN_CONSOLE"); ok {
		needLog, _ := strconv.ParseBool(logInConsole)
		if !needLog {
			c.Log.SandboxLog.Console = false
			c.Log.ContractEngineLog.Console = false
		}
	}

	if busyTimout, ok := os.LookupEnv("PROCESS_TIMEOUT"); ok {
		timeout, err := strconv.ParseInt(busyTimout, 10, 64)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to ParseInt busyTimout: %v", err))
		}
		if timeout != 0 {
			c.Process.ExecTxTimeout = time.Duration(timeout) * time.Second
		}
	}
	if slowDisable, ok := os.LookupEnv("SLOW_DISABLE"); ok {
		isDisable, err := strconv.ParseBool(slowDisable)
		if err != nil {
			errs = append(errs, fmt.Sprintf("failed to ParseBool slowDisable: %v", err))
		}
		if isDisable {
			c.Slow.Disable = true
		} else {
			if slowStepTime, ok := os.LookupEnv("SLOW_STEP_TIME"); ok {
				timeout, err := strconv.ParseInt(slowStepTime, 10, 64)
				if err != nil {
					errs = append(errs, fmt.Sprintf("failed to ParseInt slowStepTime: %v", err))
				}
				if timeout != 0 {
					c.Slow.StepTime = time.Duration(timeout) * time.Second
				}
			}

			if slowTxTime, ok := os.LookupEnv("SLOW_TX_TIME"); ok {
				timeout, err := strconv.ParseInt(slowTxTime, 10, 64)
				if err != nil {
					errs = append(errs, fmt.Sprintf("failed to ParseInt slowTxTime: %v", err))
				}
				if timeout != 0 {
					c.Slow.TxTime = time.Duration(timeout) * time.Second
				}
			}
		}
	}
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf(strings.Join(errs, ","))

}

func (c *conf) restrainConfig() {
	if c.Process.ReleaseRate < 0 {
		c.Process.ReleaseRate = 0
	} else if c.Process.ReleaseRate > 100 {
		c.Process.ReleaseRate = 100
	}
}

// GetReleaseRate returns release rate
func (c *conf) GetReleaseRate() float64 {
	return float64(c.Process.ReleaseRate) / 100.0
}

// GetMaxUserNum returns max user num
func (c *conf) GetMaxUserNum() int {
	return c.Process.MaxOriginalProcessNum * (protocol.CallContractDepth + 1)
}

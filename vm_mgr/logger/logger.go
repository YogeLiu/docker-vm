/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"
	"path/filepath"
	"time"

	rotatelogs "chainmaker.org/chainmaker/common/v2/log/file-rotatelogs"

	"chainmaker.org/chainmaker/vm-docker-go/v2/vm_mgr/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DockerLogDir = "/log"          // mount directory for log
	LogFileName  = "docker-go.log" // log file name

)
const (
	// docker module for logging
	MODULE_MANAGER             = "[Docker Manager]"
	MODULE_CONFIG              = "[Docker Config]"
	MODULE_REQUEST_SCHEDULER   = "[Docker Request Scheduler]"
	MODULE_REQUEST_GROUP       = "[Docker Request Group]"
	MODULE_USERCONTROLLER      = "[Docker User Controller]"
	MODULE_PROCESS_MANAGER     = "[Docker Process Manager]"
	MODULE_PROCESS             = "[Docker Process]"
	MODULE_SANDBOX_RPC_HANDLER = "[Docker Sandbox RPC Handler]"
	MODULE_SANDBOX_RPC_SERVER  = "[Docker Sandbox RPC Server]"
	MODULE_CHAIN_RPC_SERVER    = "[Docker Chain RPC Server]"
	MODULE_CHAIN_RPC_SERVICE   = "[Docker Chain RPC Service]"
	MODULE_SECURITY_ENV        = "[Docker Security Env]"
	MODULE_CONTRACT_MANAGER    = "[Docker Contract Manager]"
	MODULE_CONTRACT            = "[Docker Contract]"
)

const (
	maxAge       = 365
	rotationTime = 1
)

var (
	logLevelFromConfig         string
	logPathFromConfig          string
	displayInConsoleFromConfig bool
	showLineFromConfig         bool
)

func InitialConfig(logPath string) {

	// init show line config
	showLineFromConfig = true

	// init display in console config
	displayInConsoleFromConfig = config.DockerVMConfig.Log.ContractEngineLog.Console

	logName := LogFileName
	logPathFromConfig = filepath.Join(logPath, logName)

	logLevelFromConfig = config.DockerVMConfig.Log.ContractEngineLog.Level
}

func NewDockerLogger(name string) *zap.SugaredLogger {

	InitialConfig(DockerLogDir)

	var encoder zapcore.Encoder
	if name == MODULE_CONTRACT {
		encoder = getContractEncoder()
	} else {
		encoder = getEncoder()
	}

	writeSyncer := getLogWriter()

	// default log level is info
	logLevel := zapcore.InfoLevel
	if logLevelFromConfig == "DEBUG" {
		logLevel = zapcore.DebugLevel
	}

	core := zapcore.NewCore(
		encoder,
		writeSyncer,
		logLevel,
	)

	logger := zap.New(core).Named(name)
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	if showLineFromConfig {
		logger = logger.WithOptions(zap.AddCaller())
	}

	sugarLogger := logger.Sugar()

	return sugarLogger
}

func getEncoder() zapcore.Encoder {

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "line",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    CustomLevelEncoder,
		EncodeTime:     CustomTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getContractEncoder() zapcore.Encoder {
	encoderConfig := zapcore.EncoderConfig{
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    CustomLevelEncoder,
		EncodeTime:     CustomTimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
		EncodeName:     zapcore.FullNameEncoder,
	}

	return zapcore.NewConsoleEncoder(encoderConfig)
}

func getLogWriter() zapcore.WriteSyncer {

	hook, _ := rotatelogs.New(
		logPathFromConfig+".%Y%m%d%H",
		rotatelogs.WithRotationTime(time.Hour*time.Duration(rotationTime)),
		rotatelogs.WithLinkName(logPathFromConfig),
		rotatelogs.WithMaxAge(time.Hour*24*time.Duration(maxAge)),
	)

	var syncer zapcore.WriteSyncer
	if displayInConsoleFromConfig {
		syncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(hook))
	} else {
		syncer = zapcore.AddSync(hook)
	}

	return syncer
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

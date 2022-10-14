/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"
	"path/filepath"
	"strings"
	"time"

	rotatelogs "chainmaker.org/chainmaker/common/v2/log/file-rotatelogs"

	"chainmaker.org/chainmaker/vm-engine/v2/vm_mgr/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	DockerLogDir = "/log"   // mount directory for log
	LogFileName  = "go.log" // log file name
)

const (
	// docker module for logging
	MODULE_MANAGER             = "[Manager]"
	MODULE_CONFIG              = "[Config]"
	MODULE_REQUEST_SCHEDULER   = "[Request Scheduler]"
	MODULE_REQUEST_GROUP       = "[Request Group]"
	MODULE_USERCONTROLLER      = "[User Controller]"
	MODULE_PROCESS_MANAGER     = "[Process Manager]"
	MODULE_PROCESS             = "[Process]"
	MODULE_SANDBOX_RPC_SERVER  = "[Sandbox RPC Server]"
	MODULE_SANDBOX_RPC_SERVICE = "[Sandbox RPC Service]"
	MODULE_CHAIN_RPC_SERVER    = "[Chain RPC Server]"
	MODULE_CHAIN_RPC_SERVICE   = "[Chain RPC Service]"
	MODULE_SECURITY_ENV        = "[Security Env]"
	MODULE_CONTRACT_MANAGER    = "[Contract Manager]"
	MODULE_CONTRACT            = "[Contract]"
	MODULE_TEST                = "[Test]"
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
	logLevel                   zapcore.Level
	logLevelInitialized        bool
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
	return newDockerLogger(name, DockerLogDir)
}

func NewTestDockerLogger() *zap.SugaredLogger {
	basePath, _ := os.Getwd()
	return newDockerLogger(MODULE_TEST, basePath+"/")
}

func newDockerLogger(name, path string) *zap.SugaredLogger {

	InitialConfig(path)

	var encoder zapcore.Encoder
	if name == MODULE_CONTRACT {
		encoder = getContractEncoder()
	} else {
		encoder = getEncoder()
	}

	writeSyncer := getLogWriter()

	// default log level is info
	logLevel := GetLogLevel()

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

func GenerateProcessLoggerName(name string) string {
	var sb strings.Builder
	sb.WriteString("[Process ")
	sb.WriteString(name)
	sb.WriteString("]")
	return sb.String()
}

func GenerateRequestGroupLoggerName(name string) string {
	var sb strings.Builder
	sb.WriteString("[Request Group ")
	sb.WriteString(name)
	sb.WriteString("]")
	return sb.String()
}

func GetLogLevel() zapcore.Level {
	if !logLevelInitialized {
		if err := logLevel.UnmarshalText([]byte(logLevelFromConfig)); err != nil {
			logLevel = zapcore.InfoLevel
		}
		if logLevel > zapcore.ErrorLevel {
			logLevel = zapcore.ErrorLevel
		}
		logLevelInitialized = true
	}
	return logLevel
}

func DebugDynamic(l *zap.SugaredLogger, getStr func() string) {
	if logLevel == zapcore.DebugLevel {
		str := getStr()
		l.Debug(str)
	}
}

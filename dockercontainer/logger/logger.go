/*
Copyright (C) BABEC. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package logger

import (
	"os"
	"path/filepath"
	"strconv"
	"time"

	"chainmaker.org/chainmaker/vm-docker-go/dockercontainer/config"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

const (
	// docker module for logging

	MODULE_MANAGER          = "[Docker MANAGER]"
	MODULE_SCHEDULER        = "[Docker Scheduler]"
	MODULE_USERCONTROLLER   = "[Docker User Controller]"
	MODULE_PROCESS_POOL     = "[Docker Process Pool]"
	MODULE_PROCESS          = "[Docker Process]"
	MODULE_DMS_HANDLER      = "[Docker DMS Handler]"
	MODULE_DMS_SERVER       = "[Docker DMS Server]"
	MODULE_CDM_SERVER       = "[Docker CDM Server]"
	MODULE_CDM_API          = "[Docker CDM Api]"
	MODULE_SECURITY_ENV     = "[Docker Security Env]"
	MODULE_CONTRACT_MANAGER = "[Docker Contract Manager]"
	MODULE_CONTRACT         = "[Docker Contract]"
)

var (
	logLevelFromConfig         string
	logPathFromConfig          string
	displayInConsoleFromConfig bool
	showLineFromConfig         bool
)

func InitialConfig() {

	// init show line config
	showLineFromConfig = true

	// init display in console config
	b2, err := strconv.ParseBool(os.Getenv("Log_In_Console"))
	if err != nil {
		displayInConsoleFromConfig = false
	}
	if b2 {
		displayInConsoleFromConfig = true
	} else {
		displayInConsoleFromConfig = false
	}

	logName := config.LogFileName
	logPathFromConfig = filepath.Join(config.DockerLogDir, logName)

	logLevelFromConfig = os.Getenv("Log_Level")
	if logLevelFromConfig == "" {
		logLevelFromConfig = "INFO"
	}
	config.SandBoxLogLevel = logLevelFromConfig
}

func NewDockerLogger(name string) *zap.SugaredLogger {

	InitialConfig()

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

	hook := &lumberjack.Logger{
		Filename:   logPathFromConfig, //日志文件存放目录
		MaxSize:    100,               //文件大小限制,单位MB
		MaxBackups: 5,                 //最大保留日志文件数量
		MaxAge:     30,                //日志文件保留天数
		Compress:   false,             //是否压缩处理
	}

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

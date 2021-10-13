/*
Copyright (C) BABEC. All rights reserved.
Copyright (C) THL A29 Limited, a Tencent company. All rights reserved.

SPDX-License-Identifier: Apache-2.0
*/

package log

import (
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"os"
	"strings"
	"time"

	rotatelogs "chainmaker.org/chainmaker/common/v2/log/file-rotatelogs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// 日志级别，int类型，内部接口使用常量
type LOG_LEVEL int

const (
	LEVEL_DEBUG LOG_LEVEL = iota
	LEVEL_INFO
	LEVEL_WARN
	LEVEL_ERROR
)

// 日志级别，配置文件定义的常量
const (
	DEBUG = "DEBUG"
	INFO  = "INFO"
	WARN  = "WARN"
	ERROR = "ERROR"
)

func GetLogLevel(lvl string) LOG_LEVEL {
	switch lvl {
	case ERROR:
		return LEVEL_ERROR
	case WARN:
		return LEVEL_WARN
	case INFO:
		return LEVEL_INFO
	case DEBUG:
		return LEVEL_DEBUG
	}

	return LEVEL_INFO
}

// 日志切割默认配置
const (
	DEFAULT_MAX_AGE       = 365 // 日志最长保存时间，单位：天
	DEFAULT_ROTATION_TIME = 6   // 日志滚动间隔，单位：小时
	DEFAULT_ROTATION_SIZE = 100 // 默认的日志滚动大小，单位：MB
)

//日志滚动单位
const (
	ROTATION_SIZE_MB = 1024 * 1024
)

type color int

const (
	ColorBlack color = iota + 30
	ColorRed
	ColorGreen
	ColorYellow
	ColorBlue
	ColorMagenta
	ColorCyan
	ColorWhite
)

var colorList = [...]color{ColorRed, ColorGreen, ColorYellow, ColorBlue, ColorMagenta}

var hookMap = make(map[string]struct{})

type LogConfig struct {
	Module       string    // module: module name
	ChainId      string    // chainId: chain id
	LogPath      string    // logPath: log file save path
	LogLevel     LOG_LEVEL // logLevel: log level
	MaxAge       int       // maxAge: the maximum number of days to retain old log files
	RotationTime int       // RotationTime: rotation time
	RotationSize int64     // RotationSize: rotation size Mb
	JsonFormat   bool      // jsonFormat: log file use json format
	ShowLine     bool      // showLine: show filename and line number
	LogInConsole bool      // logInConsole: show logs in console at the same time
	ShowColor    bool      // if true, show color log
	IsBrief      bool      // if true, only show log, won't print log level、caller func and line
}

func InitSugarLogger(logConfig *LogConfig) (*zap.SugaredLogger, zap.AtomicLevel) {
	var level zapcore.Level
	switch logConfig.LogLevel {
	case LEVEL_DEBUG:
		level = zap.DebugLevel
	case LEVEL_INFO:
		level = zap.InfoLevel
	case LEVEL_WARN:
		level = zap.WarnLevel
	case LEVEL_ERROR:
		level = zap.ErrorLevel
	default:
		level = zap.InfoLevel
	}

	aLevel := zap.NewAtomicLevel()
	aLevel.SetLevel(level)

	sugaredLogger := newLogger(logConfig, aLevel).Sugar()

	return sugaredLogger, aLevel
}

func newLogger(logConfig *LogConfig, level zap.AtomicLevel) *zap.Logger {
	var (
		hook io.Writer
		ok   bool
		err  error
	)

	_, ok = hookMap[logConfig.LogPath]
	if !ok {
		hook, err = getHook(logConfig.LogPath, logConfig.MaxAge, logConfig.RotationTime, logConfig.RotationSize)
		if err != nil {
			log.Fatalf("new logger get hook failed, %s", err)
		}
		hookMap[logConfig.LogPath] = struct{}{}
	} else {
		hook, err = getHook(logConfig.LogPath, logConfig.MaxAge, 0, logConfig.RotationSize)
		if err != nil {
			log.Fatalf("new logger get hook failed, %s", err)
		}
	}

	var syncer zapcore.WriteSyncer
	if logConfig.LogInConsole {
		syncer = zapcore.NewMultiWriteSyncer(zapcore.AddSync(os.Stdout), zapcore.AddSync(hook))
	} else {
		syncer = zapcore.AddSync(hook)
	}

	var encoderConfig zapcore.EncoderConfig
	if logConfig.IsBrief {
		encoderConfig = zapcore.EncoderConfig{
			TimeKey:    "time",
			MessageKey: "msg",
			EncodeTime: CustomTimeEncoder,
			LineEnding: zapcore.DefaultLineEnding,
		}
	} else {
		encoderConfig = zapcore.EncoderConfig{
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
	}

	var encoder zapcore.Encoder
	if logConfig.JsonFormat {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	}

	core := zapcore.NewCore(
		encoder,
		syncer,
		level,
	)

	chainId := fmt.Sprintf("@%s", logConfig.ChainId)
	if logConfig.ShowColor {
		chainId = getColorChainId(chainId)
	}

	var name string
	if logConfig.ChainId != "" {
		name = fmt.Sprintf("%s %s", logConfig.Module, chainId)
	} else {
		name = logConfig.Module
	}

	logger := zap.New(core).Named(name)
	defer func(logger *zap.Logger) {
		_ = logger.Sync()
	}(logger)

	if logConfig.ShowLine {
		logger = logger.WithOptions(zap.AddCaller())
	}
	logger = logger.WithOptions(zap.AddStacktrace(zapcore.ErrorLevel), zap.AddCallerSkip(1))
	return logger
}

func getHook(filename string, maxAge, rotationTime int, rotationSize int64) (io.Writer, error) {

	hook, err := rotatelogs.New(
		filename+".%Y%m%d%H",
		rotatelogs.WithRotationTime(time.Hour*time.Duration(rotationTime)),
		//filename+".%Y%m%d%H%M",
		rotatelogs.WithRotationSize(rotationSize*ROTATION_SIZE_MB),
		rotatelogs.WithLinkName(filename),
		rotatelogs.WithMaxAge(time.Hour*24*time.Duration(maxAge)),
	)

	if err != nil {
		return nil, err
	}

	return hook, nil
}

// nolint: deadcode, unused
func recognizeLogLevel(l string) LOG_LEVEL {
	logLevel := strings.ToUpper(l)
	var level LOG_LEVEL
	switch logLevel {
	case DEBUG:
		level = LEVEL_DEBUG
	case INFO:
		level = LEVEL_INFO
	case WARN:
		level = LEVEL_WARN
	case ERROR:
		level = LEVEL_ERROR
	default:
		level = LEVEL_INFO
	}
	return level
}

func CustomLevelEncoder(level zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString("[" + level.CapitalString() + "]")
}

func CustomTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}

// nolint: deadcode, unused
func showColor(color color, msg string) string {
	return fmt.Sprintf("\033[%dm%s\033[0m", int(color), msg)
}

func showColorBold(color color, msg string) string {
	return fmt.Sprintf("\033[%d;1m%s\033[0m", int(color), msg)
}

func getColorChainId(chainId string) string {
	c := crc32.ChecksumIEEE([]byte(chainId))
	color := colorList[int(c)%len(colorList)]
	return showColorBold(color, chainId)
}

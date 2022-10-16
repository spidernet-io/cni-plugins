package logging

import (
	"github.com/spidernet-io/cni-plugins/pkg/constant"
	"github.com/spidernet-io/cni-plugins/pkg/types"
	"github.com/spidernet-io/spiderpool/pkg/logutils"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"k8s.io/utils/pointer"
)

var LoggerFile *zap.Logger

const (
	DefaultLogFileMaxSize    = 100
	DefaultLogFileMaxAge     = 30
	
	DefaultLogFileMaxBackups = 10
)

const (
	// DebugLevel logs are typically voluminous, and are usually disabled in
	// production.
	DebugLevel = zapcore.DebugLevel
	// InfoLevel is the default logging priority.
	InfoLevel = zapcore.InfoLevel
	// WarnLevel logs are more important than Info, but don't need individual
	// human review.
	WarnLevel = zapcore.WarnLevel
	// ErrorLevel logs are high-priority. If an application is running smoothly,
	// it shouldn't generate any error-level logs.
	ErrorLevel = zapcore.ErrorLevel
	// DPanicLevel logs are particularly important errors. In development the
	// logger panics after writing the message.
	DPanicLevel = zapcore.DPanicLevel
	// PanicLevel logs a message, then panics.
	PanicLevel = zapcore.PanicLevel
	// FatalLevel logs a message, then calls os.Exit(1).
	FatalLevel = zapcore.FatalLevel
)

// SetLogOptions set the LoggingOptions of NetConf
func SetLogOptions(options *types.LogOptions) error {
	v := logutils.ConvertLogLevel(options.LogLevel)
	if v == nil {
		// wrong logging level, give default level
		logLevel := zap.InfoLevel
		v = &logLevel
	}

	var err error
	LoggerFile, err = logutils.InitFileLogger(*v, options.LogFilePath, *options.LogFileMaxSize, *options.LogFileMaxAge, *options.LogFileMaxCount)
	if err != nil {
		return err
	}

	return nil
}

// InitLogOptions init log options from config file
func InitLogOptions(logOptions *types.LogOptions) *types.LogOptions {
	// validate logging config
	if logOptions != nil {
		if logOptions.LogLevel == "" {
			logOptions.LogLevel = constant.LogDebugLevelStr
		}
		if logOptions.LogFileMaxSize == nil {
			logOptions.LogFileMaxSize = pointer.Int(constant.LogDefaultMaxSize)
		}
		if logOptions.LogFileMaxAge == nil {
			logOptions.LogFileMaxAge = pointer.Int(constant.LogDefaultMaxAge)
		}
		if logOptions.LogFileMaxCount == nil {
			logOptions.LogFileMaxCount = pointer.Int(constant.LogDefaultMaxAge)
		}
	} else {
		logOptions = &types.LogOptions{
			LogLevel:        constant.LogDebugLevelStr,
			LogFileMaxSize:  pointer.Int(constant.LogDefaultMaxSize),
			LogFileMaxAge:   pointer.Int(constant.LogDefaultMaxAge),
			LogFileMaxCount: pointer.Int(constant.LogDefaultMaxAge),
		}
	}
	return logOptions
}

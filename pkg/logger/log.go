/*
 * Copyright 2022 Holoinsight Project Authors. Licensed under Apache-2.0.
 */

package logger

import (
	"github.com/traas-stack/holoinsight-agent/pkg/appconfig"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"os"
	"path/filepath"
)

type (
	alwaysLevel     struct{}
	loggerComposite struct {
		debug   *zap.Logger
		debugS  *zap.SugaredLogger
		info    *zap.Logger
		infoS   *zap.SugaredLogger
		warn    *zap.Logger
		warnS   *zap.SugaredLogger
		error   *zap.Logger
		errorS  *zap.SugaredLogger
		stat    *zap.Logger
		config  *zap.Logger
		configS *zap.SugaredLogger
		meta    *zap.Logger
		metaS   *zap.SugaredLogger
	}
)

var (
	zapLogger    *loggerComposite
	DebugEnabled = false
)

// init initializes default loggers (to console)
func init() {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		ConsoleSeparator: " ",
		LineEnding:       zapcore.DefaultLineEnding,
		EncodeLevel:      zapcore.LowercaseLevelEncoder,
		EncodeTime:       zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
	}

	newZapLogger2 := func() *zap.Logger {
		return zap.New(
			zapcore.NewTee(
				zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), alwaysLevel{}),
			),
		)
	}

	zapLogger = &loggerComposite{
		debug:  newZapLogger2(),
		info:   newZapLogger2(),
		warn:   newZapLogger2(),
		error:  newZapLogger2(),
		stat:   newZapLogger2(),
		config: newZapLogger2(),
		meta:   newZapLogger2(),
	}
	zapLogger.debugS = zapLogger.debug.Sugar()
	zapLogger.infoS = zapLogger.info.Sugar()
	zapLogger.warnS = zapLogger.warn.Sugar()
	zapLogger.errorS = zapLogger.error.Sugar()
	zapLogger.configS = zapLogger.config.Sugar()
	zapLogger.metaS = zapLogger.meta.Sugar()
}

func (a alwaysLevel) Enabled(level zapcore.Level) bool {
	return true
}

func SetupZapLogger() {
	if appconfig.IsDev() {
		return
	}
	setupZapLogger0(false)
	registerHttpHandler()
}

func setupZapLogger0(dev bool) {
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:          "time",
		LevelKey:         "level",
		NameKey:          "logger",
		CallerKey:        "caller",
		MessageKey:       "msg",
		StacktraceKey:    "stacktrace",
		ConsoleSeparator: " ",
		LineEnding:       zapcore.DefaultLineEnding,
		// EncodeLevel:      zapcore.LowercaseLevelEncoder, // 小写编码器
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05.000"),
		EncodeDuration: zapcore.SecondsDurationEncoder,
		// EncodeCaller:   zapcore.FullCallerEncoder, // 全路径编码器
	}

	newZapLogger2 := func(path string) *zap.Logger {
		logDir := "logs"
		w, err := NewRotateWriter(LogConfig{
			Filename:           filepath.Join(logDir, path),
			MaxSize:            1024 * 1024 * 1024,
			MaxBackupCount:     7,
			MaxBackupsSize:     8 * 1024 * 1024 * 1024,
			TimeLayout:         "2006-01-02",
			RemoveUnknownFiles: true,
			DeleteScanPatterns: []string{
				filepath.Join(logDir, filepath.Base(path)+"-*"+filepath.Ext(path)),
				filepath.Join(logDir, path) + ".*",
			},
		})
		if err != nil {
			panic(err)
		}
		if dev {
			bak := encoderConfig
			bak.EncodeLevel = zapcore.LowercaseLevelEncoder
			return zap.New(
				zapcore.NewTee(
					zapcore.NewCore(zapcore.NewConsoleEncoder(bak), zapcore.AddSync(os.Stdout), alwaysLevel{}),
					zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(w), alwaysLevel{}),
				),
			)
		} else {
			return zap.New(
				zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(w), alwaysLevel{}),
			)
		}
	}

	// 构建日志
	zapLogger = &loggerComposite{
		debug:  newZapLogger2("debug.log"),
		info:   newZapLogger2("info.log"),
		warn:   newZapLogger2("warn.log"),
		error:  newZapLogger2("error.log"),
		stat:   newZapLogger2("stat.log"),
		config: newZapLogger2("config.log"),
		meta:   newZapLogger2("meta.log"),
	}
	zapLogger.debugS = zapLogger.debug.Sugar()
	zapLogger.infoS = zapLogger.info.Sugar()
	zapLogger.warnS = zapLogger.warn.Sugar()
	zapLogger.errorS = zapLogger.error.Sugar()
	zapLogger.configS = zapLogger.config.Sugar()
	zapLogger.metaS = zapLogger.meta.Sugar()
}

func Debugz(msg string, fields ...zap.Field) {
	if DebugEnabled {
		zapLogger.debug.Info(msg, fields...)
	}
}
func Infoz(msg string, fields ...zap.Field) {
	zapLogger.info.Info(msg, fields...)
}
func Warnz(msg string, fields ...zap.Field) {
	zapLogger.warn.Info(msg, fields...)
}
func Errorz(msg string, fields ...zap.Field) {
	zapLogger.error.Info(msg, fields...)
}
func Configz(msg string, fields ...zap.Field) {
	zapLogger.config.Info(msg, fields...)
}

func Debugw(msg string, keyAndValues ...interface{}) {
	if DebugEnabled {
		zapLogger.debugS.Infow(msg, keyAndValues...)
	}
}
func Infow(msg string, keyAndValues ...interface{}) {
	zapLogger.infoS.Infow(msg, keyAndValues...)
}
func Warnw(msg string, keyAndValues ...interface{}) {
	zapLogger.warnS.Infow(msg, keyAndValues...)
}
func Errorw(msg string, keyAndValues ...interface{}) {
	zapLogger.errorS.Infow(msg, keyAndValues...)
}

func Debugf(msg string, args ...interface{}) {
	if DebugEnabled {
		zapLogger.debugS.Infof(msg, args...)
	}
}
func Infof(msg string, args ...interface{}) {
	zapLogger.infoS.Infof(msg, args...)
}
func Warnf(msg string, args ...interface{}) {
	zapLogger.warnS.Infof(msg, args...)
}
func Errorf(msg string, args ...interface{}) {
	zapLogger.errorS.Infof(msg, args...)
}
func Configf(msg string, args ...interface{}) {
	zapLogger.configS.Infof(msg, args...)
}

func Stat(msg string) {
	zapLogger.stat.Info(msg)
}

func Metaf(msg string, args ...interface{}) {
	zapLogger.metaS.Infof(msg, args...)
}

func Metaz(msg string, fields ...zap.Field) {
	zapLogger.meta.Info(msg, fields...)
}

func IsDebugEnabled() bool {
	return DebugEnabled
}

func TestMode() {
}

func init() {
	setupZapLogger0(true)
}

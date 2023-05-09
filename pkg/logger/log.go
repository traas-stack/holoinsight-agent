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
	"reflect"
	"strings"
	"unsafe"
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
	zapLogger    = &loggerComposite{}
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

	newStdoutLogger := func() *zap.Logger {
		return zap.New(
			zapcore.NewTee(
				zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), alwaysLevel{}),
			),
		)
	}

	zapLogger.buildLoggers(func(name string) *zap.Logger {
		return newStdoutLogger()
	})
}

// buildLoggers automatically build loggers using reflect
func (c *loggerComposite) buildLoggers(factory func(name string) *zap.Logger) {
	e := reflect.ValueOf(c).Elem()
	etype := e.Type()
	for i := 0; i < etype.NumField(); i++ {
		field := etype.Field(i)
		if !strings.HasSuffix(field.Name, "S") {
			zlogger := factory(field.Name)
			// c.xxx = zlogger
			*(*unsafe.Pointer)(e.Field(i).Addr().UnsafePointer()) = unsafe.Pointer(zlogger)
			s := e.FieldByName(field.Name + "S")
			if s.IsValid() {
				// c.xxxS = zlogger.Sugar()
				*(*unsafe.Pointer)(s.Addr().UnsafePointer()) = unsafe.Pointer(zlogger.Sugar())
			}
		}
	}
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

	newZapLogger := func(path string) *zap.Logger {
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

		return zap.New(
			zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(w), alwaysLevel{}),
		)
	}

	// Build loggers
	// set xxx = newZapLogger('xxx.log') using reflect
	// set xxxS = xxx.Sugar() using reflect
	// When you need to add a new logger, just add it as the field of loggerComposite
	zapLogger.buildLoggers(func(name string) *zap.Logger {
		return newZapLogger(name + ".log")
	})
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

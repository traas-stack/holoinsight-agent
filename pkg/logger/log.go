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
	LoggerComposite struct {
		Debug  *zap.Logger
		DebugS *zap.SugaredLogger
		Info   *zap.Logger
		InfoS  *zap.SugaredLogger
		Warn   *zap.Logger
		WarnS  *zap.SugaredLogger
		Error  *zap.Logger
		ErrorS *zap.SugaredLogger
		Stat   *zap.Logger
		Config *zap.Logger
		Meta   *zap.Logger
		MetaS  *zap.SugaredLogger
		Cri    *zap.Logger
	}
)

var (
	ZapLogger    = &LoggerComposite{}
	DebugEnabled = false
	writers      []*RotateWriter
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
		EncodeDuration:   zapcore.SecondsDurationEncoder,
	}

	newStdoutLogger := func() *zap.Logger {
		return zap.New(
			zapcore.NewTee(
				zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(os.Stdout), alwaysLevel{}),
			),
		)
	}

	ZapLogger.buildLoggers(func(name string) *zap.Logger {
		return newStdoutLogger()
	})
}

// buildLoggers automatically build loggers using reflect
func (c *LoggerComposite) buildLoggers(factory func(name string) *zap.Logger) {
	e := reflect.ValueOf(c).Elem()
	etype := e.Type()
	for i := 0; i < etype.NumField(); i++ {
		field := etype.Field(i)
		if !strings.HasSuffix(field.Name, "S") {
			zlogger := factory(strings.ToLower(field.Name))
			// c.xxx = zlogger
			*(*unsafe.Pointer)(e.Field(i).Addr().UnsafePointer()) = unsafe.Pointer(zlogger)
			if s := e.FieldByName(strings.ToLower(field.Name[:1]) + field.Name[1:] + "S"); s.IsValid() {
				// c.xxxS = zlogger.Sugar()
				*(*unsafe.Pointer)(s.Addr().UnsafePointer()) = unsafe.Pointer(zlogger.Sugar())
			}
			if s := e.FieldByName(strings.ToUpper(field.Name[:1]) + field.Name[1:] + "S"); s.IsValid() {
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
	setupZapLogger0()
	registerHttpHandler()
}

func DisableRotates() {
	for _, writer := range writers {
		writer.disableRotate()
	}
}

func setupZapLogger0() {
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
		writers = append(writers, w)

		return zap.New(
			zapcore.NewCore(zapcore.NewConsoleEncoder(encoderConfig), zapcore.AddSync(w), alwaysLevel{}),
		)
	}

	// Build loggers
	// set xxx = newZapLogger('xxx.log') using reflect
	// set xxxS = xxx.Sugar() using reflect
	// When you need to add a new logger, just add it as the field of LoggerComposite
	ZapLogger.buildLoggers(func(name string) *zap.Logger {
		return newZapLogger(name + ".log")
	})
}

func Debugz(msg string, fields ...zap.Field) {
	if DebugEnabled {
		ZapLogger.Debug.Info(msg, fields...)
	}
}
func Infoz(msg string, fields ...zap.Field) {
	ZapLogger.Info.Info(msg, fields...)
}

func Infozo(option zap.Option, msg string, fields ...zap.Field) {
	ZapLogger.Info.WithOptions(option).Info(msg, fields...)
}

func Warnz(msg string, fields ...zap.Field) {
	ZapLogger.Warn.Info(msg, fields...)
}
func Errorz(msg string, fields ...zap.Field) {
	ZapLogger.Error.Info(msg, fields...)
}
func Errorzo(option zap.Option, msg string, fields ...zap.Field) {
	ZapLogger.Error.WithOptions(option).Info(msg, fields...)
}

func Configz(msg string, fields ...zap.Field) {
	ZapLogger.Config.Info(msg, fields...)
}

func Debugf(msg string, args ...interface{}) {
	if DebugEnabled {
		ZapLogger.DebugS.Infof(msg, args...)
	}
}
func Infof(msg string, args ...interface{}) {
	ZapLogger.InfoS.Infof(msg, args...)
}
func Warnf(msg string, args ...interface{}) {
	ZapLogger.WarnS.Infof(msg, args...)
}
func Errorf(msg string, args ...interface{}) {
	ZapLogger.ErrorS.Infof(msg, args...)
}
func Stat(msg string) {
	ZapLogger.Stat.Info(msg)
}

func Metaz(msg string, fields ...zap.Field) {
	ZapLogger.Meta.Info(msg, fields...)
}

func IsDebugEnabled() bool {
	return DebugEnabled
}

// Criz prints logs to Cri.log
func Criz(msg string, fields ...zap.Field) {
	ZapLogger.Cri.Info(msg, fields...)
}

package log

import (
	"context"
	"fmt"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"google.golang.org/grpc/metadata"
	"time"
)

type Config struct {
	Level            string        `json:"level" yaml:"level"`
	Encoding         string        `json:"encoding" yaml:"encoding"`
	EncoderConfig    EncoderConfig `json:"encoderConfig" yaml:"encoderConfig"`
	OutputPaths      []string      `json:"outputPaths" yaml:"outputPaths"`
	ErrorOutputPaths []string      `json:"errorOutputPaths" yaml:"errorOutputPaths"`
	// InitialFields is a collection of fields to add to the root logger.
	InitialFields map[string]interface{} `json:"initialFields" yaml:"initialFields"`
}

type EncoderConfig struct {
	MessageKey    string `json:"messageKey" yaml:"messageKey"`
	LevelKey      string `json:"levelKey" yaml:"levelKey"`
	TimeKey       string `json:"timeKey" yaml:"timeKey"`
	NameKey       string `json:"nameKey" yaml:"nameKey"`
	CallerKey     string `json:"callerKey" yaml:"callerKey"`
	StacktraceKey string `json:"stacktraceKey" yaml:"stacktraceKey"`
	LineEnding    string `json:"lineEnding" yaml:"lineEnding"`
}

// LogLevel specifies the severity of a given log message
type LogLevel int

// Log levels
const (
	LogLevelDebug LogLevel = iota
	LogLevelInfo
	LogLevelWarning
	LogLevelError
)

//type Logger interface {
//	//// With creates a child logger and adds structured context to it. Fields added
//	//// to the child don't affect the parent, and vice versa.
//	With(fields ...Field) Logger
//
//	// Debug logs a message at DebugLevel. The message includes any fields passed
//	// at the log site, as well as any fields accumulated on the logger.
//	Debug(msg string, fields ...Field)
//
//	// DPanic logs a message at DPanicLevel. The message includes any fields
//	// passed at the log site, as well as any fields accumulated on the logger.
//	//
//	// If the logger is in development mode, it then panics (DPanic means
//	// "development panic"). This is useful for catching errors that are
//	// recoverable, but shouldn't ever happen.
//	DPanic(msg string, fields ...Field)
//
//	// Error logs a message at ErrorLevel. The message includes any fields passed
//	// at the log site, as well as any fields accumulated on the logger.
//	Error(msg string, fields ...Field)
//
//	// Fatal logs a message at FatalLevel. The message includes any fields passed
//	// at the log site, as well as any fields accumulated on the logger.
//	//
//	// The logger then calls os.Exit(1), even if logging at FatalLevel is
//	// disabled.
//	Fatal(msg string, fields ...Field)
//
//	// Info logs a message at InfoLevel. The message includes any fields passed
//	// at the log site, as well as any fields accumulated on the logger.
//	Info(msg string, fields ...Field)
//
//	// Warn logs a message at WarnLevel. The message includes any fields passed
//	// at the log site, as well as any fields accumulated on the logger.
//	Warn(msg string, fields ...Field)
//
//	// Sync calls the underlying Core's Sync method, flushing any buffered log
//	// entries. Applications should take care to call Sync before exiting.
//	Sync() error
//}

type Logger struct {
	*zap.Logger
}

func NewZap(logCfg *Config) (*Logger, error) {
	ll := zap.NewAtomicLevel()
	err := ll.UnmarshalText([]byte(logCfg.Level))
	if err != nil {
		return nil, err
	}
	isDebugMode := ll.Enabled(zap.InfoLevel)

	if logCfg.EncoderConfig.TimeKey == "" {
		logCfg.EncoderConfig.TimeKey = "T"
	}
	if logCfg.EncoderConfig.LevelKey == "" {
		logCfg.EncoderConfig.LevelKey = "L"
	}
	if logCfg.EncoderConfig.NameKey == "" {
		logCfg.EncoderConfig.NameKey = "N"
	}
	if logCfg.EncoderConfig.CallerKey == "" {
		logCfg.EncoderConfig.CallerKey = "C"
	}
	if logCfg.EncoderConfig.MessageKey == "" {
		logCfg.EncoderConfig.MessageKey = "M"
	}
	if logCfg.EncoderConfig.StacktraceKey == "" {
		logCfg.EncoderConfig.StacktraceKey = "S"
	}

	if len(logCfg.OutputPaths) == 0 {
		logCfg.OutputPaths = []string{"stderr"}
	}
	if len(logCfg.ErrorOutputPaths) == 0 {
		logCfg.ErrorOutputPaths = []string{"stderr"}
	}

	zapCfg := &zap.Config{
		Level:       ll,
		Development: isDebugMode,
		Sampling: &zap.SamplingConfig{
			Initial:    100,
			Thereafter: 100,
		},
		Encoding: logCfg.Encoding,
		EncoderConfig: zapcore.EncoderConfig{
			TimeKey:       logCfg.EncoderConfig.TimeKey,
			LevelKey:      logCfg.EncoderConfig.LevelKey,
			NameKey:       logCfg.EncoderConfig.NameKey,
			CallerKey:     logCfg.EncoderConfig.CallerKey,
			MessageKey:    logCfg.EncoderConfig.MessageKey,
			StacktraceKey: logCfg.EncoderConfig.StacktraceKey,
			LineEnding:    zapcore.DefaultLineEnding,
		},
		OutputPaths:      logCfg.OutputPaths,
		ErrorOutputPaths: logCfg.ErrorOutputPaths,
	}

	if isDebugMode {
		zapCfg.Development = true
		zapCfg.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = CSTTimeEncoder
		zapCfg.EncoderConfig.EncodeDuration = zapcore.StringDurationEncoder
		zapCfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	} else {
		zapCfg.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		zapCfg.EncoderConfig.EncodeTime = zapcore.EpochTimeEncoder
		zapCfg.EncoderConfig.EncodeDuration = zapcore.SecondsDurationEncoder
		zapCfg.EncoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	}

	l, err := zapCfg.Build(zap.AddStacktrace(zap.ErrorLevel))

	return &Logger{l}, err
}

func CSTTimeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.999999999"))
}

func (l *Logger) ResetLogger(logger *Logger) {
	l.Logger = logger.Logger
}

func (l *Logger) Write(p []byte) (n int, err error) {
	l.Debug(fmt.Sprintf("HTTP: %s", string(p[:])))
	return len(p), nil
}

func (l *Logger) Trace(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return l.Logger
	}

	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return l.Logger
	}

	traceId, ok := md["trace_id"]
	if !ok {
		return l.Logger
	}

	return l.With(zap.String("trace_id", traceId[0]))
}

func (l *Logger) Output(calldepth int, s string)  {
	switch s[:3] {
	case "INF":
		l.Info("output info",zap.String("msg",s))
	case "WRN":
		l.Warn("output warning", zap.String("msg", s))
	case "ERR":
		l.Error("output error", zap.String("msg", s))
	case "DBG":
		l.Debug("output debug", zap.String("msg", s))
	}
}

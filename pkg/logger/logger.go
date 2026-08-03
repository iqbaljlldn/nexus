package logger

import (
	"os"

	"github.com/iqbaljlldn/nexus/pkg/config"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Level adalah global atomic level yang bisa diekspos ke HTTP endpoint
// (misal: zap.ServeHTTP) untuk mengubah level log secara live.
var Level = zap.NewAtomicLevel()

func New(cfg config.Config) (*zap.Logger, error) {
	Level.SetLevel(parseLevel(cfg.Logger.Level))

	encoderConfig := zap.NewProductionEncoderConfig()

	encoderConfig.TimeKey = "timestamp"
	encoderConfig.LevelKey = "level"
	encoderConfig.NameKey = "logger"
	encoderConfig.MessageKey = "message"
	encoderConfig.CallerKey = "caller"
	encoderConfig.StacktraceKey = "stacktrace"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
	encoderConfig.EncodeCaller = zapcore.ShortCallerEncoder
	encoderConfig.EncodeDuration = zapcore.MillisDurationEncoder

	var encoder zapcore.Encoder
	if cfg.Env == "development" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	stdout := zapcore.AddSync(os.Stdout)

	file := zapcore.AddSync(&lumberjack.Logger{
		Filename:   cfg.Logger.LogFile,
		MaxSize:    cfg.Logger.MaxSize,
		MaxBackups: cfg.Logger.MaxBackups,
		MaxAge:     cfg.Logger.MaxAge,
		Compress:   cfg.Logger.Compress,
	})

	core := zapcore.NewTee(
		newCore(encoder, stdout, Level),
		newCore(encoder, file, Level),
	)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
		zap.AddStacktrace(zapcore.ErrorLevel),
	)

	return logger.With(
		zap.String("service", cfg.ServiceName),
		zap.String("environment", cfg.Env),
	), nil
}

func newCore(encoder zapcore.Encoder, writer zapcore.WriteSyncer, level zapcore.LevelEnabler) zapcore.Core {
	core := zapcore.NewCore(
		encoder,
		writer,
		level,
	)

	// Inject PII Redaction
	core = &redactCore{Core: core}

	return zapcore.NewSamplerWithOptions(
		core,
		1,
		100,
		100,
	)
}

// --- PII Redaction Implementation ---

var secretKeys = map[string]bool{
	"password":      true,
	"token":         true,
	"access_token":  true,
	"refresh_token": true,
	"credit_card":   true,
	"secret":        true,
	"authorization": true,
}

type redactCore struct {
	zapcore.Core
}

func (c *redactCore) With(fields []zapcore.Field) zapcore.Core {
	return &redactCore{Core: c.Core.With(redactFields(fields))}
}

func (c *redactCore) Check(ent zapcore.Entry, ce *zapcore.CheckedEntry) *zapcore.CheckedEntry {
	if c.Enabled(ent.Level) {
		return ce.AddCore(ent, c)
	}
	return ce
}

func (c *redactCore) Write(ent zapcore.Entry, fields []zapcore.Field) error {
	return c.Core.Write(ent, redactFields(fields))
}

func redactFields(fields []zapcore.Field) []zapcore.Field {
	redacted := make([]zapcore.Field, len(fields))
	for i, f := range fields {
		if secretKeys[f.Key] {
			f.String = "[REDACTED]"
			f.Type = zapcore.StringType
			f.Integer = 0
			f.Interface = nil
		}
		redacted[i] = f
	}
	return redacted
}

func parseLevel(level string) zapcore.Level {
	switch level {
	case "debug":
		return zapcore.DebugLevel
	case "info":
		return zapcore.InfoLevel
	case "warn":
		return zapcore.WarnLevel
	case "error":
		return zapcore.ErrorLevel
	case "fatal":
		return zapcore.FatalLevel
	case "panic":
		return zapcore.PanicLevel
	default:
		return zapcore.InfoLevel
	}
}

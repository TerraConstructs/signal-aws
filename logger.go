package signal

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger interface defines the logging contract for tcsignal-aws
type Logger interface {
	Debug(msg string, fields ...zap.Field)
	Info(msg string, fields ...zap.Field)
	Warn(msg string, fields ...zap.Field)
	Error(msg string, fields ...zap.Field)
	Sync() error
	With(fields ...zap.Field) Logger
}

// ZapLogger wraps zap.Logger to implement our Logger interface
type ZapLogger struct {
	logger *zap.Logger
}

// NewLogger creates a new logger based on the provided format and level
func NewLogger(format string, level string) (Logger, error) {
	var zapLevel zapcore.Level
	switch level {
	case "debug":
		zapLevel = zap.DebugLevel
	case "info":
		zapLevel = zap.InfoLevel
	case "warn":
		zapLevel = zap.WarnLevel
	case "error":
		zapLevel = zap.ErrorLevel
	default:
		zapLevel = zap.InfoLevel
	}

	var logger *zap.Logger
	var err error

	if format == "json" {
		// Production configuration with JSON output
		config := zap.NewProductionConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.MessageKey = "msg"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.LowercaseLevelEncoder
		config.InitialFields = map[string]interface{}{
			"component": "tcsignal-aws",
		}
		logger, err = config.Build()
	} else if format == "console" {
		// Console/development configuration
		config := zap.NewDevelopmentConfig()
		config.Level = zap.NewAtomicLevelAt(zapLevel)
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.LevelKey = "level"
		config.EncoderConfig.MessageKey = "msg"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
		config.EncoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder
		config.InitialFields = map[string]interface{}{
			"component": "tcsignal-aws",
		}
		logger, err = config.Build()
	} else {
		return nil, fmt.Errorf("invalid log format: %s (must be 'json' or 'console')", format)
	}

	if err != nil {
		return nil, err
	}

	return &ZapLogger{logger: logger}, nil
}

// Debug logs a debug message with fields
func (zl *ZapLogger) Debug(msg string, fields ...zap.Field) {
	zl.logger.Debug(msg, fields...)
}

// Info logs an info message with fields
func (zl *ZapLogger) Info(msg string, fields ...zap.Field) {
	zl.logger.Info(msg, fields...)
}

// Warn logs a warning message with fields
func (zl *ZapLogger) Warn(msg string, fields ...zap.Field) {
	zl.logger.Warn(msg, fields...)
}

// Error logs an error message with fields
func (zl *ZapLogger) Error(msg string, fields ...zap.Field) {
	zl.logger.Error(msg, fields...)
}

// Sync flushes any buffered log entries
func (zl *ZapLogger) Sync() error {
	return zl.logger.Sync()
}

// With creates a child logger with additional fields
func (zl *ZapLogger) With(fields ...zap.Field) Logger {
	return &ZapLogger{logger: zl.logger.With(fields...)}
}

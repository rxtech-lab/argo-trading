package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps the zap logger with additional functionality
type Logger struct {
	*zap.Logger
}

// NewLogger creates a new logger instance with production configuration
func NewLogger() (*Logger, error) {
	config := zap.NewProductionConfig()

	// Set the output to stdout
	config.OutputPaths = []string{"stdout"}

	// Set the error output to stderr
	config.ErrorOutputPaths = []string{"stderr"}

	// Set the log level
	config.Level = zap.NewAtomicLevelAt(zapcore.InfoLevel)

	// Create the logger
	zapLogger, err := config.Build()
	if err != nil {
		return nil, err
	}

	return &Logger{
		Logger: zapLogger,
	}, nil
}

// Sync flushes any buffered log entries
func (l *Logger) Sync() error {
	if l.Logger != nil {
		return l.Logger.Sync()
	}
	return nil
}

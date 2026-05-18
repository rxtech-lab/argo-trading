package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger wraps the zap logger with additional functionality.
type Logger struct {
	*zap.Logger

	file *os.File
}

// NewLogger creates a new logger instance with production configuration.
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
		file:   nil,
	}, nil
}

// EnableFileOutput tees the logger's output to the given file path in addition
// to the existing destinations (typically stdout). Subsequent log calls write
// to both. Safe to call once per logger; calling again replaces the file sink.
func (l *Logger) EnableFileOutput(filePath string) error {
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}

	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	fileEncoder := zapcore.NewJSONEncoder(encoderConfig)
	fileCore := zapcore.NewCore(fileEncoder, zapcore.AddSync(file), zapcore.InfoLevel)

	teed := zapcore.NewTee(l.Logger.Core(), fileCore)
	l.Logger = zap.New(teed)

	if l.file != nil {
		_ = l.file.Close()
	}
	l.file = file

	return nil
}

// Sync flushes any buffered log entries.
func (l *Logger) Sync() error {
	if l.Logger != nil {
		_ = l.Logger.Sync()
	}
	if l.file != nil {
		return l.file.Sync()
	}

	return nil
}

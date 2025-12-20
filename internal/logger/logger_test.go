package logger

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type LoggerTestSuite struct {
	suite.Suite
}

func TestLoggerSuite(t *testing.T) {
	suite.Run(t, new(LoggerTestSuite))
}

func (suite *LoggerTestSuite) TestNewLogger() {
	logger, err := NewLogger()
	suite.NoError(err)
	suite.NotNil(logger)
	suite.NotNil(logger.Logger)
}

func (suite *LoggerTestSuite) TestLoggerSync() {
	logger, err := NewLogger()
	suite.NoError(err)
	suite.NotNil(logger)

	// Sync should not return an error for a valid logger
	err = logger.Sync()
	// Note: Sync may return an error on some systems (e.g., when syncing stdout)
	// but it should not panic
	_ = err
}

func (suite *LoggerTestSuite) TestLoggerSyncNilLogger() {
	logger := &Logger{Logger: nil}

	// Sync should not panic and should return nil for a nil inner logger
	err := logger.Sync()
	suite.NoError(err)
}

func (suite *LoggerTestSuite) TestLoggerLogging() {
	logger, err := NewLogger()
	suite.NoError(err)
	suite.NotNil(logger)

	// These should not panic
	logger.Info("test info message")
	logger.Debug("test debug message")
	logger.Warn("test warn message")
	logger.Error("test error message")
}

func (suite *LoggerTestSuite) TestLoggerWithFields() {
	logger, err := NewLogger()
	suite.NoError(err)
	suite.NotNil(logger)

	// Should not panic
	logger.With().Info("test message with fields")
}

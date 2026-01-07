package engine

import (
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// BacktestLogTestSuite is a test suite for BacktestLog
type BacktestLogTestSuite struct {
	suite.Suite
	logStorage *BacktestLog
	logger     *logger.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *BacktestLogTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger

	logStorage, err := NewBacktestLog(suite.logger)
	suite.Require().NoError(err)
	suite.logStorage = logStorage
}

// TearDownSuite runs once after all tests in the suite
func (suite *BacktestLogTestSuite) TearDownSuite() {
	if suite.logStorage != nil {
		suite.logStorage.Close()
	}
}

// SetupTest runs before each test
func (suite *BacktestLogTestSuite) SetupTest() {
	// Cleanup before running each test
	err := suite.logStorage.Cleanup()
	suite.Require().NoError(err)
}

// TestLogAndGet tests the Log and GetLogs methods
func (suite *BacktestLogTestSuite) TestLogAndGet() {
	// Test cases
	testCases := []struct {
		name  string
		entry log.LogEntry
	}{
		{
			name: "Debug log entry",
			entry: log.LogEntry{
				Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
				Symbol:    "AAPL",
				Level:     types.LogLevelDebug,
				Message:   "Debug message",
				Fields:    map[string]string{"key1": "value1"},
			},
		},
		{
			name: "Info log entry",
			entry: log.LogEntry{
				Timestamp: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC),
				Symbol:    "MSFT",
				Level:     types.LogLevelInfo,
				Message:   "Info message",
				Fields:    map[string]string{"order_id": "12345"},
			},
		},
	}

	// Insert test log entries
	for _, tc := range testCases {
		err := suite.logStorage.Log(tc.entry)
		suite.Require().NoError(err, "Failed to insert log entry for case: %s", tc.name)
	}

	// Get all logs
	logs, err := suite.logStorage.GetLogs()
	suite.Require().NoError(err, "Failed to get logs")
	suite.Require().Equal(len(testCases), len(logs), "Number of logs does not match test cases")

	// Verify logs
	for i, tc := range testCases {
		suite.Run(tc.name, func() {
			logEntry := logs[i]
			suite.Equal(tc.entry.Timestamp.UTC(), logEntry.Timestamp.UTC())
			suite.Equal(tc.entry.Symbol, logEntry.Symbol)
			suite.Equal(tc.entry.Level, logEntry.Level)
			suite.Equal(tc.entry.Message, logEntry.Message)
			suite.Equal(tc.entry.Fields, logEntry.Fields)
		})
	}
}

// TestLogWithAllLevels tests logging with all log levels
func (suite *BacktestLogTestSuite) TestLogWithAllLevels() {
	levels := []types.LogLevel{
		types.LogLevelDebug,
		types.LogLevelInfo,
		types.LogLevelWarning,
		types.LogLevelError,
	}

	baseTime := time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC)

	for i, level := range levels {
		entry := log.LogEntry{
			Timestamp: baseTime.Add(time.Duration(i) * time.Hour),
			Symbol:    "TEST",
			Level:     level,
			Message:   "Test message for " + string(level),
			Fields:    nil,
		}
		err := suite.logStorage.Log(entry)
		suite.Require().NoError(err, "Failed to log entry with level %s", level)
	}

	logs, err := suite.logStorage.GetLogs()
	suite.Require().NoError(err)
	suite.Require().NotNil(logs)
	suite.Require().Equal(len(levels), len(logs))

	for i, level := range levels {
		suite.Equal(level, logs[i].Level)
	}
}

// TestLogWithFields tests JSON fields serialization/deserialization
func (suite *BacktestLogTestSuite) TestLogWithFields() {
	entry := log.LogEntry{
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Symbol:    "BTC",
		Level:     types.LogLevelInfo,
		Message:   "Trade executed",
		Fields: map[string]string{
			"order_id": "ord-123",
			"quantity": "100",
			"price":    "50000.50",
			"strategy": "momentum",
			"special":  "value with \"quotes\" and 'apostrophes'",
		},
	}

	err := suite.logStorage.Log(entry)
	suite.Require().NoError(err)

	logs, err := suite.logStorage.GetLogs()
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(logs))

	suite.Equal(entry.Fields["order_id"], logs[0].Fields["order_id"])
	suite.Equal(entry.Fields["quantity"], logs[0].Fields["quantity"])
	suite.Equal(entry.Fields["price"], logs[0].Fields["price"])
	suite.Equal(entry.Fields["strategy"], logs[0].Fields["strategy"])
	suite.Equal(entry.Fields["special"], logs[0].Fields["special"])
}

// TestLogWithEmptyFields tests logging with nil/empty fields
func (suite *BacktestLogTestSuite) TestLogWithEmptyFields() {
	// Test with nil fields
	entry1 := log.LogEntry{
		Timestamp: time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Symbol:    "ETH",
		Level:     types.LogLevelInfo,
		Message:   "No fields",
		Fields:    nil,
	}
	err := suite.logStorage.Log(entry1)
	suite.Require().NoError(err)

	// Test with empty fields map
	entry2 := log.LogEntry{
		Timestamp: time.Date(2023, 1, 1, 11, 0, 0, 0, time.UTC),
		Symbol:    "ETH",
		Level:     types.LogLevelInfo,
		Message:   "Empty fields",
		Fields:    map[string]string{},
	}
	err = suite.logStorage.Log(entry2)
	suite.Require().NoError(err)

	logs, err := suite.logStorage.GetLogs()
	suite.Require().NoError(err)
	suite.Require().Equal(2, len(logs))
}

// TestWriteToParquet tests the Write method
func (suite *BacktestLogTestSuite) TestWriteToParquet() {
	// Create some log entries
	entry := log.LogEntry{
		Timestamp: time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
		Symbol:    "GOOG",
		Level:     types.LogLevelWarning,
		Message:   "Testing parquet export",
		Fields:    map[string]string{"test": "value"},
	}

	err := suite.logStorage.Log(entry)
	suite.Require().NoError(err)

	// Create a temporary directory for the test
	tempDir := suite.T().TempDir()

	// Export to parquet
	err = suite.logStorage.Write(tempDir)
	suite.Require().NoError(err, "Failed to write logs to parquet")
}

// TestCleanup tests the Cleanup method
func (suite *BacktestLogTestSuite) TestCleanup() {
	// Add some entries
	entry := log.LogEntry{
		Timestamp: time.Now(),
		Symbol:    "TEST",
		Level:     types.LogLevelInfo,
		Message:   "Before cleanup",
		Fields:    nil,
	}
	err := suite.logStorage.Log(entry)
	suite.Require().NoError(err)

	// Verify entry exists
	logs, err := suite.logStorage.GetLogs()
	suite.Require().NoError(err)
	suite.Require().Equal(1, len(logs))

	// Cleanup
	err = suite.logStorage.Cleanup()
	suite.Require().NoError(err)

	// Verify logs are empty after cleanup
	logs, err = suite.logStorage.GetLogs()
	suite.Require().NoError(err)
	suite.Require().Equal(0, len(logs))
}

// TestBacktestLogSuite runs the test suite
func TestBacktestLogSuite(t *testing.T) {
	suite.Run(t, new(BacktestLogTestSuite))
}

package main

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// LogErrorTestSuite extends the base test suite
type LogErrorTestSuite struct {
	testhelper.E2ETestSuite
}

func TestLogErrorTestSuite(t *testing.T) {
	suite.Run(t, new(LogErrorTestSuite))
}

// SetupTest initializes the test with config
func (s *LogErrorTestSuite) SetupTest() {
}

func (s *LogErrorTestSuite) TestLogAndErrorMarkers() {
	s.Run("TestLogAndErrorMarkers", func() {
		s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "LogErrorStrategy", "./log_error_plugin.wasm", "")

		// 1. Verify logs.parquet exists and contains correct entries
		logs, err := testhelper.ReadLogs(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Greater(len(logs), 0, "Should have log entries")

		// Verify log levels are present
		hasInfo := false
		hasDebug := false
		hasWarning := false
		hasError := false
		for _, logEntry := range logs {
			s.Require().NotEmpty(logEntry.Symbol, "Log should have symbol from market data")
			s.Require().False(logEntry.Timestamp.IsZero(), "Log should have timestamp from market data")

			switch logEntry.Level {
			case types.LogLevelInfo:
				hasInfo = true
			case types.LogLevelDebug:
				hasDebug = true
			case types.LogLevelWarning:
				hasWarning = true
			case types.LogLevelError:
				hasError = true
			}
		}
		s.Require().True(hasInfo, "Should have INFO level log")
		s.Require().True(hasDebug, "Should have DEBUG level log")
		s.Require().True(hasWarning, "Should have WARNING level log")
		s.Require().True(hasError, "Should have ERROR level log")

		// Verify first log has fields
		firstLog := logs[0]
		s.Require().NotNil(firstLog.Fields, "First log should have fields")
		s.Require().Equal("0", firstLog.Fields["count"], "First log should have count field")
		s.Require().Equal("start", firstLog.Fields["action"], "First log should have action field")

		// 2. Verify error markers exist
		markers, err := testhelper.ReadMarker(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)

		// Find error marker
		var errorMarker *types.Mark
		for i, m := range markers {
			if m.Level == types.MarkLevelError && m.Category == "StrategyError" {
				errorMarker = &markers[i]
				break
			}
		}
		s.Require().NotNil(errorMarker, "Should have error marker for strategy error")
		s.Require().Equal("Strategy Error", errorMarker.Title)
		s.Require().Contains(errorMarker.Message, "simulated strategy error")
		s.Require().Equal(types.MarkColorRed, errorMarker.Color)

		// 3. Verify stats file has logs path
		stats, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Greater(len(stats), 0, "Should have stats")
		s.Require().NotEmpty(stats[0].LogsFilePath, "Stats should have logs file path")

		// 4. Verify processing continued after error (we should have logs after the error)
		// The recovered log should be present, indicating processing continued
		hasRecovered := false
		for _, logEntry := range logs {
			if logEntry.Message == "Recovered after error" {
				hasRecovered = true
				break
			}
		}
		s.Require().True(hasRecovered, "Should have recovered log after error, indicating processing continued")
	})
}

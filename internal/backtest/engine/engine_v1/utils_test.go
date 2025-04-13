package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
)

// MockStrategy implements TradingStrategy interface for testing
type MockStrategy struct {
	name string
}

func (m *MockStrategy) Name() string {
	return m.name
}

func (m *MockStrategy) Initialize(config string) error {
	return nil
}

func (m *MockStrategy) ProcessData(data types.MarketData) error {
	return nil
}

func (m *MockStrategy) InitializeApi(api strategy.StrategyApi) error {
	return nil
}

// UtilsTestSuite is a test suite for utils package
type UtilsTestSuite struct {
	suite.Suite
}

// TestUtilsSuite runs the test suite
func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (suite *UtilsTestSuite) TestGetResultFolder() {
	tests := []struct {
		name          string
		configPath    string
		dataPath      string
		strategyName  string
		resultsFolder string
		startTime     optional.Option[time.Time]
		endTime       optional.Option[time.Time]
		expectedPath  string
		expectedError bool
	}{
		{
			name:          "Basic case without time range",
			configPath:    "/path/to/config.json",
			dataPath:      "/path/to/data.csv",
			strategyName:  "TestStrategy",
			resultsFolder: "/results",
			startTime:     optional.None[time.Time](),
			endTime:       optional.None[time.Time](),
			expectedPath:  "/results/TestStrategy/config/data",
			expectedError: false,
		},
		{
			name:          "Case with time range",
			configPath:    "/path/to/config.json",
			dataPath:      "/path/to/data.csv",
			strategyName:  "TestStrategy",
			resultsFolder: "/results",
			startTime:     optional.Some(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
			endTime:       optional.Some(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			expectedPath:  "/results/TestStrategy/config/20230101_20231231/data",
			expectedError: false,
		},
		{
			name:          "Case with only start time",
			configPath:    "/path/to/config.json",
			dataPath:      "/path/to/data.csv",
			strategyName:  "TestStrategy",
			resultsFolder: "/results",
			startTime:     optional.Some(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
			endTime:       optional.None[time.Time](),
			expectedPath:  "/results/TestStrategy/config/20230101_all/data",
			expectedError: false,
		},
		{
			name:          "Case with only end time",
			configPath:    "/path/to/config.json",
			dataPath:      "/path/to/data.csv",
			strategyName:  "TestStrategy",
			resultsFolder: "/results",
			startTime:     optional.None[time.Time](),
			endTime:       optional.Some(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
			expectedPath:  "/results/TestStrategy/config/all_20231231/data",
			expectedError: false,
		},
		{
			name:          "Case with complex file names",
			configPath:    "/path/to/my.config.json",
			dataPath:      "/path/to/trading.data.csv",
			strategyName:  "ComplexStrategy",
			resultsFolder: "/results",
			startTime:     optional.None[time.Time](),
			endTime:       optional.None[time.Time](),
			expectedPath:  "/results/ComplexStrategy/my.config/trading.data",
			expectedError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Create a mock strategy
			mockStrategy := &MockStrategy{name: tc.strategyName}

			// Create a mock backtest engine
			mockEngine := &BacktestEngineV1{
				config: BacktestEngineV1Config{
					StartTime: tc.startTime,
					EndTime:   tc.endTime,
				},
				resultsFolder: tc.resultsFolder,
			}

			// Get the result folder path
			resultPath := getResultFolder(tc.configPath, tc.dataPath, mockEngine, mockStrategy)

			// Normalize paths for comparison
			expectedPath := filepath.Clean(tc.expectedPath)
			resultPath = filepath.Clean(resultPath)

			// Assert the paths match
			suite.Assert().Equal(expectedPath, resultPath, "Result folder path mismatch")
		})
	}
}

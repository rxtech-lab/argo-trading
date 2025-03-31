package engine

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
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

func (m *MockStrategy) ProcessData(ctx strategy.StrategyContext, data types.MarketData, targetSymbol string) ([]types.ExecuteOrder, error) {
	return nil, nil
}

// UtilsTestSuite is a test suite for utils package
type UtilsTestSuite struct {
	suite.Suite
}

// TestUtilsSuite runs the test suite
func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (suite *UtilsTestSuite) TestCalculateMaxQuantity() {
	tests := []struct {
		name          string
		balance       float64
		price         float64
		commissionFee commission_fee.CommissionFee
		expectedQty   int
		expectedError bool
	}{
		{
			name:          "Simple case with no commission",
			balance:       1000.0,
			price:         100.0,
			commissionFee: commission_fee.NewZeroCommissionFee(),
			expectedQty:   10,
			expectedError: false,
		},
		{
			name:          "Case with commission",
			balance:       1000.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   9,
			expectedError: false,
		},
		{
			name:          "Zero balance",
			balance:       0.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
		{
			name:          "Zero price",
			balance:       1000.0,
			price:         0.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
		{
			name:          "Balance less than price",
			balance:       50.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			qty := CalculateMaxQuantity(tc.balance, tc.price, tc.commissionFee)
			suite.Assert().Equal(tc.expectedQty, qty, "Quantity mismatch")
		})
	}
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

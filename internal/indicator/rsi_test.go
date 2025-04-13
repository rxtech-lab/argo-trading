package indicator

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// RSITestSuite is a test suite for the RSI indicator
type RSITestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	rsi        *RSI
	logger     *logger.Logger
}

// SetupSuite sets up the test suite
func (suite *RSITestSuite) SetupSuite() {
	// Create a no-op logger that doesn't log to console
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}      // Empty output paths to prevent console logging
	loggerConfig.ErrorOutputPaths = []string{} // Empty error output paths
	zapLogger, err := loggerConfig.Build()
	suite.Require().NoError(err)
	suite.logger = &logger.Logger{Logger: zapLogger}

	// Create an in-memory DuckDB database for testing
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Create a DuckDBDataSource
	dataSource, err := datasource.NewDataSource(":memory:", suite.logger)
	suite.Require().NoError(err)
	suite.dataSource = dataSource

	// Initialize RSI with a default period of 14
	rsi := NewRSI()
	suite.rsi = rsi.(*RSI)
	suite.Require().NotNil(suite.rsi)

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *RSITestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestRSICalcSuite runs the test suite
func TestRSICalcSuite(t *testing.T) {
	suite.Run(t, new(RSITestSuite))
}

// TestRSICalculation tests the RSI calculation against the expected values from the parquet file
func (suite *RSITestSuite) TestRSICalculation() {
	// Define test cases
	testCases := []struct {
		name              string
		symbol            string
		index             int // Index in the data to check (should be >= 14 as RSI is only available after 14 data points)
		allowedDifference float64
	}{
		{
			name:              "RSI after 14 data points",
			symbol:            "AAPL", // Using AAPL as the test symbol
			index:             14,     // Test at the first point RSI should be available
			allowedDifference: 0.1,    // Allow up to 0.1 difference
		},
		{
			name:              "RSI at mid-range point",
			symbol:            "AAPL",
			index:             7,
			allowedDifference: 0.1,
		},
		{
			name:              "RSI at later point",
			symbol:            "AAPL",
			index:             21,
			allowedDifference: 0.1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Query all RSI values from the parquet data
			columnName := fmt.Sprintf("rsi_%d", tc.index)
			expectedRSIQuery := `
				SELECT time, close, ` + columnName + `
				FROM market_data
				WHERE ` + columnName + ` IS NOT NULL
				ORDER BY time ASC
			`
			results, err := suite.dataSource.ExecuteSQL(expectedRSIQuery)
			suite.Require().NoError(err, "Failed to query expected RSI values")
			suite.Require().NotEmpty(results, "No results returned for expected RSI query")

			// Create indicator context for RSI calculation
			ctx := IndicatorContext{
				DataSource: suite.dataSource,
			}

			suite.Require().NoError(err, "Failed to query all historical data")

			// Skip the first tc.index data points as RSI is only available after that
			for i, result := range results {
				if i < tc.index {
					continue
				}
				expectedTime := result.Values["time"].(time.Time)
				expectedRSI := result.Values[columnName].(float64)

				currentMarketData, err := suite.dataSource.GetMarketData(tc.symbol, expectedTime)
				suite.Require().NoError(err, "Failed to get market data for time %v", expectedTime)
				suite.Require().NotEqual(time.Time{}, currentMarketData.Time,
					"Could not find market data for time %v", expectedTime)

				// Calculate RSI for this point
				// Configure RSI indicator with the test period
				suite.rsi.Config(tc.index)
				calculatedRSI, err := suite.rsi.RawValue(currentMarketData.Symbol, currentMarketData.Time, ctx)
				signal, err := suite.rsi.GetSignal(currentMarketData, ctx)
				suite.Require().NoError(err, "Failed to calculate RSI at point %d", i)

				// Compare expected and calculated RSI values
				diff := expectedRSI - calculatedRSI
				if diff < 0 {
					diff = -diff
				}
				suite.Assert().LessOrEqual(diff, tc.allowedDifference,
					"RSI difference too large at point %d (time: %v): expected %f, got %f, diff %f",
					i, expectedTime, expectedRSI, calculatedRSI, diff)

				// expect signal to be a buy or sell
				suite.Assert().Equal(signal.Symbol, tc.symbol)
			}
		})
	}
}

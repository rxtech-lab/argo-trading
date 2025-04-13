package indicator

import (
	"database/sql"
	"fmt"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// EMATestSuite is a test suite for the EMA indicator
type EMATestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	ema        *EMA
	logger     *logger.Logger
}

// SetupSuite sets up the test suite
func (suite *EMATestSuite) SetupSuite() {
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

	// Initialize EMA with a default period of 20
	ema := NewEMA()
	suite.ema = ema.(*EMA)
	suite.Require().NotNil(suite.ema)

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *EMATestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestEMACalcSuite runs the test suite
func TestEMACalcSuite(t *testing.T) {
	suite.Run(t, new(EMATestSuite))
}

// TestEMACalculation tests the EMA calculation against the expected values from the parquet file
func (suite *EMATestSuite) TestEMACalculation() {
	// Define test cases
	testCases := []struct {
		name              string
		symbol            string
		period            int // EMA period to test
		allowedDifference float64
	}{
		{
			name:              "EMA with period 7",
			symbol:            "AAPL",
			period:            7,
			allowedDifference: 1,
		},
		{
			name:              "EMA with period 14",
			symbol:            "AAPL",
			period:            14,
			allowedDifference: 1,
		},
		{
			name:              "EMA with period 21",
			symbol:            "AAPL",
			period:            21,
			allowedDifference: 1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Query all EMA values from the parquet data
			columnName := fmt.Sprintf("ema_%d", tc.period)
			expectedEMAQuery := `
				SELECT time, close, ` + columnName + `
				FROM market_data
				WHERE ` + columnName + ` IS NOT NULL
				AND symbol = ?
				ORDER BY time ASC
			`
			results, err := suite.dataSource.ExecuteSQL(expectedEMAQuery, tc.symbol)
			suite.Require().NoError(err, "Failed to query expected EMA values")
			suite.Require().NotEmpty(results, "No results returned for expected EMA query")

			// Create indicator context for EMA calculation
			ctx := IndicatorContext{
				DataSource: suite.dataSource,
			}

			// Configure EMA with the test period
			suite.ema.Config(tc.period)

			// Skip the first tc.period data points as EMA is only reliable after that
			for i, result := range results {
				if i < tc.period {
					continue
				}

				expectedTime := result.Values["time"].(time.Time)
				expectedEMA := result.Values[columnName].(float64)

				// Get market data for the expected time
				currentMarketData, err := suite.dataSource.GetMarketData(tc.symbol, expectedTime)
				suite.Require().NoError(err, "Failed to get market data for time %v", expectedTime)
				suite.Require().NotEqual(time.Time{}, currentMarketData.Time,
					"Could not find market data for time %v", expectedTime)

				// Calculate EMA for this point
				calculatedEMA, err := suite.ema.RawValue(currentMarketData.Symbol, currentMarketData.Time, ctx)
				suite.Require().NoError(err, "Failed to calculate EMA at point %d", i)

				// Compare expected and calculated EMA values
				diff := expectedEMA - calculatedEMA
				if diff < 0 {
					diff = -diff
				}
				suite.Assert().LessOrEqual(diff, tc.allowedDifference,
					"EMA difference too large at point %d (time: %v): expected %f, got %f, diff %f",
					i, expectedTime, expectedEMA, calculatedEMA, diff)

				// Also test getting signal
				signal, err := suite.ema.GetSignal(currentMarketData, ctx)
				suite.Require().NoError(err, "Failed to get signal at point %d", i)

				// Verify signal properties
				suite.Assert().Equal(currentMarketData.Symbol, signal.Symbol)
				suite.Assert().Equal(currentMarketData.Time, signal.Time)
				suite.Assert().Equal(string(types.IndicatorTypeEMA), signal.Name)

				// Verify EMA value in signal matches calculated value
				rawValue, ok := signal.RawValue.(map[string]float64)
				suite.Assert().True(ok, "Failed to cast RawValue to map[string]float64")
				emaFromSignal, exists := rawValue["ema"]
				suite.Assert().True(exists, "EMA value not found in signal raw values")
				suite.Assert().Equal(calculatedEMA, emaFromSignal, "EMA values don't match")
			}
		})
	}
}

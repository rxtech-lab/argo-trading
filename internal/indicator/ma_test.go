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

// MATestSuite is a test suite for the MA indicator
type MATestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	ma         *MA
	logger     *logger.Logger
}

// SetupSuite sets up the test suite
func (suite *MATestSuite) SetupSuite() {
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

	// Initialize MA with a default period of 20
	ma := NewMA()
	suite.ma = ma.(*MA)
	suite.Require().NotNil(suite.ma)

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *MATestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestMACalcSuite runs the test suite
func TestMACalcSuite(t *testing.T) {
	suite.Run(t, new(MATestSuite))
}

// TestMACalculation tests the MA calculation against the expected values from the parquet file
func (suite *MATestSuite) TestMACalculation() {
	// Define test cases
	testCases := []struct {
		name              string
		symbol            string
		period            int // MA period to test
		allowedDifference float64
	}{
		{
			name:              "MA with period 7",
			symbol:            "AAPL",
			period:            7,
			allowedDifference: 1,
		},
		{
			name:              "MA with period 14",
			symbol:            "AAPL",
			period:            14,
			allowedDifference: 1,
		},
		{
			name:              "MA with period 21",
			symbol:            "AAPL",
			period:            21,
			allowedDifference: 1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Query all MA values from the parquet data
			columnName := fmt.Sprintf("sma_%d", tc.period)
			expectedMAQuery := `
				SELECT time, close, ` + columnName + `
				FROM market_data
				WHERE ` + columnName + ` IS NOT NULL
				AND symbol = ?
				ORDER BY time ASC
			`
			results, err := suite.dataSource.ExecuteSQL(expectedMAQuery, tc.symbol)
			suite.Require().NoError(err, "Failed to query expected MA values")
			suite.Require().NotEmpty(results, "No results returned for expected MA query")

			// Create indicator context for MA calculation
			ctx := IndicatorContext{
				DataSource: suite.dataSource,
			}

			// Configure MA with the test period
			suite.ma.Config(tc.period)

			// Skip the first tc.period data points as MA is only reliable after that
			for i, result := range results {
				if i < tc.period {
					continue
				}

				expectedTime := result.Values["time"].(time.Time)
				expectedMA := result.Values[columnName].(float64)

				// Get market data for the expected time
				currentMarketData, err := suite.dataSource.GetMarketData(tc.symbol, expectedTime)
				suite.Require().NoError(err, "Failed to get market data for time %v", expectedTime)
				suite.Require().NotEqual(time.Time{}, currentMarketData.Time,
					"Could not find market data for time %v", expectedTime)

				// Calculate MA for this point
				calculatedMA, err := suite.ma.RawValue(currentMarketData.Symbol, currentMarketData.Time, ctx)
				suite.Require().NoError(err, "Failed to calculate MA at point %d", i)

				// Compare expected and calculated MA values
				diff := expectedMA - calculatedMA
				if diff < 0 {
					diff = -diff
				}
				suite.Assert().LessOrEqual(diff, tc.allowedDifference,
					"MA difference too large at point %d (time: %v): expected %f, got %f, diff %f",
					i, expectedTime, expectedMA, calculatedMA, diff)

				// Also test getting signal
				signal, err := suite.ma.GetSignal(currentMarketData, ctx)
				suite.Require().NoError(err, "Failed to get signal at point %d", i)

				// Verify signal properties
				suite.Assert().Equal(currentMarketData.Symbol, signal.Symbol)
				suite.Assert().Equal(currentMarketData.Time, signal.Time)
				suite.Assert().Equal(string(types.IndicatorTypeMA), signal.Name)

				// Verify MA value in signal matches calculated value
				rawValue, ok := signal.RawValue.(map[string]float64)
				suite.Assert().True(ok, "Failed to cast RawValue to map[string]float64")
				maFromSignal, exists := rawValue["ma"]
				suite.Assert().True(exists, "MA value not found in signal raw values")
				suite.Assert().Equal(calculatedMA, maFromSignal, "MA values don't match")
			}
		})
	}
}

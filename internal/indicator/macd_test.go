package indicator

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// MACDTestSuite is a test suite for the MACD indicator
type MACDTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	macd       *MACD
	logger     *logger.Logger
	registry   IndicatorRegistry
}

// SetupSuite sets up the test suite
func (suite *MACDTestSuite) SetupSuite() {
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

	// Initialize MACD with default parameters
	macd := NewMACD()
	suite.macd = macd.(*MACD)
	suite.Require().NotNil(suite.macd)

	// Create indicator registry and register EMA (needed by MACD)
	suite.registry = NewIndicatorRegistry()
	suite.registry.RegisterIndicator(NewEMA())

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *MACDTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestMACDCalcSuite runs the test suite
func TestMACDCalcSuite(t *testing.T) {
	suite.Run(t, new(MACDTestSuite))
}

// TestMACDCalculation tests the MACD calculation against the expected values from the parquet file
func (suite *MACDTestSuite) TestMACDCalculation() {
	// Define test cases with different MACD parameters
	testCases := []struct {
		name              string
		symbol            string
		fastPeriod        int
		slowPeriod        int
		signalPeriod      int
		allowedDifference float64
	}{
		{
			name:              "MACD with default parameters (12, 26, 9)",
			symbol:            "AAPL",
			fastPeriod:        12,
			slowPeriod:        26,
			signalPeriod:      9,
			allowedDifference: 1,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Query MACD line values from the parquet data
			macdLineQuery := `
				SELECT time, close, macd_line
				FROM market_data
				WHERE macd_line IS NOT NULL
				AND symbol = ?
				ORDER BY time ASC
			`
			results, err := suite.dataSource.ExecuteSQL(macdLineQuery, tc.symbol)
			suite.Require().NoError(err, "Failed to query expected MACD line values")
			suite.Require().NotEmpty(results, "No results returned for expected MACD line query")

			// Create indicator context for MACD calculation
			ctx := IndicatorContext{
				DataSource:        suite.dataSource,
				IndicatorRegistry: suite.registry,
			}

			// Configure MACD with the test parameters
			suite.macd.Config(tc.fastPeriod, tc.slowPeriod, tc.signalPeriod)

			// Skip the first few data points as MACD needs enough data points to calculate
			skipPoints := tc.slowPeriod // Skip at least slow period data points
			for i, result := range results {
				if i < skipPoints {
					continue
				}

				expectedTime := result.Values["time"].(time.Time)
				expectedMACDLine := result.Values["macd_line"].(float64)

				// Get market data for the expected time
				currentMarketData, err := suite.dataSource.GetMarketData(tc.symbol, expectedTime)
				suite.Require().NoError(err, "Failed to get market data for time %v", expectedTime)
				suite.Require().NotEqual(time.Time{}, currentMarketData.Time,
					"Could not find market data for time %v", expectedTime)

				// Calculate MACD for this point
				calculatedMACD, err := suite.macd.RawValue(currentMarketData.Symbol, currentMarketData.Time, ctx)
				suite.Require().NoError(err, "Failed to calculate MACD at point %d", i)

				// Compare expected and calculated MACD values
				diff := expectedMACDLine - calculatedMACD
				if diff < 0 {
					diff = -diff
				}
				suite.Assert().LessOrEqual(diff, tc.allowedDifference,
					"MACD difference too large at point %d (time: %v): expected %f, got %f, diff %f",
					i, expectedTime, expectedMACDLine, calculatedMACD, diff)
			}
		})
	}
}

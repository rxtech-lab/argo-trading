package indicator

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// ATRIntegrationTestSuite is an integration test suite for the ATR indicator
type ATRIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	atr        *ATR
	logger     *logger.Logger
	registry   IndicatorRegistry
}

// SetupSuite sets up the test suite
func (suite *ATRIntegrationTestSuite) SetupSuite() {
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

	// Initialize ATR with a default period of 14
	atr := NewATR()
	suite.atr = atr.(*ATR)
	suite.Require().NotNil(suite.atr)

	// Create indicator registry and register EMA (needed by ATR)
	suite.registry = NewIndicatorRegistry()
	suite.registry.RegisterIndicator(NewEMA())

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *ATRIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestATRIntegrationSuite runs the integration test suite
func TestATRIntegrationSuite(t *testing.T) {
	suite.Run(t, new(ATRIntegrationTestSuite))
}

// TestATRGetSignal tests the GetSignal method of the ATR indicator
func (suite *ATRIntegrationTestSuite) TestATRGetSignal() {
	// Query data for testing
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 50 OFFSET 30
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource:        suite.dataSource,
		IndicatorRegistry: suite.registry,
	}

	// Configure ATR with period 14
	err = suite.atr.Config(14)
	suite.Require().NoError(err)

	// Test GetSignal for each data point
	for _, result := range results {
		marketData := types.MarketData{
			Time:   result.Values["time"].(time.Time),
			Symbol: result.Values["symbol"].(string),
			Open:   result.Values["open"].(float64),
			High:   result.Values["high"].(float64),
			Low:    result.Values["low"].(float64),
			Close:  result.Values["close"].(float64),
			Volume: result.Values["volume"].(float64),
		}

		signal, err := suite.atr.GetSignal(marketData, ctx)
		suite.Require().NoError(err, "Failed to get ATR signal")

		// Verify signal properties
		suite.Equal(marketData.Symbol, signal.Symbol)
		suite.Equal(marketData.Time, signal.Time)
		suite.Equal(types.IndicatorTypeATR, signal.Indicator)
		suite.Equal(types.SignalTypeNoAction, signal.Type) // ATR always returns NoAction

		// Verify raw value contains ATR
		rawValue, ok := signal.RawValue.(map[string]float64)
		suite.True(ok, "Failed to cast RawValue to map[string]float64")
		atrValue, exists := rawValue["atr"]
		suite.True(exists, "ATR value not found in signal raw values")
		suite.Greater(atrValue, 0.0, "ATR value should be positive")
	}
}

// TestATRRawValueWithContext tests RawValue with a valid context
func (suite *ATRIntegrationTestSuite) TestATRRawValueWithContext() {
	// Query data for testing
	query := `
		SELECT time, symbol
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 1 OFFSET 50
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource:        suite.dataSource,
		IndicatorRegistry: suite.registry,
	}

	// Configure ATR with period 14
	err = suite.atr.Config(14)
	suite.Require().NoError(err)

	result := results[0]
	symbol := result.Values["symbol"].(string)
	timestamp := result.Values["time"].(time.Time)

	// Test RawValue
	atrValue, err := suite.atr.RawValue(symbol, timestamp, ctx)
	suite.Require().NoError(err, "Failed to calculate ATR")
	suite.Greater(atrValue, 0.0, "ATR value should be positive")
}

// TestATRRawValueInvalidContext tests RawValue with invalid context
func (suite *ATRIntegrationTestSuite) TestATRRawValueInvalidContext() {
	atr := NewATR()

	// Test with invalid context type
	_, err := atr.RawValue("AAPL", time.Now(), "invalid-context")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
}

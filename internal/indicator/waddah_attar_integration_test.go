package indicator

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// WaddahAttarIntegrationTestSuite is an integration test suite for the Waddah Attar indicator
type WaddahAttarIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	wa         *WaddahAttar
	logger     *logger.Logger
	registry   IndicatorRegistry
	cache      cache.Cache
}

// SetupSuite sets up the test suite
func (suite *WaddahAttarIntegrationTestSuite) SetupSuite() {
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

	// Initialize Waddah Attar with default configuration
	wa := NewWaddahAttar()
	suite.wa = wa.(*WaddahAttar)
	suite.Require().NotNil(suite.wa)

	// Create indicator registry and register required indicators
	suite.registry = NewIndicatorRegistry()
	suite.registry.RegisterIndicator(NewEMA())
	suite.registry.RegisterIndicator(NewMACD())
	suite.registry.RegisterIndicator(NewATR())

	// Create cache
	suite.cache = cache.NewCacheV1()

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *WaddahAttarIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestWaddahAttarIntegrationSuite runs the integration test suite
func TestWaddahAttarIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WaddahAttarIntegrationTestSuite))
}

// TestWaddahAttarGetSignal tests the GetSignal method of the Waddah Attar indicator
func (suite *WaddahAttarIntegrationTestSuite) TestWaddahAttarGetSignal() {
	// Query data for testing - need enough data points
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 50 OFFSET 100
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource:        suite.dataSource,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Configure Waddah Attar with smaller periods for testing
	err = suite.wa.Config(10, 20, 5, 10, 100.0)
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

		signal, err := suite.wa.GetSignal(marketData, ctx)
		suite.Require().NoError(err, "Failed to get Waddah Attar signal")

		// Verify signal properties
		suite.Equal(marketData.Symbol, signal.Symbol)
		suite.Equal(marketData.Time, signal.Time)
		suite.Equal(types.IndicatorTypeWaddahAttar, signal.Indicator)

		// Signal type can be BuyLong, SellShort, or NoAction
		suite.True(signal.Type == types.SignalTypeBuyLong ||
			signal.Type == types.SignalTypeSellShort ||
			signal.Type == types.SignalTypeNoAction,
			"Signal type should be BuyLong, SellShort, or NoAction")
	}
}

// TestWaddahAttarRawValueWithContext tests RawValue with a valid context
func (suite *WaddahAttarIntegrationTestSuite) TestWaddahAttarRawValueWithContext() {
	// Query data for testing
	query := `
		SELECT time, symbol
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 1 OFFSET 150
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource:        suite.dataSource,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Configure Waddah Attar
	err = suite.wa.Config(10, 20, 5, 10, 100.0)
	suite.Require().NoError(err)

	result := results[0]
	symbol := result.Values["symbol"].(string)
	timestamp := result.Values["time"].(time.Time)

	// Test RawValue
	value, err := suite.wa.RawValue(symbol, timestamp, ctx)
	suite.Require().NoError(err, "Failed to calculate Waddah Attar")
	// Value could be positive or negative
	_ = value // Just verify it doesn't error
}

// TestWaddahAttarRawValueInvalidContext tests RawValue with invalid context
func (suite *WaddahAttarIntegrationTestSuite) TestWaddahAttarRawValueInvalidContext() {
	wa := NewWaddahAttar()

	// Test with invalid context type
	_, err := wa.RawValue("AAPL", time.Now(), "invalid-context")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
}

// TestWaddahAttarMultipleSignals tests getting multiple signals over time
func (suite *WaddahAttarIntegrationTestSuite) TestWaddahAttarMultipleSignals() {
	// Query a larger dataset
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 100 OFFSET 50
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")

	// Create a new cache for this test
	testCache := cache.NewCacheV1()

	// Create indicator context
	ctx := IndicatorContext{
		DataSource:        suite.dataSource,
		IndicatorRegistry: suite.registry,
		Cache:             testCache,
	}

	// Create new Waddah Attar for this test
	wa := NewWaddahAttar()
	err = wa.Config(10, 20, 5, 10, 100.0)
	suite.Require().NoError(err)

	signalTypeCounts := map[types.SignalType]int{}

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

		signal, err := wa.GetSignal(marketData, ctx)
		suite.Require().NoError(err)
		signalTypeCounts[signal.Type]++
	}

	// Should see at least some signals after processing enough data
	totalSignals := 0
	for _, count := range signalTypeCounts {
		totalSignals += count
	}
	suite.Equal(len(results), totalSignals, "Should have processed all results")
}

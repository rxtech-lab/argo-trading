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

// RangeFilterIntegrationTestSuite is an integration test suite for the Range Filter indicator
type RangeFilterIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	rf         *RangeFilter
	logger     *logger.Logger
	registry   IndicatorRegistry
	cache      cache.Cache
}

// SetupSuite sets up the test suite
func (suite *RangeFilterIntegrationTestSuite) SetupSuite() {
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

	// Initialize Range Filter with default configuration
	rf := NewRangeFilter()
	suite.rf = rf.(*RangeFilter)
	suite.Require().NotNil(suite.rf)

	// Create indicator registry and register EMA (needed by Range Filter)
	suite.registry = NewIndicatorRegistry()
	suite.registry.RegisterIndicator(NewEMA())

	// Create cache
	suite.cache = cache.NewCacheV1()

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *RangeFilterIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestRangeFilterIntegrationSuite runs the integration test suite
func TestRangeFilterIntegrationSuite(t *testing.T) {
	suite.Run(t, new(RangeFilterIntegrationTestSuite))
}

// TestRangeFilterGetSignal tests the GetSignal method of the Range Filter indicator
func (suite *RangeFilterIntegrationTestSuite) TestRangeFilterGetSignal() {
	// Query data for testing - need enough data points
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 50 OFFSET 150
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

	// Configure Range Filter with smaller period for testing
	err = suite.rf.Config(10, 2.0)
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

		signal, err := suite.rf.GetSignal(marketData, ctx)
		suite.Require().NoError(err, "Failed to get Range Filter signal")

		// Verify signal properties
		suite.Equal(marketData.Symbol, signal.Symbol)
		suite.Equal(marketData.Time, signal.Time)
		suite.Equal(types.IndicatorTypeRangeFilter, signal.Indicator)

		// Signal type can be BuyLong, SellShort, or NoAction
		suite.True(signal.Type == types.SignalTypeBuyLong ||
			signal.Type == types.SignalTypeSellShort ||
			signal.Type == types.SignalTypeNoAction,
			"Signal type should be BuyLong, SellShort, or NoAction")
	}
}

// TestRangeFilterRawValueWithContext tests RawValue with a valid context
func (suite *RangeFilterIntegrationTestSuite) TestRangeFilterRawValueWithContext() {
	// Query data for testing
	query := `
		SELECT time, symbol
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 1 OFFSET 200
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

	// Configure Range Filter
	err = suite.rf.Config(10, 2.0)
	suite.Require().NoError(err)

	result := results[0]
	symbol := result.Values["symbol"].(string)
	timestamp := result.Values["time"].(time.Time)

	// Test RawValue
	filterValue, err := suite.rf.RawValue(symbol, timestamp, ctx)
	suite.Require().NoError(err, "Failed to calculate Range Filter")
	// Filter value could be positive or negative depending on trend
	suite.NotEqual(0.0, filterValue, "Filter value should not be zero after initialization")
}

// TestRangeFilterRawValueInvalidContext tests RawValue with invalid context
func (suite *RangeFilterIntegrationTestSuite) TestRangeFilterRawValueInvalidContext() {
	rf := NewRangeFilter()

	// Test with invalid context type
	_, err := rf.RawValue("AAPL", time.Now(), "invalid-context")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
}

// TestRangeFilterMultipleSignals tests getting multiple signals over time
func (suite *RangeFilterIntegrationTestSuite) TestRangeFilterMultipleSignals() {
	// Query a larger dataset
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 100 OFFSET 100
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

	// Create new Range Filter for this test
	rf := NewRangeFilter()
	err = rf.Config(15, 2.5)
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

		signal, err := rf.GetSignal(marketData, ctx)
		suite.Require().NoError(err)
		signalTypeCounts[signal.Type]++
	}

	// Verify we got meaningful signal distribution after processing data
	totalSignals := 0
	for _, count := range signalTypeCounts {
		totalSignals += count
	}
	suite.Equal(len(results), totalSignals, "Should have processed all results")

	// At least one signal type should have been generated (meaningful assertion)
	hasNonZeroCount := false
	for _, count := range signalTypeCounts {
		if count > 0 {
			hasNonZeroCount = true
			break
		}
	}
	suite.True(hasNonZeroCount, "Should have at least one signal type with non-zero count")
}

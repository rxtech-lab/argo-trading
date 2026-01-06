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

// BollingerBandsIntegrationTestSuite is an integration test suite for the Bollinger Bands indicator
type BollingerBandsIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	bb         *BollingerBands
	logger     *logger.Logger
}

// SetupSuite sets up the test suite
func (suite *BollingerBandsIntegrationTestSuite) SetupSuite() {
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

	// Initialize Bollinger Bands
	bb := NewBollingerBands()
	suite.bb = bb.(*BollingerBands)
	suite.Require().NotNil(suite.bb)

	// Initialize data source with parquet file
	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *BollingerBandsIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
	if suite.db != nil {
		suite.db.Close()
	}
}

// TestBollingerBandsIntegrationSuite runs the integration test suite
func TestBollingerBandsIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BollingerBandsIntegrationTestSuite))
}

// TestBollingerBandsGetSignal tests the GetSignal method of the Bollinger Bands indicator
func (suite *BollingerBandsIntegrationTestSuite) TestBollingerBandsGetSignal() {
	// Query data for testing - need enough data points for the lookback period
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 30 OFFSET 50
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource: suite.dataSource,
	}

	// Configure Bollinger Bands with period 20, 2 std devs, 24 hour lookback
	err = suite.bb.Config(20, 2.0, time.Hour*24)
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

		signal, err := suite.bb.GetSignal(marketData, ctx)
		suite.Require().NoError(err, "Failed to get Bollinger Bands signal")

		// Verify signal properties
		suite.Equal(marketData.Symbol, signal.Symbol)
		suite.Equal(marketData.Time, signal.Time)
		suite.Equal(types.IndicatorTypeBollingerBands, signal.Indicator)

		// Signal type can be BuyLong, SellLong, or NoAction
		suite.True(signal.Type == types.SignalTypeBuyLong ||
			signal.Type == types.SignalTypeSellLong ||
			signal.Type == types.SignalTypeNoAction,
			"Signal type should be BuyLong, SellLong, or NoAction")

		// If signal is not NoAction with empty reason (insufficient data), verify raw values
		if signal.RawValue != nil {
			rawValue, ok := signal.RawValue.(map[string]float64)
			suite.True(ok, "Failed to cast RawValue to map[string]float64")

			// Verify band values
			upper, upperExists := rawValue["upper"]
			middle, middleExists := rawValue["middle"]
			lower, lowerExists := rawValue["lower"]

			suite.True(upperExists, "Upper band not found in signal raw values")
			suite.True(middleExists, "Middle band not found in signal raw values")
			suite.True(lowerExists, "Lower band not found in signal raw values")

			// Verify band ordering
			suite.Greater(upper, middle, "Upper band should be greater than middle")
			suite.Greater(middle, lower, "Middle band should be greater than lower")
		}
	}
}

// TestBollingerBandsRawValueWithContext tests RawValue with a valid context
func (suite *BollingerBandsIntegrationTestSuite) TestBollingerBandsRawValueWithContext() {
	// Query data for testing
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 1 OFFSET 100
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource: suite.dataSource,
	}

	// Configure Bollinger Bands
	err = suite.bb.Config(10, 2.0, time.Hour*48)
	suite.Require().NoError(err)

	result := results[0]
	marketData := types.MarketData{
		Time:   result.Values["time"].(time.Time),
		Symbol: result.Values["symbol"].(string),
		Open:   result.Values["open"].(float64),
		High:   result.Values["high"].(float64),
		Low:    result.Values["low"].(float64),
		Close:  result.Values["close"].(float64),
		Volume: result.Values["volume"].(float64),
	}

	// Test RawValue - returns the middle band (SMA)
	middleBand, err := suite.bb.RawValue(marketData, ctx)
	suite.Require().NoError(err, "Failed to calculate Bollinger Bands middle band")
	suite.Greater(middleBand, 0.0, "Middle band value should be positive")
}

// TestBollingerBandsGetSignalInsufficientData tests GetSignal with insufficient data
func (suite *BollingerBandsIntegrationTestSuite) TestBollingerBandsGetSignalInsufficientData() {
	// Query the first data point (which won't have enough historical data)
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 1
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")
	suite.Require().NotEmpty(results, "No results returned for test query")

	// Create indicator context
	ctx := IndicatorContext{
		DataSource: suite.dataSource,
	}

	// Configure Bollinger Bands with a very short lookback
	err = suite.bb.Config(20, 2.0, time.Minute*5)
	suite.Require().NoError(err)

	result := results[0]
	marketData := types.MarketData{
		Time:   result.Values["time"].(time.Time),
		Symbol: result.Values["symbol"].(string),
		Open:   result.Values["open"].(float64),
		High:   result.Values["high"].(float64),
		Low:    result.Values["low"].(float64),
		Close:  result.Values["close"].(float64),
		Volume: result.Values["volume"].(float64),
	}

	// GetSignal should return NoAction for insufficient data
	signal, err := suite.bb.GetSignal(marketData, ctx)
	suite.Require().NoError(err, "GetSignal should not error on insufficient data")
	suite.Equal(types.SignalTypeNoAction, signal.Type)
}

// TestBollingerBandsGetSignalBuySignal tests that buy signal is generated when price is below lower band
func (suite *BollingerBandsIntegrationTestSuite) TestBollingerBandsSignalTypes() {
	// Create indicator context
	ctx := IndicatorContext{
		DataSource: suite.dataSource,
	}

	// Configure Bollinger Bands with a large lookback to get enough data
	err := suite.bb.Config(20, 2.0, time.Hour*24*7)
	suite.Require().NoError(err)

	// Query enough data points to find different signal types
	query := `
		SELECT time, symbol, open, high, low, close, volume
		FROM market_data
		WHERE symbol = 'AAPL'
		ORDER BY time ASC
		LIMIT 100 OFFSET 50
	`
	results, err := suite.dataSource.ExecuteSQL(query)
	suite.Require().NoError(err, "Failed to query test data")

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

		signal, err := suite.bb.GetSignal(marketData, ctx)
		suite.Require().NoError(err)
		signalTypeCounts[signal.Type]++
	}

	// We should see at least some NoAction signals (price within bands)
	suite.Greater(signalTypeCounts[types.SignalTypeNoAction], 0, "Should have at least some NoAction signals")
}

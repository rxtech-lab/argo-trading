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

// TestMACDNameAndConfig tests the Name and Config methods of MACD
func (suite *MACDTestSuite) TestMACDNameAndConfig() {
	macd := NewMACD()
	suite.NotNil(macd)

	// Test Name
	suite.Equal(types.IndicatorTypeMACD, macd.Name())

	// Test default values
	macdImpl := macd.(*MACD)
	suite.Equal(12, macdImpl.fastPeriod)
	suite.Equal(26, macdImpl.slowPeriod)
	suite.Equal(9, macdImpl.signalPeriod)
}

func (suite *MACDTestSuite) TestMACDConfigValid() {
	macd := NewMACD()
	macdImpl := macd.(*MACD)

	err := macd.Config(10, 20, 5)
	suite.NoError(err)
	suite.Equal(10, macdImpl.fastPeriod)
	suite.Equal(20, macdImpl.slowPeriod)
	suite.Equal(5, macdImpl.signalPeriod)
}

func (suite *MACDTestSuite) TestMACDConfigInvalidParamCount() {
	macd := NewMACD()

	// Too few params
	err := macd.Config(10, 20)
	suite.Error(err)
	suite.Contains(err.Error(), "expects 3 parameters")

	// Too many params
	err = macd.Config(10, 20, 5, 100)
	suite.Error(err)
}

func (suite *MACDTestSuite) TestMACDConfigInvalidFastPeriodType() {
	macd := NewMACD()
	err := macd.Config("invalid", 20, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for fastPeriod")
}

func (suite *MACDTestSuite) TestMACDConfigInvalidFastPeriodValue() {
	macd := NewMACD()
	err := macd.Config(0, 20, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "fastPeriod must be a positive integer")

	err = macd.Config(-5, 20, 5)
	suite.Error(err)
}

func (suite *MACDTestSuite) TestMACDConfigInvalidSlowPeriodType() {
	macd := NewMACD()
	err := macd.Config(10, "invalid", 5)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for slowPeriod")
}

func (suite *MACDTestSuite) TestMACDConfigInvalidSlowPeriodValue() {
	macd := NewMACD()
	err := macd.Config(10, 0, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "slowPeriod must be a positive integer")

	err = macd.Config(10, -20, 5)
	suite.Error(err)
}

func (suite *MACDTestSuite) TestMACDConfigInvalidSignalPeriodType() {
	macd := NewMACD()
	err := macd.Config(10, 20, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for signalPeriod")
}

func (suite *MACDTestSuite) TestMACDConfigInvalidSignalPeriodValue() {
	macd := NewMACD()
	err := macd.Config(10, 20, 0)
	suite.Error(err)
	suite.Contains(err.Error(), "signalPeriod must be a positive integer")

	err = macd.Config(10, 20, -5)
	suite.Error(err)
}

func (suite *MACDTestSuite) TestMACDRawValueInvalidParams() {
	macd := NewMACD()

	// Too few params
	_, err := macd.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = macd.RawValue("symbol")
	suite.Error(err)

	_, err = macd.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = macd.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	// Invalid second param type
	_, err = macd.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")

	// Invalid third param type
	_, err = macd.RawValue("symbol", time.Now(), "not-a-ctx")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
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

// TestMACDGetSignal tests the GetSignal method of the MACD indicator
func (suite *MACDTestSuite) TestMACDGetSignal() {
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

	// Configure MACD with default parameters
	err = suite.macd.Config(12, 26, 9)
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

		signal, err := suite.macd.GetSignal(marketData, ctx)
		suite.Require().NoError(err, "Failed to get MACD signal")

		// Verify signal properties
		suite.Equal(marketData.Symbol, signal.Symbol)
		suite.Equal(marketData.Time, signal.Time)
		suite.Equal(types.IndicatorTypeMACD, signal.Indicator)

		// Signal type can be BuyLong, SellShort, or NoAction
		suite.True(signal.Type == types.SignalTypeBuyLong ||
			signal.Type == types.SignalTypeSellShort ||
			signal.Type == types.SignalTypeNoAction,
			"Signal type should be BuyLong, SellShort, or NoAction")

		// Verify raw value contains MACD
		rawValue, ok := signal.RawValue.(map[string]float64)
		suite.True(ok, "Failed to cast RawValue to map[string]float64")
		_, exists := rawValue["macd"]
		suite.True(exists, "MACD value not found in signal raw values")
	}
}

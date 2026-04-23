package indicator

import (
	"database/sql"
	"math"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// WRIntegrationTestSuite is an integration test suite for the Williams %R
// indicator that exercises the indicator against the real DuckDB datasource
// backed by the checked-in parquet file.
type WRIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	wr         *WR
	logger     *logger.Logger
}

func (suite *WRIntegrationTestSuite) SetupSuite() {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, err := loggerConfig.Build()
	suite.Require().NoError(err)
	suite.logger = &logger.Logger{Logger: zapLogger}

	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	suite.db = db

	dataSource, err := datasource.NewDataSource(":memory:", suite.logger)
	suite.Require().NoError(err)
	suite.dataSource = dataSource

	wr := NewWR()
	suite.wr = wr.(*WR)
	suite.Require().NotNil(suite.wr)

	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

func (suite *WRIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}

	if suite.db != nil {
		suite.db.Close()
	}
}

func TestWRIntegrationSuite(t *testing.T) {
	suite.Run(t, new(WRIntegrationTestSuite))
}

// TestWRCalculation exercises Williams %R over a range of market data points
// from the parquet dataset, validating the implementation by recomputing the
// expected value from the same historical window and comparing.
func (suite *WRIntegrationTestSuite) TestWRCalculation() {
	testCases := []struct {
		name   string
		symbol string
		period int
	}{
		{name: "WR period 7", symbol: "AAPL", period: 7},
		{name: "WR period 14", symbol: "AAPL", period: 14},
		{name: "WR period 21", symbol: "AAPL", period: 21},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			query := `
				SELECT time, symbol, open, high, low, close, volume
				FROM market_data
				WHERE symbol = ?
				ORDER BY time ASC
				LIMIT 50 OFFSET 100
			`
			results, err := suite.dataSource.ExecuteSQL(query, tc.symbol)
			suite.Require().NoError(err, "Failed to query market data")
			suite.Require().NotEmpty(results)

			ctx := IndicatorContext{DataSource: suite.dataSource}
			suite.Require().NoError(suite.wr.Config(tc.period))

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

				calculated, err := suite.wr.RawValue(marketData.Symbol, marketData.Time, ctx)
				suite.Require().NoError(err)

				// %R must be in the range [-100, 0].
				suite.Assert().GreaterOrEqual(calculated, -100.0)
				suite.Assert().LessOrEqual(calculated, 0.0)

				// Independently recompute the expected %R from the same historical window.
				history, err := suite.dataSource.GetPreviousNumberOfDataPoints(marketData.Time, marketData.Symbol, tc.period)
				suite.Require().NoError(err)
				suite.Require().Len(history, tc.period)

				high := history[0].High
				low := history[0].Low

				for _, bar := range history {
					if bar.High > high {
						high = bar.High
					}

					if bar.Low < low {
						low = bar.Low
					}
				}

				currentClose := history[len(history)-1].Close
				denom := high - low
				expected := 0.0

				if denom != 0 {
					expected = ((high - currentClose) / denom) * -100
				}

				suite.Assert().LessOrEqual(math.Abs(expected-calculated), 1e-9,
					"WR mismatch: expected %f got %f", expected, calculated)

				// Verify signal is consistent with the computed value and thresholds.
				signal, err := suite.wr.GetSignal(marketData, ctx)
				suite.Require().NoError(err)
				suite.Equal(marketData.Symbol, signal.Symbol)
				suite.Equal(marketData.Time, signal.Time)
				suite.Equal(types.IndicatorTypeWilliamsR, signal.Indicator)

				raw, ok := signal.RawValue.(map[string]float64)
				suite.Require().True(ok)
				suite.Assert().Equal(calculated, raw["wr"])

				switch {
				case calculated < suite.wr.oversoldThreshold:
					suite.Assert().Equal(types.SignalTypeBuyLong, signal.Type)
				case calculated > suite.wr.overboughtThreshold:
					suite.Assert().Equal(types.SignalTypeSellShort, signal.Type)
				default:
					suite.Assert().Equal(types.SignalTypeNoAction, signal.Type)
				}
			}
		})
	}
}

// TestWRRawValueInvalidContext verifies that an incorrect context type is
// rejected with a clear error.
func (suite *WRIntegrationTestSuite) TestWRRawValueInvalidContext() {
	wr := NewWR()
	_, err := wr.RawValue("AAPL", time.Now(), "invalid-context")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
}

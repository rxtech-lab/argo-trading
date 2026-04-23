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

// PSYIntegrationTestSuite is an integration test suite for the Psychological
// Line indicator that exercises it against the real DuckDB datasource.
type PSYIntegrationTestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	psy        *PSY
	logger     *logger.Logger
}

func (suite *PSYIntegrationTestSuite) SetupSuite() {
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

	psy := NewPSY()
	suite.psy = psy.(*PSY)
	suite.Require().NotNil(suite.psy)

	testDataPath := "./test_data/test_data.parquet"
	err = suite.dataSource.Initialize(testDataPath)
	suite.Require().NoError(err)
}

func (suite *PSYIntegrationTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}

	if suite.db != nil {
		suite.db.Close()
	}
}

func TestPSYIntegrationSuite(t *testing.T) {
	suite.Run(t, new(PSYIntegrationTestSuite))
}

// TestPSYCalculation exercises PSY over a range of market data points and
// verifies the implementation by recomputing the expected value from the same
// historical window.
func (suite *PSYIntegrationTestSuite) TestPSYCalculation() {
	testCases := []struct {
		name   string
		symbol string
		period int
	}{
		{name: "PSY period 6", symbol: "AAPL", period: 6},
		{name: "PSY period 12", symbol: "AAPL", period: 12},
		{name: "PSY period 24", symbol: "AAPL", period: 24},
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
			suite.Require().NoError(suite.psy.Config(tc.period))

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

				calculated, err := suite.psy.RawValue(marketData.Symbol, marketData.Time, ctx)
				suite.Require().NoError(err)

				// PSY must be in the range [0, 100].
				suite.Assert().GreaterOrEqual(calculated, 0.0)
				suite.Assert().LessOrEqual(calculated, 100.0)

				// Independently recompute the expected PSY.
				history, err := suite.dataSource.GetPreviousNumberOfDataPoints(marketData.Time, marketData.Symbol, tc.period+1)
				suite.Require().NoError(err)
				suite.Require().Len(history, tc.period+1)

				upCount := 0

				for i := 1; i < len(history); i++ {
					if history[i].Close > history[i-1].Close {
						upCount++
					}
				}

				expected := (float64(upCount) / float64(tc.period)) * 100

				suite.Assert().LessOrEqual(math.Abs(expected-calculated), 1e-9,
					"PSY mismatch: expected %f got %f", expected, calculated)

				// Verify signal type matches thresholds.
				signal, err := suite.psy.GetSignal(marketData, ctx)
				suite.Require().NoError(err)
				suite.Equal(marketData.Symbol, signal.Symbol)
				suite.Equal(marketData.Time, signal.Time)
				suite.Equal(types.IndicatorTypePSY, signal.Indicator)

				raw, ok := signal.RawValue.(map[string]float64)
				suite.Require().True(ok)
				suite.Assert().Equal(calculated, raw["psy"])

				switch {
				case calculated < suite.psy.lowerThreshold:
					suite.Assert().Equal(types.SignalTypeBuyLong, signal.Type)
				case calculated > suite.psy.upperThreshold:
					suite.Assert().Equal(types.SignalTypeSellShort, signal.Type)
				default:
					suite.Assert().Equal(types.SignalTypeNoAction, signal.Type)
				}
			}
		})
	}
}

// TestPSYRawValueInvalidContext verifies that an incorrect context type is
// rejected with a clear error.
func (suite *PSYIntegrationTestSuite) TestPSYRawValueInvalidContext() {
	psy := NewPSY()
	_, err := psy.RawValue("AAPL", time.Now(), "invalid-context")
	suite.Error(err)
	suite.Contains(err.Error(), "third parameter must be of type IndicatorContext")
}

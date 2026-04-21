package engine

import (
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// MonthlyStatsTestSuite covers the monthly breakdowns and median/percentile
// statistics added to the backtest stats output.
type MonthlyStatsTestSuite struct {
	suite.Suite
	state  *BacktestState
	logger *logger.Logger
}

func TestMonthlyStatsSuite(t *testing.T) {
	suite.Run(t, new(MonthlyStatsTestSuite))
}

func (suite *MonthlyStatsTestSuite) SetupSuite() {
	lg, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = lg

	st, err := NewBacktestState(suite.logger)
	suite.Require().NoError(err)
	suite.state = st
}

func (suite *MonthlyStatsTestSuite) TearDownSuite() {
	if suite.state != nil && suite.state.db != nil {
		suite.state.db.Close()
	}
}

func (suite *MonthlyStatsTestSuite) SetupTest() {
	suite.Require().NoError(suite.state.Initialize())
}

func (suite *MonthlyStatsTestSuite) TearDownTest() {
	suite.Require().NoError(suite.state.Cleanup())
}

// makeLongOrder is a small helper to build a long-side order with sane defaults.
func makeLongOrder(symbol string, side types.PurchaseType, qty, price float64, ts time.Time) types.Order {
	return types.Order{
		Symbol:       symbol,
		Side:         side,
		PositionType: types.PositionTypeLong,
		Quantity:     qty,
		Price:        price,
		Timestamp:    ts,
		IsCompleted:  true,
		StrategyName: "test",
		Reason:       types.Reason{Reason: "test", Message: "test"},
	}
}

// TestMonthlyAndPercentileStats exercises monthly trade counts, monthly
// balance evolution, monthly holding time, plus median/percentile fields on
// TradeHoldingTime and TradePnl. Three round-trip trades are issued across
// two different months so that both monthly buckets and percentile distributions
// are populated.
func (suite *MonthlyStatsTestSuite) TestMonthlyAndPercentileStats() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	// End time used for open-position holding time; no open positions in this test.
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  120.0,
		Time:   time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	suite.state.SetInitialBalance(100_000)

	// Three round-trip trades:
	//   January: buy@100 (10:00), sell@110 (11:00) -> +1000 PnL, hold 3600s
	//   January: buy@100 (Jan 10), sell@90  (Jan 10 +2h) -> -1000 PnL, hold 7200s
	//   February: buy@100 (Feb 1 10:00), sell@120 (Feb 1 11:00) -> +2000 PnL, hold 3600s
	orders := []types.Order{
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 110, time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 10, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 90, time.Date(2024, 1, 10, 12, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 2, 1, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 120, time.Date(2024, 2, 1, 11, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "test-run-id", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	stat := stats[0]

	// --- Monthly trades ---
	suite.Require().Len(stat.MonthlyTrades, 2, "expected two months of trade activity")

	jan := stat.MonthlyTrades[0]
	suite.Equal("2024-01", jan.Month)
	suite.Equal(4, jan.NumberOfTrades, "January should have 4 fills (2 buys + 2 sells)")
	suite.Equal(2, jan.NumberOfTradingPairs, "January should have 2 closing trades")
	suite.Equal(1, jan.NumberOfWinningTrades)
	suite.Equal(1, jan.NumberOfLosingTrades)

	feb := stat.MonthlyTrades[1]
	suite.Equal("2024-02", feb.Month)
	suite.Equal(2, feb.NumberOfTrades)
	suite.Equal(1, feb.NumberOfTradingPairs)
	suite.Equal(1, feb.NumberOfWinningTrades)
	suite.Equal(0, feb.NumberOfLosingTrades)

	// --- Monthly balance ---
	suite.Require().Len(stat.MonthlyBalance, 2)

	janBal := stat.MonthlyBalance[0]
	suite.Equal("2024-01", janBal.Month)
	suite.Equal(100_000.0, janBal.StartingBalance, "first month should start at initial balance")
	// End of January realized PnL = +1000 - 1000 = 0 -> ending balance = initial balance.
	suite.Equal(100_000.0, janBal.EndingBalance)
	suite.Equal(0.0, janBal.Change)
	suite.Equal(0.0, janBal.RealizedPnL)

	febBal := stat.MonthlyBalance[1]
	suite.Equal("2024-02", febBal.Month)
	// February starts where January ended.
	suite.Equal(janBal.EndingBalance, febBal.StartingBalance)
	// February books +2000 of realized PnL on top of the cumulative 0.
	suite.Equal(102_000.0, febBal.EndingBalance)
	suite.Equal(2000.0, febBal.Change)
	suite.Equal(2000.0, febBal.RealizedPnL)

	// Final balance should match the last monthly ending balance.
	suite.Equal(febBal.EndingBalance, stat.FinalBalance)

	// --- Monthly holding time ---
	suite.Require().Len(stat.MonthlyHoldingTime, 2)

	janHold := stat.MonthlyHoldingTime[0]
	suite.Equal("2024-01", janHold.Month)
	suite.Equal(3600, janHold.Min)
	suite.Equal(7200, janHold.Max)
	suite.Equal(5400, janHold.Avg)
	// Median of two values is the average of them for quantile_cont.
	suite.Equal(5400, janHold.Median)

	febHold := stat.MonthlyHoldingTime[1]
	suite.Equal("2024-02", febHold.Month)
	suite.Equal(3600, febHold.Min)
	suite.Equal(3600, febHold.Max)
	suite.Equal(3600, febHold.Avg)
	suite.Equal(3600, febHold.Median)

	// --- Aggregated trade holding time median + percentiles ---
	// Closing trade durations across all months: [3600, 7200, 3600] seconds.
	suite.Equal(3600, stat.TradeHoldingTime.Median, "overall median holding time")
	// quantile_cont over sorted [3600, 3600, 7200]: p25=3600, p50=3600, p75=5400.
	suite.InDelta(3600.0, stat.TradeHoldingTime.Percentiles.P25, 0.001)
	suite.InDelta(3600.0, stat.TradeHoldingTime.Percentiles.P50, 0.001)
	suite.InDelta(5400.0, stat.TradeHoldingTime.Percentiles.P75, 0.001)

	// --- PnL median + percentiles ---
	// Closing trade PnLs: [+1000, -1000, +2000]. Sorted: [-1000, 1000, 2000].
	suite.Equal(1000.0, stat.TradePnl.MedianPnL)
	suite.InDelta(0.0, stat.TradePnl.Percentiles.P25, 0.001)
	suite.InDelta(1000.0, stat.TradePnl.Percentiles.P50, 0.001)
	suite.InDelta(1500.0, stat.TradePnl.Percentiles.P75, 0.001)
}

// TestPnLPercentage verifies that PnLPercentage is computed as TotalPnL /
// TotalInvestment (the gross capital deployed across BUY entries) and not
// against the initial cash balance. Three buys of 100 shares at $100 each
// (one of which closes profitably, one at a loss, one open) deploy a gross
// investment of $30,000, against which TotalPnL is measured.
func (suite *MonthlyStatsTestSuite) TestPnLPercentage() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  100.0, // last price equals entry price -> open position has 0 unrealized PnL
		Time:   time.Date(2024, 2, 15, 12, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	// Use an initial balance much larger than the investment to make sure the
	// percentage is NOT being computed against the balance.
	suite.state.SetInitialBalance(1_000_000)

	orders := []types.Order{
		// Round trip 1: buy 100 @ 100, sell 100 @ 110 -> +1000
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 110, time.Date(2024, 1, 5, 11, 0, 0, 0, time.UTC)),
		// Round trip 2: buy 100 @ 100, sell 100 @ 95 -> -500
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 95, time.Date(2024, 1, 6, 11, 0, 0, 0, time.UTC)),
		// Open: buy 100 @ 100 -> unrealized 0 (close = 100)
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "test-run-id", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	stat := stats[0]

	// Total gross investment = 3 buys * 100 shares * $100 = $30,000.
	suite.Equal(30_000.0, stat.TradePnl.TotalInvestment)

	// Realized PnL across the two closed round trips = +1000 - 500 = +500.
	// Unrealized = 0 (last price equals entry). Total PnL = +500.
	suite.Equal(500.0, stat.TradePnl.TotalPnL)

	// PnL % is over investment (500 / 30000), NOT over initial balance.
	suite.InDelta(500.0/30_000.0, stat.TradePnl.PnLPercentage, 1e-9)
	// Sanity: must NOT be the value you'd get if dividing by initial balance.
	suite.NotEqual(500.0/1_000_000.0, stat.TradePnl.PnLPercentage,
		"pnl_percentage must be over total investment, not initial balance")
}

// TestPnLPercentageNoInvestment verifies the divide-by-zero guard: when there
// are no entry trades the percentage stays at zero rather than NaN/Inf.
func (suite *MonthlyStatsTestSuite) TestPnLPercentageNoInvestment() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().GetAllSymbols().Return([]string{"SPY"}, nil).AnyTimes()
	mockSource.EXPECT().ReadLastData("SPY").Return(types.MarketData{
		Symbol: "SPY",
		Close:  450.0,
		Time:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "test-run-id", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	suite.Equal(0.0, stats[0].TradePnl.TotalInvestment)
	suite.Equal(0.0, stats[0].TradePnl.PnLPercentage)
}

// monthly slices and zero-valued median/percentile fields.
func (suite *MonthlyStatsTestSuite) TestMonthlyStatsNoTrades() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().GetAllSymbols().Return([]string{"SPY"}, nil).AnyTimes()
	mockSource.EXPECT().ReadLastData("SPY").Return(types.MarketData{
		Symbol: "SPY",
		Close:  450.0,
		Time:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "test-run-id", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	stat := stats[0]
	suite.Empty(stat.MonthlyTrades)
	suite.Empty(stat.MonthlyBalance)
	suite.Empty(stat.MonthlyHoldingTime)
	suite.Equal(0, stat.TradeHoldingTime.Median)
	suite.Equal(types.Percentiles{}, stat.TradeHoldingTime.Percentiles)
	suite.Equal(0.0, stat.TradePnl.MedianPnL)
	suite.Equal(types.Percentiles{}, stat.TradePnl.Percentiles)
}

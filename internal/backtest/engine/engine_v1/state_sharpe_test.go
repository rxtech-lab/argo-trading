package engine

import (
	"math"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type SharpeRatioTestSuite struct {
	suite.Suite
	state  *BacktestState
	logger *logger.Logger
}

func TestSharpeRatioSuite(t *testing.T) {
	suite.Run(t, new(SharpeRatioTestSuite))
}

func (suite *SharpeRatioTestSuite) SetupSuite() {
	lg, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = lg

	st, err := NewBacktestState(suite.logger)
	suite.Require().NoError(err)
	suite.state = st
}

func (suite *SharpeRatioTestSuite) TearDownSuite() {
	if suite.state != nil && suite.state.db != nil {
		suite.state.db.Close()
	}
}

func (suite *SharpeRatioTestSuite) SetupTest() {
	suite.Require().NoError(suite.state.Initialize())
}

func (suite *SharpeRatioTestSuite) TearDownTest() {
	suite.Require().NoError(suite.state.Cleanup())
}

// expectedSharpe replicates calculateSharpeRatio's formula in plain Go so the
// hand-computed expectation tracks any intentional formula change and keeps
// the test readable.
func expectedSharpe(equities []float64, riskFreeRate float64, annualization int) float64 {
	if len(equities) < 2 {
		return 0
	}

	returns := make([]float64, 0, len(equities)-1)

	for i := 1; i < len(equities); i++ {
		prev := equities[i-1]
		if prev == 0 {
			continue
		}

		returns = append(returns, equities[i]/prev-1)
	}

	if len(returns) < 2 {
		return 0
	}

	var sum float64
	for _, r := range returns {
		sum += r
	}

	mean := sum / float64(len(returns))

	var sqSum float64
	for _, r := range returns {
		diff := r - mean
		sqSum += diff * diff
	}

	variance := sqSum / float64(len(returns)-1)
	if variance <= 0 {
		return 0
	}

	stdev := math.Sqrt(variance)

	var periodRiskFree float64
	if annualization > 0 {
		periodRiskFree = riskFreeRate / float64(annualization)
	}

	sharpe := (mean - periodRiskFree) / stdev
	if annualization > 0 {
		sharpe *= math.Sqrt(float64(annualization))
	}

	return sharpe
}

// TestSharpeRatioMultipleDays verifies that the Sharpe ratio is computed from
// daily equity returns across multiple trading days using the default 252
// annualization factor and a zero risk-free rate.
func (suite *SharpeRatioTestSuite) TestSharpeRatioMultipleDays() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  100.0,
		Time:   time.Date(2024, 1, 10, 16, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	initialBalance := 100_000.0
	suite.state.SetInitialBalance(initialBalance)
	suite.state.SetRiskFreeRate(0)
	suite.state.SetSharpeAnnualizationFactor(252)

	// Three trading days, each with a completed round trip so end-of-day
	// cumulative PnL is well-defined.
	//   Day 1 (Jan 5): +100 PnL (buy@100, sell@101)
	//   Day 2 (Jan 6): +200 PnL (buy@100, sell@102) -> cumulative +300
	//   Day 3 (Jan 7): -100 PnL (buy@100, sell@99)  -> cumulative +200
	orders := []types.Order{
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 101, time.Date(2024, 1, 5, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 102, time.Date(2024, 1, 6, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 99, time.Date(2024, 1, 7, 15, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "run-sharpe", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	equities := []float64{
		initialBalance + 100,
		initialBalance + 300,
		initialBalance + 200,
	}
	expected := expectedSharpe(equities, 0, 252)

	suite.InDelta(expected, stats[0].TradeResult.SharpeRatio, 1e-9)
	suite.NotEqual(0.0, stats[0].TradeResult.SharpeRatio, "Sharpe should be populated when there are multiple trading days")
}

// TestSharpeRatioSingleDayIsZero verifies that with only one trading day (and
// therefore at most one daily return observation), the Sharpe ratio is
// reported as 0 rather than NaN/Inf.
func (suite *SharpeRatioTestSuite) TestSharpeRatioSingleDayIsZero() {
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  100.0,
		Time:   time.Date(2024, 1, 5, 16, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	suite.state.SetInitialBalance(100_000)
	suite.state.SetRiskFreeRate(0)
	suite.state.SetSharpeAnnualizationFactor(252)

	orders := []types.Order{
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 101, time.Date(2024, 1, 5, 15, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "run-sharpe-single", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	suite.Equal(0.0, stats[0].TradeResult.SharpeRatio)
}

// TestSharpeRatioRiskFreeRateReducesRatio verifies that raising the risk-free
// rate decreases the reported Sharpe ratio for a profitable equity curve.
func (suite *SharpeRatioTestSuite) TestSharpeRatioRiskFreeRateReducesRatio() {
	initialBalance := 100_000.0
	equities := []float64{
		initialBalance + 100,
		initialBalance + 300,
		initialBalance + 200,
	}
	zeroRf := expectedSharpe(equities, 0, 252)
	highRf := expectedSharpe(equities, 0.10, 252)

	suite.Less(highRf, zeroRf, "higher risk-free rate should reduce Sharpe on profitable equity curve")

	// And exercise the engine end-to-end to confirm the state plumbing.
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  100.0,
		Time:   time.Date(2024, 1, 10, 16, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	suite.state.SetInitialBalance(initialBalance)
	suite.state.SetRiskFreeRate(0.10)
	suite.state.SetSharpeAnnualizationFactor(252)

	orders := []types.Order{
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 101, time.Date(2024, 1, 5, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 102, time.Date(2024, 1, 6, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 99, time.Date(2024, 1, 7, 15, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "run-sharpe-rf", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	suite.InDelta(highRf, stats[0].TradeResult.SharpeRatio, 1e-9)
}

// TestSharpeRatioNoAnnualization verifies that setting the annualization factor
// to a negative value (via SetSharpeAnnualizationFactor) disables annualization
// and excess-return adjustment, returning the raw mean/stdev ratio.
func (suite *SharpeRatioTestSuite) TestSharpeRatioNoAnnualization() {
	initialBalance := 100_000.0
	suite.state.SetInitialBalance(initialBalance)
	suite.state.SetRiskFreeRate(0.10) // should be ignored when annualization is 0
	suite.state.SetSharpeAnnualizationFactor(-1)

	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	mockSource := mocks.NewMockDataSource(ctrl)
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  100.0,
		Time:   time.Date(2024, 1, 10, 16, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()
	mockSource.EXPECT().GetAllSymbols().Return([]string{"AAPL"}, nil).AnyTimes()

	orders := []types.Order{
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 101, time.Date(2024, 1, 5, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 6, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 102, time.Date(2024, 1, 6, 15, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeBuy, 100, 100, time.Date(2024, 1, 7, 10, 0, 0, 0, time.UTC)),
		makeLongOrder("AAPL", types.PurchaseTypeSell, 100, 99, time.Date(2024, 1, 7, 15, 0, 0, 0, time.UTC)),
	}
	for _, o := range orders {
		_, err := suite.state.Update([]types.Order{o})
		suite.Require().NoError(err)
	}

	mockStrategy := &testMockStrategyRuntime{}
	stats, err := suite.state.GetStats(runtime.RuntimeContext{DataSource: mockSource},
		mockStrategy, "run-sharpe-noann", "", "", "", "", "", "")
	suite.Require().NoError(err)
	suite.Require().Len(stats, 1)

	equities := []float64{
		initialBalance + 100,
		initialBalance + 300,
		initialBalance + 200,
	}
	// Pass 0 to expectedSharpe: both the engine (via -1 -> 0) and the helper
	// treat zero annualization identically (raw mean/stdev, no rf adjustment).
	expected := expectedSharpe(equities, 0, 0)

	suite.InDelta(expected, stats[0].TradeResult.SharpeRatio, 1e-9)
}

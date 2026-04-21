package engine

import (
	"math"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// withAverageCostStrategy temporarily switches the suite state to use
// average-cost PnL for the scope of fn, restoring FIFO afterwards so other
// tests (which assume FIFO at the state level) are unaffected.
func (suite *BacktestStateTestSuite) withAverageCostStrategy(fn func()) {
	prev := suite.state.PortfolioCalculationStrategy()
	suite.state.SetPortfolioCalculationStrategy(PortfolioCalculationAverageCost)

	defer suite.state.SetPortfolioCalculationStrategy(prev)

	fn()
}

func approxEqual(a, b float64) bool {
	return math.Abs(a-b) < 1e-6
}

// TestAverageCostPnL_LongPosition tests average-cost individual PnL calculations
// for long positions. Unlike FIFO, average-cost uses the running weighted
// average cost basis of the currently-open position.
func (suite *BacktestStateTestSuite) TestAverageCostPnL_LongPosition() {
	tests := []struct {
		name           string
		orders         []types.Order
		expectedPnL    []float64 // Expected per-trade PnL
		expectedCumPnL []float64 // Expected cumulative PnL
	}{
		{
			name: "Single buy - no PnL",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy"},
				},
			},
			expectedPnL:    []float64{0},
			expectedCumPnL: []float64{0},
		},
		{
			name: "Single buy-sell pair - equal to FIFO",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 110.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell"},
				},
			},
			// avg_cost = (100*100 + 1)/100 = 100.01
			// PnL = 110*100 - 1 - 100.01*100 = 11000 - 1 - 10001 = 998
			expectedPnL:    []float64{0, 998},
			expectedCumPnL: []float64{0, 998},
		},
		{
			name: "Two buys at different prices then full sell - average cost basis",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 50.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 150.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell2"},
				},
			},
			// Cost basis after both buys = 50*100 + 1 + 150*100 + 1 = 20002 over 200 qty
			// avg_cost_per_unit = 100.01
			// sell1 (100 @ 120): PnL = 120*100 - 1 - 100.01*100 = 12000 - 1 - 10001 = 1998
			// Avg unchanged by sell. Remaining basis = 100.01 * 100 = 10001 over 100 qty.
			// sell2 (100 @ 120): PnL = 120*100 - 1 - 100.01*100 = 1998
			// Both sells have identical PnL under average cost, unlike FIFO.
			expectedPnL:    []float64{0, 0, 1998, 1998},
			expectedCumPnL: []float64{0, 0, 1998, 3996},
		},
		{
			name: "Partial sell crosses multiple buys - one average basis",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 50, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 50, Price: 200.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 70, Price: 160.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
				},
			},
			// Cost basis = (100*50 + 1) + (200*50 + 1) = 15002 over 100 qty
			// avg_cost_per_unit = 150.02
			// sell (70 @ 160): PnL = 160*70 - 1 - 150.02*70 = 11200 - 1 - 10501.4 = 697.6
			expectedPnL:    []float64{0, 0, 697.6},
			expectedCumPnL: []float64{0, 0, 697.6},
		},
		{
			name: "Two separate round-trips (position closes then reopens)",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 110.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 105.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 115.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell2"},
				},
			},
			// Round-trip 1: avg = 100.01, PnL = 110*100 - 1 - 10001 = 998
			// Position flat -> basis resets.
			// Round-trip 2: avg = 105.01, PnL = 115*100 - 1 - 10501 = 998
			expectedPnL:    []float64{0, 998, 0, 998},
			expectedCumPnL: []float64{0, 998, 998, 1996},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.withAverageCostStrategy(func() {
				err := suite.state.Cleanup()
				suite.Require().NoError(err)

				for _, order := range tc.orders {
					_, err := suite.state.Update([]types.Order{order})
					suite.Require().NoError(err)
				}

				trades, err := suite.state.GetAllTrades()
				suite.Require().NoError(err)
				suite.Require().Equal(len(tc.expectedPnL), len(trades), "Number of trades mismatch")

				for i, trade := range trades {
					suite.Assert().True(approxEqual(tc.expectedPnL[i], trade.PnL),
						"Average-cost PnL mismatch at trade %d: expected %v got %v", i, tc.expectedPnL[i], trade.PnL)
					suite.Assert().True(approxEqual(tc.expectedCumPnL[i], trade.CumulativePnL),
						"Cumulative PnL mismatch at trade %d: expected %v got %v", i, tc.expectedCumPnL[i], trade.CumulativePnL)
				}
			})
		})
	}
}

// TestAverageCostPnL_ShortPosition tests average-cost individual PnL for short
// positions (BUY entries, SELL exits).
func (suite *BacktestStateTestSuite) TestAverageCostPnL_ShortPosition() {
	tests := []struct {
		name           string
		orders         []types.Order
		expectedPnL    []float64
		expectedCumPnL []float64
	}{
		{
			name: "Single short entry and exit",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "open"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 90.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "close"},
				},
			},
			// entry_value = 100*100 - 1 = 9999 over 100 -> avg = 99.99
			// PnL = 99.99*100 - 90*100 - 1 = 9999 - 9000 - 1 = 998
			expectedPnL:    []float64{0, 998},
			expectedCumPnL: []float64{0, 998},
		},
		{
			name: "Two short entries at different prices - average basis",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 200.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "open1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "open2"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "close1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "close2"},
				},
			},
			// entry basis = (200*100 - 1) + (100*100 - 1) = 29998 over 200 -> avg = 149.99
			// close1 (100 @ 120): PnL = 149.99*100 - 120*100 - 1 = 14999 - 12000 - 1 = 2998
			// avg unchanged. close2 same: PnL = 2998
			expectedPnL:    []float64{0, 0, 2998, 2998},
			expectedCumPnL: []float64{0, 0, 2998, 5996},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.withAverageCostStrategy(func() {
				err := suite.state.Cleanup()
				suite.Require().NoError(err)

				for _, order := range tc.orders {
					_, err := suite.state.Update([]types.Order{order})
					suite.Require().NoError(err)
				}

				trades, err := suite.state.GetAllTrades()
				suite.Require().NoError(err)
				suite.Require().Equal(len(tc.expectedPnL), len(trades), "Number of trades mismatch")

				for i, trade := range trades {
					suite.Assert().True(approxEqual(tc.expectedPnL[i], trade.PnL),
						"Average-cost PnL mismatch at trade %d: expected %v got %v", i, tc.expectedPnL[i], trade.PnL)
					suite.Assert().True(approxEqual(tc.expectedCumPnL[i], trade.CumulativePnL),
						"Cumulative PnL mismatch at trade %d: expected %v got %v", i, tc.expectedCumPnL[i], trade.CumulativePnL)
				}
			})
		})
	}
}

// TestAverageCostPnL_TotalEqualsFIFO verifies that over a fully-closed sequence
// total PnL is identical for FIFO and average-cost strategies (only per-trade
// allocation differs).
func (suite *BacktestStateTestSuite) TestAverageCostPnL_TotalEqualsFIFO() {
	orders := []types.Order{
		{
			Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 50.0,
			Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			IsCompleted: true, PositionType: types.PositionTypeLong,
			StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
		},
		{
			Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 150.0,
			Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
			IsCompleted: true, PositionType: types.PositionTypeLong,
			StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
		},
		{
			Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
			Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			IsCompleted: true, PositionType: types.PositionTypeLong,
			StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
		},
		{
			Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
			Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
			IsCompleted: true, PositionType: types.PositionTypeLong,
			StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell2"},
		},
	}

	runTotals := func(strategy PortfolioCalculationStrategy) float64 {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)

		prev := suite.state.PortfolioCalculationStrategy()
		suite.state.SetPortfolioCalculationStrategy(strategy)
		defer suite.state.SetPortfolioCalculationStrategy(prev)

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		trades, err := suite.state.GetAllTrades()
		suite.Require().NoError(err)

		var total float64
		for _, t := range trades {
			total += t.PnL
		}

		return total
	}

	fifoTotal := runTotals(PortfolioCalculationFIFO)
	avgTotal := runTotals(PortfolioCalculationAverageCost)

	suite.Assert().True(approxEqual(fifoTotal, avgTotal),
		"Total PnL should be equal under FIFO (%v) and AverageCost (%v) for a fully-closed sequence", fifoTotal, avgTotal)
}

// TestPortfolioCalculationStrategy_DefaultAndSetter verifies the default
// strategy on a fresh BacktestState and that the setter coerces unknown values
// to AverageCost.
func (suite *BacktestStateTestSuite) TestPortfolioCalculationStrategy_DefaultAndSetter() {
	state, err := NewBacktestState(suite.logger)
	suite.Require().NoError(err)
	defer state.db.Close()

	// Fresh state defaults to FIFO so that existing state-level tests behave
	// unchanged; the engine Initialize method overrides this from config.
	suite.Assert().Equal(PortfolioCalculationFIFO, state.PortfolioCalculationStrategy())

	state.SetPortfolioCalculationStrategy(PortfolioCalculationAverageCost)
	suite.Assert().Equal(PortfolioCalculationAverageCost, state.PortfolioCalculationStrategy())

	state.SetPortfolioCalculationStrategy("garbage")
	suite.Assert().Equal(PortfolioCalculationAverageCost, state.PortfolioCalculationStrategy(),
		"Unknown strategy should coerce to average_cost")

	state.SetPortfolioCalculationStrategy(PortfolioCalculationFIFO)
	suite.Assert().Equal(PortfolioCalculationFIFO, state.PortfolioCalculationStrategy())
}

package engine

import (
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// TestHoldTime_LongPosition verifies the FIFO-weighted-average hold time on closing trades.
func (suite *BacktestStateTestSuite) TestHoldTime_LongPosition() {
	tests := []struct {
		name             string
		orders           []types.Order
		expectedHoldTime []int // expected HoldTime (seconds) per trade in order
	}{
		{
			name: "Buy then sell after 1 hour",
			orders: []types.Order{
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy"},
				},
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell"},
				},
			},
			expectedHoldTime: []int{0, 3600},
		},
		{
			name: "Two buys then one sell consumes from oldest first (FIFO)",
			orders: []types.Order{
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					// Sells 150: 100 from buy1 (held 7200s) + 50 from buy2 (held 3600s)
					// Weighted avg = (7200*100 + 3600*50)/150 = 6000
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     150,
					Price:        115.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell"},
				},
			},
			expectedHoldTime: []int{0, 0, 6000},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			for _, order := range tc.orders {
				_, err := suite.state.Update([]types.Order{order})
				suite.Require().NoError(err)
			}

			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)
			suite.Require().Equal(len(tc.expectedHoldTime), len(trades), "Number of trades mismatch")

			for i, trade := range trades {
				suite.Assert().Equal(tc.expectedHoldTime[i], trade.HoldTime, "HoldTime mismatch at trade %d", i)
			}
		})
	}
}

// TestHoldTime_LongPosition_AverageCostLIFO verifies that under the average-cost
// portfolio calculation strategy, per-trade HoldTime is computed with LIFO lot
// attribution (newest entries are consumed first).
func (suite *BacktestStateTestSuite) TestHoldTime_LongPosition_AverageCostLIFO() {
	tests := []struct {
		name             string
		orders           []types.Order
		expectedHoldTime []int
	}{
		{
			name: "Single buy then sell - identical to FIFO",
			orders: []types.Order{
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy"},
				},
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell"},
				},
			},
			expectedHoldTime: []int{0, 3600},
		},
		{
			name: "Two buys then one sell consumes newest first (LIFO)",
			orders: []types.Order{
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					// Sells 150 at t=12:00: LIFO consumes 100 from buy2 (held 3600s)
					// and 50 from buy1 (held 7200s). Weighted avg = (3600*100 + 7200*50)/150 = 4800.
					// (FIFO would produce 6000.)
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     150,
					Price:        115.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell"},
				},
			},
			expectedHoldTime: []int{0, 0, 4800},
		},
		{
			name: "Partial exit then full exit - LIFO stack replay",
			orders: []types.Order{
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					// sell1 at 12:00 consumes 50 of buy2 (held 3600s) -> HoldTime = 3600
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     50,
					Price:        115.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					// sell2 at 13:00 consumes remaining 50 of buy2 (held 7200s)
					// + 100 of buy1 (held 10800s). Weighted = (7200*50 + 10800*100)/150 = 9600.
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     150,
					Price:        120.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell2"},
				},
			},
			expectedHoldTime: []int{0, 0, 3600, 9600},
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			suite.withAverageCostStrategy(func() {
				for _, order := range tc.orders {
					_, err := suite.state.Update([]types.Order{order})
					suite.Require().NoError(err)
				}

				trades, err := suite.state.GetAllTrades()
				suite.Require().NoError(err)
				suite.Require().Equal(len(tc.expectedHoldTime), len(trades), "Number of trades mismatch")

				for i, trade := range trades {
					suite.Assert().Equal(tc.expectedHoldTime[i], trade.HoldTime, "HoldTime mismatch at trade %d", i)
				}
			})
		})
	}
}

// TestCalculateTradeHoldingTime_AverageCostLIFO verifies the stats-level
// min/max/avg holding time under LIFO lot attribution (avg-cost mode).
// Scenario:
//
//	buy1 100@10:00, buy2 100@11:00, sell 150@12:00, endTime=15:00
//
// LIFO match emits: 100 * (12:00-11:00) = 3600s and 50 * (12:00-10:00) = 7200s
// for consumed slices, then the remaining 50 of buy1 emits (15:00-10:00) =
// 18000s as an open-position duration.
func (suite *BacktestStateTestSuite) TestCalculateTradeHoldingTime_AverageCostLIFO() {
	err := suite.state.Cleanup()
	suite.Require().NoError(err)

	orders := []types.Order{
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        100.0,
			Fee:          1.0,
			Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			IsCompleted:  true,
			PositionType: types.PositionTypeLong,
			StrategyName: "test",
			Reason:       types.Reason{Reason: "test", Message: "buy1"},
		},
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        105.0,
			Fee:          1.0,
			Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
			IsCompleted:  true,
			PositionType: types.PositionTypeLong,
			StrategyName: "test",
			Reason:       types.Reason{Reason: "test", Message: "buy2"},
		},
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeSell,
			Quantity:     150,
			Price:        115.0,
			Fee:          1.0,
			Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
			IsCompleted:  true,
			PositionType: types.PositionTypeLong,
			StrategyName: "test",
			Reason:       types.Reason{Reason: "test", Message: "sell"},
		},
	}

	suite.withAverageCostStrategy(func() {
		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		endTime := time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC)
		holdingTime, err := suite.state.calculateTradeHoldingTime("AAPL", endTime, PortfolioCalculationAverageCost)
		suite.Require().NoError(err)

		// durations = [3600, 7200, 18000]; avg = 9600.
		suite.Assert().Equal(3600, holdingTime.Min, "Min mismatch")
		suite.Assert().Equal(18000, holdingTime.Max, "Max mismatch")
		suite.Assert().Equal(9600, holdingTime.Avg, "Avg mismatch")
	})
}

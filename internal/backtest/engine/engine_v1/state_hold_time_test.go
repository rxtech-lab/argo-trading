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

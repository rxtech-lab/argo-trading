package engine

import (
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// TestLIFOPnL_LongPosition tests LIFO-based individual PnL calculations for long positions.
// LIFO matches a closing trade against the most recent (last) buy entries first.
func (suite *BacktestStateTestSuite) TestLIFOPnL_LongPosition() {
	tests := []struct {
		name            string
		orders          []types.Order
		expectedLIFOPnL []float64 // Expected LIFO PnL for each trade in order
	}{
		{
			name: "Single buy - no LIFO PnL",
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
			},
			expectedLIFOPnL: []float64{0},
		},
		{
			name: "Single buy-sell pair - LIFO equals FIFO",
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
			// LIFO: sell matches the only buy (100@$100, fee=1)
			//   PnL = (110*100 - 1) - (100*100 + 1) = 11000 - 1 - 10001 = 998
			expectedLIFOPnL: []float64{0, 998},
		},
		{
			name: "Two buys then full sell - LIFO matches last buy first",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 105.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					// Partial sell: 100 units @ $115 -> matches buy2 (last buy) entirely
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 115.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					// Final sell: 100 units @ $120 -> matches buy1 (the remaining lot)
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 120.0,
					Fee: 1.0, Timestamp: time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell2"},
				},
			},
			// LIFO sell1 matches buy2 (100@$105, fee=1):
			//   PnL = (115*100 - 1) - (105*100 + 1) = 11499 - 10501 = 998
			// LIFO sell2 matches buy1 (100@$100, fee=1):
			//   PnL = (120*100 - 1) - (100*100 + 1) = 11999 - 10001 = 1998
			expectedLIFOPnL: []float64{0, 0, 998, 1998},
		},
		{
			name: "Partial sell across multiple buys - LIFO consumes newest first",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 50, Price: 110.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					// Sell 80 @ $120: LIFO consumes 50 from buy2 (@110) and 30 from buy1 (@100)
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 80, Price: 120.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeLong,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "sell1"},
				},
			},
			// matched cost = 50*110 + 30*100 = 5500 + 3000 = 8500
			// sell value   = 80*120 = 9600
			// LIFO PnL     = 9600 - 0 - 8500 = 1100
			expectedLIFOPnL: []float64{0, 0, 1100},
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
			suite.Require().Equal(len(tc.expectedLIFOPnL), len(trades), "Number of trades mismatch")

			for i, trade := range trades {
				suite.Assert().InDelta(tc.expectedLIFOPnL[i], trade.LIFOPnL, 1e-9, "LIFO PnL mismatch at trade %d", i)
			}
		})
	}
}

// TestLIFOPnL_ShortPosition tests LIFO-based individual PnL calculations for short positions.
// Short positions: BUY entries, SELL exits.
func (suite *BacktestStateTestSuite) TestLIFOPnL_ShortPosition() {
	tests := []struct {
		name            string
		orders          []types.Order
		expectedLIFOPnL []float64
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
			// Short LIFO: matched entry value = 100*100 - 1 = 9999
			// exit value = 90*100 + 1 (sell fee) = 9001
			// PnL = 9999 - 9001 = 998
			expectedLIFOPnL: []float64{0, 998},
		},
		{
			name: "Two short entries then exit - LIFO matches last buy first",
			orders: []types.Order{
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 100.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "open1"},
				},
				{
					Symbol: "AAPL", Side: types.PurchaseTypeBuy, Quantity: 100, Price: 110.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "open2"},
				},
				{
					// Sell 100 @ 95 -> LIFO matches latest open (100@110)
					Symbol: "AAPL", Side: types.PurchaseTypeSell, Quantity: 100, Price: 95.0,
					Fee: 0.0, Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true, PositionType: types.PositionTypeShort,
					StrategyName: "test", Reason: types.Reason{Reason: "test", Message: "close1"},
				},
			},
			// Short LIFO close1: matched entry = 100*110 = 11000; exit = 100*95 = 9500
			// PnL = 11000 - 9500 = 1500
			expectedLIFOPnL: []float64{0, 0, 1500},
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
			suite.Require().Equal(len(tc.expectedLIFOPnL), len(trades), "Number of trades mismatch")

			for i, trade := range trades {
				suite.Assert().InDelta(tc.expectedLIFOPnL[i], trade.LIFOPnL, 1e-9, "LIFO PnL mismatch at trade %d", i)
			}
		})
	}
}

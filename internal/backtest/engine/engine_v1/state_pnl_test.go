package engine

import (
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// TestFIFOPnL_LongPosition tests FIFO-based individual PnL calculations for long positions.
func (suite *BacktestStateTestSuite) TestFIFOPnL_LongPosition() {
	tests := []struct {
		name           string
		orders         []types.Order
		expectedPnL    []float64 // Expected FIFO PnL for each trade
		expectedCumPnL []float64 // Expected cumulative PnL for each trade
	}{
		{
			name: "Single buy - no PnL",
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
			expectedPnL:    []float64{0},
			expectedCumPnL: []float64{0},
		},
		{
			name: "Single buy-sell pair - FIFO equals cumulative",
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
			// FIFO PnL = (110*100 - 1) - (100*100 + 1) = 10999 - 10001 = 998
			expectedPnL:    []float64{0, 998},
			expectedCumPnL: []float64{0, 998},
		},
		{
			name: "Multiple buys at different prices, FIFO matches first buy to first sell",
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
			// FIFO: sell1 matches buy1 (100@$50, fee $1)
			//   PnL = (120*100 - 1) - (50*100 + 1) = 11999 - 5001 = 6998
			// FIFO: sell2 matches buy2 (100@$150, fee $1)
			//   PnL = (120*100 - 1) - (150*100 + 1) = 11999 - 15001 = -3002
			// Cumulative: avg entry = (50*100 + 150*100 + 2) / 200 = 20002/200 = 100.01
			//   sell1 PnL = (120*100 - 1) - (100.01*100) = 11999 - 10001 = 1998
			//   sell2 PnL = (120*100 - 1) - (100.01*100) = 11999 - 10001 = 1998
			expectedPnL:    []float64{0, 0, 6998, -3002},
			expectedCumPnL: []float64{0, 0, 1998, 1998},
		},
		{
			name: "Partial sell crosses multiple buys (FIFO)",
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
			// FIFO: sell 70 matches buy1 (50@$100) + buy2 (20@$200)
			//   buy1 cost: 100*50 + 1*50/50 = 5001
			//   buy2 cost: 200*20 + 1*20/50 = 4000.4
			//   total cost: 5001 + 4000.4 = 9001.4
			//   PnL = 160*70 - 1 - 9001.4 = 11200 - 1 - 9001.4 = 2197.6
			// Cumulative: avg entry = (100*50 + 200*50 + 2) / 100 = 15002/100 = 150.02
			//   PnL = (160*70 - 1) - (150.02*70) = 11199 - 10501.4 = 697.6
			expectedPnL:    []float64{0, 0, 2197.6},
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
			// FIFO: sell1 matches buy1 (100@$100)
			//   PnL = (110*100 - 1) - (100*100 + 1) = 998
			// FIFO: sell2 matches buy2 (100@$105) - buy1 is consumed
			//   PnL = (115*100 - 1) - (105*100 + 1) = 998
			// Cumulative: after buy2 avg entry = (100*100+105*100+2)/200 = 20502/200 = 102.51
			//   sell2 PnL = (115*100 - 1) - (102.51*100) = 11499 - 10251 = 1248
			expectedPnL:    []float64{0, 998, 0, 998},
			expectedCumPnL: []float64{0, 998, 0, 1248},
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
			suite.Require().Equal(len(tc.expectedPnL), len(trades), "Number of trades mismatch")

			for i, trade := range trades {
				suite.Assert().Equal(tc.expectedPnL[i], trade.PnL, "FIFO PnL mismatch at trade %d", i)
				suite.Assert().Equal(tc.expectedCumPnL[i], trade.CumulativePnL, "Cumulative PnL mismatch at trade %d", i)
			}
		})
	}
}

// TestFIFOPnL_ShortPosition tests FIFO-based individual PnL calculations for short positions.
func (suite *BacktestStateTestSuite) TestFIFOPnL_ShortPosition() {
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
			// Short: entry=buy, exit=sell. Profit when price goes down.
			// FIFO: (100*100 - 1) - (90*100 + 1) = 9999 - 9001 = 998
			expectedPnL:    []float64{0, 998},
			expectedCumPnL: []float64{0, 998},
		},
		{
			name: "Multiple short entries at different prices, FIFO matches first",
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
			// FIFO: close1 matches open1 (100@$200)
			//   PnL = (200*100 - 1) - (120*100 + 1) = 19999 - 12001 = 7998
			// FIFO: close2 matches open2 (100@$100)
			//   PnL = (100*100 - 1) - (120*100 + 1) = 9999 - 12001 = -2002
			// Cumulative: avg entry = (200*100+100*100-2)/200 = 29998/200 = 149.99
			//   close1 PnL = 149.99*100 - (120*100+1) = 14999 - 12001 = 2998
			//   close2 PnL = 149.99*100 - (120*100+1) = 14999 - 12001 = 2998
			expectedPnL:    []float64{0, 0, 7998, -2002},
			expectedCumPnL: []float64{0, 0, 2998, 2998},
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
			suite.Require().Equal(len(tc.expectedPnL), len(trades), "Number of trades mismatch")

			for i, trade := range trades {
				suite.Assert().Equal(tc.expectedPnL[i], trade.PnL, "FIFO PnL mismatch at trade %d", i)
				suite.Assert().Equal(tc.expectedCumPnL[i], trade.CumulativePnL, "Cumulative PnL mismatch at trade %d", i)
			}
		})
	}
}

// TestFIFOPnL_TotalConsistency verifies that sum of FIFO PnL equals sum of cumulative PnL
// for fully closed positions.
func (suite *BacktestStateTestSuite) TestFIFOPnL_TotalConsistency() {
	suite.Run("Long position total PnL consistency", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)

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

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		trades, err := suite.state.GetAllTrades()
		suite.Require().NoError(err)

		var totalFIFO, totalCumulative float64
		for _, trade := range trades {
			totalFIFO += trade.PnL
			totalCumulative += trade.CumulativePnL
		}

		// Both should equal the same total for a fully closed position
		suite.Assert().Equal(totalCumulative, totalFIFO, "Total FIFO PnL should equal total cumulative PnL for fully closed positions")
	})

	suite.Run("Short position total PnL consistency", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)

		orders := []types.Order{
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
		}

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		trades, err := suite.state.GetAllTrades()
		suite.Require().NoError(err)

		var totalFIFO, totalCumulative float64
		for _, trade := range trades {
			totalFIFO += trade.PnL
			totalCumulative += trade.CumulativePnL
		}

		suite.Assert().Equal(totalCumulative, totalFIFO, "Total FIFO PnL should equal total cumulative PnL for fully closed positions")
	})
}

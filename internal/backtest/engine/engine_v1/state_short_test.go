package engine

import (
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

func (suite *BacktestStateTestSuite) TestUpdate_ShortPosition() {

	type ExpectPosition struct {
		types.Position
		TotalPnL float64
	}

	tests := []struct {
		name             string
		orders           []types.Order
		expectedTrades   []types.Trade
		expectedPosition ExpectPosition
		expectError      bool
	}{
		{
			name: "Single entry with fee",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeShort,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
						OrderID:      "order1",
						Symbol:       "AAPL",
						Side:         types.PurchaseTypeBuy,
						Quantity:     100,
						Price:        100.0,
						Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted:  true,
						PositionType: types.PositionTypeShort,
						StrategyName: "a",
						Reason: types.Reason{
							Reason:  "test",
							Message: "reason",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:                     "AAPL",
					TotalShortPositionQuantity: 100,
					OpenTimestamp:              time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: 0,
			},
		},
		{
			name: "Single entry and exit with fee",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeShort,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				{
					OrderID:      "order2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeShort,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
						OrderID:      "order1",
						Symbol:       "AAPL",
						Side:         types.PurchaseTypeBuy,
						Quantity:     100,
						Price:        100.0,
						Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted:  true,
						PositionType: types.PositionTypeShort,
						StrategyName: "a",
						Reason: types.Reason{
							Reason:  "test",
							Message: "reason",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					Order: types.Order{
						OrderID:      "order2",
						Symbol:       "AAPL",
						Side:         types.PurchaseTypeSell,
						Quantity:     100,
						Price:        110.0,
						Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted:  true,
						PositionType: types.PositionTypeShort,
						StrategyName: "a",
						Reason: types.Reason{
							Reason:  "test",
							Message: "reason",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					PnL:           -1002,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:                     "AAPL",
					TotalShortPositionQuantity: 0,
					OpenTimestamp:              time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: -1002,
			},
		},
		{
			name: "Single entry and partial close with fee",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeShort,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				{
					OrderID:      "order2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     50,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					PositionType: types.PositionTypeShort,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
						OrderID:      "order1",
						Symbol:       "AAPL",
						Side:         types.PurchaseTypeBuy,
						Quantity:     100,
						Price:        100.0,
						Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted:  true,
						PositionType: types.PositionTypeShort,
						StrategyName: "a",
						Reason: types.Reason{
							Reason:  "test",
							Message: "reason",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					Order: types.Order{
						OrderID:      "order2",
						Symbol:       "AAPL",
						Side:         types.PurchaseTypeSell,
						Quantity:     50,
						Price:        110.0,
						Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted:  true,
						PositionType: types.PositionTypeShort,
						StrategyName: "a",
						Reason: types.Reason{
							Reason:  "test",
							Message: "reason",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   50,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					PnL:           -501.5,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:                     "AAPL",
					TotalShortPositionQuantity: 50,
					OpenTimestamp:              time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: -501.5,
			},
		},
		{
			name: "Multiple entry and close long position with fee",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeShort,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
					Quantity:    100,
					Price:       90.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeShort,
					Fee:          1.0,
				},
				{
					OrderID:     "order3",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
					Quantity:    100,
					Price:       80.0,
					Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeShort,
					Fee:          1.0,
				},
				{
					OrderID:     "order4",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    100,
					Price:       110.0,
					Timestamp:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeShort,
					Fee:          1.0,
				},
				{
					OrderID:     "order5",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    100,
					Price:       120.0,
					Timestamp:   time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeShort,
					Fee:          1.0,
				},
				{
					OrderID:     "order6",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    100,
					Price:       130.0,
					Timestamp:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeShort,
					Fee:          1.0,
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
						OrderID:     "order1",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeBuy,
						Quantity:    100,
						Price:       100.0,
						Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					Order: types.Order{
						OrderID:     "order2",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeBuy,
						Quantity:    100,
						Price:       90.0,
						Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 90.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					Order: types.Order{
						OrderID:     "order3",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeBuy,
						Quantity:    100,
						Price:       80.0,
						Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 80.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					Order: types.Order{
						OrderID:     "order4",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeSell,
						Quantity:    100,
						Price:       110.0,
						Timestamp:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					PnL:           -2002,
				},
				{
					Order: types.Order{
						OrderID:     "order5",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeSell,
						Quantity:    100,
						Price:       120.0,
						Timestamp:   time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 120.0,
					Fee:           1.0,
					PnL:           -3002,
				},
				{
					Order: types.Order{
						OrderID:     "order6",
						Symbol:      "AAPL",
						Side:        types.PurchaseTypeSell,
						Quantity:    100,
						Price:       130.0,
						Timestamp:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
					ExecutedAt:    time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 130.0,
					Fee:           1.0,
					PnL:           -4002,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:                     "AAPL",
					TotalShortPositionQuantity: 0,
					OpenTimestamp:              time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: -9006,
			},
		},
	}

	// Run Update tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset state before each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			// Process orders one by one to simulate real trading
			var allResults []UpdateResult
			for _, order := range tc.orders {
				results, err := suite.state.Update([]types.Order{order})
				if tc.expectError {
					suite.Assert().Error(err)
					return
				}
				suite.Assert().NoError(err)
				suite.Assert().Equal(1, len(results), "Expected one result per order")
				allResults = append(allResults, results[0])
			}

			// Verify trades
			trades, err := suite.state.GetAllTrades()
			suite.Assert().NoError(err)
			suite.Assert().Equal(len(tc.expectedTrades), len(trades), "Number of trades mismatch")

			if len(tc.expectedTrades) > 0 {
				for i, expected := range tc.expectedTrades {
					// Skip order_id comparison since it's auto-generated
					suite.Assert().Equal(expected.ExecutedAt.UTC(), trades[i].ExecutedAt.UTC(), "ExecutedAt mismatch")
					suite.Assert().Equal(expected.ExecutedQty, trades[i].ExecutedQty, "ExecutedQty mismatch")
					suite.Assert().Equal(expected.ExecutedPrice, trades[i].ExecutedPrice, "ExecutedPrice mismatch")
					suite.Assert().Equal(expected.Fee, trades[i].Fee, "Commission mismatch")
					suite.Assert().Equal(expected.PnL, trades[i].PnL, "PnL mismatch")
				}
			}

			// Verify positions
			position, err := suite.state.GetPosition(tc.expectedPosition.Symbol)
			suite.Assert().NoError(err)
			suite.Assert().Equal(tc.expectedPosition.Symbol, position.Symbol, "Position symbol mismatch")
			suite.Assert().Equal(tc.expectedPosition.TotalShortPositionQuantity, position.TotalShortPositionQuantity, "Position quantity mismatch")
			suite.Assert().Equal(tc.expectedPosition.OpenTimestamp.UTC(), position.OpenTimestamp.UTC(), "Position open timestamp mismatch")
			suite.Assert().Equal(tc.expectedPosition.TotalPnL, position.GetTotalPnL(), "Position total PnL mismatch")

			// Verify results
			for i, result := range allResults {
				// Verify order
				suite.Assert().Equal(tc.orders[i].Symbol, result.Order.Symbol, "Result order symbol mismatch")
				suite.Assert().Equal(tc.orders[i].Side, result.Order.Side, "Result order type mismatch")
				suite.Assert().Equal(tc.orders[i].Quantity, result.Order.Quantity, "Result order quantity mismatch")
				suite.Assert().Equal(tc.orders[i].Price, result.Order.Price, "Result order price mismatch")
				suite.Assert().Equal(tc.orders[i].Timestamp.UTC(), result.Order.Timestamp.UTC(), "Result order timestamp mismatch")
				suite.Assert().Equal(tc.orders[i].IsCompleted, result.Order.IsCompleted, "Result order is_completed mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Reason, result.Order.Reason.Reason, "Result order reason mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Message, result.Order.Reason.Message, "Result order message mismatch")
				suite.Assert().Equal(tc.orders[i].StrategyName, result.Order.StrategyName, "Result order strategy name mismatch")
				suite.Assert().Equal(tc.orders[i].PositionType, result.Order.PositionType, "Result order position type mismatch")

				// Verify trade
				suite.Assert().Equal(tc.orders[i].Symbol, result.Trade.Order.Symbol, "Result trade symbol mismatch")
				suite.Assert().Equal(tc.orders[i].Side, result.Trade.Order.Side, "Result trade type mismatch")
				suite.Assert().Equal(tc.orders[i].Quantity, result.Trade.ExecutedQty, "Result trade quantity mismatch")
				suite.Assert().Equal(tc.orders[i].Price, result.Trade.ExecutedPrice, "Result trade price mismatch")
				suite.Assert().Equal(tc.orders[i].Timestamp.UTC(), result.Trade.ExecutedAt.UTC(), "Result trade timestamp mismatch")

				// Verify IsNewPosition
				if i == 0 && tc.orders[i].Side == types.PurchaseTypeBuy {
					suite.Assert().True(result.IsNewPosition, "Expected IsNewPosition to be true for first buy order")
				} else {
					suite.Assert().False(result.IsNewPosition, "Expected IsNewPosition to be false for subsequent orders")
				}
			}
		})
	}
}

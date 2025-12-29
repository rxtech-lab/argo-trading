package engine

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// BacktestTradingTestSuite is a test suite for BacktestTrading
type BacktestTradingTestSuite struct {
	suite.Suite
	state          *BacktestState
	logger         *logger.Logger
	trading        *BacktestTrading
	commission     commission_fee.CommissionFee
	initialBalance float64
}

// SetupSuite runs once before all tests in the suite
func (suite *BacktestTradingTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger

	var stateErr error
	suite.state, stateErr = NewBacktestState(suite.logger)
	suite.Require().NoError(stateErr)
	suite.Require().NotNil(suite.state)

	suite.initialBalance = 10000.0
	suite.commission = commission_fee.NewZeroCommissionFee()
}

// TearDownSuite runs once after all tests in the suite
func (suite *BacktestTradingTestSuite) TearDownSuite() {
	if suite.state != nil && suite.state.db != nil {
		suite.state.db.Close()
	}
}

// SetupTest runs before each test
func (suite *BacktestTradingTestSuite) SetupTest() {
	err := suite.state.Initialize()
	suite.Require().NoError(err)
	suite.trading = &BacktestTrading{
		state:            suite.state,
		balance:          suite.initialBalance,
		marketData:       types.MarketData{},
		pendingOrders:    []types.ExecuteOrder{},
		commission:       suite.commission,
		decimalPrecision: 1, // Default to 1 decimal place
	}
}

// TearDownTest runs after each test
func (suite *BacktestTradingTestSuite) TearDownTest() {
	err := suite.state.Cleanup()
	suite.Require().NoError(err)
}

// TestBacktestTradingTestSuite runs the test suite
func TestBacktestTradingTestSuite(t *testing.T) {
	suite.Run(t, new(BacktestTradingTestSuite))
}

func (suite *BacktestTradingTestSuite) TestUpdateCurrentMarketData() {
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
		Close:  95.0,
	}

	suite.trading.UpdateCurrentMarketData(marketData)
	suite.Assert().Equal(marketData, suite.trading.marketData)
}

func (suite *BacktestTradingTestSuite) TestCancelAllOrders() {
	// Setup test data
	suite.trading.pendingOrders = []types.ExecuteOrder{
		{
			ID:     "order1",
			Symbol: "AAPL",
		},
		{
			ID:     "order2",
			Symbol: "GOOGL",
		},
	}

	err := suite.trading.CancelAllOrders()
	suite.Require().NoError(err)
	suite.Assert().Empty(suite.trading.pendingOrders)
}

func (suite *BacktestTradingTestSuite) TestCancelOrder() {
	// Setup test data
	suite.trading.pendingOrders = []types.ExecuteOrder{
		{
			ID:     "order1",
			Symbol: "AAPL",
		},
		{
			ID:     "order2",
			Symbol: "GOOGL",
		},
	}

	// Test canceling existing order
	err := suite.trading.CancelOrder("order1")
	suite.Require().NoError(err)
	suite.Assert().Len(suite.trading.pendingOrders, 1)
	suite.Assert().Equal("order2", suite.trading.pendingOrders[0].ID)

	// Test canceling non-existent order
	err = suite.trading.CancelOrder("non-existent")
	suite.Require().NoError(err)
	suite.Assert().Len(suite.trading.pendingOrders, 1)
}

// TestPlaceOrder_Simple_Comparison tests the place order function with simple comparison
// No placed order checking.
func (suite *BacktestTradingTestSuite) TestPlaceOrder_Simple_Comparison() {

	type Test struct {
		name                 string
		marketData           types.MarketData
		order                types.ExecuteOrder
		marketDataAfterOrder optional.Option[types.MarketData]
		decimalPrecision     int
		expectError          bool
		executedOrderNumber  int
		pendingOrderNumber   int
		expectFailedOrder    bool
		expectedFailReason   string
	}

	testRunner := func(tests []Test) {
		for _, tc := range tests {
			suite.Run(tc.name, func() {
				suite.trading.decimalPrecision = tc.decimalPrecision
				suite.trading.UpdateCurrentMarketData(tc.marketData)
				err := suite.trading.PlaceOrder(tc.order)
				if tc.expectError {
					suite.Assert().Error(err)
				} else {
					suite.Assert().NoError(err)
				}

				if tc.marketDataAfterOrder.IsSome() {
					suite.trading.UpdateCurrentMarketData(tc.marketDataAfterOrder.Unwrap())
				}

				suite.Assert().Equal(tc.pendingOrderNumber, len(suite.trading.pendingOrders))

				allOrders, err := suite.state.GetAllOrders()
				suite.Require().NoError(err)
				suite.Assert().Equal(tc.executedOrderNumber, len(allOrders))

				// Check for failed order
				if tc.expectFailedOrder {
					var failedOrder *types.Order
					for i := range allOrders {
						if allOrders[i].Status == types.OrderStatusFailed {
							failedOrder = &allOrders[i]
							break
						}
					}
					suite.Require().NotNil(failedOrder, "Expected a failed order but none found")
					suite.Assert().Equal(types.OrderStatusFailed, failedOrder.Status)
					suite.Assert().Equal(tc.expectedFailReason, failedOrder.Reason.Reason)
				}

				if err := suite.state.Cleanup(); err != nil {
					suite.T().Fatalf("failed to cleanup state: %v", err)
				}
			})
		}
	}

	suite.Run("Market Price order - Long", func() {
		tests := []Test{
			{
				name: "Valid order within price range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
			},
			{
				name: "Order price above range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        110.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
			},
			{
				name: "Order price below range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        85.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
			},
			{
				name: "Order quantity exceeds buying power",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     1000000, // Very large quantity
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false, // No error, but order is failed
				executedOrderNumber:  1,     // Failed order is stored
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInsufficientBuyPower,
			},
			{
				name: "Order quantity exceeds selling power",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					OrderType:    types.OrderTypeMarket,
					Quantity:     1000, // Selling more than held
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false, // No error, but order is failed
				executedOrderNumber:  1,     // Failed order is stored
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInsufficientSellPower,
			},
			{
				name: "Buy order with zero quantity",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     0,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidQuantity,
			},
			{
				name: "Buy order with negative quantity",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     -10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidQuantity,
			},
			{
				name: "Buy order with zero price",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidPrice,
			},
			{
				name: "Buy order with negative price",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        -95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidPrice,
			},
			{
				name: "Sell order with zero quantity",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					OrderType:    types.OrderTypeMarket,
					Quantity:     0,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidQuantity,
			},
			{
				name: "Sell order with negative quantity",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					OrderType:    types.OrderTypeMarket,
					Quantity:     -10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidQuantity,
			},
			{
				name: "Sell order with zero price",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidPrice,
			},
			{
				name: "Sell order with negative price",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					OrderType:    types.OrderTypeMarket,
					Quantity:     10,
					Price:        -95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
				expectFailedOrder:    true,
				expectedFailReason:   types.OrderReasonInvalidPrice,
			},
		}

		testRunner(tests)
	})

	suite.Run("Limit order - Long", func() {
		tests := []Test{
			{
				name: "Limit order within price range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   100.0,
					Low:    90.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeLimit,
					Quantity:     10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				marketDataAfterOrder: optional.None[types.MarketData](),
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  1,
				pendingOrderNumber:   0,
			}, {
				name: "Limit order outside price range but after within the range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   101.0,
					Low:    100.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeLimit,
					Quantity:     10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				marketDataAfterOrder: optional.Some(types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					High:   100.0,
					Low:    94.0,
				}),
				decimalPrecision:    0,
				expectError:         false,
				executedOrderNumber: 1,
				pendingOrderNumber:  0,
			},
			{
				name: "Limit order outside price range and never within the range",
				marketData: types.MarketData{
					Symbol: "AAPL",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   101.0,
					Low:    100.0,
				},
				order: types.ExecuteOrder{
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					OrderType:    types.OrderTypeLimit,
					Quantity:     10,
					Price:        95.0,
					StrategyName: "test_strategy",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				marketDataAfterOrder: optional.None[types.MarketData](),
				decimalPrecision:     0,
				expectError:          false,
				executedOrderNumber:  0,
				pendingOrderNumber:   1,
			},
		}

		testRunner(tests)
	})

}

func (suite *BacktestTradingTestSuite) TestPlaceOrder_With_Market_Price_Order_Buy() {
	// Setup market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
	}
	suite.trading.UpdateCurrentMarketData(marketData)

	// Calculate expected average price
	avgPrice := (marketData.High + marketData.Low) / 2 // 95.0

	testCases := []struct {
		name               string
		balance            float64
		quantity           float64
		expectError        bool
		errorMessage       string
		expectFailedOrder  bool
		expectedFailReason string
	}{
		{
			name:        "Successful buy - sufficient balance",
			balance:     10000.0,
			quantity:    10.0, // Total cost: 950.0 (10 * 95.0)
			expectError: false,
		},
		{
			name:               "Failed buy - insufficient balance",
			balance:            500.0,
			quantity:           10.0, // Total cost: 950.0 (10 * 95.0)
			expectError:        false,
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientBuyPower,
		},
		{
			name:        "Successful buy - exact balance",
			balance:     950.0,
			quantity:    10.0, // Total cost: 950.0 (10 * 95.0)
			expectError: false,
		},
		{
			name:         "Failed buy - zero quantity after rounding",
			balance:      1000.0,
			quantity:     0.01, // Will round to 0 with precision of 1
			expectError:  true,
			errorMessage: "order quantity is too small or zero after rounding to configured precision",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Set balance for this test
			suite.trading.UpdateBalance(tc.balance)

			// Create order
			order := types.ExecuteOrder{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeMarket,
				Quantity:     tc.quantity,
				Price:        avgPrice, // Add the price field which is required
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "reason",
				},
			}

			// Execute the order
			err = suite.trading.PlaceOrder(order)

			if tc.expectError {
				suite.Assert().Error(err)
				if tc.errorMessage != "" {
					suite.Assert().Contains(err.Error(), tc.errorMessage)
				}
			} else {
				suite.Assert().NoError(err)

				if tc.expectFailedOrder {
					// Check for failed order
					allOrders, err := suite.state.GetAllOrders()
					suite.Require().NoError(err)
					suite.Require().Len(allOrders, 1)
					suite.Assert().Equal(types.OrderStatusFailed, allOrders[0].Status)
					suite.Assert().Equal(tc.expectedFailReason, allOrders[0].Reason.Reason)
				} else {
					// Verify the trade was executed correctly
					trades, err := suite.state.GetAllTrades()
					suite.Require().NoError(err)
					suite.Require().NotEmpty(trades)

					// Check that the trade was executed at the average price
					suite.Assert().Equal(avgPrice, trades[0].Order.Price)
					suite.Assert().Equal(tc.quantity, trades[0].Order.Quantity)
					suite.Assert().Equal(types.PurchaseTypeBuy, trades[0].Order.Side)
				}
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestPlaceOrder_With_Market_Price_Order_Sell() {
	// Setup initial position
	initialQuantity := 50.0
	initialOrder := types.Order{
		Symbol:       "AAPL",
		Side:         types.PurchaseTypeBuy,
		Quantity:     initialQuantity,
		Price:        90.0,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		StrategyName: "test_strategy",
		PositionType: types.PositionTypeLong,
		Reason: types.Reason{
			Reason:  "test",
			Message: "reason",
		},
	}

	// Setup market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
	}

	// Calculate expected average price
	avgPrice := (marketData.High + marketData.Low) / 2 // 95.0

	testCases := []struct {
		name               string
		sellQuantity       float64
		expectError        bool
		expectedQuantity   float64 // Expected quantity to be sold after adjustment
		expectFailedOrder  bool
		expectedFailReason string
		addInitialPosition bool // Whether to add initial position before the test
	}{
		{
			name:               "Successful sell - within holdings",
			sellQuantity:       20.0,
			expectError:        false,
			expectedQuantity:   20.0,
			addInitialPosition: true,
		},
		{
			name:               "Successful sell - exact holdings",
			sellQuantity:       50.0,
			expectError:        false,
			expectedQuantity:   50.0,
			addInitialPosition: true,
		},
		{
			name:               "Failed sell - quantity exceeds holdings",
			sellQuantity:       100.0, // More than we have (we have 50)
			expectError:        false,
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientSellPower,
			addInitialPosition: true,
		},
		{
			name:               "Failed sell - no holdings",
			sellQuantity:       10.0,
			expectError:        false, // No error, but order is failed
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientSellPower,
			addInitialPosition: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Add initial position if specified
			if tc.addInitialPosition {
				_, err = suite.state.Update([]types.Order{initialOrder})
				suite.Require().NoError(err)
			}

			// Update market data
			suite.trading.UpdateCurrentMarketData(marketData)

			// Create sell order
			order := types.ExecuteOrder{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeSell,
				OrderType:    types.OrderTypeMarket,
				Quantity:     tc.sellQuantity,
				Price:        avgPrice, // Add the price field which is required
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "reason",
				},
			}

			// Execute the order
			err = suite.trading.PlaceOrder(order)

			if tc.expectError {
				suite.Assert().Error(err)
			} else {
				suite.Assert().NoError(err)

				if tc.expectFailedOrder {
					// Check for failed order
					allOrders, err := suite.state.GetAllOrders()
					suite.Require().NoError(err)

					// Find the failed order among all orders
					var failedOrder *types.Order
					for i := range allOrders {
						if allOrders[i].Status == types.OrderStatusFailed {
							failedOrder = &allOrders[i]
							break
						}
					}
					suite.Require().NotNil(failedOrder, "Expected a failed order but none found")
					suite.Assert().Equal(tc.expectedFailReason, failedOrder.Reason.Reason)
				} else {
					// Verify the trade was executed correctly
					trades, err := suite.state.GetAllTrades()
					suite.Require().NoError(err)
					suite.Assert().GreaterOrEqual(len(trades), 2) // Initial buy + our sell

					// Find our sell trade (most recent one)
					var sellTrade types.Trade
					for _, trade := range trades {
						if trade.Order.Side == types.PurchaseTypeSell {
							sellTrade = trade
							break
						}
					}

					// Check that the trade was executed properly
					suite.Assert().Equal(tc.expectedQuantity, sellTrade.Order.Quantity)
					// Check that the avg price was used
					suite.Assert().Equal((marketData.High+marketData.Low)/2, sellTrade.Order.Price)
				}
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestPlaceOrder_With_Limit_Price_Order_Buy() {
	// Setup initial market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
	}

	testCases := []struct {
		name               string
		balance            float64
		limitPrice         float64
		quantity           float64
		updatedMarketData  *types.MarketData // To simulate price changes
		expectError        bool
		errorMessage       string
		shouldExecute      bool    // Should the order execute immediately or be pending?
		executionPrice     float64 // Expected execution price
		expectFailedOrder  bool
		expectedFailReason string
	}{
		{
			name:           "Buy limit below market - execute immediately",
			balance:        10000.0,
			limitPrice:     95.0,
			quantity:       10.0,
			expectError:    false,
			shouldExecute:  true,
			executionPrice: 95.0, // Average market price is 95.0, but limit is 95.0 too
		},
		{
			name:           "Buy limit below market - lower price execution",
			balance:        10000.0,
			limitPrice:     98.0,
			quantity:       10.0,
			expectError:    false,
			shouldExecute:  true,
			executionPrice: 95.0, // Should use the lower average market price (95.0) instead of limit (98.0)
		},
		{
			name:          "Buy limit above current market - order pending",
			balance:       10000.0,
			limitPrice:    85.0, // Below current market low of 90
			quantity:      10.0,
			expectError:   false,
			shouldExecute: false, // Should be pending
		},
		{
			name:       "Buy limit above market initially pending, then executed",
			balance:    10000.0,
			limitPrice: 85.0, // Below current market
			quantity:   10.0,
			updatedMarketData: &types.MarketData{
				Symbol: "AAPL",
				Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
				High:   90.0,
				Low:    80.0, // Now below our limit price
			},
			expectError:    false,
			shouldExecute:  true,
			executionPrice: 85.0, // Should use our limit price
		},
		{
			name:               "Buy limit - insufficient balance",
			balance:            500.0,
			limitPrice:         95.0,
			quantity:           10.0, // Total cost: 950.0
			expectError:        false,
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientBuyPower,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Set balance and market data
			suite.trading.UpdateBalance(tc.balance)
			suite.trading.UpdateCurrentMarketData(marketData)

			// Create limit order
			order := types.ExecuteOrder{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeLimit,
				Price:        tc.limitPrice,
				Quantity:     tc.quantity,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "reason",
				},
			}

			// Place the order
			err = suite.trading.PlaceOrder(order)

			if tc.expectError {
				suite.Assert().Error(err)
				if tc.errorMessage != "" {
					suite.Assert().Contains(err.Error(), tc.errorMessage)
				}
				return
			}

			suite.Assert().NoError(err)

			// Check for failed order first
			if tc.expectFailedOrder {
				allOrders, err := suite.state.GetAllOrders()
				suite.Require().NoError(err)

				// Find the failed order among all orders
				var failedOrder *types.Order
				for i := range allOrders {
					if allOrders[i].Status == types.OrderStatusFailed {
						failedOrder = &allOrders[i]
						break
					}
				}
				suite.Require().NotNil(failedOrder, "Expected a failed order but none found")
				suite.Assert().Equal(tc.expectedFailReason, failedOrder.Reason.Reason)
				return
			}

			// If we have updated market data, simulate a price change
			if tc.updatedMarketData != nil {
				suite.trading.UpdateCurrentMarketData(*tc.updatedMarketData)
			}

			// Get all trades
			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)

			if tc.shouldExecute {
				// Order should be executed
				suite.Assert().NotEmpty(trades)
				suite.Assert().Equal(tc.executionPrice, trades[0].Order.Price)
				suite.Assert().Equal(tc.quantity, trades[0].Order.Quantity)
				suite.Assert().Equal(types.PurchaseTypeBuy, trades[0].Order.Side)
			} else {
				// Order should be pending
				suite.Assert().Empty(trades)
				suite.Assert().NotEmpty(suite.trading.pendingOrders)
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestPlaceOrder_With_Limit_Price_Order_Sell() {
	// Setup initial position
	initialQuantity := 50.0
	initialOrder := types.Order{
		Symbol:       "AAPL",
		Side:         types.PurchaseTypeBuy,
		PositionType: types.PositionTypeLong,
		Quantity:     initialQuantity,
		Price:        90.0,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		StrategyName: "test_strategy",
		Reason: types.Reason{
			Reason:  "test",
			Message: "reason",
		},
	}

	// Setup market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
	}

	testCases := []struct {
		name               string
		limitPrice         float64
		sellQuantity       float64
		updatedMarketData  *types.MarketData // To simulate price changes
		expectError        bool
		expectedQuantity   float64 // Expected quantity after adjustment
		shouldExecute      bool    // Should execute immediately or be pending
		executionPrice     float64 // Expected execution price
		expectFailedOrder  bool
		expectedFailReason string
		addInitialPosition bool // Whether to add initial position before the test
	}{
		{
			name:               "Sell limit below current high - execute immediately",
			limitPrice:         95.0,
			sellQuantity:       20.0,
			expectError:        false,
			expectedQuantity:   20.0,
			shouldExecute:      true,
			executionPrice:     95.0, // Should use limit price
			addInitialPosition: true,
		},
		{
			name:               "Sell limit above current high - order pending",
			limitPrice:         110.0,
			sellQuantity:       20.0,
			expectError:        false,
			expectedQuantity:   20.0,
			shouldExecute:      false, // Should be pending
			addInitialPosition: true,
		},
		{
			name:         "Sell limit initially pending, then executed",
			limitPrice:   110.0,
			sellQuantity: 20.0,
			updatedMarketData: &types.MarketData{
				Symbol: "AAPL",
				Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
				High:   120.0, // Now above our limit price
				Low:    100.0,
			},
			expectError:        false,
			expectedQuantity:   20.0,
			shouldExecute:      true,
			executionPrice:     110.0, // Should use our limit price
			addInitialPosition: true,
		},
		{
			name:               "Failed sell limit - quantity exceeds holdings",
			limitPrice:         95.0,
			sellQuantity:       100.0, // More than we have (we have 50)
			expectError:        false,
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientSellPower,
			addInitialPosition: true,
		},
		{
			name:               "Sell limit with no holdings",
			limitPrice:         95.0,
			sellQuantity:       10.0,
			expectError:        false,
			expectFailedOrder:  true,
			expectedFailReason: types.OrderReasonInsufficientSellPower,
			addInitialPosition: false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Add initial position if specified
			if tc.addInitialPosition {
				_, err = suite.state.Update([]types.Order{initialOrder})
				suite.Require().NoError(err)
			}

			// Update market data
			suite.trading.UpdateCurrentMarketData(marketData)

			// Create sell limit order
			order := types.ExecuteOrder{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeSell,
				OrderType:    types.OrderTypeLimit,
				Price:        tc.limitPrice,
				Quantity:     tc.sellQuantity,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "reason",
				},
			}

			// Place the order
			err = suite.trading.PlaceOrder(order)

			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)

			// Check for failed order first
			if tc.expectFailedOrder {
				allOrders, err := suite.state.GetAllOrders()
				suite.Require().NoError(err)

				// Find the failed order among all orders
				var failedOrder *types.Order
				for i := range allOrders {
					if allOrders[i].Status == types.OrderStatusFailed {
						failedOrder = &allOrders[i]
						break
					}
				}
				suite.Require().NotNil(failedOrder, "Expected a failed order but none found")
				suite.Assert().Equal(tc.expectedFailReason, failedOrder.Reason.Reason)
				return
			}

			// If we have updated market data, simulate a price change
			if tc.updatedMarketData != nil {
				suite.trading.UpdateCurrentMarketData(*tc.updatedMarketData)
			}

			// Get all trades
			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)

			// Initial buy should always be there if not a failed order case
			suite.Assert().GreaterOrEqual(len(trades), 1)

			if tc.shouldExecute {
				// Should have at least 2 trades (initial buy + our sell)
				suite.Assert().GreaterOrEqual(len(trades), 2)

				// Find our sell trade
				var sellTrade types.Trade
				for _, trade := range trades {
					if trade.Order.Side == types.PurchaseTypeSell {
						sellTrade = trade
						break
					}
				}

				// Check trade details
				suite.Assert().Equal(tc.executionPrice, sellTrade.Order.Price)
				suite.Assert().Equal(tc.expectedQuantity, sellTrade.Order.Quantity)
				suite.Assert().Equal(types.PurchaseTypeSell, sellTrade.Order.Side)
			} else {
				// Order should be pending - there should only be the initial buy trade
				sellTrades := 0
				for _, trade := range trades {
					if trade.Order.Side == types.PurchaseTypeSell {
						sellTrades++
					}
				}
				suite.Assert().Equal(0, sellTrades)
				suite.Assert().NotEmpty(suite.trading.pendingOrders)
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestPlaceMultipleOrders() {
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
		High:   100.0,
		Low:    90.0,
	}
	suite.trading.UpdateCurrentMarketData(marketData)

	orders := []types.ExecuteOrder{
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			OrderType:    types.OrderTypeLimit,
			Quantity:     10,
			Price:        95.0,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "reason",
			},
		},
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			OrderType:    types.OrderTypeLimit,
			Quantity:     5,
			Price:        93.0,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "reason",
			},
		},
	}

	err := suite.trading.PlaceMultipleOrders(orders)
	suite.Require().NoError(err)

	// Test with order that exceeds buying power - should not return error but create a failed order
	ordersWithFailingOrder := []types.ExecuteOrder{
		{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			OrderType:    types.OrderTypeLimit,
			Quantity:     1000000, // Very large quantity that exceeds buying power
			Price:        95.0,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "reason",
			},
		},
	}
	err = suite.trading.PlaceMultipleOrders(ordersWithFailingOrder)
	suite.Assert().NoError(err) // No error returned, order is just marked as failed

	// Verify that a failed order was stored
	allOrders, err := suite.state.GetAllOrders()
	suite.Require().NoError(err)
	var failedOrder *types.Order
	for i := range allOrders {
		if allOrders[i].Status == types.OrderStatusFailed {
			failedOrder = &allOrders[i]
			break
		}
	}
	suite.Require().NotNil(failedOrder)
	suite.Assert().Equal(types.OrderReasonInsufficientBuyPower, failedOrder.Reason.Reason)
}

func (suite *BacktestTradingTestSuite) TestGetOrderStatus() {
	// Setup test data
	completedOrder := types.Order{
		OrderID:      "completed",
		Symbol:       "AAPL",
		IsCompleted:  true,
		StrategyName: "test_strategy",
		Side:         types.PurchaseTypeBuy,
		Quantity:     10,
		Price:        95.0,
		Timestamp:    time.Now(),
		PositionType: types.PositionTypeLong,
		Reason: types.Reason{
			Reason:  "test",
			Message: "test",
		},
	}
	result, err := suite.state.Update([]types.Order{completedOrder})
	suite.Require().NoError(err)
	completedOrder.OrderID = result[0].Order.OrderID
	// should not test pending order here
	// Add the pending order to the trading system's pending orders
	suite.trading.pendingOrders = []types.ExecuteOrder{
		{
			ID:           "pending",
			Symbol:       "GOOGL",
			StrategyName: "test_strategy",
		},
	}

	tests := []struct {
		name           string
		orderID        string
		expectedStatus types.OrderStatus
		expectError    bool
	}{
		{
			name:           "Completed order",
			orderID:        completedOrder.OrderID,
			expectedStatus: types.OrderStatusFilled,
			expectError:    false,
		},
		{
			name:           "Non-existent order",
			orderID:        "non-existent",
			expectedStatus: types.OrderStatusFailed,
			expectError:    true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			status, err := suite.trading.GetOrderStatus(tc.orderID)
			if tc.expectError {
				suite.Assert().Error(err)
			} else {
				suite.Assert().NoError(err)
				suite.Assert().Equal(tc.expectedStatus, status)
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestDecimalPrecisionHandling() {
	// Test with different decimal precision settings
	testCases := []struct {
		name             string
		decimalPrecision int
		quantity         float64
		expectedQuantity float64
	}{
		{
			name:             "Integer precision (0 decimals)",
			decimalPrecision: 0,
			quantity:         1.9,
			expectedQuantity: 1.0,
		},
		{
			name:             "1 decimal precision",
			decimalPrecision: 1,
			quantity:         1.95,
			expectedQuantity: 1.9,
		},
		{
			name:             "2 decimal precision",
			decimalPrecision: 2,
			quantity:         1.956,
			expectedQuantity: 1.95,
		},
		{
			name:             "8 decimal precision (crypto standard)",
			decimalPrecision: 8,
			quantity:         0.123456789,
			expectedQuantity: 0.12345678,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Create a fresh trading instance with the proper decimal precision
			trading := &BacktestTrading{
				state:   suite.state,
				balance: 1000000.0, // Set a high balance to allow all trades
				marketData: types.MarketData{
					Symbol: "BTC/USD",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   40000.0,
					Low:    39000.0,
					Close:  39500.0,
				},
				pendingOrders:    []types.ExecuteOrder{},
				commission:       suite.commission,
				decimalPrecision: tc.decimalPrecision,
			}

			order := types.ExecuteOrder{
				Symbol:       "BTC/USD",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeLimit,
				Quantity:     tc.quantity,
				Price:        39500.0,
				StrategyName: "crypto_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "test",
				},
			}

			// Execute the order
			err = trading.PlaceOrder(order)
			suite.Require().NoError(err)

			// Retrieve trades and verify the quantity was correctly rounded
			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)
			suite.Require().NotEmpty(trades)

			suite.Assert().Equal(tc.expectedQuantity, trades[0].Order.Quantity)
		})
	}
}

// TestPlaceOrderWithDecimalPrecision tests placing orders with different decimal precision settings
func (suite *BacktestTradingTestSuite) TestPlaceOrderWithDecimalPrecision() {
	testCases := []struct {
		name             string
		decimalPrecision int
		orderQuantity    float64
		expectedQuantity float64
	}{
		{
			name:             "BTC/USD with precision rounding down",
			decimalPrecision: 8,
			orderQuantity:    0.123456789,
			expectedQuantity: 0.12345678,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)

			// Create a fresh trading instance
			trading := &BacktestTrading{
				state:   suite.state,
				balance: 1000000.0, // Set a high balance for crypto trading
				marketData: types.MarketData{
					Symbol: "BTC/USD",
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					High:   40000.0,
					Low:    39000.0,
					Close:  39500.0,
				},
				pendingOrders:    []types.ExecuteOrder{},
				commission:       suite.commission,
				decimalPrecision: tc.decimalPrecision,
			}

			order := types.ExecuteOrder{
				Symbol:       "BTC/USD",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeLimit,
				Quantity:     tc.orderQuantity,
				Price:        39500.0,
				StrategyName: "crypto_strategy",
				PositionType: types.PositionTypeLong,
				Reason: types.Reason{
					Reason:  "test",
					Message: "reason",
				},
			}

			// Place the order
			err = trading.PlaceOrder(order)
			suite.Require().NoError(err)

			// Verify the order quantity was rounded correctly
			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)
			suite.Require().NotEmpty(trades)

			suite.Assert().Equal(tc.expectedQuantity, trades[0].Order.Quantity)
		})
	}
}

func (suite *BacktestTradingTestSuite) TestGetPosition() {
	// Setup test data
	order := types.Order{
		Symbol:       "AAPL",
		Side:         types.PurchaseTypeBuy,
		Quantity:     100,
		Price:        100.0,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		StrategyName: "test_strategy",
		PositionType: types.PositionTypeLong,
		Reason: types.Reason{
			Reason:  "test",
			Message: "test",
		},
	}
	_, err := suite.state.Update([]types.Order{order})
	suite.Require().NoError(err)

	// Test getting existing position
	position, err := suite.trading.GetPosition("AAPL")
	suite.Require().NoError(err)
	suite.Assert().Equal("AAPL", position.Symbol)
	suite.Assert().Equal(float64(100), position.TotalLongPositionQuantity)

	// Test getting non-existent position
	position, err = suite.trading.GetPosition("GOOGL")
	suite.Require().NoError(err)
	suite.Assert().Equal("GOOGL", position.Symbol)
	suite.Assert().Equal(float64(0), position.TotalLongPositionQuantity)
}

func (suite *BacktestTradingTestSuite) TestGetPositions() {
	// Setup test data with unique order IDs
	orders := []types.Order{
		{
			OrderID:      "order1",
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        100.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "reason",
			},
		},
		{
			OrderID:      "order2",
			Symbol:       "GOOGL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     50,
			Price:        2000.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "reason",
			},
		},
	}

	// Update state with each order individually to avoid batch issues
	for _, order := range orders {
		_, err := suite.state.Update([]types.Order{order})
		suite.Require().NoError(err)
	}

	positions, err := suite.trading.GetPositions()
	suite.Require().NoError(err)
	suite.Assert().Len(positions, 2)

	// Verify positions
	for _, position := range positions {
		switch position.Symbol {
		case "AAPL":
			suite.Assert().Equal(float64(100), position.TotalLongPositionQuantity)
		case "GOOGL":
			suite.Assert().Equal(float64(50), position.TotalLongPositionQuantity)
		default:
			suite.Fail("Unexpected position symbol")
		}
	}
}

func (suite *BacktestTradingTestSuite) TestNewBacktestTrading() {
	// Test that the factory function correctly initializes the trading system
	state := suite.state
	initialBalance := 20000.0
	commission := commission_fee.NewZeroCommissionFee()
	decimalPrecision := 4

	tradingSystem := NewBacktestTrading(state, initialBalance, commission, decimalPrecision)

	// Type assertion to check the concrete implementation
	backtest, ok := tradingSystem.(*BacktestTrading)
	suite.Require().True(ok, "Expected trading system to be of type *BacktestTrading")

	// Verify the fields were correctly initialized
	suite.Assert().Equal(state, backtest.state)
	suite.Assert().Equal(initialBalance, backtest.balance)
	suite.Assert().Equal(commission, backtest.commission)
	suite.Assert().Equal(decimalPrecision, backtest.decimalPrecision)
	suite.Assert().Empty(backtest.pendingOrders)
}

func (suite *BacktestTradingTestSuite) TestGetAccountInfo() {
	// Test empty account (no positions)
	suite.Run("Empty account", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		suite.trading.UpdateBalance(10000.0)
		suite.trading.UpdateCurrentMarketData(types.MarketData{
			Symbol: "AAPL",
			Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			High:   100.0,
			Low:    90.0,
			Close:  95.0,
		})

		info, err := suite.trading.GetAccountInfo()
		suite.Require().NoError(err)
		suite.Assert().Equal(10000.0, info.Balance)
		suite.Assert().Equal(10000.0, info.Equity)
		suite.Assert().Equal(0.0, info.RealizedPnL)
		suite.Assert().Equal(0.0, info.UnrealizedPnL)
		suite.Assert().Equal(0.0, info.TotalFees)
	})

	// Test account with open position
	suite.Run("Account with open position", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		// Create an initial buy order to establish a position
		initialOrder := types.Order{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        90.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Fee:          1.0,
			Reason: types.Reason{
				Reason:  "test",
				Message: "test",
			},
		}
		_, err = suite.state.Update([]types.Order{initialOrder})
		suite.Require().NoError(err)

		// Update balance to reflect the purchase
		suite.trading.UpdateBalance(1000.0)
		suite.trading.UpdateCurrentMarketData(types.MarketData{
			Symbol: "AAPL",
			Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			High:   100.0,
			Low:    90.0,
			Close:  95.0, // Price increased from entry price of 90
		})

		info, err := suite.trading.GetAccountInfo()
		suite.Require().NoError(err)
		suite.Assert().Equal(1000.0, info.Balance)
		suite.Assert().Greater(info.Equity, info.Balance) // Should have unrealized profit
		suite.Assert().Greater(info.UnrealizedPnL, 0.0)   // Price went up
		suite.Assert().Equal(1.0, info.TotalFees)
	})
}

func (suite *BacktestTradingTestSuite) TestGetOpenOrders() {
	// Test with no pending orders
	suite.Run("No pending orders", func() {
		suite.trading.pendingOrders = []types.ExecuteOrder{}

		orders, err := suite.trading.GetOpenOrders()
		suite.Require().NoError(err)
		suite.Assert().Empty(orders)
	})

	// Test with pending orders
	suite.Run("With pending orders", func() {
		suite.trading.pendingOrders = []types.ExecuteOrder{
			{
				ID:           "order1",
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeLimit,
				Quantity:     10,
				Price:        85.0,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
			},
			{
				ID:           "order2",
				Symbol:       "GOOGL",
				Side:         types.PurchaseTypeBuy,
				OrderType:    types.OrderTypeLimit,
				Quantity:     5,
				Price:        2000.0,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
			},
		}

		orders, err := suite.trading.GetOpenOrders()
		suite.Require().NoError(err)
		suite.Assert().Len(orders, 2)
		suite.Assert().Equal("order1", orders[0].ID)
		suite.Assert().Equal("order2", orders[1].ID)
	})

	// Test that returned orders are a copy (modification doesn't affect original)
	suite.Run("Returns copy of orders", func() {
		suite.trading.pendingOrders = []types.ExecuteOrder{
			{
				ID:     "order1",
				Symbol: "AAPL",
			},
		}

		orders, err := suite.trading.GetOpenOrders()
		suite.Require().NoError(err)

		// Modify the returned slice
		orders[0].ID = "modified"

		// Original should not be affected
		suite.Assert().Equal("order1", suite.trading.pendingOrders[0].ID)
	})
}

func (suite *BacktestTradingTestSuite) TestGetTrades() {
	// Setup: Create some trades
	suite.Run("Get all trades without filter", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		// Create multiple trades
		orders := []types.Order{
			{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     100,
				Price:        90.0,
				Timestamp:    time.Now().Add(-2 * time.Hour),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
			{
				Symbol:       "GOOGL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     50,
				Price:        2000.0,
				Timestamp:    time.Now().Add(-1 * time.Hour),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
		}

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		// Get all trades
		trades, err := suite.trading.GetTrades(types.TradeFilter{})
		suite.Require().NoError(err)
		suite.Assert().Len(trades, 2)
	})

	// Test filter by symbol
	suite.Run("Filter by symbol", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		orders := []types.Order{
			{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     100,
				Price:        90.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
			{
				Symbol:       "GOOGL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     50,
				Price:        2000.0,
				Timestamp:    time.Now(),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
		}

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		// Filter by AAPL
		trades, err := suite.trading.GetTrades(types.TradeFilter{Symbol: "AAPL"})
		suite.Require().NoError(err)
		suite.Assert().Len(trades, 1)
		suite.Assert().Equal("AAPL", trades[0].Order.Symbol)
	})

	// Test limit
	suite.Run("Filter with limit", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		// Create 5 trades
		for i := 0; i < 5; i++ {
			order := types.Order{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     float64(10 + i),
				Price:        90.0,
				Timestamp:    time.Now().Add(time.Duration(i) * time.Minute),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			}
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		// Get only 3 trades
		trades, err := suite.trading.GetTrades(types.TradeFilter{Limit: 3})
		suite.Require().NoError(err)
		suite.Assert().Len(trades, 3)
	})

	// Test time range filter
	suite.Run("Filter by time range", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		baseTime := time.Now()

		orders := []types.Order{
			{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     100,
				Price:        90.0,
				Timestamp:    baseTime.Add(-3 * time.Hour),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
			{
				Symbol:       "AAPL",
				Side:         types.PurchaseTypeBuy,
				Quantity:     50,
				Price:        91.0,
				Timestamp:    baseTime.Add(-1 * time.Hour),
				IsCompleted:  true,
				StrategyName: "test_strategy",
				PositionType: types.PositionTypeLong,
				Reason:       types.Reason{Reason: "test", Message: "test"},
			},
		}

		for _, order := range orders {
			_, err := suite.state.Update([]types.Order{order})
			suite.Require().NoError(err)
		}

		// Filter to get only recent trades (last 2 hours)
		trades, err := suite.trading.GetTrades(types.TradeFilter{
			StartTime: baseTime.Add(-2 * time.Hour),
		})
		suite.Require().NoError(err)
		suite.Assert().Len(trades, 1)
		suite.Assert().Equal(50.0, trades[0].Order.Quantity)
	})
}

func (suite *BacktestTradingTestSuite) TestGetMaxBuyQuantity() {
	suite.Run("Valid price with sufficient balance", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		suite.trading.UpdateBalance(10000.0)
		suite.trading.UpdateCurrentMarketData(types.MarketData{
			Symbol: "AAPL",
			High:   100.0,
			Low:    90.0,
		})

		// At price 100, with balance 10000, max quantity should be 100
		maxQty, err := suite.trading.GetMaxBuyQuantity("AAPL", 100.0)
		suite.Require().NoError(err)
		suite.Assert().Equal(100.0, maxQty)
	})

	suite.Run("Zero balance returns zero", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		suite.trading.UpdateBalance(0)

		maxQty, err := suite.trading.GetMaxBuyQuantity("AAPL", 100.0)
		suite.Require().NoError(err)
		suite.Assert().Equal(0.0, maxQty)
	})

	suite.Run("Zero price returns error", func() {
		suite.trading.UpdateBalance(10000.0)

		_, err := suite.trading.GetMaxBuyQuantity("AAPL", 0)
		suite.Assert().Error(err)
	})

	suite.Run("Negative price returns error", func() {
		suite.trading.UpdateBalance(10000.0)

		_, err := suite.trading.GetMaxBuyQuantity("AAPL", -10.0)
		suite.Assert().Error(err)
	})

	suite.Run("Respects decimal precision", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		trading := &BacktestTrading{
			state:            suite.state,
			balance:          1000.0,
			commission:       suite.commission,
			decimalPrecision: 2,
		}

		// 1000 / 33 = 30.303..., should round down to 30.30
		maxQty, err := trading.GetMaxBuyQuantity("AAPL", 33.0)
		suite.Require().NoError(err)
		suite.Assert().Equal(30.30, maxQty)
	})
}

func (suite *BacktestTradingTestSuite) TestGetMaxSellQuantity() {
	suite.Run("With existing position", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		// Create a position by adding a buy order
		order := types.Order{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        90.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason:       types.Reason{Reason: "test", Message: "test"},
		}
		_, err = suite.state.Update([]types.Order{order})
		suite.Require().NoError(err)

		maxQty, err := suite.trading.GetMaxSellQuantity("AAPL")
		suite.Require().NoError(err)
		suite.Assert().Equal(100.0, maxQty)
	})

	suite.Run("No position returns zero", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		maxQty, err := suite.trading.GetMaxSellQuantity("AAPL")
		suite.Require().NoError(err)
		suite.Assert().Equal(0.0, maxQty)
	})

	suite.Run("Different symbol returns zero", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		// Create a position for AAPL
		order := types.Order{
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        90.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason:       types.Reason{Reason: "test", Message: "test"},
		}
		_, err = suite.state.Update([]types.Order{order})
		suite.Require().NoError(err)

		// Query for GOOGL should return 0
		maxQty, err := suite.trading.GetMaxSellQuantity("GOOGL")
		suite.Require().NoError(err)
		suite.Assert().Equal(0.0, maxQty)
	})

	suite.Run("Respects decimal precision", func() {
		err := suite.state.Cleanup()
		suite.Require().NoError(err)
		err = suite.state.Initialize()
		suite.Require().NoError(err)

		trading := &BacktestTrading{
			state:            suite.state,
			balance:          10000.0,
			commission:       suite.commission,
			decimalPrecision: 2,
		}

		// Create a position with a fractional quantity
		order := types.Order{
			Symbol:       "BTC/USD",
			Side:         types.PurchaseTypeBuy,
			Quantity:     1.23456789,
			Price:        40000.0,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			StrategyName: "test_strategy",
			PositionType: types.PositionTypeLong,
			Reason:       types.Reason{Reason: "test", Message: "test"},
		}
		_, err = suite.state.Update([]types.Order{order})
		suite.Require().NoError(err)

		maxQty, err := trading.GetMaxSellQuantity("BTC/USD")
		suite.Require().NoError(err)
		suite.Assert().Equal(1.23, maxQty) // Rounded to 2 decimal places
	})
}

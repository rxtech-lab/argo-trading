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
				suite.Assert().Equal(tc.executedOrderNumber, len(allOrders))
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
						Reason: "test",
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
						Reason: "test",
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
						Reason: "test",
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
						Reason: "test",
					},
				},
				decimalPrecision:     0,
				expectError:          true,
				executedOrderNumber:  0,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
			},
			{
				name: "Order quantity exceeds selling power",
				marketData: types.MarketData{
					Symbol: "AAPL",
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
						Reason: "test",
					},
				},
				decimalPrecision:     0,
				expectError:          true,
				executedOrderNumber:  0,
				pendingOrderNumber:   0,
				marketDataAfterOrder: optional.None[types.MarketData](),
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
						Reason: "test",
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
						Reason: "test",
					},
				},
				marketDataAfterOrder: optional.Some(types.MarketData{
					Symbol: "AAPL",
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
						Reason: "test",
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
		High:   100.0,
		Low:    90.0,
	}
	suite.trading.UpdateCurrentMarketData(marketData)
	
	// Calculate expected average price
	avgPrice := (marketData.High + marketData.Low) / 2 // 95.0
	
	testCases := []struct {
		name         string
		balance      float64
		quantity     float64
		expectError  bool
		errorMessage string
	}{
		{
			name:         "Successful buy - sufficient balance",
			balance:      10000.0,
			quantity:     10.0,  // Total cost: 950.0 (10 * 95.0)
			expectError:  false,
		},
		{
			name:         "Failed buy - insufficient balance",
			balance:      500.0,
			quantity:     10.0, // Total cost: 950.0 (10 * 95.0)
			expectError:  true,
			errorMessage: "market buy order cost (950.00) exceeds available balance (500.00)",
		},
		{
			name:         "Successful buy - exact balance",
			balance:      950.0,
			quantity:     10.0, // Total cost: 950.0 (10 * 95.0)
			expectError:  false,
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
					Reason: "test",
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
				
				// Verify the trade was executed correctly
				trades, err := suite.state.GetAllTrades()
				suite.Require().NoError(err)
				suite.Require().NotEmpty(trades)
				
				// Check that the trade was executed at the average price
				suite.Assert().Equal(avgPrice, trades[0].Order.Price)
				suite.Assert().Equal(tc.quantity, trades[0].Order.Quantity)
				suite.Assert().Equal(types.PurchaseTypeBuy, trades[0].Order.Side)
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
	}
	
	// Setup market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		High:   100.0,
		Low:    90.0,
	}
	
	// Calculate expected average price
	avgPrice := (marketData.High + marketData.Low) / 2 // 95.0
	
	testCases := []struct {
		name            string
		sellQuantity    float64
		expectError     bool
		expectedQuantity float64 // Expected quantity to be sold after adjustment
	}{
		{
			name:            "Successful sell - within holdings",
			sellQuantity:    20.0,
			expectError:     false,
			expectedQuantity: 20.0,
		},
		{
			name:            "Successful sell - exact holdings",
			sellQuantity:    50.0,
			expectError:     false,
			expectedQuantity: 50.0,
		},
		{
			name:            "Successful sell - adjusted to max holdings",
			sellQuantity:    100.0, // More than we have
			expectError:     false,
			expectedQuantity: 50.0, // Should adjust to available 50.0
		},
		{
			name:            "Failed sell - no holdings",
			sellQuantity:    10.0,
			expectError:     true, // No shares in a clean state
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)
			
			// Only add initial position for non-error cases or when testing oversize sells
			if !tc.expectError {
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
					Reason: "test",
				},
			}
			
			// Execute the order
			err = suite.trading.PlaceOrder(order)
			
			if tc.expectError {
				suite.Assert().Error(err)
			} else {
				suite.Assert().NoError(err)
				
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
				
				// Check that the trade was adjusted properly
				suite.Assert().Equal(tc.expectedQuantity, sellTrade.Order.Quantity)
				// Check that the avg price was used
				suite.Assert().Equal((marketData.High+marketData.Low)/2, sellTrade.Order.Price)
			}
		})
	}
}

func (suite *BacktestTradingTestSuite) TestPlaceOrder_With_Limit_Price_Order_Buy() {
	// Setup initial market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		High:   100.0,
		Low:    90.0,
	}
	
	testCases := []struct {
		name            string
		balance         float64
		limitPrice      float64
		quantity        float64
		updatedMarketData *types.MarketData  // To simulate price changes
		expectError     bool
		errorMessage    string
		shouldExecute   bool      // Should the order execute immediately or be pending?
		executionPrice  float64   // Expected execution price
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
			name:           "Buy limit above current market - order pending",
			balance:        10000.0,
			limitPrice:     85.0, // Below current market low of 90
			quantity:       10.0,
			expectError:    false,
			shouldExecute:  false, // Should be pending
		},
		{
			name:            "Buy limit above market initially pending, then executed",
			balance:         10000.0,
			limitPrice:      85.0, // Below current market
			quantity:        10.0,
			updatedMarketData: &types.MarketData{
				Symbol: "AAPL",
				High:   90.0,
				Low:    80.0, // Now below our limit price
			},
			expectError:    false,
			shouldExecute:  true,
			executionPrice: 85.0, // Should use our limit price
		},
		{
			name:           "Buy limit - insufficient balance",
			balance:        500.0,
			limitPrice:     95.0,
			quantity:       10.0, // Total cost: 950.0
			expectError:    true,
			errorMessage:   "limit buy order cost (950.00) exceeds available balance (500.00)",
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
					Reason: "test",
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
		Quantity:     initialQuantity,
		Price:        90.0,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		StrategyName: "test_strategy",
	}
	
	// Setup market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		High:   100.0,
		Low:    90.0,
	}
	
	testCases := []struct {
		name             string
		limitPrice       float64
		sellQuantity     float64
		updatedMarketData *types.MarketData // To simulate price changes
		expectError      bool
		expectedQuantity float64            // Expected quantity after adjustment
		shouldExecute    bool               // Should execute immediately or be pending
		executionPrice   float64            // Expected execution price
	}{
		{
			name:             "Sell limit below current high - execute immediately",
			limitPrice:       95.0,
			sellQuantity:     20.0,
			expectError:      false,
			expectedQuantity: 20.0,
			shouldExecute:    true,
			executionPrice:   95.0, // Should use limit price
		},
		{
			name:             "Sell limit above current high - order pending",
			limitPrice:       110.0,
			sellQuantity:     20.0,
			expectError:      false,
			expectedQuantity: 20.0,
			shouldExecute:    false, // Should be pending
		},
		{
			name:             "Sell limit initially pending, then executed",
			limitPrice:       110.0,
			sellQuantity:     20.0,
			updatedMarketData: &types.MarketData{
				Symbol: "AAPL",
				High:   120.0, // Now above our limit price
				Low:    100.0,
			},
			expectError:      false,
			expectedQuantity: 20.0,
			shouldExecute:    true,
			executionPrice:   110.0, // Should use our limit price
		},
		{
			name:             "Sell limit with quantity adjustment",
			limitPrice:       95.0,
			sellQuantity:     100.0, // More than we have
			expectError:      false,
			expectedQuantity: 50.0, // Should adjust to available
			shouldExecute:    true,
			executionPrice:   95.0,
		},
		{
			name:             "Sell limit with no holdings",
			limitPrice:       95.0,
			sellQuantity:     10.0,
			expectError:      true, // No shares in a clean state
		},
	}
	
	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Reset state for each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)
			err = suite.state.Initialize()
			suite.Require().NoError(err)
			
			// Only add initial position for non-error cases or when testing oversize sells
			if !tc.expectError {
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
					Reason: "test",
				},
			}
			
			// Place the order
			err = suite.trading.PlaceOrder(order)
			
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}
			
			suite.Assert().NoError(err)
			
			// If we have updated market data, simulate a price change
			if tc.updatedMarketData != nil {
				suite.trading.UpdateCurrentMarketData(*tc.updatedMarketData)
			}
			
			// Get all trades
			trades, err := suite.state.GetAllTrades()
			suite.Require().NoError(err)
			
			// Initial buy should always be there if not an error case
			if !tc.expectError {
				suite.Assert().GreaterOrEqual(len(trades), 1)
			}
			
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
				Reason: "test",
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
				Reason: "test",
			},
		},
	}

	err := suite.trading.PlaceMultipleOrders(orders)
	suite.Require().NoError(err)

	// Test with invalid order
	invalidOrders := append(orders, types.ExecuteOrder{
		Symbol:       "AAPL",
		Side:         types.PurchaseTypeBuy,
		OrderType:    types.OrderTypeLimit,
		Quantity:     1000000, // Very large quantity
		Price:        95.0,
		StrategyName: "test_strategy",
		PositionType: types.PositionTypeLong,
	})
	err = suite.trading.PlaceMultipleOrders(invalidOrders)
	suite.Assert().Error(err)
}

func (suite *BacktestTradingTestSuite) TestGetOrderStatus() {
	// Setup test data
	completedOrder := types.Order{
		OrderID:      "completed",
		Symbol:       "AAPL",
		IsCompleted:  true,
		StrategyName: "test_strategy",
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
					Reason: "test",
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
					Reason: "test",
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
	}
	_, err := suite.state.Update([]types.Order{order})
	suite.Require().NoError(err)

	// Test getting existing position
	position, err := suite.trading.GetPosition("AAPL")
	suite.Require().NoError(err)
	suite.Assert().Equal("AAPL", position.Symbol)
	suite.Assert().Equal(float64(100), position.Quantity)

	// Test getting non-existent position
	position, err = suite.trading.GetPosition("GOOGL")
	suite.Require().NoError(err)
	suite.Assert().Equal("GOOGL", position.Symbol)
	suite.Assert().Equal(float64(0), position.Quantity)
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
			suite.Assert().Equal(float64(100), position.Quantity)
		case "GOOGL":
			suite.Assert().Equal(float64(50), position.Quantity)
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

package engine

import (
	"testing"
	"time"

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

func (suite *BacktestTradingTestSuite) TestPlaceOrder() {
	tests := []struct {
		name             string
		marketData       types.MarketData
		order            types.ExecuteOrder
		decimalPrecision int
		expectError      bool
	}{
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
				OrderType:    types.OrderTypeLimit,
				Quantity:     10,
				Price:        95.0,
				StrategyName: "test_strategy",
				Reason: types.Reason{
					Reason: "test",
				},
			},
			decimalPrecision: 0,
			expectError:      false,
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
				OrderType:    types.OrderTypeLimit,
				Quantity:     10,
				Price:        110.0,
				StrategyName: "test_strategy",
				Reason: types.Reason{
					Reason: "test",
				},
			},
			decimalPrecision: 0,
			expectError:      false,
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
				OrderType:    types.OrderTypeLimit,
				Quantity:     10,
				Price:        85.0,
				StrategyName: "test_strategy",
				Reason: types.Reason{
					Reason: "test",
				},
			},
			decimalPrecision: 0,
			expectError:      true,
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
				OrderType:    types.OrderTypeLimit,
				Quantity:     1000000, // Very large quantity
				Price:        95.0,
				StrategyName: "test_strategy",
				Reason: types.Reason{
					Reason: "test",
				},
			},
			decimalPrecision: 0,
			expectError:      true,
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
				OrderType:    types.OrderTypeLimit,
				Quantity:     1000, // Selling more than held
				Price:        95.0,
				StrategyName: "test_strategy",
				Reason: types.Reason{
					Reason: "test",
				},
			},
			decimalPrecision: 0,
			expectError:      true,
		},
	}

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

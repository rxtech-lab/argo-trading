package engine

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/suite"
)

// BacktestStateTestSuite is a test suite for BacktestState
type BacktestStateTestSuite struct {
	suite.Suite
	state  *BacktestState
	logger *logger.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *BacktestStateTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
	suite.state = NewBacktestState(suite.logger)
	suite.Require().NotNil(suite.state)
}

// TearDownSuite runs once after all tests in the suite
func (suite *BacktestStateTestSuite) TearDownSuite() {
	if suite.state != nil && suite.state.db != nil {
		suite.state.db.Close()
	}
}

// SetupTest runs before each test
func (suite *BacktestStateTestSuite) SetupTest() {
	// Initialize the state before each test
	err := suite.state.Initialize()
	suite.Require().NoError(err)
}

// TearDownTest runs after each test
func (suite *BacktestStateTestSuite) TearDownTest() {
	// Cleanup the state after each test
	err := suite.state.Cleanup()
	suite.Require().NoError(err)
}

// TestBacktestStateSuite runs the test suite
func TestBacktestStateSuite(t *testing.T) {
	suite.Run(t, new(BacktestStateTestSuite))
}

func (suite *BacktestStateTestSuite) TestUpdate() {

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
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       100.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expectedTrades: []types.Trade{
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:        "AAPL",
					Quantity:      100,
					OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: 0,
			},
		},
		{
			name: "Single entry and exit with fee",
			orders: []types.Order{
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       100.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeSell,
					Quantity:    100,
					Price:       110.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expectedTrades: []types.Trade{
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					PnL:           (110*100 - 1) - (100*100 + 1),
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:        "AAPL",
					Quantity:      0,
					OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: (110*100 - 1) - (100*100 + 1),
			},
		},
		{
			name: "Single entry and partial close with fee",
			orders: []types.Order{
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       100.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
				{
					Symbol:    "AAPL",
					OrderType: types.OrderTypeSell,
					Quantity:  50,
					Price:     110.0,
					Fee:       1.0,
					Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
			},
			expectedTrades: []types.Trade{
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   50,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					PnL:           498.5,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:        "AAPL",
					Quantity:      50,
					OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				//((110*50-1)/50 - (100*100+1)/100) * 50
				TotalPnL: 498.5,
			},
		},
		{
			name: "Multiple entry and close long position with fee",
			orders: []types.Order{
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       100.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       90.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeBuy,
					Quantity:    100,
					Price:       80.0,
					Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeSell,
					Quantity:    100,
					Price:       110.0,
					Timestamp:   time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeSell,
					Quantity:    100,
					Price:       120.0,
					Timestamp:   time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeSell,
					Quantity:    100,
					Price:       130.0,
					Timestamp:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
					Fee:          1.0,
				},
			},
			expectedTrades: []types.Trade{
				{
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 90.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 80.0,
					Fee:           1.0,
					PnL:           0,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 13, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 110.0,
					Fee:           1.0,
					// ((110*100-1)/100 - (100*100+1 + 90*100+1 + 80*100+1)/300) * 100
					PnL: 1998,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 120.0,
					Fee:           1.0,
					//  ((120*100-1)/100 - (100*100+1 + 90*100+1 + 80*100+1)/300) * 100
					PnL: 2998,
				},
				{
					ExecutedAt:    time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 130.0,
					Fee:           1.0,
					// ((130*100-1)/100 -  (100*100+1 + 90*100+1 + 80*100+1)/300) * 100
					PnL: 3998,
				},
			},
			expectedPosition: ExpectPosition{
				Position: types.Position{
					Symbol:        "AAPL",
					Quantity:      0,
					OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				},
				TotalPnL: 8994,
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
			suite.Assert().Equal(tc.expectedPosition.Quantity, position.Quantity, "Position quantity mismatch")
			suite.Assert().Equal(tc.expectedPosition.OpenTimestamp.UTC(), position.OpenTimestamp.UTC(), "Position open timestamp mismatch")
			suite.Assert().Equal(tc.expectedPosition.TotalPnL, position.GetTotalPnL(), "Position total PnL mismatch")

			// Verify results
			for i, result := range allResults {
				// Verify order
				suite.Assert().Equal(tc.orders[i].Symbol, result.Order.Symbol, "Result order symbol mismatch")
				suite.Assert().Equal(tc.orders[i].OrderType, result.Order.OrderType, "Result order type mismatch")
				suite.Assert().Equal(tc.orders[i].Quantity, result.Order.Quantity, "Result order quantity mismatch")
				suite.Assert().Equal(tc.orders[i].Price, result.Order.Price, "Result order price mismatch")
				suite.Assert().Equal(tc.orders[i].Timestamp.UTC(), result.Order.Timestamp.UTC(), "Result order timestamp mismatch")
				suite.Assert().Equal(tc.orders[i].IsCompleted, result.Order.IsCompleted, "Result order is_completed mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Reason, result.Order.Reason.Reason, "Result order reason mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Message, result.Order.Reason.Message, "Result order message mismatch")
				suite.Assert().Equal(tc.orders[i].StrategyName, result.Order.StrategyName, "Result order strategy name mismatch")

				// Verify trade
				suite.Assert().Equal(tc.orders[i].Symbol, result.Trade.Order.Symbol, "Result trade symbol mismatch")
				suite.Assert().Equal(tc.orders[i].OrderType, result.Trade.Order.OrderType, "Result trade type mismatch")
				suite.Assert().Equal(tc.orders[i].Quantity, result.Trade.ExecutedQty, "Result trade quantity mismatch")
				suite.Assert().Equal(tc.orders[i].Price, result.Trade.ExecutedPrice, "Result trade price mismatch")
				suite.Assert().Equal(tc.orders[i].Timestamp.UTC(), result.Trade.ExecutedAt.UTC(), "Result trade timestamp mismatch")

				// Verify IsNewPosition
				if i == 0 && tc.orders[i].OrderType == types.OrderTypeBuy {
					suite.Assert().True(result.IsNewPosition, "Expected IsNewPosition to be true for first buy order")
				} else {
					suite.Assert().False(result.IsNewPosition, "Expected IsNewPosition to be false for subsequent orders")
				}
			}
		})
	}
}

func (suite *BacktestStateTestSuite) TestWrite() {
	// Create a temporary directory for test files
	tmpDir := suite.T().TempDir()

	// Create some test data
	orders := []types.Order{
		{
			Symbol:      "AAPL",
			OrderType:   types.OrderTypeBuy,
			Quantity:    100,
			Price:       100.0,
			Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			IsCompleted: true,
			Reason: types.Reason{
				Reason:  "test",
				Message: "test message",
			},
			StrategyName: "test_strategy",
		},
		{
			Symbol:      "AAPL",
			OrderType:   types.OrderTypeSell,
			Quantity:    50,
			Price:       110.0,
			Timestamp:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
			IsCompleted: true,
			Reason: types.Reason{
				Reason:  "test",
				Message: "test message",
			},
			StrategyName: "test_strategy",
		},
	}

	// Process orders to create test data
	for _, order := range orders {
		_, err := suite.state.Update([]types.Order{order})
		suite.Require().NoError(err)
	}

	// Test writing to Parquet files
	err := suite.state.Write(tmpDir)
	suite.Require().NoError(err)

	// Verify that all three files were created
	tradesPath := filepath.Join(tmpDir, "trades.parquet")
	ordersPath := filepath.Join(tmpDir, "orders.parquet")

	// Check if files exist
	suite.Require().FileExists(tradesPath, "trades.parquet file should exist")
	suite.Require().FileExists(ordersPath, "orders.parquet file should exist")

	// Verify the data in the files using DuckDB
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	// Read and verify trades
	_, err = db.Exec(fmt.Sprintf("CREATE VIEW trades AS SELECT * FROM read_parquet('%s')", tradesPath))
	suite.Require().NoError(err)

	var tradeCount int
	err = db.QueryRow("SELECT COUNT(*) FROM trades").Scan(&tradeCount)
	suite.Require().NoError(err)
	suite.Require().Equal(2, tradeCount, "Should have 2 trades")

	// Read and verify orders
	_, err = db.Exec(fmt.Sprintf("CREATE VIEW orders AS SELECT * FROM read_parquet('%s')", ordersPath))
	suite.Require().NoError(err)

	var orderCount int
	err = db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&orderCount)
	suite.Require().NoError(err)
	suite.Require().Equal(2, orderCount, "Should have 2 orders")

	// Verify data in trades
	var symbol string
	var orderTypeStr string
	var quantity float64
	var price float64
	err = db.QueryRow(`
		SELECT symbol, order_type, quantity, price 
		FROM trades 
		ORDER BY timestamp ASC 
		LIMIT 1
	`).Scan(&symbol, &orderTypeStr, &quantity, &price)
	suite.Require().NoError(err)
	suite.Require().Equal("AAPL", symbol, "Trade symbol mismatch")
	suite.Require().Equal(string(types.OrderTypeBuy), orderTypeStr, "Trade order type mismatch")
	suite.Require().Equal(100.0, quantity, "Trade quantity mismatch")
	suite.Require().Equal(100.0, price, "Trade price mismatch")

	// Verify data in orders
	err = db.QueryRow(`
		SELECT symbol, order_type, quantity, price 
		FROM orders 
		ORDER BY timestamp ASC 
		LIMIT 1
	`).Scan(&symbol, &orderTypeStr, &quantity, &price)
	suite.Require().NoError(err)
	suite.Require().Equal("AAPL", symbol, "Order symbol mismatch")
	suite.Require().Equal(string(types.OrderTypeBuy), orderTypeStr, "Order type mismatch")
	suite.Require().Equal(100.0, quantity, "Order quantity mismatch")
	suite.Require().Equal(100.0, price, "Order price mismatch")
}

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
	tests := []struct {
		name             string
		orders           []types.Order
		expectedTrades   []types.Trade
		expectedPosition types.Position
		expectError      bool
	}{
		{
			name: "Open long position",
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
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
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
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Commission:    0.0,
					PnL:           0.0,
				},
			},
			expectedPosition: types.Position{
				Symbol:        "AAPL",
				Quantity:      100,
				AveragePrice:  100.0,
				OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			expectError: false,
		},
		{
			name: "Close long position with profit",
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
				},
				{
					Symbol:      "AAPL",
					OrderType:   types.OrderTypeSell,
					Quantity:    100,
					Price:       110.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason:  "test",
						Message: "test message",
					},
					StrategyName: "test_strategy",
				},
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
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
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Commission:    0.0,
					PnL:           0.0,
				},
				{
					Order: types.Order{
						Symbol:      "AAPL",
						OrderType:   types.OrderTypeSell,
						Quantity:    100,
						Price:       110.0,
						Timestamp:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason:  "test",
							Message: "test message",
						},
						StrategyName: "test_strategy",
					},
					ExecutedAt:    time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 110.0,
					Commission:    0.0,
					PnL:           1000.0, // (110 - 100) * 100
				},
			},
			expectedPosition: types.Position{}, // Position should be closed
			expectError:      false,
		},
		{
			name: "Partial close of long position",
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
			},
			expectedTrades: []types.Trade{
				{
					Order: types.Order{
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
					ExecutedAt:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					ExecutedQty:   100,
					ExecutedPrice: 100.0,
					Commission:    0.0,
					PnL:           0.0,
				},
				{
					Order: types.Order{
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
					ExecutedAt:    time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					ExecutedQty:   50,
					ExecutedPrice: 110.0,
					Commission:    0.0,
					PnL:           500.0, // (110 - 100) * 50
				},
			},
			expectedPosition: types.Position{
				Symbol:        "AAPL",
				Quantity:      50,
				AveragePrice:  100.0,
				OpenTimestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			},
			expectError: false,
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
					suite.Assert().Equal(expected.Order.Symbol, trades[i].Order.Symbol, "Symbol mismatch")
					suite.Assert().Equal(expected.Order.OrderType, trades[i].Order.OrderType, "Order type mismatch")
					suite.Assert().Equal(expected.Order.Quantity, trades[i].Order.Quantity, "Quantity mismatch")
					suite.Assert().Equal(expected.Order.Price, trades[i].Order.Price, "Price mismatch")
					suite.Assert().Equal(expected.Order.Timestamp.UTC(), trades[i].Order.Timestamp.UTC(), "Timestamp mismatch")
					suite.Assert().Equal(expected.Order.IsCompleted, trades[i].Order.IsCompleted, "IsCompleted mismatch")
					suite.Assert().Equal(expected.Order.Reason.Reason, trades[i].Order.Reason.Reason, "Reason mismatch")
					suite.Assert().Equal(expected.Order.Reason.Message, trades[i].Order.Reason.Message, "Message mismatch")
					suite.Assert().Equal(expected.Order.StrategyName, trades[i].Order.StrategyName, "Strategy name mismatch")
					suite.Assert().Equal(expected.ExecutedAt.UTC(), trades[i].ExecutedAt.UTC(), "ExecutedAt mismatch")
					suite.Assert().Equal(expected.ExecutedQty, trades[i].ExecutedQty, "ExecutedQty mismatch")
					suite.Assert().Equal(expected.ExecutedPrice, trades[i].ExecutedPrice, "ExecutedPrice mismatch")
					suite.Assert().Equal(expected.Commission, trades[i].Commission, "Commission mismatch")
					suite.Assert().Equal(expected.PnL, trades[i].PnL, "PnL mismatch")
				}
			}

			// Verify positions
			if tc.expectedPosition != (types.Position{}) {
				position, err := suite.state.GetPosition(tc.expectedPosition.Symbol)
				suite.Assert().NoError(err)
				suite.Assert().Equal(tc.expectedPosition.Symbol, position.Symbol, "Position symbol mismatch")
				suite.Assert().Equal(tc.expectedPosition.Quantity, position.Quantity, "Position quantity mismatch")
				suite.Assert().Equal(tc.expectedPosition.AveragePrice, position.AveragePrice, "Position average price mismatch")
				suite.Assert().Equal(tc.expectedPosition.OpenTimestamp.UTC(), position.OpenTimestamp.UTC(), "Position open timestamp mismatch")
			} else {
				// Verify position is closed
				position, err := suite.state.GetPosition("AAPL")
				suite.Assert().NoError(err)
				suite.Assert().Equal(types.Position{}, position, "Expected no position")
			}

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

				// Verify position
				if tc.expectedPosition != (types.Position{}) {
					if i == len(tc.orders)-1 {
						// Only verify final position state
						suite.Assert().Equal(tc.expectedPosition.Symbol, result.Position.Symbol, "Result position symbol mismatch")
						suite.Assert().Equal(tc.expectedPosition.Quantity, result.Position.Quantity, "Result position quantity mismatch")
						suite.Assert().Equal(tc.expectedPosition.AveragePrice, result.Position.AveragePrice, "Result position average price mismatch")
						suite.Assert().Equal(tc.expectedPosition.OpenTimestamp.UTC(), result.Position.OpenTimestamp.UTC(), "Result position open timestamp mismatch")
					}
				}

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
	positionsPath := filepath.Join(tmpDir, "positions.parquet")
	ordersPath := filepath.Join(tmpDir, "orders.parquet")

	// Check if files exist
	suite.Require().FileExists(tradesPath, "trades.parquet file should exist")
	suite.Require().FileExists(positionsPath, "positions.parquet file should exist")
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

	// Read and verify positions
	_, err = db.Exec(fmt.Sprintf("CREATE VIEW positions AS SELECT * FROM read_parquet('%s')", positionsPath))
	suite.Require().NoError(err)

	var positionCount int
	err = db.QueryRow("SELECT COUNT(*) FROM positions").Scan(&positionCount)
	suite.Require().NoError(err)
	suite.Require().Equal(1, positionCount, "Should have 1 position")

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

	// Verify data in positions
	err = db.QueryRow(`
		SELECT symbol, quantity, average_price 
		FROM positions 
		LIMIT 1
	`).Scan(&symbol, &quantity, &price)
	suite.Require().NoError(err)
	suite.Require().Equal("AAPL", symbol, "Position symbol mismatch")
	suite.Require().Equal(50.0, quantity, "Position quantity mismatch")
	suite.Require().Equal(100.0, price, "Position average price mismatch")

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

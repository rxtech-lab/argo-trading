package engine

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/strategy"
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
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
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
						Side:        types.PurchaseTypeSell,
						Quantity:    100,
						Price:       110.0,
						Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
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
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:   "order2",
					Symbol:    "AAPL",
					Side:      types.PurchaseTypeSell,
					Quantity:  50,
					Price:     110.0,
					Fee:       1.0,
					Timestamp: time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
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
						Side:        types.PurchaseTypeSell,
						Quantity:    50,
						Price:       110.0,
						Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
						IsCompleted: true,
						Reason: types.Reason{
							Reason: "test",
						},
					},
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
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					PnL:           1998,
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
					PnL:           2998,
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
					PnL:           3998,
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
				suite.Assert().Equal(tc.orders[i].Side, result.Order.Side, "Result order type mismatch")
				suite.Assert().Equal(tc.orders[i].Quantity, result.Order.Quantity, "Result order quantity mismatch")
				suite.Assert().Equal(tc.orders[i].Price, result.Order.Price, "Result order price mismatch")
				suite.Assert().Equal(tc.orders[i].Timestamp.UTC(), result.Order.Timestamp.UTC(), "Result order timestamp mismatch")
				suite.Assert().Equal(tc.orders[i].IsCompleted, result.Order.IsCompleted, "Result order is_completed mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Reason, result.Order.Reason.Reason, "Result order reason mismatch")
				suite.Assert().Equal(tc.orders[i].Reason.Message, result.Order.Reason.Message, "Result order message mismatch")
				suite.Assert().Equal(tc.orders[i].StrategyName, result.Order.StrategyName, "Result order strategy name mismatch")

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

func (suite *BacktestStateTestSuite) TestWrite() {
	// Create a temporary directory for test files
	tmpDir := suite.T().TempDir()

	// Create some test data
	orders := []types.Order{
		{
			OrderID:     "order1",
			Symbol:      "AAPL",
			Side:        types.PurchaseTypeBuy,
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
			OrderID:     "order2",
			Symbol:      "AAPL",
			Side:        types.PurchaseTypeSell,
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
	suite.Require().Equal(string(types.PurchaseTypeBuy), orderTypeStr, "Trade order type mismatch")
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
	suite.Require().Equal(string(types.PurchaseTypeBuy), orderTypeStr, "Order type mismatch")
	suite.Require().Equal(100.0, quantity, "Order quantity mismatch")
	suite.Require().Equal(100.0, price, "Order price mismatch")
}

// MockDataSource implements datasource.DataSource for testing
type MockDataSource struct {
	lastData map[string]types.MarketData
}

func (m *MockDataSource) Initialize(path string) error { return nil }
func (m *MockDataSource) ReadAll(start, end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return nil
}
func (m *MockDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[datasource.Interval]) ([]types.MarketData, error) {
	return nil, nil
}
func (m *MockDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	if data, ok := m.lastData[symbol]; ok {
		return data, nil
	}
	return types.MarketData{}, fmt.Errorf("no data for symbol %s", symbol)
}
func (m *MockDataSource) ExecuteSQL(query string, params ...interface{}) ([]datasource.SQLResult, error) {
	return nil, nil
}
func (m *MockDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	return 0, nil
}
func (m *MockDataSource) Close() error { return nil }

func (suite *BacktestStateTestSuite) TestGetStats() {
	// Create mock data source
	mockSource := &MockDataSource{
		lastData: map[string]types.MarketData{
			"AAPL": {
				Symbol: "AAPL",
				Close:  120.0,
			},
			"GOOGL": {
				Symbol: "GOOGL",
				Close:  2100.0,
			},
		},
	}

	tests := []struct {
		name          string
		orders        []types.Order
		expectedStats []types.TradeStats
		expectError   bool
	}{
		{
			name: "Single symbol with multiple trades",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    50,
					Price:       110.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "AAPL",
					TradePnl: types.TradePnl{
						RealizedPnL:   498.5, // ((50 * 110 -1)/50 - (100 * 100 + 1)/100)*50
						TotalPnL:      1498,
						UnrealizedPnL: 999.5, // ((50 * 120)/50 - (100 * 100 + 1)/100)*50
						MaximumLoss:   0,
						MaximumProfit: 498.5,
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        2,
						NumberOfWinningTrades: 1,
						NumberOfLosingTrades:  0,
						WinRate:               0.5,
						MaxDrawdown:           0,
					},
					TotalFees: 2.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 1, // 1 hour between buy and sell
						Max: 1,
						Avg: 1,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Multiple symbols with trades",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "GOOGL",
					Side:        types.PurchaseTypeBuy,
					Quantity:    50,
					Price:       2000.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "AAPL",
					TradePnl: types.TradePnl{
						RealizedPnL:   0,
						TotalPnL:      1999,
						UnrealizedPnL: 1999,
						MaximumLoss:   0,
						MaximumProfit: 0,
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        1,
						NumberOfWinningTrades: 0,
						NumberOfLosingTrades:  0,
						WinRate:               0,
						MaxDrawdown:           0,
					},
					TotalFees: 1.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 0,
						Max: 0,
						Avg: 0,
					},
				},
				{
					Symbol: "GOOGL",
					TradePnl: types.TradePnl{
						RealizedPnL:   0,
						TotalPnL:      4999,
						UnrealizedPnL: 4999,
						MaximumLoss:   0,
						MaximumProfit: 0,
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        1,
						NumberOfWinningTrades: 0,
						NumberOfLosingTrades:  0,
						WinRate:               0,
						MaxDrawdown:           0,
					},
					TotalFees: 1.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 0,
						Max: 0,
						Avg: 0,
					},
				},
			},
			expectError: false,
		},
		{
			name:          "No trades",
			orders:        []types.Order{},
			expectedStats: []types.TradeStats{},
			expectError:   false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset state before each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			// Process orders
			for _, order := range tc.orders {
				_, err := suite.state.Update([]types.Order{order})
				suite.Require().NoError(err)
			}

			// Get stats
			stats, err := suite.state.GetStats(strategy.StrategyContext{
				DataSource: mockSource,
			})
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(len(tc.expectedStats), len(stats), "Number of stats mismatch")

			// Compare stats
			for i, expected := range tc.expectedStats {
				if i >= len(stats) {
					suite.T().Fatalf("Expected stats[%d] but got no more stats", i)
				}
				actual := stats[i]
				suite.Assert().Equal(expected.Symbol, actual.Symbol, "Symbol mismatch")
				suite.Assert().Equal(expected.TradeResult.NumberOfTrades, actual.TradeResult.NumberOfTrades, "Number of trades mismatch")
				suite.Assert().Equal(expected.TradeResult.NumberOfWinningTrades, actual.TradeResult.NumberOfWinningTrades, "Number of winning trades mismatch")
				suite.Assert().Equal(expected.TradeResult.NumberOfLosingTrades, actual.TradeResult.NumberOfLosingTrades, "Number of losing trades mismatch")
				suite.Assert().Equal(expected.TradeResult.WinRate, actual.TradeResult.WinRate, "Win rate mismatch")
				suite.Assert().Equal(expected.TradePnl.TotalPnL, actual.TradePnl.TotalPnL, "Total profit loss mismatch")
				suite.Assert().Equal(expected.TradePnl.RealizedPnL, actual.TradePnl.RealizedPnL, "Realized profit loss mismatch")
				suite.Assert().Equal(expected.TradePnl.UnrealizedPnL, actual.TradePnl.UnrealizedPnL, "Unrealized profit loss mismatch")
				suite.Assert().Equal(expected.TradeResult.MaxDrawdown, actual.TradeResult.MaxDrawdown, "Max drawdown mismatch")
				suite.Assert().Equal(expected.TradePnl.MaximumLoss, actual.TradePnl.MaximumLoss, "Maximum loss mismatch")
				suite.Assert().Equal(expected.TradePnl.MaximumProfit, actual.TradePnl.MaximumProfit, "Maximum profit mismatch")
				suite.Assert().Equal(expected.TotalFees, actual.TotalFees, "Total fees mismatch")
				suite.Assert().Equal(expected.TradeHoldingTime.Min, actual.TradeHoldingTime.Min, "Min holding time mismatch")
				suite.Assert().Equal(expected.TradeHoldingTime.Max, actual.TradeHoldingTime.Max, "Max holding time mismatch")
				suite.Assert().Equal(expected.TradeHoldingTime.Avg, actual.TradeHoldingTime.Avg, "Avg holding time mismatch")
			}
		})
	}
}

func (suite *BacktestStateTestSuite) TestGetOrderById() {

	tests := []struct {
		name        string
		orders      []types.Order
		expected    optional.Option[types.Order]
		expectError bool
		isExisting  bool
	}{
		{
			name: "Existing order",
			orders: []types.Order{
				{
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
			isExisting:  true,
			expectError: false,
		},
		{
			name: "Non-existing order",
			orders: []types.Order{
				{
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
			expectError: false,
			isExisting:  false,
		},
	}

	existingOrderID := ""
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset state before each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			// Process orders and store generated order IDs
			for _, order := range tc.orders {
				results, err := suite.state.Update([]types.Order{order})
				existingOrderID = results[0].Order.OrderID
				suite.Require().NoError(err)
				suite.Require().Len(results, 1)
			}

			// For the first test case, we'll look up the existing order
			// For the second test case, we'll look up a non-existent order
			var orderID string
			if tc.isExisting {
				orderID = existingOrderID
			} else {
				orderID = uuid.New().String()
			}

			// Get order by ID
			result, err := suite.state.GetOrderById(orderID)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)

			if tc.name == "Existing order" {
				suite.Assert().True(result.IsSome(), "Expected order to exist")
				actualOrder := result.Unwrap()

				// Verify the order details match the input order (except for the generated ID)
				suite.Assert().Equal(tc.orders[0].Symbol, actualOrder.Symbol, "Symbol mismatch")
				suite.Assert().Equal(tc.orders[0].Side, actualOrder.Side, "Side mismatch")
				suite.Assert().Equal(tc.orders[0].Quantity, actualOrder.Quantity, "Quantity mismatch")
				suite.Assert().Equal(tc.orders[0].Price, actualOrder.Price, "Price mismatch")
				suite.Assert().Equal(tc.orders[0].Timestamp.UTC(), actualOrder.Timestamp.UTC(), "Timestamp mismatch")
				suite.Assert().Equal(tc.orders[0].IsCompleted, actualOrder.IsCompleted, "IsCompleted mismatch")
				suite.Assert().Equal(tc.orders[0].Reason.Reason, actualOrder.Reason.Reason, "Reason mismatch")
				suite.Assert().Equal(tc.orders[0].Reason.Message, actualOrder.Reason.Message, "Message mismatch")
				suite.Assert().Equal(tc.orders[0].StrategyName, actualOrder.StrategyName, "Strategy name mismatch")
			} else {
				suite.Assert().False(result.IsSome(), "Expected order to not exist")
			}
		})
	}
}

func (suite *BacktestStateTestSuite) TestGetAllPositions() {
	tests := []struct {
		name        string
		orders      []types.Order
		expected    []types.Position
		expectError bool
	}{
		{
			name: "Single open position",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
			expected: []types.Position{
				{
					Symbol:           "AAPL",
					Quantity:         100,
					TotalInQuantity:  100,
					TotalOutQuantity: 0,
					TotalInAmount:    10000.0,
					TotalOutAmount:   0,
					TotalInFee:       1.0,
					TotalOutFee:      0,
					OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:     "",
				},
			},
			expectError: false,
		},
		{
			name: "Multiple open positions",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "GOOGL",
					Side:        types.PurchaseTypeBuy,
					Quantity:    50,
					Price:       2000.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expected: []types.Position{
				{
					Symbol:           "AAPL",
					Quantity:         100,
					TotalInQuantity:  100,
					TotalOutQuantity: 0,
					TotalInAmount:    10000.0,
					TotalOutAmount:   0,
					TotalInFee:       1.0,
					TotalOutFee:      0,
					OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:     "",
				},
				{
					Symbol:           "GOOGL",
					Quantity:         50,
					TotalInQuantity:  50,
					TotalOutQuantity: 0,
					TotalInAmount:    100000.0,
					TotalOutAmount:   0,
					TotalInFee:       1.0,
					TotalOutFee:      0,
					OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:     "",
				},
			},
			expectError: false,
		},
		{
			name: "Closed position",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    100,
					Price:       110.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			expected:    []types.Position{},
			expectError: false,
		},
		{
			name:        "No positions",
			orders:      []types.Order{},
			expected:    []types.Position{},
			expectError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset state before each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			// Process orders
			for _, order := range tc.orders {
				_, err := suite.state.Update([]types.Order{order})
				suite.Require().NoError(err)
			}

			// Get all positions
			positions, err := suite.state.GetAllPositions()
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(len(tc.expected), len(positions), "Number of positions mismatch")

			// Compare positions
			for i, expected := range tc.expected {
				if i >= len(positions) {
					suite.T().Fatalf("Expected positions[%d] but got no more positions", i)
				}
				actual := positions[i]
				suite.Assert().Equal(expected.Symbol, actual.Symbol, "Symbol mismatch")
				suite.Assert().Equal(expected.Quantity, actual.Quantity, "Quantity mismatch")
				suite.Assert().Equal(expected.TotalInQuantity, actual.TotalInQuantity, "Total in quantity mismatch")
				suite.Assert().Equal(expected.TotalOutQuantity, actual.TotalOutQuantity, "Total out quantity mismatch")
				suite.Assert().Equal(expected.TotalInAmount, actual.TotalInAmount, "Total in amount mismatch")
				suite.Assert().Equal(expected.TotalOutAmount, actual.TotalOutAmount, "Total out amount mismatch")
				suite.Assert().Equal(expected.TotalInFee, actual.TotalInFee, "Total in fee mismatch")
				suite.Assert().Equal(expected.TotalOutFee, actual.TotalOutFee, "Total out fee mismatch")
				suite.Assert().Equal(expected.OpenTimestamp.UTC(), actual.OpenTimestamp.UTC(), "Open timestamp mismatch")
				suite.Assert().Equal(expected.StrategyName, actual.StrategyName, "Strategy name mismatch")
			}
		})
	}
}

func (suite *BacktestStateTestSuite) TestGetPosition() {
	tests := []struct {
		name        string
		orders      []types.Order
		symbol      string
		expected    types.Position
		expectError bool
	}{
		{
			name: "Single buy order",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
			symbol: "AAPL",
			expected: types.Position{
				Symbol:           "AAPL",
				Quantity:         100,
				TotalInQuantity:  100,
				TotalOutQuantity: 0,
				TotalInAmount:    10000.0,
				TotalOutAmount:   0,
				TotalInFee:       1.0,
				TotalOutFee:      0,
				OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:     "",
			},
			expectError: false,
		},
		{
			name: "Multiple buys and sells",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
					Quantity:    50,
					Price:       110.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
				{
					OrderID:     "order3",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    75,
					Price:       120.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			symbol: "AAPL",
			expected: types.Position{
				Symbol:           "AAPL",
				Quantity:         75,  // 100 + 50 - 75
				TotalInQuantity:  150, // 100 + 50
				TotalOutQuantity: 75,
				TotalInAmount:    15500.0, // (100 * 100) + (50 * 110)
				TotalOutAmount:   9000.0,  // 75 * 120
				TotalInFee:       2.0,     // 1 + 1
				TotalOutFee:      1.0,
				OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:     "",
			},
			expectError: false,
		},
		{
			name: "Fully closed position",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
					OrderID:     "order2",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeSell,
					Quantity:    100,
					Price:       110.0,
					Fee:         1.0,
					Timestamp:   time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted: true,
					Reason: types.Reason{
						Reason: "test",
					},
				},
			},
			symbol: "AAPL",
			expected: types.Position{
				Symbol:           "AAPL",
				Quantity:         0,
				TotalInQuantity:  100,
				TotalOutQuantity: 100,
				TotalInAmount:    10000.0,
				TotalOutAmount:   11000.0,
				TotalInFee:       1.0,
				TotalOutFee:      1.0,
				OpenTimestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:     "",
			},
			expectError: false,
		},
		{
			name: "Non-existent symbol",
			orders: []types.Order{
				{
					OrderID:     "order1",
					Symbol:      "AAPL",
					Side:        types.PurchaseTypeBuy,
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
			symbol: "GOOGL",
			expected: types.Position{
				Symbol:           "GOOGL",
				Quantity:         0,
				TotalInQuantity:  0,
				TotalOutQuantity: 0,
				TotalInAmount:    0,
				TotalOutAmount:   0,
				TotalInFee:       0,
				TotalOutFee:      0,
				OpenTimestamp:    time.Time{},
				StrategyName:     "",
			},
			expectError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Reset state before each test case
			err := suite.state.Cleanup()
			suite.Require().NoError(err)

			// Process orders
			for _, order := range tc.orders {
				_, err := suite.state.Update([]types.Order{order})
				suite.Require().NoError(err)
			}

			// Get position
			position, err := suite.state.GetPosition(tc.symbol)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(tc.expected.Symbol, position.Symbol, "Symbol mismatch")
			suite.Assert().Equal(tc.expected.Quantity, position.Quantity, "Quantity mismatch")
			suite.Assert().Equal(tc.expected.TotalInQuantity, position.TotalInQuantity, "Total in quantity mismatch")
			suite.Assert().Equal(tc.expected.TotalOutQuantity, position.TotalOutQuantity, "Total out quantity mismatch")
			suite.Assert().Equal(tc.expected.TotalInAmount, position.TotalInAmount, "Total in amount mismatch")
			suite.Assert().Equal(tc.expected.TotalOutAmount, position.TotalOutAmount, "Total out amount mismatch")
			suite.Assert().Equal(tc.expected.TotalInFee, position.TotalInFee, "Total in fee mismatch")
			suite.Assert().Equal(tc.expected.TotalOutFee, position.TotalOutFee, "Total out fee mismatch")
			suite.Assert().Equal(tc.expected.OpenTimestamp.UTC(), position.OpenTimestamp.UTC(), "Open timestamp mismatch")
			suite.Assert().Equal(tc.expected.StrategyName, position.StrategyName, "Strategy name mismatch")
		})
	}
}

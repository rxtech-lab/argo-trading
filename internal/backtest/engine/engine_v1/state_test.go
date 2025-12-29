package engine

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
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

	var stateErr error
	suite.state, stateErr = NewBacktestState(suite.logger)
	suite.Require().NoError(stateErr)
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

func (suite *BacktestStateTestSuite) TestWrite() {
	// Create a temporary directory for test files
	tmpDir := suite.T().TempDir()

	// Create some test data
	orders := []types.Order{
		{
			OrderID:      "order1",
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeBuy,
			Quantity:     100,
			Price:        100.0,
			Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			IsCompleted:  true,
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  "test",
				Message: "test message",
			},
			StrategyName: "test_strategy",
		},
		{
			OrderID:      "order2",
			Symbol:       "AAPL",
			Side:         types.PurchaseTypeSell,
			Quantity:     50,
			Price:        110.0,
			Timestamp:    time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
			IsCompleted:  true,
			PositionType: types.PositionTypeLong,
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

// TestGetStats runs before each test
func (suite *BacktestStateTestSuite) TestGetStats() {
	// Create mock controller
	ctrl := gomock.NewController(suite.T())
	defer ctrl.Finish()

	// Create mock data source using gomock
	mockSource := mocks.NewMockDataSource(ctrl)

	// Set up mock behavior for ReadLastData
	// End time is 15:00 for holding time calculations on open positions
	mockSource.EXPECT().ReadLastData("AAPL").Return(types.MarketData{
		Symbol: "AAPL",
		Close:  120.0,
		Time:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()

	mockSource.EXPECT().ReadLastData("GOOGL").Return(types.MarketData{
		Symbol: "GOOGL",
		Close:  2100.0,
		Time:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()

	mockSource.EXPECT().ReadLastData("TSLA").Return(types.MarketData{
		Symbol: "TSLA",
		Close:  800.0,
		Time:   time.Date(2024, 1, 1, 15, 0, 0, 0, time.UTC),
	}, nil).AnyTimes()

	// IBM mock for multi-day hold time test - end time is 3 days after buy
	mockSource.EXPECT().ReadLastData("IBM").Return(types.MarketData{
		Symbol: "IBM",
		Close:  150.0,
		Time:   time.Date(2024, 1, 4, 10, 0, 0, 0, time.UTC), // Jan 4, 10:00
	}, nil).AnyTimes()

	// For GetPreviousNumberOfDataPoints (required by interface but not used in test)
	mockSource.EXPECT().GetPreviousNumberOfDataPoints(gomock.Any(), gomock.Any(), gomock.Any()).Return(
		[]types.MarketData{}, nil,
	).AnyTimes()

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
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
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
					PositionType: types.PositionTypeLong,
					Quantity:     50,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
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
						Min: 3600, // 1 hour between buy and sell = 3600 seconds
						Max: 3600,
						Avg: 3600,
					},
					BuyAndHoldPnl: 2000.0, // (120 - 100) * 100
				},
			},
			expectError: false,
		},
		{
			name: "Multiple symbols with trades",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				{
					OrderID:      "order2",
					Symbol:       "GOOGL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     50,
					Price:        2000.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
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
						Min: 18000, // Open position: buy at 10:00, end at 15:00 = 5 hours = 18000 seconds
						Max: 18000,
						Avg: 18000,
					},
					BuyAndHoldPnl: 2000.0,
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
						Min: 18000, // Open position: buy at 10:00, end at 15:00 = 5 hours = 18000 seconds
						Max: 18000,
						Avg: 18000,
					},
					BuyAndHoldPnl: 5000,
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
		{
			name: "Short position calculation",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "TSLA",
					Side:         types.PurchaseTypeSell,
					PositionType: types.PositionTypeShort,
					Quantity:     10,
					Price:        1000.0,
					Fee:          5.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "short_strategy",
					Reason: types.Reason{
						Reason:  "test",
						Message: "short position test",
					},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "TSLA",
					TradePnl: types.TradePnl{
						RealizedPnL:   0,
						TotalPnL:      2000, // (1000 - 800) * 10 = 2000 profit for a short position
						UnrealizedPnL: 2000,
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
					TotalFees: 5.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 0,
						Max: 0,
						Avg: 0,
					},
					BuyAndHoldPnl: 2000.0, // (1000 - 800) * 10 = positive 2000 for a short position
				},
			},
			expectError: false,
		},
		{
			name: "All realized hold time - all positions closed",
			orders: []types.Order{
				{
					OrderID:      "buy1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					OrderID:      "sell1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), // 1 hour later
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					OrderID:      "buy2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
				{
					OrderID:      "sell2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        115.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 14, 0, 0, 0, time.UTC), // 2 hours later
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell2"},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "AAPL",
					TradePnl: types.TradePnl{
						RealizedPnL:   0, // Position fully closed, system shows 0
						TotalPnL:      0, // Position fully closed, system shows 0
						UnrealizedPnL: 0,
						MaximumLoss:   0,
						MaximumProfit: 1248.0, // From trades table MAX(pnl)
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        4,
						NumberOfWinningTrades: 2,
						NumberOfLosingTrades:  0,
						WinRate:               0.5,
						MaxDrawdown:           0,
					},
					TotalFees: 4.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 3600, // First trade: 1 hour = 3600 seconds
						Max: 7200, // Second trade: 2 hours = 7200 seconds
						Avg: 5400, // Average: (3600+7200)/2 = 5400 seconds
					},
					BuyAndHoldPnl: 2000.0, // (120 - 100) * 100 based on first buy
				},
			},
			expectError: false,
		},
		{
			name: "Partial realized hold time - mix of closed and open positions",
			orders: []types.Order{
				{
					OrderID:      "buy1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					OrderID:      "sell1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC), // 1 hour later
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "sell1"},
				},
				{
					OrderID:      "buy2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), // No matching sell, open until end time (15:00) = 3 hours
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "AAPL",
					TradePnl: types.TradePnl{
						RealizedPnL:   748.0,
						TotalPnL:      2497.0, // Realized + unrealized
						UnrealizedPnL: 1749.0,
						MaximumLoss:   0,
						MaximumProfit: 998.0, // From trades table MAX(pnl)
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        3,
						NumberOfWinningTrades: 1,
						NumberOfLosingTrades:  0,
						WinRate:               0.3333333333333333,
						MaxDrawdown:           0,
					},
					TotalFees: 3.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 3600,  // First trade: 1 hour (closed) = 3600 seconds
						Max: 10800, // Second trade: 3 hours (open, end time 15:00 - buy time 12:00) = 10800 seconds
						Avg: 7200,  // Average: (3600+10800)/2 = 7200 seconds
					},
					BuyAndHoldPnl: 2000.0, // (120 - 100) * 100
				},
			},
			expectError: false,
		},
		{
			name: "Non-realized hold time only - all positions open",
			orders: []types.Order{
				{
					OrderID:      "buy1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), // Open until end time (15:00) = 5 hours
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
				{
					OrderID:      "buy2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        105.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC), // Open until end time (15:00) = 3 hours
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy2"},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "AAPL",
					TradePnl: types.TradePnl{
						RealizedPnL:   0,
						TotalPnL:      3498.0, // Unrealized only
						UnrealizedPnL: 3498.0, // (120 - 100.01) * 100 + (120 - 105.01) * 100
						MaximumLoss:   0,
						MaximumProfit: 0,
					},
					TradeResult: types.TradeResult{
						NumberOfTrades:        2,
						NumberOfWinningTrades: 0,
						NumberOfLosingTrades:  0,
						WinRate:               0,
						MaxDrawdown:           0,
					},
					TotalFees: 2.0,
					TradeHoldingTime: types.TradeHoldingTime{
						Min: 10800, // Second buy: 3 hours (end time 15:00 - buy time 12:00) = 10800 seconds
						Max: 18000, // First buy: 5 hours (end time 15:00 - buy time 10:00) = 18000 seconds
						Avg: 14400, // Average: (18000+10800)/2 = 14400 seconds
					},
					BuyAndHoldPnl: 2000.0, // (120 - 100) * 100 based on first buy
				},
			},
			expectError: false,
		},
		{
			name: "Multi-day hold time - open position across multiple days",
			orders: []types.Order{
				{
					OrderID:      "buy1",
					Symbol:       "IBM",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        140.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC), // Jan 1, 10:00
					IsCompleted:  true,
					StrategyName: "test",
					Reason:       types.Reason{Reason: "test", Message: "buy1"},
				},
			},
			expectedStats: []types.TradeStats{
				{
					Symbol: "IBM",
					TradePnl: types.TradePnl{
						RealizedPnL:   0,
						TotalPnL:      999.0, // (150 - 140.01) * 100
						UnrealizedPnL: 999.0,
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
						// Buy: Jan 1, 10:00, End: Jan 4, 10:00 = exactly 72 hours (3 days) = 259200 seconds
						Min: 259200,
						Max: 259200,
						Avg: 259200,
					},
					BuyAndHoldPnl: 1000.0, // (150 - 140) * 100
				},
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

			// Get stats
			stats, err := suite.state.GetStats(runtime.RuntimeContext{
				DataSource: mockSource,
			}, "test-run-id", "/path/to/trades.parquet", "/path/to/orders.parquet", "/path/to/marks.parquet", "/path/to/strategy.wasm", "/path/to/data.parquet")
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
				suite.Assert().Equal(expected.BuyAndHoldPnl, actual.BuyAndHoldPnl, "Buy and hold PnL mismatch")
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
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
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
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
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
				suite.Assert().Equal(tc.orders[0].Quantity, actualOrder.Quantity, "TotalLongPositionQuantity mismatch")
				suite.Assert().Equal(tc.orders[0].Price, actualOrder.Price, "Price mismatch")
				suite.Assert().Equal(tc.orders[0].Timestamp.UTC(), actualOrder.Timestamp.UTC(), "Timestamp mismatch")
				suite.Assert().Equal(tc.orders[0].IsCompleted, actualOrder.IsCompleted, "IsCompleted mismatch")
				suite.Assert().Equal(tc.orders[0].Reason.Reason, actualOrder.Reason.Reason, "Reason mismatch")
				suite.Assert().Equal(tc.orders[0].Reason.Message, actualOrder.Reason.Message, "Message mismatch")
				suite.Assert().Equal(tc.orders[0].StrategyName, actualOrder.StrategyName, "Strategy name mismatch")
				suite.Assert().Equal(tc.orders[0].PositionType, actualOrder.PositionType, "Position type mismatch")
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
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			expected: []types.Position{
				{
					Symbol:                       "AAPL",
					TotalLongPositionQuantity:    100,
					TotalLongInPositionQuantity:  100,
					TotalLongOutPositionQuantity: 0,
					TotalLongInPositionAmount:    10000.0,
					TotalLongOutPositionAmount:   0,
					TotalLongInFee:               1.0,
					TotalLongOutFee:              0,
					OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:                 "a",
				},
			},
			expectError: false,
		},
		{
			name: "Multiple open positions",
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
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
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
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			expected: []types.Position{
				{
					Symbol:                       "AAPL",
					TotalLongPositionQuantity:    100,
					TotalLongInPositionQuantity:  100,
					TotalLongOutPositionQuantity: 0,
					TotalLongInPositionAmount:    10000.0,
					TotalLongOutPositionAmount:   0,
					TotalLongInFee:               1.0,
					TotalLongOutFee:              0,
					OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:                 "a",
				},
				{
					Symbol:                       "GOOGL",
					TotalLongPositionQuantity:    50,
					TotalLongInPositionQuantity:  50,
					TotalLongOutPositionQuantity: 0,
					TotalLongInPositionAmount:    100000.0,
					TotalLongOutPositionAmount:   0,
					TotalLongInFee:               1.0,
					TotalLongOutFee:              0,
					OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					StrategyName:                 "a",
				},
			},
			expectError: false,
		},
		{
			name: "Closed position",
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
					PositionType: types.PositionTypeLong,
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
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					PositionType: types.PositionTypeLong,
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
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
				suite.Assert().Equal(expected.TotalLongPositionQuantity, actual.TotalLongPositionQuantity, "TotalLongPositionQuantity mismatch")
				suite.Assert().Equal(expected.TotalLongInPositionQuantity, actual.TotalLongInPositionQuantity, "Total in quantity mismatch")
				suite.Assert().Equal(expected.TotalLongOutPositionQuantity, actual.TotalLongOutPositionQuantity, "Total out quantity mismatch")
				suite.Assert().Equal(expected.TotalLongInPositionAmount, actual.TotalLongInPositionAmount, "Total in amount mismatch")
				suite.Assert().Equal(expected.TotalLongOutPositionAmount, actual.TotalLongOutPositionAmount, "Total out amount mismatch")
				suite.Assert().Equal(expected.TotalLongInFee, actual.TotalLongInFee, "Total in fee mismatch")
				suite.Assert().Equal(expected.TotalLongOutFee, actual.TotalLongOutFee, "Total out fee mismatch")
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
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			symbol: "AAPL",
			expected: types.Position{
				Symbol:                       "AAPL",
				TotalLongPositionQuantity:    100,
				TotalLongInPositionQuantity:  100,
				TotalLongOutPositionQuantity: 0,
				TotalLongInPositionAmount:    10000.0,
				TotalLongOutPositionAmount:   0,
				TotalLongInFee:               1.0,
				TotalLongOutFee:              0,
				OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:                 "a",
			},
			expectError: false,
		},
		{
			name: "Multiple buys and sells",
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
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				{
					OrderID:      "order2",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					Quantity:     50,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
				{
					OrderID:      "order3",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeSell,
					Quantity:     75,
					Price:        120.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					PositionType: types.PositionTypeLong,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			symbol: "AAPL",
			expected: types.Position{
				Symbol:                       "AAPL",
				TotalLongPositionQuantity:    75,  // 100 + 50 - 75
				TotalLongInPositionQuantity:  150, // 100 + 50
				TotalLongOutPositionQuantity: 75,
				TotalLongInPositionAmount:    15500.0, // (100 * 100) + (50 * 110)
				TotalLongOutPositionAmount:   9000.0,  // 75 * 120
				TotalLongInFee:               2.0,     // 1 + 1
				TotalLongOutFee:              1.0,
				OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:                 "a",
			},
			expectError: false,
		},
		{
			name: "Fully closed position",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
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
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        110.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 11, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			symbol: "AAPL",
			expected: types.Position{
				Symbol:                       "AAPL",
				TotalLongPositionQuantity:    0,
				TotalLongInPositionQuantity:  100,
				TotalLongOutPositionQuantity: 100,
				TotalLongInPositionAmount:    10000.0,
				TotalLongOutPositionAmount:   11000.0,
				TotalLongInFee:               1.0,
				TotalLongOutFee:              1.0,
				OpenTimestamp:                time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
				StrategyName:                 "a",
			},
			expectError: false,
		},
		{
			name: "Non-existent symbol",
			orders: []types.Order{
				{
					OrderID:      "order1",
					Symbol:       "AAPL",
					Side:         types.PurchaseTypeBuy,
					PositionType: types.PositionTypeLong,
					Quantity:     100,
					Price:        100.0,
					Fee:          1.0,
					Timestamp:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					IsCompleted:  true,
					StrategyName: "a",
					Reason: types.Reason{
						Reason:  "test",
						Message: "reason",
					},
				},
			},
			symbol: "GOOGL",
			expected: types.Position{
				Symbol:                       "GOOGL",
				TotalLongPositionQuantity:    0,
				TotalLongInPositionQuantity:  0,
				TotalLongOutPositionQuantity: 0,
				TotalLongInPositionAmount:    0,
				TotalLongOutPositionAmount:   0,
				TotalLongInFee:               0,
				TotalLongOutFee:              0,
				OpenTimestamp:                time.Time{},
				StrategyName:                 "",
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
			suite.Assert().Equal(tc.expected.TotalLongPositionQuantity, position.TotalLongPositionQuantity, "TotalLongPositionQuantity mismatch")
			suite.Assert().Equal(tc.expected.TotalLongInPositionQuantity, position.TotalLongInPositionQuantity, "Total in quantity mismatch")
			suite.Assert().Equal(tc.expected.TotalLongOutPositionQuantity, position.TotalLongOutPositionQuantity, "Total out quantity mismatch")
			suite.Assert().Equal(tc.expected.TotalLongInPositionAmount, position.TotalLongInPositionAmount, "Total in amount mismatch")
			suite.Assert().Equal(tc.expected.TotalLongOutPositionAmount, position.TotalLongOutPositionAmount, "Total out amount mismatch")
			suite.Assert().Equal(tc.expected.TotalLongInFee, position.TotalLongInFee, "Total in fee mismatch")
			suite.Assert().Equal(tc.expected.TotalLongOutFee, position.TotalLongOutFee, "Total out fee mismatch")
			suite.Assert().Equal(tc.expected.OpenTimestamp.UTC(), position.OpenTimestamp.UTC(), "Open timestamp mismatch")
			suite.Assert().Equal(tc.expected.StrategyName, position.StrategyName, "Strategy name mismatch")
		})
	}
}

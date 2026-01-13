package stats

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type StatsTrackerTestSuite struct {
	suite.Suite
	tempDir string
	logger  *logger.Logger
}

func (s *StatsTrackerTestSuite) SetupSuite() {
	log, err := logger.NewLogger()
	s.Require().NoError(err)
	s.logger = log
}

func (s *StatsTrackerTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "stats_tracker_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *StatsTrackerTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func TestStatsTrackerTestSuite(t *testing.T) {
	suite.Run(t, new(StatsTrackerTestSuite))
}

func (s *StatsTrackerTestSuite) TestInitialize() {
	st := NewStatsTracker(s.logger)

	symbols := []string{"BTCUSDT", "ETHUSDT"}
	runID := "run_1"
	sessionStart := time.Now()
	strategyInfo := types.StrategyInfo{
		ID:      "test-strategy",
		Version: "1.0.0",
		Name:    "Test Strategy",
	}

	st.Initialize(symbols, runID, sessionStart, strategyInfo)

	s.Equal(runID, st.GetRunID())
	s.Equal(sessionStart.Format("2006-01-02"), st.GetCurrentDate())
}

func (s *StatsTrackerTestSuite) TestRecordTrade_WinningTrade() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           500.0, // Winning trade
	}

	st.RecordTrade(trade)

	stats := st.GetCumulativeStats()
	s.Equal(1, stats.TradeResult.NumberOfTrades)
	s.Equal(1, stats.TradeResult.NumberOfWinningTrades)
	s.Equal(0, stats.TradeResult.NumberOfLosingTrades)
	s.Equal(1.0, stats.TradeResult.WinRate)
	s.Equal(500.0, stats.TradePnl.RealizedPnL)
	s.Equal(10.0, stats.TotalFees)
}

func (s *StatsTrackerTestSuite) TestRecordTrade_LosingTrade() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        49000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 49000.0,
		Fee:           10.0,
		PnL:           -200.0, // Losing trade
	}

	st.RecordTrade(trade)

	stats := st.GetCumulativeStats()
	s.Equal(1, stats.TradeResult.NumberOfTrades)
	s.Equal(0, stats.TradeResult.NumberOfWinningTrades)
	s.Equal(1, stats.TradeResult.NumberOfLosingTrades)
	s.Equal(0.0, stats.TradeResult.WinRate)
	s.Equal(-200.0, stats.TradePnl.RealizedPnL)
	s.Equal(-200.0, stats.TradePnl.MaximumLoss)
}

func (s *StatsTrackerTestSuite) TestRecordTrade_WinRate() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Record 3 winning and 2 losing trades
	trades := []types.Trade{
		{Order: types.Order{OrderID: "1", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 100},
		{Order: types.Order{OrderID: "2", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 200},
		{Order: types.Order{OrderID: "3", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 50},
		{Order: types.Order{OrderID: "4", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: -100},
		{Order: types.Order{OrderID: "5", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: -50},
	}

	for _, trade := range trades {
		st.RecordTrade(trade)
	}

	stats := st.GetCumulativeStats()
	s.Equal(5, stats.TradeResult.NumberOfTrades)
	s.Equal(3, stats.TradeResult.NumberOfWinningTrades)
	s.Equal(2, stats.TradeResult.NumberOfLosingTrades)
	s.Equal(0.6, stats.TradeResult.WinRate) // 3/5 = 0.6
}

func (s *StatsTrackerTestSuite) TestRecordTrade_MaxDrawdown() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Simulate: +100, +200 (peak=300), -150 (drawdown=150), +50 (drawdown=100)
	trades := []types.Trade{
		{Order: types.Order{OrderID: "1", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 100},
		{Order: types.Order{OrderID: "2", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 200},
		{Order: types.Order{OrderID: "3", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: -150},
		{Order: types.Order{OrderID: "4", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: time.Now(), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: time.Now(), ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 50},
	}

	for _, trade := range trades {
		st.RecordTrade(trade)
	}

	stats := st.GetCumulativeStats()
	s.Equal(200.0, stats.TradePnl.RealizedPnL) // 100+200-150+50 = 200
	s.Equal(150.0, stats.TradeResult.MaxDrawdown)
}

func (s *StatsTrackerTestSuite) TestHandleDateBoundary() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Record a trade
	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           500.0,
	}
	st.RecordTrade(trade)

	// Verify cumulative stats
	cumulativeStats := st.GetCumulativeStats()
	s.Equal(1, cumulativeStats.TradeResult.NumberOfTrades)

	// Handle date boundary
	newDate := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	st.HandleDateBoundary(newDate)

	// Daily stats should be reset
	dailyStats := st.GetDailyStats()
	s.Equal(0, dailyStats.TradeResult.NumberOfTrades)
	s.Equal(0.0, dailyStats.TradePnl.RealizedPnL)

	// Cumulative stats should remain
	cumulativeStats = st.GetCumulativeStats()
	s.Equal(1, cumulativeStats.TradeResult.NumberOfTrades)
	s.Equal(500.0, cumulativeStats.TradePnl.RealizedPnL)
}

func (s *StatsTrackerTestSuite) TestSetUnrealizedPnL() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Record a trade with some realized PnL
	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           500.0,
	}
	st.RecordTrade(trade)

	// Set unrealized PnL
	st.SetUnrealizedPnL(200.0)

	stats := st.GetCumulativeStats()
	s.Equal(500.0, stats.TradePnl.RealizedPnL)
	s.Equal(200.0, stats.TradePnl.UnrealizedPnL)
	s.Equal(700.0, stats.TradePnl.TotalPnL)
}

func (s *StatsTrackerTestSuite) TestWriteStatsYAML() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{
		ID:      "test-strategy",
		Version: "1.0.0",
		Name:    "Test Strategy",
	})

	statsPath := filepath.Join(s.tempDir, "stats.yaml")
	st.SetFilePaths(
		filepath.Join(s.tempDir, "orders.parquet"),
		filepath.Join(s.tempDir, "trades.parquet"),
		filepath.Join(s.tempDir, "marks.parquet"),
		filepath.Join(s.tempDir, "logs.parquet"),
		filepath.Join(s.tempDir, "market_data.parquet"),
		statsPath,
	)

	// Record a trade
	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           500.0,
	}
	st.RecordTrade(trade)

	// Write stats to YAML
	err := st.WriteStatsYAML()
	s.Require().NoError(err)

	// Verify file exists
	s.FileExists(statsPath)

	// Read and verify content
	readStats, err := types.ReadLiveTradeStats(statsPath)
	s.Require().NoError(err)
	s.Equal("run_1", readStats.ID)
	s.Equal(1, readStats.TradeResult.NumberOfTrades)
	s.Equal(500.0, readStats.TradePnl.RealizedPnL)
}

func (s *StatsTrackerTestSuite) TestHoldingTimeCalculation() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	baseTime := time.Now()

	// Record trades with different holding times
	trades := []types.Trade{
		{Order: types.Order{OrderID: "1", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: baseTime.Add(-60 * time.Second), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: baseTime, ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 100},  // 60 seconds
		{Order: types.Order{OrderID: "2", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: baseTime.Add(-120 * time.Second), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: baseTime, ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 100}, // 120 seconds
		{Order: types.Order{OrderID: "3", Symbol: "BTCUSDT", Side: types.PurchaseTypeSell, Quantity: 1, Price: 50000, Timestamp: baseTime.Add(-180 * time.Second), IsCompleted: true, Status: types.OrderStatusFilled, Reason: types.Reason{Reason: "s", Message: "t"}, StrategyName: "t", Fee: 1, PositionType: types.PositionTypeLong}, ExecutedAt: baseTime, ExecutedQty: 1, ExecutedPrice: 50000, Fee: 1, PnL: 100}, // 180 seconds
	}

	for _, trade := range trades {
		st.RecordTrade(trade)
	}

	stats := st.GetCumulativeStats()
	s.Equal(60, stats.TradeHoldingTime.Min)
	s.Equal(180, stats.TradeHoldingTime.Max)
	s.Equal(120, stats.TradeHoldingTime.Avg) // (60+120+180)/3 = 120
}

func (s *StatsTrackerTestSuite) TestZeroTrades_WinRateIsZero() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	stats := st.GetCumulativeStats()
	s.Equal(0, stats.TradeResult.NumberOfTrades)
	s.Equal(0.0, stats.TradeResult.WinRate) // Should not divide by zero
}

// ============================================================================
// Additional Edge Case Tests
// ============================================================================

func (s *StatsTrackerTestSuite) TestWriteStatsYAML_NoOutputPath() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Don't set file paths - statsOutputPath will be empty

	// Should return nil without error
	err := st.WriteStatsYAML()
	s.NoError(err)
}

func (s *StatsTrackerTestSuite) TestGetStatsOutputPath() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Before setting paths
	s.Equal("", st.GetStatsOutputPath())

	// After setting paths
	statsPath := filepath.Join(s.tempDir, "stats.yaml")
	st.SetFilePaths("", "", "", "", "", statsPath)

	s.Equal(statsPath, st.GetStatsOutputPath())
}

func (s *StatsTrackerTestSuite) TestSetFilePaths() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	ordersPath := "/data/orders.parquet"
	tradesPath := "/data/trades.parquet"
	marksPath := "/data/marks.parquet"
	logsPath := "/data/logs.parquet"
	marketDataPath := "/data/market.parquet"
	statsPath := "/data/stats.yaml"

	st.SetFilePaths(ordersPath, tradesPath, marksPath, logsPath, marketDataPath, statsPath)

	// Verify through GetCumulativeStats which includes file paths
	stats := st.GetCumulativeStats()
	s.Equal(ordersPath, stats.OrdersFilePath)
	s.Equal(tradesPath, stats.TradesFilePath)
	s.Equal(marksPath, stats.MarksFilePath)
	s.Equal(logsPath, stats.LogsFilePath)
	s.Equal(marketDataPath, stats.MarketDataFilePath)
}

func (s *StatsTrackerTestSuite) TestGetDailyStats() {
	st := NewStatsTracker(s.logger)
	sessionStart := time.Now()
	st.Initialize([]string{"BTCUSDT"}, "run_1", sessionStart, types.StrategyInfo{
		Name: "TestStrategy",
	})

	// Record a trade
	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        51000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 51000.0,
		Fee:           10.0,
		PnL:           500.0,
	}
	st.RecordTrade(trade)

	dailyStats := st.GetDailyStats()
	s.Equal(1, dailyStats.TradeResult.NumberOfTrades)
	s.Equal(500.0, dailyStats.TradePnl.RealizedPnL)
	s.Equal("run_1", dailyStats.ID)
	s.Equal("TestStrategy", dailyStats.Strategy.Name)
}

func (s *StatsTrackerTestSuite) TestRecordTrade_ZeroPnL() {
	st := NewStatsTracker(s.logger)
	st.Initialize([]string{"BTCUSDT"}, "run_1", time.Now(), types.StrategyInfo{})

	// Trade with zero PnL (breakeven)
	trade := types.Trade{
		Order: types.Order{
			OrderID:      "order-1",
			Symbol:       "BTCUSDT",
			Side:         types.PurchaseTypeSell,
			Quantity:     1.0,
			Price:        50000.0,
			Timestamp:    time.Now().Add(-time.Hour),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       types.Reason{Reason: "strategy", Message: "Test"},
			StrategyName: "test",
			Fee:          10.0,
			PositionType: types.PositionTypeLong,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   1.0,
		ExecutedPrice: 50000.0,
		Fee:           10.0,
		PnL:           0.0, // Breakeven trade
	}

	st.RecordTrade(trade)

	stats := st.GetCumulativeStats()
	s.Equal(1, stats.TradeResult.NumberOfTrades)
	// Zero PnL should not count as winning or losing
	s.Equal(0, stats.TradeResult.NumberOfWinningTrades)
	s.Equal(0, stats.TradeResult.NumberOfLosingTrades)
	s.Equal(0.0, stats.TradeResult.WinRate)
}

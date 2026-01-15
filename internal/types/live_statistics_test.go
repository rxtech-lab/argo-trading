package types

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type LiveStatisticsTestSuite struct {
	suite.Suite
	tempDir string
}

func (s *LiveStatisticsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "live_statistics_test_*")
	s.Require().NoError(err)
	s.tempDir = tempDir
}

func (s *LiveStatisticsTestSuite) TearDownTest() {
	if s.tempDir != "" {
		os.RemoveAll(s.tempDir)
	}
}

func TestLiveStatisticsTestSuite(t *testing.T) {
	suite.Run(t, new(LiveStatisticsTestSuite))
}

// ============================================================================
// WriteLiveTradeStats Tests
// ============================================================================

func (s *LiveStatisticsTestSuite) TestWriteLiveTradeStats_Success() {
	path := filepath.Join(s.tempDir, "stats.yaml")
	stats := LiveTradeStats{
		ID:           "run_1",
		Date:         "2025-01-13",
		SessionStart: time.Now(),
		LastUpdated:  time.Now(),
		Symbols:      []string{"BTCUSDT", "ETHUSDT"},
		TradeResult: TradeResult{
			NumberOfTrades:        10,
			NumberOfWinningTrades: 6,
			NumberOfLosingTrades:  4,
			WinRate:               0.6,
			MaxDrawdown:           100.0,
		},
		TradePnl: TradePnl{
			RealizedPnL:   500.0,
			UnrealizedPnL: 50.0,
			TotalPnL:      550.0,
			MaximumLoss:   -200.0,
			MaximumProfit: 300.0,
		},
		TradeHoldingTime: TradeHoldingTime{
			Min: 60,
			Max: 3600,
			Avg: 1800,
		},
		TotalFees:          25.0,
		OrdersFilePath:     "/data/orders.parquet",
		TradesFilePath:     "/data/trades.parquet",
		MarksFilePath:      "/data/marks.parquet",
		LogsFilePath:       "/data/logs.parquet",
		MarketDataFilePath: "/data/market.parquet",
		Strategy: StrategyInfo{
			Name:    "TestStrategy",
			Version: "1.0.0",
		},
	}

	err := WriteLiveTradeStats(path, stats)
	s.Require().NoError(err)

	// Verify file exists
	_, err = os.Stat(path)
	s.NoError(err)
}

func (s *LiveStatisticsTestSuite) TestWriteLiveTradeStats_InvalidPath() {
	// Use a path that cannot be written to (non-existent directory)
	path := filepath.Join(s.tempDir, "nonexistent", "subdir", "stats.yaml")
	stats := LiveTradeStats{
		ID: "run_1",
	}

	err := WriteLiveTradeStats(path, stats)
	s.Error(err)
	s.Contains(err.Error(), "failed to write live trade stats to file")
}

// ============================================================================
// ReadLiveTradeStats Tests
// ============================================================================

func (s *LiveStatisticsTestSuite) TestReadLiveTradeStats_Success() {
	path := filepath.Join(s.tempDir, "stats.yaml")

	// Write stats first
	originalStats := LiveTradeStats{
		ID:           "run_1",
		Date:         "2025-01-13",
		SessionStart: time.Date(2025, 1, 13, 10, 0, 0, 0, time.UTC),
		LastUpdated:  time.Date(2025, 1, 13, 12, 0, 0, 0, time.UTC),
		Symbols:      []string{"BTCUSDT"},
		TradeResult: TradeResult{
			NumberOfTrades:        5,
			NumberOfWinningTrades: 3,
			NumberOfLosingTrades:  2,
			WinRate:               0.6,
			MaxDrawdown:           50.0,
		},
		TradePnl: TradePnl{
			RealizedPnL:   200.0,
			UnrealizedPnL: 20.0,
			TotalPnL:      220.0,
			MaximumLoss:   -100.0,
			MaximumProfit: 150.0,
		},
		TradeHoldingTime: TradeHoldingTime{
			Min: 30,
			Max: 1800,
			Avg: 900,
		},
		TotalFees:      10.0,
		OrdersFilePath: "/data/orders.parquet",
		Strategy: StrategyInfo{
			Name:    "TestStrategy",
			Version: "1.0.0",
		},
	}

	err := WriteLiveTradeStats(path, originalStats)
	s.Require().NoError(err)

	// Read stats back
	readStats, err := ReadLiveTradeStats(path)
	s.Require().NoError(err)

	// Verify fields match
	s.Equal(originalStats.ID, readStats.ID)
	s.Equal(originalStats.Date, readStats.Date)
	s.Equal(originalStats.Symbols, readStats.Symbols)
	s.Equal(originalStats.TradeResult.NumberOfTrades, readStats.TradeResult.NumberOfTrades)
	s.Equal(originalStats.TradeResult.WinRate, readStats.TradeResult.WinRate)
	s.Equal(originalStats.TradePnl.TotalPnL, readStats.TradePnl.TotalPnL)
	s.Equal(originalStats.TradeHoldingTime.Avg, readStats.TradeHoldingTime.Avg)
	s.Equal(originalStats.TotalFees, readStats.TotalFees)
	s.Equal(originalStats.Strategy.Name, readStats.Strategy.Name)
}

func (s *LiveStatisticsTestSuite) TestReadLiveTradeStats_FileNotFound() {
	path := filepath.Join(s.tempDir, "nonexistent.yaml")

	_, err := ReadLiveTradeStats(path)
	s.Error(err)
	s.Contains(err.Error(), "failed to read live trade stats file")
}

func (s *LiveStatisticsTestSuite) TestReadLiveTradeStats_InvalidYAML() {
	path := filepath.Join(s.tempDir, "invalid.yaml")

	// Write invalid YAML content
	err := os.WriteFile(path, []byte("invalid: yaml: content: [broken"), 0644)
	s.Require().NoError(err)

	_, err = ReadLiveTradeStats(path)
	s.Error(err)
	s.Contains(err.Error(), "failed to unmarshal live trade stats")
}

// ============================================================================
// NewLiveTradeStats Tests
// ============================================================================

func (s *LiveStatisticsTestSuite) TestNewLiveTradeStats() {
	runID := "run_5"
	symbols := []string{"BTCUSDT", "ETHUSDT", "SOLUSDT"}
	strategy := StrategyInfo{
		Name:    "MyStrategy",
		Version: "2.0.0",
	}

	stats := NewLiveTradeStats(runID, symbols, strategy)

	// Verify basic fields
	s.Equal(runID, stats.ID)
	s.Equal(symbols, stats.Symbols)
	s.Equal(strategy.Name, stats.Strategy.Name)
	s.Equal(strategy.Version, stats.Strategy.Version)

	// Verify date is set to today
	s.Equal(time.Now().Format("2006-01-02"), stats.Date)

	// Verify timestamps are set
	s.False(stats.SessionStart.IsZero())
	s.False(stats.LastUpdated.IsZero())
}

func (s *LiveStatisticsTestSuite) TestNewLiveTradeStats_VerifyDefaults() {
	stats := NewLiveTradeStats("run_1", []string{"BTCUSDT"}, StrategyInfo{})

	// Verify all numeric fields default to 0
	s.Equal(0, stats.TradeResult.NumberOfTrades)
	s.Equal(0, stats.TradeResult.NumberOfWinningTrades)
	s.Equal(0, stats.TradeResult.NumberOfLosingTrades)
	s.Equal(float64(0), stats.TradeResult.WinRate)
	s.Equal(float64(0), stats.TradeResult.MaxDrawdown)

	s.Equal(float64(0), stats.TradePnl.RealizedPnL)
	s.Equal(float64(0), stats.TradePnl.UnrealizedPnL)
	s.Equal(float64(0), stats.TradePnl.TotalPnL)
	s.Equal(float64(0), stats.TradePnl.MaximumLoss)
	s.Equal(float64(0), stats.TradePnl.MaximumProfit)

	s.Equal(0, stats.TradeHoldingTime.Min)
	s.Equal(0, stats.TradeHoldingTime.Max)
	s.Equal(0, stats.TradeHoldingTime.Avg)

	s.Equal(float64(0), stats.TotalFees)

	// Verify file paths are empty strings
	s.Equal("", stats.OrdersFilePath)
	s.Equal("", stats.TradesFilePath)
	s.Equal("", stats.MarksFilePath)
	s.Equal("", stats.LogsFilePath)
	s.Equal("", stats.MarketDataFilePath)
}

// ============================================================================
// EngineStatus Tests
// ============================================================================

func (s *LiveStatisticsTestSuite) TestEngineStatus_Values() {
	// Verify enum values are correct strings
	s.Equal(EngineStatus("prefetching"), EngineStatusPrefetching)
	s.Equal(EngineStatus("gap_filling"), EngineStatusGapFilling)
	s.Equal(EngineStatus("running"), EngineStatusRunning)
	s.Equal(EngineStatus("stopped"), EngineStatusStopped)
}

// ============================================================================
// ProviderConnectionStatus Tests
// ============================================================================

func (s *LiveStatisticsTestSuite) TestProviderConnectionStatus_Values() {
	// Verify enum values are correct strings
	s.Equal(ProviderConnectionStatus("connected"), ProviderStatusConnected)
	s.Equal(ProviderConnectionStatus("disconnected"), ProviderStatusDisconnected)
}

func (s *LiveStatisticsTestSuite) TestProviderStatusUpdate_Struct() {
	// Test that ProviderStatusUpdate can be created with both statuses
	update := ProviderStatusUpdate{
		MarketDataStatus: ProviderStatusConnected,
		TradingStatus:    ProviderStatusDisconnected,
	}

	s.Equal(ProviderStatusConnected, update.MarketDataStatus)
	s.Equal(ProviderStatusDisconnected, update.TradingStatus)
}

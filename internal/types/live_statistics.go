package types

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// EngineStatus represents the current state of the live trading engine.
type EngineStatus string

const (
	// EngineStatusPrefetching indicates the engine is downloading historical data.
	EngineStatusPrefetching EngineStatus = "prefetching"

	// EngineStatusGapFilling indicates the engine is filling gaps in historical data.
	EngineStatusGapFilling EngineStatus = "gap_filling"

	// EngineStatusRunning indicates the engine is processing live market data.
	EngineStatusRunning EngineStatus = "running"

	// EngineStatusStopped indicates the engine has stopped.
	EngineStatusStopped EngineStatus = "stopped"
)

// LiveTradeStats contains statistics for a live trading session.
type LiveTradeStats struct {
	// ID is the unique identifier for this trading session (e.g., "run_1").
	ID string `yaml:"id" json:"id"`

	// Date is the date of this statistics record in YYYY-MM-DD format.
	Date string `yaml:"date" json:"date"`

	// SessionStart is when this trading session started.
	SessionStart time.Time `yaml:"session_start" json:"session_start"`

	// LastUpdated is when these statistics were last updated.
	LastUpdated time.Time `yaml:"last_updated" json:"last_updated"`

	// Symbols being traded in this session.
	Symbols []string `yaml:"symbols" json:"symbols"`

	// TradeResult contains trade counts and win rate.
	TradeResult TradeResult `yaml:"trade_result" json:"trade_result"`

	// TradePnl contains profit/loss breakdown.
	TradePnl TradePnl `yaml:"trade_pnl" json:"trade_pnl"`

	// TradeHoldingTime contains holding time statistics.
	TradeHoldingTime TradeHoldingTime `yaml:"trade_holding_time" json:"trade_holding_time"`

	// TotalFees is the sum of all trading fees paid.
	TotalFees float64 `yaml:"total_fees" json:"total_fees"`

	// OrdersFilePath is the path to the orders parquet file.
	OrdersFilePath string `yaml:"orders_file_path" json:"orders_file_path"`

	// TradesFilePath is the path to the trades parquet file.
	TradesFilePath string `yaml:"trades_file_path" json:"trades_file_path"`

	// MarksFilePath is the path to the marks parquet file.
	MarksFilePath string `yaml:"marks_file_path" json:"marks_file_path"`

	// LogsFilePath is the path to the logs parquet file.
	LogsFilePath string `yaml:"logs_file_path" json:"logs_file_path"`

	// MarketDataFilePath is the path to the market data parquet file.
	MarketDataFilePath string `yaml:"market_data_file_path" json:"market_data_file_path"`

	// Strategy contains metadata about the strategy that generated these stats.
	Strategy StrategyInfo `yaml:"strategy" json:"strategy"`
}

// DailyLiveTradeStats contains both daily and cumulative statistics for a session.
// Daily stats are reset at the start of each day, while cumulative stats track
// the entire session from start to finish.
type DailyLiveTradeStats struct {
	// Daily statistics for this specific day only.
	Daily LiveTradeStats `yaml:"daily" json:"daily"`

	// Cumulative statistics from session start.
	Cumulative LiveTradeStats `yaml:"cumulative" json:"cumulative"`
}

// WriteLiveTradeStats writes live trade statistics to a YAML file.
func WriteLiveTradeStats(path string, stats LiveTradeStats) error {
	data, err := yaml.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal live trade stats to YAML: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write live trade stats to file: %w", err)
	}

	return nil
}

// ReadLiveTradeStats reads live trade statistics from a YAML file.
func ReadLiveTradeStats(path string) (LiveTradeStats, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return LiveTradeStats{}, fmt.Errorf("failed to read live trade stats file: %w", err)
	}

	var stats LiveTradeStats
	if err := yaml.Unmarshal(data, &stats); err != nil {
		return LiveTradeStats{}, fmt.Errorf("failed to unmarshal live trade stats: %w", err)
	}

	return stats, nil
}

// NewLiveTradeStats creates a new LiveTradeStats with initialized values.
func NewLiveTradeStats(runID string, symbols []string, strategy StrategyInfo) LiveTradeStats {
	now := time.Now()

	return LiveTradeStats{
		ID:           runID,
		Date:         now.Format("2006-01-02"),
		SessionStart: now,
		LastUpdated:  now,
		Symbols:      symbols,
		TradeResult: TradeResult{
			NumberOfTrades:        0,
			NumberOfWinningTrades: 0,
			NumberOfLosingTrades:  0,
			WinRate:               0,
			MaxDrawdown:           0,
		},
		TradePnl: TradePnl{
			RealizedPnL:   0,
			UnrealizedPnL: 0,
			TotalPnL:      0,
			MaximumLoss:   0,
			MaximumProfit: 0,
		},
		TradeHoldingTime: TradeHoldingTime{
			Min: 0,
			Max: 0,
			Avg: 0,
		},
		TotalFees:          0,
		OrdersFilePath:     "",
		TradesFilePath:     "",
		MarksFilePath:      "",
		LogsFilePath:       "",
		MarketDataFilePath: "",
		Strategy:           strategy,
	}
}

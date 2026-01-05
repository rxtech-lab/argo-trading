package types

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type TradeHoldingTime struct {
	// Minimum holding time of a trade in seconds
	Min int `yaml:"min"`
	// Maximum holding time of a trade in seconds
	Max int `yaml:"max"`
	// Average holding time of a trade in seconds
	Avg int `yaml:"avg"`
}

type TradePnl struct {
	// Realized PnL. By adding all the sell trades' pnl.
	RealizedPnL float64 `yaml:"realized_pnl"`
	// Unrealized PnL. By adding all the (buy trades' amount - sell trades' amount) * end trading market price
	UnrealizedPnL float64 `yaml:"unrealized_pnl"`
	// Total PnL. By adding RealizedPnL and UnrealizedPnL.
	TotalPnL float64 `yaml:"total_pnl"`
	// Maximum loss. Find all realized pnl's minimum value.
	MaximumLoss float64 `yaml:"maximum_loss"`
	// Maximum profit. Find all realized pnl's maximum value.
	MaximumProfit float64 `yaml:"maximum_profit"`
}

type TradeResult struct {
	// Count of all trades.
	NumberOfTrades int `yaml:"number_of_trades"`
	// Count of winning trades that has positive pnl.
	NumberOfWinningTrades int `yaml:"number_of_winning_trades"`
	// Count of losing trades that has negative pnl.
	NumberOfLosingTrades int `yaml:"number_of_losing_trades"`
	// Win rate.
	WinRate float64 `yaml:"win_rate"`
	// Maximum drawdown.
	MaxDrawdown float64 `yaml:"max_drawdown"`
}

// StrategyInfo contains metadata about the strategy that generated stats.
type StrategyInfo struct {
	// ID is the unique identifier for the strategy (e.g., "com.example.strategy.sma")
	ID string `yaml:"id" json:"id"`
	// Version is the version of the strategy (from GetRuntimeEngineVersion)
	Version string `yaml:"version" json:"version"`
	// Name is the human-readable name of the strategy
	Name string `yaml:"name" json:"name"`
}

type TradeStats struct {
	// ID is the unique identifier for this backtest run.
	ID string `yaml:"id" json:"id"`
	// Timestamp is when this backtest run was executed.
	Timestamp time.Time `yaml:"timestamp" json:"timestamp"`
	// Symbol of the trading pair.
	Symbol string `yaml:"symbol"`
	// Result of all trades.
	TradeResult TradeResult `yaml:"trade_result"`
	// Total fees.
	TotalFees float64 `yaml:"total_fees"`
	// Holding time of all trades.
	TradeHoldingTime TradeHoldingTime `yaml:"trade_holding_time"`
	// PnL of all trades.
	TradePnl TradePnl `yaml:"trade_pnl"`
	// Buy and hold PnL.
	BuyAndHoldPnl float64 `yaml:"buy_and_hold_pnl"`
	// TradesFilePath is the path to the trades parquet file.
	TradesFilePath string `yaml:"trades_file_path" json:"trades_file_path"`
	// OrdersFilePath is the path to the orders parquet file.
	OrdersFilePath string `yaml:"orders_file_path" json:"orders_file_path"`
	// MarksFilePath is the path to the marks parquet file.
	MarksFilePath string `yaml:"marks_file_path" json:"marks_file_path"`
	// Strategy contains metadata about the strategy that generated these stats.
	Strategy StrategyInfo `yaml:"strategy" json:"strategy"`
	// StrategyPath is the path to the strategy WASM file.
	StrategyPath string `yaml:"strategy_path" json:"strategy_path"`
	// DataPath is the path to the market data file used for this backtest.
	DataPath string `yaml:"data_path" json:"data_path"`
}

func WriteTradeStats(path string, stats []TradeStats) error {
	// Marshal the struct to YAML
	data, err := yaml.Marshal(stats)
	if err != nil {
		return fmt.Errorf("failed to marshal trade stats to YAML: %w", err)
	}

	// Write the YAML data to the file
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write trade stats to file: %w", err)
	}

	return nil
}

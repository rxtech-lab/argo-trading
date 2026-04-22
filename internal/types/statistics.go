package types

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Percentiles captures the 25th, 50th (median), 75th, 90th, 95th and 99th
// percentiles of a numeric distribution. Values are stored as float64 so the
// same struct can be reused for durations (in seconds) and for monetary PnL.
type Percentiles struct {
	P25 float64 `yaml:"p25"`
	P50 float64 `yaml:"p50"`
	P75 float64 `yaml:"p75"`
	P90 float64 `yaml:"p90"`
	P95 float64 `yaml:"p95"`
	P99 float64 `yaml:"p99"`
}

type TradeHoldingTime struct {
	// Minimum holding time of a trade in seconds
	Min int `yaml:"min"`
	// Maximum holding time of a trade in seconds
	Max int `yaml:"max"`
	// Average holding time of a trade in seconds
	Avg int `yaml:"avg"`
	// Median holding time of a trade in seconds
	Median int `yaml:"median"`
	// Percentiles of holding time (in seconds).
	Percentiles Percentiles `yaml:"percentiles"`
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
	// Median PnL across all closing trades.
	MedianPnL float64 `yaml:"median_pnl"`
	// Percentiles of per-trade realized PnL across all closing trades.
	Percentiles Percentiles `yaml:"percentiles"`
	// TotalInvestment is the gross capital deployed across all entry trades
	// (sum of executed_qty * executed_price for BUY fills, which represent
	// the entries for both long and short positions in this engine).
	TotalInvestment float64 `yaml:"total_investment"`
	// PnLPercentage is TotalPnL divided by TotalInvestment, expressed as a
	// fraction (e.g. 0.12 = +12%). It is intentionally NOT computed against
	// the initial cash balance — only the capital actually put to work counts.
	// Zero when TotalInvestment is zero.
	PnLPercentage float64 `yaml:"pnl_percentage"`
}

// MonthlyTradeStats summarises trade activity for a single calendar month.
type MonthlyTradeStats struct {
	// Month in YYYY-MM format (e.g. "2024-01").
	Month string `yaml:"month"`
	// Total number of trade fills (entries and exits) executed during the month.
	NumberOfTrades int `yaml:"number_of_trades"`
	// Number of closing trades (round trips) completed during the month.
	NumberOfTradingPairs int `yaml:"number_of_trading_pairs"`
	// Number of closing trades with a positive realized PnL.
	NumberOfWinningTrades int `yaml:"number_of_winning_trades"`
	// Number of closing trades with a negative realized PnL.
	NumberOfLosingTrades int `yaml:"number_of_losing_trades"`
}

// MonthlyBalanceChange captures equity balance evolution within a single month.
// StartingBalance is the equity (initial balance + cumulative PnL) entering the
// month and EndingBalance is the equity at the last trade of the month.
type MonthlyBalanceChange struct {
	// Month in YYYY-MM format (e.g. "2024-01").
	Month string `yaml:"month"`
	// Equity at the start of the month (before any trades that month).
	StartingBalance float64 `yaml:"starting_balance"`
	// Equity at the end of the month (after the last trade of the month).
	EndingBalance float64 `yaml:"ending_balance"`
	// EndingBalance - StartingBalance.
	Change float64 `yaml:"change"`
	// Sum of per-trade realized PnL booked during the month.
	RealizedPnL float64 `yaml:"realized_pnl"`
}

// MonthlyHoldingTime captures holding time statistics for closing trades whose
// exit happened within a given calendar month.
type MonthlyHoldingTime struct {
	// Month in YYYY-MM format (e.g. "2024-01").
	Month string `yaml:"month"`
	// Minimum holding time in seconds for closing trades exited that month.
	Min int `yaml:"min"`
	// Maximum holding time in seconds for closing trades exited that month.
	Max int `yaml:"max"`
	// Average holding time in seconds for closing trades exited that month.
	Avg int `yaml:"avg"`
	// Median holding time in seconds for closing trades exited that month.
	Median int `yaml:"median"`
}

type TradeResult struct {
	// Count of all trades (both entry and exit fills).
	NumberOfTrades int `yaml:"number_of_trades"`
	// Count of trading pairs (round trips). Each exit trade closes a pair
	// against one or more prior entry trades. Open positions are not counted.
	NumberOfTradingPairs int `yaml:"number_of_trading_pairs"`
	// Count of winning trading pairs (exit trades with positive pnl).
	NumberOfWinningTrades int `yaml:"number_of_winning_trades"`
	// Count of losing trading pairs (exit trades with negative pnl).
	NumberOfLosingTrades int `yaml:"number_of_losing_trades"`
	// Win rate (winning trading pairs / total trading pairs).
	WinRate float64 `yaml:"win_rate"`
	// Maximum drawdown.
	MaxDrawdown float64 `yaml:"max_drawdown"`
	// SharpeRatio is the annualized Sharpe ratio computed from daily equity
	// returns: (mean(daily_return) - rf/N) / stdev(daily_return) * sqrt(N),
	// where rf is the configured annual risk-free rate and N is the
	// annualization factor (e.g. 252 for trading-day returns). Zero when
	// fewer than two daily return observations exist or when the return
	// series has zero variance.
	SharpeRatio float64 `yaml:"sharpe_ratio"`
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
	// LogsFilePath is the path to the logs parquet file.
	LogsFilePath string `yaml:"logs_file_path" json:"logs_file_path"`
	// Strategy contains metadata about the strategy that generated these stats.
	Strategy StrategyInfo `yaml:"strategy" json:"strategy"`
	// StrategyPath is the path to the strategy WASM file.
	StrategyPath string `yaml:"strategy_path" json:"strategy_path"`
	// DataPath is the path to the market data file used for this backtest.
	DataPath string `yaml:"data_path" json:"data_path"`
	// InitialBalance is the starting cash balance for this backtest run.
	InitialBalance float64 `yaml:"initial_balance" json:"initial_balance"`
	// FinalBalance is the portfolio equity at the end of this backtest run (initial_balance + total_pnl).
	FinalBalance float64 `yaml:"final_balance" json:"final_balance"`
	// PortfolioCalculation identifies the portfolio PnL calculation strategy
	// used by the backtest run (e.g., "fifo" or "average_cost"). It records
	// which accounting method was used to compute the per-trade and cumulative
	// PnL values in this stats record, so consumers can interpret them
	// consistently.
	PortfolioCalculation string `yaml:"portfolio_calculation" json:"portfolio_calculation"`
	// MonthlyTrades is a per-month breakdown of trade counts (totals, pairs, wins, losses).
	MonthlyTrades []MonthlyTradeStats `yaml:"monthly_trades"`
	// MonthlyBalance is a per-month breakdown of equity balance changes.
	MonthlyBalance []MonthlyBalanceChange `yaml:"monthly_balance"`
	// MonthlyHoldingTime is a per-month breakdown of closing-trade holding times.
	MonthlyHoldingTime []MonthlyHoldingTime `yaml:"monthly_holding_time"`
	// BacktestConfig is the backtest engine configuration used for this run.
	// Stored as a YAML node so the original structure is preserved verbatim
	// in stats.yaml without coupling this package to the engine package.
	BacktestConfig *yaml.Node `yaml:"backtest_config,omitempty" json:"backtest_config,omitempty"`
	// StrategyConfig is the strategy-specific configuration used for this run,
	// captured from the YAML config file/content supplied to the engine.
	StrategyConfig *yaml.Node `yaml:"strategy_config,omitempty" json:"strategy_config,omitempty"`
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

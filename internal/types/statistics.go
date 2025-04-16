package types

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type TradeHoldingTime struct {
	// Minimum holding time of a trade
	Min int `yaml:"min"`
	// Maximum holding time of a trade
	Max int `yaml:"max"`
	// Average holding time of a trade
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

type TradeStats struct {
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

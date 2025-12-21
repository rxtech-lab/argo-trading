package types

import "time"

// AccountInfo represents the current account state including balance, equity, and P&L information.
type AccountInfo struct {
	// Balance is the current cash balance (excluding unrealized P&L)
	Balance float64 `json:"balance" yaml:"balance"`
	// Equity is the total account value (balance + unrealized P&L)
	Equity float64 `json:"equity" yaml:"equity"`
	// BuyingPower is the available amount for new purchases
	BuyingPower float64 `json:"buying_power" yaml:"buying_power"`
	// RealizedPnL is the total realized profit/loss from closed positions
	RealizedPnL float64 `json:"realized_pnl" yaml:"realized_pnl"`
	// UnrealizedPnL is the total unrealized profit/loss from open positions
	UnrealizedPnL float64 `json:"unrealized_pnl" yaml:"unrealized_pnl"`
	// TotalFees is the total fees paid
	TotalFees float64 `json:"total_fees" yaml:"total_fees"`
	// MarginUsed is the margin currently in use (for margin trading)
	MarginUsed float64 `json:"margin_used" yaml:"margin_used"`
}

// TradeFilter is used to filter trades when querying trade history.
type TradeFilter struct {
	// Symbol filters trades by symbol (empty string means no filter)
	Symbol string `json:"symbol" yaml:"symbol"`
	// StartTime filters trades executed after this time (zero time means no filter)
	StartTime time.Time `json:"start_time" yaml:"start_time"`
	// EndTime filters trades executed before this time (zero time means no filter)
	EndTime time.Time `json:"end_time" yaml:"end_time"`
	// Limit limits the number of trades returned (0 means no limit)
	Limit int `json:"limit" yaml:"limit"`
}

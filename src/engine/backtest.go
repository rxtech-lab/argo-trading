package engine

import (
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
)

type MarketDataSource interface {
	Iterator() func(yield func(types.MarketData) bool)
}

type BacktestEngine interface {
	// SetInitialCapital sets the initial capital for the backtest
	SetInitialCapital(amount float64) error

	// AddStrategy adds a strategy to be tested
	AddStrategy(strategy strategy.TradingStrategy, config string) error

	AddMarketDataSource(source MarketDataSource) error

	// Run executes the backtest
	Run() error

	// GetTrades returns all trades that occurred during the backtest
	GetTrades() []types.Trade

	// GetTradeStats returns statistics about the backtest
	GetTradeStats() types.TradeStats

	// GetEquityCurve returns the equity curve data
	GetEquityCurve() []float64

	// GetTradeStatsByStrategy returns statistics for a specific strategy
	GetTradeStatsByStrategy(strategyName string) types.TradeStats
}

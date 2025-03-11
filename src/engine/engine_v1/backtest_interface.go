package engine

import (
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
)

type StrategyFactory func() strategy.TradingStrategy

type BacktestEngine interface {
	Initialize(config string) error

	// AddStrategy adds a strategy to be tested
	AddStrategy(strategy StrategyFactory, config string) error

	AddMarketDataSource(source types.MarketDataSource) error
	// Run executes the backtest
	Run() error
	// GetTradeStatsByStrategy returns statistics for a specific strategy
	GetTradeStatsByStrategy(strategyName string) types.TradeStats
}

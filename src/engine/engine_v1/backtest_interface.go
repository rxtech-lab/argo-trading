package engine

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
)

type StrategyFactory func() strategy.TradingStrategy

type BacktestEngine interface {
	Name() string
	Initialize(config string) error

	// AddStrategy adds a strategy to be tested
	AddStrategy(strategy StrategyFactory, config string) error

	AddMarketDataSource(source types.MarketDataSource) error
	// Run executes the backtest
	Run() error
	// GetTradeStatsByStrategy returns statistics for a specific strategy
	GetTradeStatsByStrategy(strategyName string) types.TradeStats
	// TestCalculateCommission calculates commission for testing purposes
	TestCalculateCommission(order types.Order, executionPrice float64) (float64, error)
	// TestSetConfig sets configuration values for testing purposes
	TestSetConfig(initialCapital, currentCapital float64, resultsFolder string, startTime, endTime time.Time, commissionFormula string) error
}

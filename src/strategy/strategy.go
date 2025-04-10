package strategy

import (
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/cache"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/indicator"
	"github.com/sirily11/argo-trading-go/src/trading"
	"github.com/sirily11/argo-trading-go/src/types"
)

type StrategyContext struct {
	// DataSource provides the market data as well as the historical data
	DataSource datasource.DataSource
	// IndicatorRegistry is the registry of all indicators
	IndicatorRegistry *indicator.IndicatorRegistry
	// GetPosition returns the current position of the symbol
	GetPosition func(symbol string) (types.Position, error)
	// Cache is the cache of the strategy
	Cache cache.Cache
	// Trading System is used to place orders
	TradingSystem trading.TradingSystem
}

// TradingStrategy interface defines methods that any trading strategy must implement
// Strategies should be stateless - position and order management is handled by the trading system
type TradingStrategy interface {
	// Initialize sets up the strategy with a configuration string and initial context
	// The trading system is responsible for decoding the config string
	Initialize(config string) error

	// ProcessData processes new market data and generates signals
	// It receives a context object with all necessary information to make decisions
	ProcessData(ctx StrategyContext, data types.MarketData, targetSymbol string) ([]types.ExecuteOrder, error)

	// Name returns the name of the strategy
	Name() string
}

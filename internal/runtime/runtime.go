package runtime

import (
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/indicator"
	"github.com/rxtech-lab/argo-trading/internal/marker"
	"github.com/rxtech-lab/argo-trading/internal/trading"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type StrategyRuntime interface {
	// Initialize initializes the strategy with the given config
	Initialize(config string) error
	// InitializeApi initializes the strategy with the given api
	InitializeApi(api strategy.StrategyApi) error
	// ProcessData processes the market data
	ProcessData(data types.MarketData) error
	// GetConfigSchema returns the schema of the strategy config
	GetConfigSchema() (string, error)
	// Name returns the name of the strategy
	Name() string
}

type RuntimeContext struct {
	// DataSource provides the market data as well as the historical data
	DataSource datasource.DataSource
	// IndicatorRegistry is the registry of all indicators
	IndicatorRegistry indicator.IndicatorRegistry
	// Cache is the cache of the strategy
	Cache cache.Cache
	// Trading System is used to place orders
	TradingSystem trading.TradingSystem
	// Marker is used to mark a point in time with a signal and a reason
	Marker marker.Marker
}

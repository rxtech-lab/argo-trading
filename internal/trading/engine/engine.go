package engine

import (
	"context"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/runtime"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// Lifecycle callback types for live trading phases.
// All callbacks with error return can abort execution if they return an error.

// OnEngineStartCallback is called when the engine starts successfully.
// previousDataPath contains the path to the parquet file with historical data if persistence is enabled,
// or an empty string if persistence is disabled.
type OnEngineStartCallback func(symbols []string, interval string, previousDataPath string) error

// OnEngineStopCallback is called when the engine stops (always called via defer).
type OnEngineStopCallback func(err error)

// OnMarketDataCallback is called for each market data point received.
// runID is the session run ID (UUID) when persistence is enabled, or empty string otherwise.
type OnMarketDataCallback func(runID string, data types.MarketData) error

// OnOrderPlacedCallback is called when an order is placed by the strategy.
type OnOrderPlacedCallback func(order types.ExecuteOrder) error

// OnOrderFilledCallback is called when an order is filled.
type OnOrderFilledCallback func(order types.Order) error

// OnErrorCallback is called when a non-fatal error occurs.
type OnErrorCallback func(err error)

// OnStrategyErrorCallback is called when the strategy returns an error.
type OnStrategyErrorCallback func(data types.MarketData, err error)

// OnStatsUpdateCallback is called when trading statistics are updated.
type OnStatsUpdateCallback func(stats types.LiveTradeStats) error

// OnStatusUpdateCallback is called when engine status changes.
type OnStatusUpdateCallback func(status types.EngineStatus) error

// OnProviderStatusChangeCallback is called when provider connection status changes.
// It receives the current status of both market data and trading providers.
type OnProviderStatusChangeCallback func(status types.ProviderStatusUpdate) error

// LiveTradingCallbacks holds all lifecycle callback functions for the live trading engine.
// All fields are pointers - nil means no callback will be invoked.
type LiveTradingCallbacks struct {
	// OnEngineStart is called when the engine starts successfully.
	OnEngineStart *OnEngineStartCallback

	// OnEngineStop is called when the engine stops (always called via defer).
	OnEngineStop *OnEngineStopCallback

	// OnMarketData is called for each market data point received.
	OnMarketData *OnMarketDataCallback

	// OnOrderPlaced is called when an order is placed by the strategy.
	OnOrderPlaced *OnOrderPlacedCallback

	// OnOrderFilled is called when an order is filled.
	OnOrderFilled *OnOrderFilledCallback

	// OnError is called when a non-fatal error occurs.
	OnError *OnErrorCallback

	// OnStrategyError is called when the strategy returns an error.
	OnStrategyError *OnStrategyErrorCallback

	// OnStatsUpdate is called when trading statistics are updated.
	OnStatsUpdate *OnStatsUpdateCallback

	// OnStatusUpdate is called when engine status changes.
	OnStatusUpdate *OnStatusUpdateCallback

	// OnProviderStatusChange is called when provider connection status changes.
	// It receives the current status of both market data and trading providers.
	OnProviderStatusChange *OnProviderStatusChangeCallback
}

// PrefetchConfig holds configuration for historical data prefetching.
type PrefetchConfig struct {
	// Enabled enables historical data prefetching before live trading starts
	Enabled bool `json:"enabled" yaml:"enabled" jsonschema:"description=Enable historical data prefetch"`

	// StartTimeType specifies how to determine the prefetch start time.
	// "date" uses StartTime as absolute time, "days" uses Days relative to now.
	StartTimeType string `json:"start_time_type" yaml:"start_time_type" jsonschema:"description=How to specify start time,enum=date,enum=days"`

	// StartTime is the absolute start time for prefetching (used when StartTimeType is "date")
	StartTime time.Time `json:"start_time" yaml:"start_time" jsonschema:"description=Absolute start time (when type is date)"`

	// Days is the number of days of history to prefetch (used when StartTimeType is "days")
	Days int `json:"days" yaml:"days" jsonschema:"description=Number of days to prefetch (when type is days)"`
}

// LiveTradingEngineConfig holds the configuration for the live trading engine.
type LiveTradingEngineConfig struct {
	// MarketDataCacheSize is the number of historical data points to cache per symbol
	// for indicator calculations (default: 1000)
	MarketDataCacheSize int `json:"market_data_cache_size" yaml:"market_data_cache_size" jsonschema:"description=Number of market data points to cache per symbol,default=1000"`

	// EnableLogging enables strategy log storage
	EnableLogging bool `json:"enable_logging" yaml:"enable_logging" jsonschema:"description=Enable strategy log storage,default=true"`

	// Prefetch configures historical data prefetching for indicator accuracy
	Prefetch PrefetchConfig `json:"prefetch" yaml:"prefetch" jsonschema:"description=Historical data prefetch configuration"`
}

// GetConfigSchema returns the JSON schema for LiveTradingEngineConfig.
func GetConfigSchema() (string, error) {
	return strategy.ToJSONSchema(&LiveTradingEngineConfig{}) //nolint:exhaustruct // Empty config for schema generation
}

// LiveTradingEngine orchestrates real-time strategy execution with streaming market data.
//
//nolint:interfacebloat // Engine is a core interface that naturally requires multiple methods
type LiveTradingEngine interface {
	// Initialize sets up the engine with the given configuration.
	Initialize(config LiveTradingEngineConfig) error

	// LoadStrategyFromFile loads a WASM strategy from a file path.
	LoadStrategyFromFile(strategyPath string) error

	// LoadStrategyFromBytes loads a WASM strategy from bytes.
	LoadStrategyFromBytes(strategyBytes []byte) error

	// LoadStrategy loads a pre-created strategy runtime.
	LoadStrategy(strategy runtime.StrategyRuntime) error

	// SetStrategyConfig sets the strategy configuration (YAML/JSON string).
	SetStrategyConfig(config string) error

	// SetMarketDataProvider configures the market data provider.
	// The provider must support the Stream() method.
	SetMarketDataProvider(provider provider.Provider) error

	// SetTradingProvider configures the trading provider.
	SetTradingProvider(provider tradingprovider.TradingSystemProvider) error

	// SetDataOutputPath sets the base directory for session data output (orders, trades, marks, logs, stats).
	// Must be called before Run() if persistence is desired.
	SetDataOutputPath(path string) error

	// Run starts the live trading engine.
	// Blocks until context is cancelled or a fatal error occurs.
	Run(ctx context.Context, callbacks LiveTradingCallbacks) error

	// GetConfigSchema returns the JSON schema for engine configuration.
	GetConfigSchema() (string, error)
}

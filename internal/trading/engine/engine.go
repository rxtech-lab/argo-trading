package engine

import (
	"context"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/trading/wallet"
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

// OnPrefetchProgressCallback is called during historical data prefetch and gap-fill
// downloads with per-symbol progress. current/total are in the provider's reported units
// (Binance reports time elapsed in ms vs. total range; Polygon reports request counts).
type OnPrefetchProgressCallback func(symbol string, current float64, total float64, message string) error

// OnProviderStatusChangeCallback is called when provider connection status changes.
// It receives the current status of both market data and trading providers.
type OnProviderStatusChangeCallback func(status types.ProviderStatusUpdate) error

// LiveTradingDataCategory identifies a logical group of persisted live-trading data
// that has changed. Consumers use these to invalidate the matching UI surfaces.
type LiveTradingDataCategory string

const (
	LiveTradingDataCategoryMarketData LiveTradingDataCategory = "market_data"
	LiveTradingDataCategoryTrades     LiveTradingDataCategory = "trades"
	LiveTradingDataCategoryOrders     LiveTradingDataCategory = "orders"
	LiveTradingDataCategoryMarks      LiveTradingDataCategory = "marks"
	LiveTradingDataCategoryLogs       LiveTradingDataCategory = "logs"
	LiveTradingDataCategoryStats      LiveTradingDataCategory = "stats"
)

// OnLiveDataChangedCallback is invoked after a logical batch of persistence writes
// completes. It is a coalesced reload/invalidation hint, not a per-write notification.
//
//   - runID:      session run ID; empty when persistence is disabled.
//   - categories: which data groups were written in this batch.
//   - finalized:  true on the final emission after the engine stops and all writers
//     have flushed. Consumers should treat this as a "force full reload" signal.
//   - sequence:   monotonically increasing within a single Run(), starting at 1.
//
// Consumers are expected to debounce reloads (e.g. ~500ms) and only refresh the
// surfaces matching the reported categories.
type OnLiveDataChangedCallback func(runID string, categories []LiveTradingDataCategory, finalized bool, sequence int64) error

// Wallet change callbacks are coalesced invalidation hints — they carry no
// payload and signal that the consumer should re-fetch from the wallet API.
// They fire only when the underlying value has changed since the previous tick
// or order event.

// OnOrderChangedCallback fires when an order is placed, filled, or its status
// otherwise changes.
type OnOrderChangedCallback func() error

// OnBalanceChangedCallback fires when the combined value of all assets changes.
type OnBalanceChangedCallback func() error

// OnBuyingPowerChangedCallback fires when the broker-reported buying power changes.
type OnBuyingPowerChangedCallback func() error

// OnAssetsChangedCallback fires when the set of held assets or their quantities
// changes.
type OnAssetsChangedCallback func() error

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

	// OnPrefetchProgress is called during prefetch and gap-fill downloads.
	// The current EngineStatus (from OnStatusUpdate) distinguishes prefetch vs. gap-fill.
	OnPrefetchProgress *OnPrefetchProgressCallback

	// OnProviderStatusChange is called when provider connection status changes.
	// It receives the current status of both market data and trading providers.
	OnProviderStatusChange *OnProviderStatusChangeCallback

	// OnLiveDataChanged is a coalesced invalidation hint emitted after each
	// per-tick batch of persistence writes completes, and once more after the
	// engine stops with finalized=true.
	OnLiveDataChanged *OnLiveDataChangedCallback

	// OnOrderChanged signals that the set of orders has changed (placed, filled,
	// or status update) and the UI should re-fetch from the wallet API.
	OnOrderChanged *OnOrderChangedCallback

	// OnBalanceChanged signals that the combined value of all assets has changed.
	OnBalanceChanged *OnBalanceChangedCallback

	// OnBuyingPowerChanged signals that the broker-reported buying power has changed.
	OnBuyingPowerChanged *OnBuyingPowerChangedCallback

	// OnAssetsChanged signals that the set or quantities of held assets has changed.
	OnAssetsChanged *OnAssetsChangedCallback
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

	// Wallet returns a read-only wallet facade over the currently configured
	// trading provider. Returns an error if no trading provider has been set.
	// The wallet is callable outside Run() so the UI can show balance/assets
	// without an active session.
	Wallet() (wallet.Wallet, error)
}

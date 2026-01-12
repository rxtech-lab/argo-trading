package engine

import (
	"context"

	"github.com/invopop/jsonschema"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

// Lifecycle callback types for live trading phases.
// All callbacks with error return can abort execution if they return an error.

// OnEngineStartCallback is called when the engine starts successfully.
type OnEngineStartCallback func(symbols []string, interval string) error

// OnEngineStopCallback is called when the engine stops (always called via defer).
type OnEngineStopCallback func(err error)

// OnMarketDataCallback is called for each market data point received.
type OnMarketDataCallback func(data types.MarketData) error

// OnOrderPlacedCallback is called when an order is placed by the strategy.
type OnOrderPlacedCallback func(order types.ExecuteOrder) error

// OnOrderFilledCallback is called when an order is filled.
type OnOrderFilledCallback func(order types.Order) error

// OnErrorCallback is called when a non-fatal error occurs.
type OnErrorCallback func(err error)

// OnStrategyErrorCallback is called when the strategy returns an error.
type OnStrategyErrorCallback func(data types.MarketData, err error)

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
}

// LiveTradingEngineConfig holds the configuration for the live trading engine.
type LiveTradingEngineConfig struct {
	// Symbols to trade/monitor
	Symbols []string `json:"symbols" yaml:"symbols" jsonschema:"description=List of symbols to stream and trade" validate:"required,min=1"`

	// Interval for market data streaming (e.g., "1m", "5m", "1h")
	Interval string `json:"interval" yaml:"interval" jsonschema:"description=Candlestick interval for streaming data,default=1m" validate:"required"`

	// MarketDataCacheSize is the number of historical data points to cache per symbol
	// for indicator calculations (default: 1000)
	MarketDataCacheSize int `json:"market_data_cache_size" yaml:"market_data_cache_size" jsonschema:"description=Number of market data points to cache per symbol,default=1000"`

	// EnableLogging enables strategy log storage
	EnableLogging bool `json:"enable_logging" yaml:"enable_logging" jsonschema:"description=Enable strategy log storage,default=true"`

	// LogOutputPath is the directory for log files (optional)
	LogOutputPath string `json:"log_output_path" yaml:"log_output_path" jsonschema:"description=Directory for log output files"`
}

// GetConfigSchema returns the JSON schema for LiveTradingEngineConfig.
func GetConfigSchema() (string, error) {
	reflector := jsonschema.Reflector{}                     //nolint:exhaustruct // Using default reflector settings
	schema := reflector.Reflect(&LiveTradingEngineConfig{}) //nolint:exhaustruct // Empty config for schema generation

	schemaBytes, err := schema.MarshalJSON()
	if err != nil {
		return "", err
	}

	return string(schemaBytes), nil
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

	// Run starts the live trading engine.
	// Blocks until context is cancelled or a fatal error occurs.
	Run(ctx context.Context, callbacks LiveTradingCallbacks) error

	// GetConfigSchema returns the JSON schema for engine configuration.
	GetConfigSchema() (string, error)
}

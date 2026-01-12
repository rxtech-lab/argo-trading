package swiftargo

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	engine_v1 "github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

// TradingEngineHelper is the callback interface for live trading lifecycle events.
// Swift consumers implement this interface to receive notifications during trading.
type TradingEngineHelper interface {
	// OnEngineStart is called when the engine starts successfully.
	// symbols: list of symbols being traded
	// interval: the candlestick interval (e.g., "1m", "5m")
	OnEngineStart(symbols StringCollection, interval string) error

	// OnEngineStop is called when the engine stops (always called via defer).
	OnEngineStop(err error)

	// OnMarketData is called for each market data point received.
	OnMarketData(symbol string, timestamp int64, open, high, low, close, volume float64) error

	// OnOrderPlaced is called when an order is placed by the strategy.
	// orderJSON is the JSON representation of the ExecuteOrder.
	OnOrderPlaced(orderJSON string) error

	// OnOrderFilled is called when an order is filled.
	// orderJSON is the JSON representation of the filled Order.
	OnOrderFilled(orderJSON string) error

	// OnError is called when a non-fatal error occurs.
	OnError(err error)

	// OnStrategyError is called when the strategy returns an error.
	OnStrategyError(symbol string, timestamp int64, err error)
}

// TradingProviderInfo contains metadata about a trading provider.
type TradingProviderInfo struct {
	Name           string
	DisplayName    string
	Description    string
	IsPaperTrading bool
}

// TradingEngine wraps the live trading engine for Swift consumers.
type TradingEngine struct {
	helper TradingEngineHelper
	engine engine.LiveTradingEngine

	// Cancellation support
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

// GetSupportedTradingProviders returns a StringCollection of all supported trading provider names.
// This follows the gomobile pattern where slices cannot be returned directly.
func GetSupportedTradingProviders() StringCollection {
	providers := tradingprovider.GetSupportedProviders()

	return &StringArray{items: providers}
}

// GetTradingProviderSchema returns the JSON schema for a specific trading provider's configuration.
// The providerName should be one of the values returned by GetSupportedTradingProviders().
// Returns empty string if the provider is not found.
func GetTradingProviderSchema(providerName string) string {
	schema, err := tradingprovider.GetProviderConfigSchema(providerName)
	if err != nil {
		return ""
	}

	return schema
}

// GetTradingProviderInfo returns metadata for a specific trading provider.
// Returns nil and an error if the provider is not found.
func GetTradingProviderInfo(providerName string) (*TradingProviderInfo, error) {
	info, err := tradingprovider.GetProviderInfo(providerName)
	if err != nil {
		return nil, err
	}

	return &TradingProviderInfo{
		Name:           info.Name,
		DisplayName:    info.DisplayName,
		Description:    info.Description,
		IsPaperTrading: info.IsPaperTrading,
	}, nil
}

// GetLiveTradingEngineConfigSchema returns the JSON schema for the live trading engine configuration.
func GetLiveTradingEngineConfigSchema() string {
	schema, err := engine.GetConfigSchema()
	if err != nil {
		return ""
	}

	return schema
}

// GetSupportedMarketDataProviders returns a StringCollection of all supported market data provider names
// that can be used with SetMarketDataProvider.
func GetSupportedMarketDataProviders() StringCollection {
	return &StringArray{items: []string{
		string(provider.ProviderBinance),
		string(provider.ProviderPolygon),
	}}
}

// NewTradingEngine creates a new TradingEngine with the given helper for callbacks.
// The helper can be nil if no callbacks are needed.
func NewTradingEngine(helper TradingEngineHelper) (*TradingEngine, error) {
	eng, err := engine_v1.NewLiveTradingEngineV1()
	if err != nil {
		return nil, err
	}

	return &TradingEngine{
		helper:     helper,
		engine:     eng,
		mu:         sync.Mutex{},
		cancelFunc: nil,
	}, nil
}

// Initialize sets up the engine with the given JSON configuration.
// The configJSON must conform to GetLiveTradingEngineConfigSchema().
func (t *TradingEngine) Initialize(configJSON string) error {
	var config engine.LiveTradingEngineConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return fmt.Errorf("failed to parse engine config: %w", err)
	}

	return t.engine.Initialize(config)
}

// SetTradingProvider sets the trading provider using provider name and JSON config.
// The providerName should be one of the values returned by GetSupportedTradingProviders().
// The configJSON must conform to GetTradingProviderSchema(providerName).
func (t *TradingEngine) SetTradingProvider(providerName string, configJSON string) error {
	config, err := tradingprovider.ParseProviderConfig(providerName, configJSON)
	if err != nil {
		return fmt.Errorf("failed to parse trading provider config: %w", err)
	}

	tradingProvider, err := tradingprovider.NewTradingSystemProvider(
		tradingprovider.ProviderType(providerName),
		config,
	)
	if err != nil {
		return fmt.Errorf("failed to create trading provider: %w", err)
	}

	return t.engine.SetTradingProvider(tradingProvider)
}

// SetMarketDataProvider sets the market data provider for streaming data.
// providerName: "binance" or "polygon"
// configJSON: JSON config (polygon requires {"apiKey": "your-key"}, binance can be empty "{}").
func (t *TradingEngine) SetMarketDataProvider(providerName string, configJSON string) error {
	var marketProvider provider.Provider
	var err error

	switch provider.ProviderType(providerName) {
	case provider.ProviderBinance:
		marketProvider, err = provider.NewBinanceClient()
	case provider.ProviderPolygon:
		if configJSON == "" || configJSON == "{}" {
			return fmt.Errorf("polygon provider requires apiKey in config")
		}
		var config struct {
			ApiKey string `json:"apiKey"`
		}
		if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
			return fmt.Errorf("failed to parse polygon config: %w", err)
		}
		if config.ApiKey == "" {
			return fmt.Errorf("polygon provider requires apiKey in config")
		}
		marketProvider, err = provider.NewPolygonClient(config.ApiKey)
	default:
		return fmt.Errorf("unsupported market data provider: %s", providerName)
	}

	if err != nil {
		return fmt.Errorf("failed to create market data provider: %w", err)
	}

	return t.engine.SetMarketDataProvider(marketProvider)
}

// SetWasm loads a WASM strategy from the given file path.
func (t *TradingEngine) SetWasm(wasmPath string) error {
	return t.engine.LoadStrategyFromFile(wasmPath)
}

// SetStrategyConfig sets the strategy configuration (YAML or JSON string).
func (t *TradingEngine) SetStrategyConfig(config string) error {
	return t.engine.SetStrategyConfig(config)
}

// Run starts the live trading engine. This method is blocking.
// Can be cancelled by calling Cancel() from another goroutine.
func (t *TradingEngine) Run() error {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Store cancel function with mutex protection
	t.mu.Lock()
	t.cancelFunc = cancel
	t.mu.Unlock()

	// Ensure we clean up the cancel function when done
	defer func() {
		t.mu.Lock()
		t.cancelFunc = nil
		t.mu.Unlock()
	}()

	// Create callbacks from helper interface
	callbacks := t.createCallbacks()

	return t.engine.Run(ctx, callbacks)
}

// Cancel cancels any in-progress run.
// This method is safe to call from any goroutine (e.g., Swift's main thread).
// Returns true if a run was cancelled, false if no run was in progress.
func (t *TradingEngine) Cancel() bool {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.cancelFunc != nil {
		t.cancelFunc()
		t.cancelFunc = nil

		return true
	}

	return false
}

// createCallbacks creates engine.LiveTradingCallbacks from the helper interface.
func (t *TradingEngine) createCallbacks() engine.LiveTradingCallbacks {
	var callbacks engine.LiveTradingCallbacks

	if t.helper == nil {
		return callbacks
	}

	// OnEngineStart callback
	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string) error {
		symbolsCollection := &StringArray{items: symbols}

		return t.helper.OnEngineStart(symbolsCollection, interval)
	})
	callbacks.OnEngineStart = &onStart

	// OnEngineStop callback
	onStop := engine.OnEngineStopCallback(func(err error) {
		t.helper.OnEngineStop(err)
	})
	callbacks.OnEngineStop = &onStop

	// OnMarketData callback
	onMarketData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		return t.helper.OnMarketData(
			data.Symbol,
			data.Time.UnixMilli(),
			data.Open,
			data.High,
			data.Low,
			data.Close,
			data.Volume,
		)
	})
	callbacks.OnMarketData = &onMarketData

	// OnOrderPlaced callback
	onOrderPlaced := engine.OnOrderPlacedCallback(func(order types.ExecuteOrder) error {
		orderJSON, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %w", err)
		}

		return t.helper.OnOrderPlaced(string(orderJSON))
	})
	callbacks.OnOrderPlaced = &onOrderPlaced

	// OnOrderFilled callback
	onOrderFilled := engine.OnOrderFilledCallback(func(order types.Order) error {
		orderJSON, err := json.Marshal(order)
		if err != nil {
			return fmt.Errorf("failed to marshal order: %w", err)
		}

		return t.helper.OnOrderFilled(string(orderJSON))
	})
	callbacks.OnOrderFilled = &onOrderFilled

	// OnError callback
	onError := engine.OnErrorCallback(func(err error) {
		t.helper.OnError(err)
	})
	callbacks.OnError = &onError

	// OnStrategyError callback
	onStrategyError := engine.OnStrategyErrorCallback(func(data types.MarketData, err error) {
		t.helper.OnStrategyError(data.Symbol, data.Time.UnixMilli(), err)
	})
	callbacks.OnStrategyError = &onStrategyError

	return callbacks
}

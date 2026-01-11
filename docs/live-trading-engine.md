---
title: Live Trading Engine Architecture
description: Design and implementation plan for the live trading engine with streaming market data
---

# Live Trading Engine Architecture

This document outlines the architecture and implementation plan for the Live Trading Engine in Argo Trading. The engine enables real-time strategy execution using streaming market data and live/paper trading providers, following patterns established by the existing backtest engine.

## Overview

The Live Trading Engine is designed to:

1. **Load and initialize WASM strategies** - Similar to the backtest engine
2. **Connect to streaming market data providers** - Using the existing `Stream()` interface
3. **Execute trades via trading providers** - Using the existing `TradingSystemProvider` interface
4. **Support configurable providers** - Both market data and trading providers accept configuration

### Key Design Principles

- **Provider-based architecture** - Consistent with existing market data and trading provider patterns
- **Configuration-driven** - All providers receive typed configuration at initialization
- **Reuse existing interfaces** - Leverage `Provider.Stream()` and `TradingSystemProvider`
- **Strategy compatibility** - Same WASM strategies work in both backtest and live modes
- **Graceful lifecycle management** - Proper startup, shutdown, and error handling

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Live Trading Engine                             │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                          Engine Configuration                          │ │
│  │  - Market Data Provider Type + Config                                  │ │
│  │  - Trading Provider Type + Config                                      │ │
│  │  - Strategy WASM Path                                                  │ │
│  │  - Strategy Config                                                     │ │
│  │  - Symbols to trade                                                    │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
                                      ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                            Strategy (WASM Plugin)                            │
│  ┌─────────────────────────────────────────────────────────────────────────┐│
│  │ ProcessData(MarketData) → Analyze → PlaceOrder / GetPositions / etc.   ││
│  └─────────────────────────────────────────────────────────────────────────┘│
└─────────────────────────────────────────────────────────────────────────────┘
                                      │
              ┌───────────────────────┴───────────────────────┐
              │                                               │
              ▼                                               ▼
┌──────────────────────────────┐            ┌──────────────────────────────────┐
│   Market Data Provider       │            │      Trading Provider            │
│   (with Stream support)      │            │   (TradingSystemProvider)        │
│                              │            │                                  │
│  ┌────────────────────────┐  │            │  ┌────────────────────────────┐  │
│  │ Stream(ctx, symbols,   │  │            │  │ PlaceOrder(order)          │  │
│  │        interval)       │  │            │  │ GetPositions()             │  │
│  │ → iter.Seq2[MarketData]│  │            │  │ CancelOrder(id)            │  │
│  └────────────────────────┘  │            │  │ GetAccountInfo()           │  │
│                              │            │  └────────────────────────────┘  │
│  Providers:                  │            │                                  │
│  - Binance (WebSocket)       │            │  Providers:                      │
│  - Polygon (WebSocket)       │            │  - binance-paper                 │
│                              │            │  - binance-live                  │
│  Config passed at init       │            │  - ibkr-paper                    │
│                              │            │  - ibkr-live                     │
└──────────────────────────────┘            └──────────────────────────────────┘
              │                                               │
              ▼                                               ▼
┌──────────────────────────────┐            ┌──────────────────────────────────┐
│   WebSocket Connections      │            │      Broker/Exchange APIs        │
│   - Binance: wss://stream.   │            │   - Binance REST + WebSocket     │
│     binance.com:9443/ws      │            │   - IBKR TWS/Gateway             │
│   - Polygon: wss://socket.   │            │                                  │
│     polygon.io/stocks        │            │                                  │
└──────────────────────────────┘            └──────────────────────────────────┘
```

## Core Interfaces

### LiveTradingEngine Interface

```go
package engine

import (
    "context"

    "github.com/rxtech-lab/argo-trading/internal/runtime"
    "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
    tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
)

// LiveTradingEngine orchestrates real-time strategy execution with streaming market data.
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
    SetMarketDataProvider(providerType provider.ProviderType, config any) error

    // SetTradingProvider configures the trading provider.
    SetTradingProvider(providerType tradingprovider.ProviderType, config any) error

    // Run starts the live trading engine.
    // Blocks until context is cancelled or a fatal error occurs.
    Run(ctx context.Context, callbacks LiveTradingCallbacks) error

    // GetConfigSchema returns the JSON schema for engine configuration.
    GetConfigSchema() (string, error)
}
```

### LiveTradingEngineConfig

```go
// LiveTradingEngineConfig holds the configuration for the live trading engine.
type LiveTradingEngineConfig struct {
    // Symbols to trade/monitor
    Symbols []string `json:"symbols" yaml:"symbols" jsonschema:"description=List of symbols to stream and trade" validate:"required,min=1"`

    // Interval for market data streaming (e.g., "1m", "5m", "1h")
    Interval string `json:"interval" yaml:"interval" jsonschema:"description=Candlestick interval for streaming data,default=1m" validate:"required"`

    // MarketDataCacheSize is the number of historical data points to cache per symbol
    // for indicator calculations (default: 1000)
    MarketDataCacheSize int `json:"market_data_cache_size" yaml:"market_data_cache_size" jsonschema:"description=Number of market data points to cache per symbol,default=1000"`

    // DecimalPrecision for order quantities (0 = integers only)
    DecimalPrecision int `json:"decimal_precision" yaml:"decimal_precision" jsonschema:"description=Decimal precision for order quantities,default=8"`

    // EnableLogging enables strategy log storage
    EnableLogging bool `json:"enable_logging" yaml:"enable_logging" jsonschema:"description=Enable strategy log storage,default=true"`

    // LogOutputPath is the directory for log files (optional)
    LogOutputPath string `json:"log_output_path" yaml:"log_output_path" jsonschema:"description=Directory for log output files"`
}
```

### Lifecycle Callbacks

```go
// LiveTradingCallbacks holds lifecycle callback functions for the live trading engine.
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

type OnEngineStartCallback func(symbols []string, interval string) error
type OnEngineStopCallback func(err error)
type OnMarketDataCallback func(data types.MarketData) error
type OnOrderPlacedCallback func(order types.ExecuteOrder) error
type OnOrderFilledCallback func(order types.Order) error
type OnErrorCallback func(err error)
type OnStrategyErrorCallback func(data types.MarketData, err error)
```

## Provider Configuration

### Market Data Provider Configuration

The market data provider is configured with a type and provider-specific configuration:

```go
// MarketDataProviderConfig holds the configuration for a market data provider.
type MarketDataProviderConfig struct {
    // ProviderType identifies the provider (e.g., "binance", "polygon")
    ProviderType provider.ProviderType `json:"provider_type" yaml:"provider_type" validate:"required"`

    // Config is the provider-specific configuration
    Config any `json:"config" yaml:"config"`
}
```

#### Binance Market Data Configuration

```go
// BinanceMarketDataConfig for Binance streaming.
type BinanceMarketDataConfig struct {
    // UseTestnet uses the Binance testnet WebSocket endpoint
    UseTestnet bool `json:"use_testnet" yaml:"use_testnet" jsonschema:"description=Use Binance testnet for streaming,default=false"`
}
```

#### Polygon Market Data Configuration

```go
// PolygonMarketDataConfig for Polygon streaming.
type PolygonMarketDataConfig struct {
    // ApiKey is the Polygon.io API key (required)
    ApiKey string `json:"api_key" yaml:"api_key" jsonschema:"description=Polygon.io API key" validate:"required"`
}
```

### Trading Provider Configuration

Trading providers follow the existing pattern in [trading-system.md](./trading-system.md):

```go
// TradingProviderConfig holds the configuration for a trading provider.
type TradingProviderConfig struct {
    // ProviderType identifies the provider (e.g., "binance-paper", "binance-live")
    ProviderType tradingprovider.ProviderType `json:"provider_type" yaml:"provider_type" validate:"required"`

    // Config is the provider-specific configuration
    Config any `json:"config" yaml:"config" validate:"required"`
}
```

Existing trading provider configurations:

- **BinanceProviderConfig**: `ApiKey`, `SecretKey`
- **IBKRProviderConfig**: `Host`, `Port`, `ClientID`, `AccountID`

## Implementation Plan

### Phase 1: Core Engine Structure

**Files to create:**

```
internal/
└── live/
    └── engine/
        ├── engine.go           # LiveTradingEngine interface
        ├── engine_v1/
        │   ├── live_trading_v1.go    # Main engine implementation
        │   ├── config.go             # Configuration structs
        │   ├── callbacks.go          # Callback type definitions
        │   └── streaming_datasource.go # Real-time data source adapter
        └── engine_test.go      # Engine tests
```

**Tasks:**

1. Define `LiveTradingEngine` interface in `engine.go`
2. Create `LiveTradingEngineConfig` struct with JSON schema support
3. Define lifecycle callbacks for real-time events
4. Create `LiveTradingEngineV1` struct implementing the interface

### Phase 2: Market Data Provider Integration

**Tasks:**

1. Create `SetMarketDataProvider()` implementation that:
   - Accepts provider type and configuration
   - Creates the appropriate provider instance using the existing factory
   - Validates the provider supports streaming

2. Create `StreamingDataSource` adapter that:
   - Wraps the `iter.Seq2[MarketData, error]` from `Provider.Stream()`
   - Implements `datasource.DataSource` interface for strategy compatibility
   - Maintains a sliding window cache of recent data for indicators
   - Provides `GetRange()`, `GetPreviousNumberOfDataPoints()`, etc.

```go
// StreamingDataSource adapts real-time streaming data to the DataSource interface.
type StreamingDataSource struct {
    provider      provider.Provider
    symbols       []string
    interval      string
    cache         *SlidingWindowCache
    currentData   map[string]types.MarketData // Latest data per symbol
    mu            sync.RWMutex
}

func (s *StreamingDataSource) Initialize(ctx context.Context) error
func (s *StreamingDataSource) GetRange(start, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error)
func (s *StreamingDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error)
func (s *StreamingDataSource) ReadLastData(symbol string) (types.MarketData, error)
func (s *StreamingDataSource) AddToCache(data types.MarketData)
```

### Phase 3: Trading Provider Integration

**Tasks:**

1. Create `SetTradingProvider()` implementation that:
   - Accepts provider type and configuration
   - Uses existing `tradingprovider.NewTradingSystemProvider()` factory
   - Validates connection to the trading provider

2. Integrate with `RuntimeContext`:
   - Set `TradingSystem` field with the configured provider
   - Strategy's `PlaceOrder()`, `GetPositions()`, etc. will use this provider

### Phase 4: Strategy Loading and Initialization

**Tasks:**

1. Implement `LoadStrategyFromFile()`:
   - Reuse existing `wasm.NewStrategyWasmRuntime()`
   - Store strategy reference for later initialization

2. Implement `LoadStrategyFromBytes()`:
   - Reuse existing `wasm.NewStrategyWasmRuntimeFromBytes()`

3. Implement `SetStrategyConfig()`:
   - Store configuration string for strategy initialization

4. Strategy initialization in `Run()`:
   - Create `RuntimeContext` with all dependencies
   - Call `strategy.InitializeApi()` with WASM API
   - Check version compatibility
   - Call `strategy.Initialize()` with strategy config

### Phase 5: Main Run Loop

**Tasks:**

1. Implement the main `Run()` method:

```go
func (e *LiveTradingEngineV1) Run(ctx context.Context, callbacks LiveTradingCallbacks) error {
    // 1. Validate all components are configured
    if err := e.preRunCheck(); err != nil {
        return err
    }

    // 2. Initialize the strategy
    if err := e.initializeStrategy(); err != nil {
        return err
    }

    // 3. Call OnEngineStart callback
    if callbacks.OnEngineStart != nil {
        if err := (*callbacks.OnEngineStart)(e.config.Symbols, e.config.Interval); err != nil {
            return err
        }
    }

    // 4. Ensure OnEngineStop is always called
    defer func() {
        if callbacks.OnEngineStop != nil {
            (*callbacks.OnEngineStop)(nil)
        }
    }()

    // 5. Start streaming market data
    stream := e.marketDataProvider.Stream(ctx, e.config.Symbols, e.config.Interval)

    // 6. Process each market data point
    for data, err := range stream {
        // Handle context cancellation
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }

        // Handle stream errors
        if err != nil {
            if callbacks.OnError != nil {
                (*callbacks.OnError)(err)
            }
            continue // or break based on error severity
        }

        // Add to cache for indicator calculations
        e.streamingDataSource.AddToCache(data)

        // Update current market data in trading system if needed
        e.updateCurrentMarketData(data)

        // Invoke OnMarketData callback
        if callbacks.OnMarketData != nil {
            if err := (*callbacks.OnMarketData)(data); err != nil {
                return err
            }
        }

        // Execute strategy
        if err := e.strategy.ProcessData(data); err != nil {
            if callbacks.OnStrategyError != nil {
                (*callbacks.OnStrategyError)(data, err)
            }
            // Continue processing - don't abort on strategy errors
        }
    }

    return nil
}
```

### Phase 6: CLI Command

**Files to create:**

```
cmd/
└── live/
    └── main.go    # CLI entry point for live trading
```

**Usage:**

```bash
# Live trading with Binance
go run cmd/live/main.go \
    --strategy-wasm ./examples/strategy/strategy.wasm \
    --strategy-config ./config/strategy/my-strategy.yaml \
    --market-data-provider binance \
    --trading-provider binance-paper \
    --trading-config ./config/trading/binance-testnet.yaml \
    --symbols BTCUSDT,ETHUSDT \
    --interval 1m

# Live trading with IBKR + Polygon data
go run cmd/live/main.go \
    --strategy-wasm ./examples/strategy/strategy.wasm \
    --strategy-config ./config/strategy/my-strategy.yaml \
    --market-data-provider polygon \
    --market-data-config ./config/marketdata/polygon.yaml \
    --trading-provider ibkr-paper \
    --trading-config ./config/trading/ibkr-paper.yaml \
    --symbols AAPL,GOOGL \
    --interval 1m
```

### Phase 7: Testing

**Test Categories:**

1. **Unit Tests**
   - Configuration parsing and validation
   - Provider initialization
   - Streaming data source caching
   - Lifecycle callback invocation

2. **Integration Tests**
   - End-to-end with mock providers
   - Strategy execution with simulated market data
   - Order flow testing

3. **Paper Trading Tests**
   - Connect to Binance testnet
   - Execute sample strategy
   - Verify order placement and position tracking

## Configuration Examples

### Full YAML Configuration

```yaml
# config/live-trading.yaml
engine:
  symbols:
    - BTCUSDT
    - ETHUSDT
  interval: "1m"
  market_data_cache_size: 1000
  decimal_precision: 8
  enable_logging: true
  log_output_path: "./logs"

market_data:
  provider_type: binance
  config:
    use_testnet: true

trading:
  provider_type: binance-paper
  config:
    api_key: ${BINANCE_TESTNET_API_KEY}
    secret_key: ${BINANCE_TESTNET_SECRET_KEY}

strategy:
  wasm_path: ./examples/strategy/strategy.wasm
  config_path: ./config/strategy/my-strategy.yaml
```

### Programmatic Configuration

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"

    "github.com/rxtech-lab/argo-trading/internal/live/engine"
    "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
    tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
)

func main() {
    // Create engine
    eng, err := engine.NewLiveTradingEngineV1()
    if err != nil {
        log.Fatal(err)
    }

    // Configure engine
    config := engine.LiveTradingEngineConfig{
        Symbols:             []string{"BTCUSDT", "ETHUSDT"},
        Interval:            "1m",
        MarketDataCacheSize: 1000,
        DecimalPrecision:    8,
        EnableLogging:       true,
    }

    if err := eng.Initialize(config); err != nil {
        log.Fatal(err)
    }

    // Set market data provider with configuration
    marketDataConfig := provider.BinanceMarketDataConfig{
        UseTestnet: true,
    }
    if err := eng.SetMarketDataProvider(provider.ProviderBinance, marketDataConfig); err != nil {
        log.Fatal(err)
    }

    // Set trading provider with configuration
    tradingConfig := tradingprovider.BinanceProviderConfig{
        ApiKey:    os.Getenv("BINANCE_TESTNET_API_KEY"),
        SecretKey: os.Getenv("BINANCE_TESTNET_SECRET_KEY"),
    }
    if err := eng.SetTradingProvider(tradingprovider.ProviderBinancePaper, tradingConfig); err != nil {
        log.Fatal(err)
    }

    // Load strategy
    if err := eng.LoadStrategyFromFile("./examples/strategy/strategy.wasm"); err != nil {
        log.Fatal(err)
    }

    // Set strategy config
    strategyConfig, _ := os.ReadFile("./config/strategy/my-strategy.yaml")
    if err := eng.SetStrategyConfig(string(strategyConfig)); err != nil {
        log.Fatal(err)
    }

    // Setup context with cancellation
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Handle shutdown signals
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        log.Println("Shutting down...")
        cancel()
    }()

    // Define callbacks
    onStart := func(symbols []string, interval string) error {
        log.Printf("Engine started: symbols=%v, interval=%s\n", symbols, interval)
        return nil
    }
    onMarketData := func(data types.MarketData) error {
        log.Printf("Market data: %s %.2f\n", data.Symbol, data.Close)
        return nil
    }
    onError := func(err error) {
        log.Printf("Error: %v\n", err)
    }

    callbacks := engine.LiveTradingCallbacks{
        OnEngineStart: &onStart,
        OnMarketData:  &onMarketData,
        OnError:       &onError,
    }

    // Run the engine
    if err := eng.Run(ctx, callbacks); err != nil {
        log.Printf("Engine stopped: %v\n", err)
    }
}
```

## Error Handling

### Connection Errors

```go
// Reconnection strategy for WebSocket disconnections
type ReconnectionConfig struct {
    MaxRetries     int           `json:"max_retries" yaml:"max_retries" jsonschema:"default=5"`
    InitialBackoff time.Duration `json:"initial_backoff" yaml:"initial_backoff" jsonschema:"default=1s"`
    MaxBackoff     time.Duration `json:"max_backoff" yaml:"max_backoff" jsonschema:"default=30s"`
    BackoffFactor  float64       `json:"backoff_factor" yaml:"backoff_factor" jsonschema:"default=2.0"`
}
```

### Error Types

```go
const (
    ErrCodeProviderConnectionFailed = "PROVIDER_CONNECTION_FAILED"
    ErrCodeStreamDisconnected       = "STREAM_DISCONNECTED"
    ErrCodeStrategyInitFailed       = "STRATEGY_INIT_FAILED"
    ErrCodeTradingProviderError     = "TRADING_PROVIDER_ERROR"
    ErrCodeInvalidConfiguration     = "INVALID_CONFIGURATION"
    ErrCodeContextCancelled         = "CONTEXT_CANCELLED"
)
```

## File Structure

```
internal/
└── live/
    └── engine/
        ├── engine.go                    # Interface definitions
        ├── engine_v1/
        │   ├── live_trading_v1.go       # Main engine implementation
        │   ├── config.go                # Configuration structs + JSON schema
        │   ├── callbacks.go             # Callback type definitions
        │   ├── streaming_datasource.go  # Real-time data source adapter
        │   └── sliding_window_cache.go  # Cache for indicator calculations
        └── engine_test.go               # Engine tests

cmd/
└── live/
    └── main.go                          # CLI entry point

config/
├── live-trading.yaml                    # Example full configuration
├── trading/
│   ├── binance-testnet.yaml            # Binance testnet config
│   └── ibkr-paper.yaml                 # IBKR paper config
└── marketdata/
    └── polygon.yaml                     # Polygon config
```

## Comparison with Backtest Engine

| Feature | Backtest Engine | Live Trading Engine |
|---------|-----------------|---------------------|
| Data Source | Parquet files (historical) | WebSocket stream (real-time) |
| Trading Provider | BacktestTrading (simulated) | Binance/IBKR (real/paper) |
| Strategy Loading | Same WASM runtime | Same WASM runtime |
| Strategy API | Same RuntimeContext | Same RuntimeContext |
| Data Iteration | `datasource.ReadAll()` | `provider.Stream()` |
| Caching | SlidingWindowDataSource | StreamingDataSource |
| Results | Files (stats, marks, logs) | Real-time callbacks |
| Execution | Batch (finite) | Continuous (until cancelled) |

## Security Considerations

1. **API Key Management**
   - Never log API keys or secrets
   - Support environment variable substitution in configs
   - Validate keys before starting the engine

2. **Paper Trading Safety**
   - Default to paper trading providers
   - Require explicit confirmation for live trading
   - Log all orders with provider type

3. **Rate Limiting**
   - Respect exchange rate limits
   - Implement order throttling if needed
   - Handle rate limit errors gracefully

## Future Enhancements

1. **Multi-strategy support** - Run multiple strategies on different symbols
2. **Position synchronization** - Sync positions with exchange on startup
3. **Risk management** - Built-in stop-loss and position limits
4. **Performance metrics** - Real-time P&L tracking and reporting
5. **Alert system** - Notifications for orders, errors, and events
6. **Web dashboard** - Real-time visualization of trading activity

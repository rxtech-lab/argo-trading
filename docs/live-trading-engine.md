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

type OnEngineStartCallback func(symbols []string, interval string, previousDataPath string) error
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
## Implementation Plan

### Phase 1: Core Engine Structure

**Files to create:**

```
internal/
└── trading/
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

Create `SetMarketDataProvider()` implementation that:
   - Accepts provider type and configuration
   - Creates the appropriate provider instance using the existing factory
   - Validates the provider supports streaming


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
        // Determine previousDataPath - if persistence is enabled, provide the parquet file path
        previousDataPath := ""
        if e.streamingWriter != nil {
            previousDataPath = e.streamingWriter.GetOutputPath()
        }
        
        if err := (*callbacks.OnEngineStart)(e.config.Symbols, e.config.Interval, previousDataPath); err != nil {
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
└── trading/
    └── main.go    # CLI entry point for live trading
```

**Usage:**

```bash
# Live trading with Binance
go run cmd/trading/main.go \
    --strategy-wasm ./examples/strategy/strategy.wasm \
    --strategy-config ./config/strategy/my-strategy.yaml \
    --market-data-provider binance \
    --trading-provider binance-paper \
    --trading-config ./config/trading/binance-testnet.yaml \
    --symbols BTCUSDT,ETHUSDT \
    --interval 1m

# Live trading with IBKR + Polygon data
go run cmd/trading/main.go \
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

The testing strategy for the Live Trading Engine uses mock providers to simulate real-time market data and trading without connecting to actual exchanges. This enables fast, reproducible, and deterministic E2E tests.

#### 7.1 Test Architecture Overview

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              E2E Test Setup                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────┐      ┌─────────────────────────────────────────┐  │
│  │  Test Suite         │      │  LiveTradingEngineV1                    │  │
│  │  (testify/suite)    │─────▶│                                         │  │
│  └─────────────────────┘      │  ┌─────────────────────────────────┐    │  │
│                               │  │  WASM Strategy                  │    │  │
│                               │  │  (Same as production)           │    │  │
│                               │  └─────────────────────────────────┘    │  │
│                               └──────────────┬──────────────────────────┘  │
│                                              │                              │
│                    ┌─────────────────────────┴─────────────────────────┐   │
│                    │                                                   │   │
│                    ▼                                                   ▼   │
│  ┌─────────────────────────────────────┐   ┌────────────────────────────┐ │
│  │  MockMarketDataProvider             │   │  MockTradingProvider       │ │
│  │  ┌───────────────────────────────┐  │   │  ┌──────────────────────┐  │ │
│  │  │ MockDataGenerator             │  │   │  │ In-memory positions  │  │ │
│  │  │ - PatternIncreasing           │  │   │  │ In-memory balance    │  │ │
│  │  │ - PatternDecreasing           │  │   │  │ Trade recording      │  │ │
│  │  │ - PatternVolatile             │  │   │  │ Instant execution    │  │ │
│  │  └───────────────────────────────┘  │   │  └──────────────────────┘  │ │
│  └─────────────────────────────────────┘   └────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

**Key Testing Principles:**

- **Fast execution**: Mock data yields as fast as possible (no artificial delays)
- **Reproducible**: Seed-based random generation ensures deterministic results
- **Isolated**: Each test gets fresh mock instances
- **Pattern-based**: Test strategy behavior under different market conditions

#### 7.2 Mock Market Data Provider

The `MockMarketDataProvider` implements the `Provider` interface using the existing `MockDataGenerator` from the backtest test helpers.

```go
package testhelper

import (
    "context"
    "iter"

    "github.com/rxtech-lab/argo-trading/internal/types"
)

// MockMarketDataProvider implements provider.Provider for testing
type MockMarketDataProvider struct {
    configs map[string]MockMarketDataConfig  // Config per symbol
}

// MockMarketDataConfig extends MockDataConfig for streaming with error injection
type MockMarketDataConfig struct {
    // Symbol for this configuration
    Symbol string

    // Pattern determines price movement (increasing, decreasing, volatile)
    Pattern SimulationPattern

    // InitialPrice is the starting price
    InitialPrice float64

    // NumDataPoints is the total number of data points to generate
    NumDataPoints int

    // TrendStrength controls how strong the trend is (0.0-1.0)
    TrendStrength float64

    // VolatilityPercent controls price volatility
    VolatilityPercent float64

    // MaxDrawdownPercent limits drawdown for volatile pattern
    MaxDrawdownPercent float64

    // Seed for reproducible random generation
    Seed int64

    // ErrorAfterN injects an error after N data points (0 = no error)
    ErrorAfterN int

    // ErrorToReturn is the error to inject
    ErrorToReturn error
}

// NewMockMarketDataProvider creates a new mock provider with the given configurations
func NewMockMarketDataProvider(configs ...MockMarketDataConfig) *MockMarketDataProvider {
    p := &MockMarketDataProvider{
        configs: make(map[string]MockMarketDataConfig),
    }
    for _, c := range configs {
        p.configs[c.Symbol] = c
    }
    return p
}

// Stream implements provider.Provider.Stream
// Yields generated market data as fast as possible for quick test execution
func (p *MockMarketDataProvider) Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error] {
    return func(yield func(types.MarketData, error) bool) {
        // Generate data for each symbol using MockDataGenerator
        for _, symbol := range symbols {
            config, ok := p.configs[symbol]
            if !ok {
                yield(types.MarketData{}, fmt.Errorf("no config for symbol: %s", symbol))
                return
            }

            generator := NewMockDataGenerator(MockDataConfig{
                Symbol:             config.Symbol,
                Pattern:            config.Pattern,
                InitialPrice:       config.InitialPrice,
                NumDataPoints:      config.NumDataPoints,
                TrendStrength:      config.TrendStrength,
                VolatilityPercent:  config.VolatilityPercent,
                MaxDrawdownPercent: config.MaxDrawdownPercent,
                Seed:               config.Seed,
            })

            data, err := generator.Generate()
            if err != nil {
                yield(types.MarketData{}, err)
                return
            }

            for i, d := range data {
                // Check for context cancellation
                select {
                case <-ctx.Done():
                    return
                default:
                }

                // Inject error if configured
                if config.ErrorAfterN > 0 && i >= config.ErrorAfterN {
                    yield(types.MarketData{}, config.ErrorToReturn)
                    return
                }

                if !yield(d, nil) {
                    return
                }
            }
        }
    }
}

// ConfigWriter implements provider.Provider (no-op for mock)
func (p *MockMarketDataProvider) ConfigWriter(writer writer.MarketDataWriter) {}

// Download implements provider.Provider (not used in live trading)
func (p *MockMarketDataProvider) Download(ctx context.Context, ticker string, startDate time.Time, endDate time.Time,
    multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (string, error) {
    return "", fmt.Errorf("download not supported in mock provider")
}
```

#### 7.3 Mock Trading Provider

The `MockTradingProvider` implements `TradingSystemProvider` with instant order execution and in-memory state tracking.

```go
package testhelper

import (
    "fmt"
    "sync"
    "time"

    "github.com/rxtech-lab/argo-trading/internal/types"
)

// MockTradingProvider implements TradingSystemProvider for testing
type MockTradingProvider struct {
    mu sync.RWMutex

    // State
    balance   float64
    positions map[string]*types.Position
    orders    []types.ExecuteOrder
    trades    []types.Trade
    openOrders []types.ExecuteOrder

    // Current market data (updated by engine)
    currentPrice map[string]float64

    // Behavior configuration
    FailAllOrders bool
    FailReason    string
}

// NewMockTradingProvider creates a new mock trading provider
func NewMockTradingProvider(initialBalance float64) *MockTradingProvider {
    return &MockTradingProvider{
        balance:      initialBalance,
        positions:    make(map[string]*types.Position),
        orders:       make([]types.ExecuteOrder, 0),
        trades:       make([]types.Trade, 0),
        openOrders:   make([]types.ExecuteOrder, 0),
        currentPrice: make(map[string]float64),
    }
}

// SetCurrentPrice updates the current price for a symbol (called by test or engine)
func (m *MockTradingProvider) SetCurrentPrice(symbol string, price float64) {
    m.mu.Lock()
    defer m.mu.Unlock()
    m.currentPrice[symbol] = price
}

// PlaceOrder executes an order instantly at the current price
func (m *MockTradingProvider) PlaceOrder(order types.ExecuteOrder) error {
    m.mu.Lock()
    defer m.mu.Unlock()

    // Check for configured failure
    if m.FailAllOrders {
        return fmt.Errorf("order failed: %s", m.FailReason)
    }

    // Record order
    m.orders = append(m.orders, order)

    // Get execution price
    price := order.Price
    if price == 0 {
        price = m.currentPrice[order.Symbol]
    }
    if price == 0 {
        return fmt.Errorf("no price available for %s", order.Symbol)
    }

    // Calculate cost
    cost := price * order.Quantity

    // Execute based on side
    if order.Side == types.SideBuy {
        if cost > m.balance {
            return fmt.Errorf("insufficient balance: need %.2f, have %.2f", cost, m.balance)
        }
        m.balance -= cost

        // Update position
        pos := m.getOrCreatePosition(order.Symbol)
        pos.TotalLongPositionQuantity += order.Quantity
        pos.TotalLongInPositionQuantity += order.Quantity
        pos.TotalLongInPositionAmount += cost
    } else {
        // Sell
        pos := m.getOrCreatePosition(order.Symbol)
        if order.Quantity > pos.TotalLongPositionQuantity {
            return fmt.Errorf("insufficient position: need %.2f, have %.2f",
                order.Quantity, pos.TotalLongPositionQuantity)
        }

        m.balance += cost
        pos.TotalLongPositionQuantity -= order.Quantity
        pos.TotalLongOutPositionQuantity += order.Quantity
        pos.TotalLongOutPositionAmount += cost
    }

    // Record trade
    trade := types.Trade{
        Order: types.Order{
            OrderID:      order.ID,
            Symbol:       order.Symbol,
            Side:         order.Side,
            Quantity:     order.Quantity,
            Price:        price,
            Timestamp:    time.Now(),
            IsCompleted:  true,
            Status:       types.OrderStatusFilled,
            StrategyName: order.StrategyName,
        },
        ExecutedAt:    time.Now(),
        ExecutedQty:   order.Quantity,
        ExecutedPrice: price,
    }
    m.trades = append(m.trades, trade)

    return nil
}

func (m *MockTradingProvider) getOrCreatePosition(symbol string) *types.Position {
    if pos, ok := m.positions[symbol]; ok {
        return pos
    }
    pos := &types.Position{Symbol: symbol}
    m.positions[symbol] = pos
    return pos
}

// GetPositions returns all positions
func (m *MockTradingProvider) GetPositions() ([]types.Position, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    result := make([]types.Position, 0, len(m.positions))
    for _, pos := range m.positions {
        result = append(result, *pos)
    }
    return result, nil
}

// GetPosition returns a specific position
func (m *MockTradingProvider) GetPosition(symbol string) (types.Position, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    if pos, ok := m.positions[symbol]; ok {
        return *pos, nil
    }
    return types.Position{Symbol: symbol}, nil
}

// GetAccountInfo returns account information
func (m *MockTradingProvider) GetAccountInfo() (types.AccountInfo, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return types.AccountInfo{
        Balance:     m.balance,
        Equity:      m.balance, // Simplified
        BuyingPower: m.balance,
    }, nil
}

// GetTrades returns all executed trades (for test assertions)
func (m *MockTradingProvider) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()

    return m.trades, nil
}

// GetAllTrades returns all trades without filter (convenience for tests)
func (m *MockTradingProvider) GetAllTrades() []types.Trade {
    m.mu.RLock()
    defer m.mu.RUnlock()

    result := make([]types.Trade, len(m.trades))
    copy(result, m.trades)
    return result
}

// Additional TradingSystemProvider methods...
func (m *MockTradingProvider) PlaceMultipleOrders(orders []types.ExecuteOrder) error {
    for _, o := range orders {
        if err := m.PlaceOrder(o); err != nil {
            return err
        }
    }
    return nil
}

func (m *MockTradingProvider) CancelOrder(orderID string) error { return nil }
func (m *MockTradingProvider) CancelAllOrders() error { return nil }
func (m *MockTradingProvider) GetOrderStatus(orderID string) (types.OrderStatus, error) {
    return types.OrderStatusFilled, nil
}
func (m *MockTradingProvider) GetOpenOrders() ([]types.ExecuteOrder, error) { return nil, nil }
func (m *MockTradingProvider) GetMaxBuyQuantity(symbol string, price float64) (float64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    return m.balance / price, nil
}
func (m *MockTradingProvider) GetMaxSellQuantity(symbol string) (float64, error) {
    m.mu.RLock()
    defer m.mu.RUnlock()
    if pos, ok := m.positions[symbol]; ok {
        return pos.TotalLongPositionQuantity, nil
    }
    return 0, nil
}
```

#### 7.4 E2E Test Scenarios

##### Test Scenario 1: Strategy Execution in Increasing Market

Tests that a strategy correctly identifies and profits from an upward trend.

```go
func (s *LiveTradingE2ETestSuite) TestStrategyInIncreasingMarket() {
    // Setup: Generate increasing price data
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            TrendStrength: 0.02,  // 2% increase per candle
            NumDataPoints: 100,
            Seed:          42,    // Reproducible
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)

    // Configure engine
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./trend_following_strategy.wasm")

    // Run
    err := s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
    s.Require().NoError(err)

    // Assertions
    trades := mockTrading.GetAllTrades()
    s.Greater(len(trades), 0, "Strategy should place trades in increasing market")

    // Verify profitability
    accountInfo, _ := mockTrading.GetAccountInfo()
    s.Greater(accountInfo.Balance, 10000.0, "Balance should increase in uptrend")
}
```

##### Test Scenario 2: Strategy Execution in Decreasing Market

Tests that a strategy limits losses or avoids trades in a downward trend.

```go
func (s *LiveTradingE2ETestSuite) TestStrategyInDecreasingMarket() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternDecreasing,
            InitialPrice:  50000.0,
            TrendStrength: 0.02,  // 2% decrease per candle
            NumDataPoints: 100,
            Seed:          42,
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./trend_following_strategy.wasm")

    err := s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
    s.Require().NoError(err)

    // A good strategy should either:
    // 1. Not trade (wait for uptrend)
    // 2. Trade but exit quickly to limit losses
    accountInfo, _ := mockTrading.GetAccountInfo()

    // Allow some loss but not catastrophic
    s.GreaterOrEqual(accountInfo.Balance, 8000.0, "Strategy should limit losses in downtrend")
}
```

##### Test Scenario 3: Strategy Execution in Volatile Market

Tests strategy behavior with high volatility and drawdown constraints.

```go
func (s *LiveTradingE2ETestSuite) TestStrategyInVolatileMarket() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:             "BTCUSDT",
            Pattern:            testhelper.PatternVolatile,
            InitialPrice:       50000.0,
            MaxDrawdownPercent: 10.0,  // Max 10% drawdown from peak
            VolatilityPercent:  3.0,   // 3% volatility per candle
            NumDataPoints:      200,
            Seed:               42,
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./volatility_strategy.wasm")

    err := s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
    s.Require().NoError(err)

    // Verify strategy handles volatility
    trades := mockTrading.GetAllTrades()
    s.NotEmpty(trades, "Strategy should trade in volatile market")

    // Check that stop-losses were respected (no single trade > 5% loss)
    for _, trade := range trades {
        if trade.PnL < 0 {
            lossPct := (trade.PnL / trade.ExecutedPrice) * 100
            s.GreaterOrEqual(lossPct, -5.0, "Individual trade loss should be limited")
        }
    }
}
```

##### Test Scenario 4: Multi-Symbol Streaming

Tests engine handling multiple symbols with different market conditions.

```go
func (s *LiveTradingE2ETestSuite) TestMultiSymbolStreaming() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            NumDataPoints: 50,
            Seed:          42,
        },
        testhelper.MockMarketDataConfig{
            Symbol:             "ETHUSDT",
            Pattern:            testhelper.PatternVolatile,
            InitialPrice:       3000.0,
            MaxDrawdownPercent: 15.0,
            NumDataPoints:      50,
            Seed:               43,
        },
    )

    symbolsSeen := make(map[string]int)
    var mu sync.Mutex

    callbacks := engine.LiveTradingCallbacks{
        OnMarketData: func(data types.MarketData) error {
            mu.Lock()
            symbolsSeen[data.Symbol]++
            mu.Unlock()
            return nil
        },
    }

    mockTrading := testhelper.NewMockTradingProvider(20000.0)
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./multi_symbol_strategy.wasm")

    err := s.engine.Run(context.Background(), callbacks)
    s.Require().NoError(err)

    // Verify both symbols were processed
    s.Equal(50, symbolsSeen["BTCUSDT"], "Should process all BTCUSDT data points")
    s.Equal(50, symbolsSeen["ETHUSDT"], "Should process all ETHUSDT data points")
}
```

##### Test Scenario 5: Error Handling

Tests engine behavior when stream errors occur.

```go
func (s *LiveTradingE2ETestSuite) TestStreamErrorHandling() {
    expectedErr := errors.New("simulated connection lost")

    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            NumDataPoints: 100,
            Seed:          42,
            ErrorAfterN:   50,  // Inject error after 50 data points
            ErrorToReturn: expectedErr,
        },
    )

    var errorReceived error
    var dataCountBeforeError int

    callbacks := engine.LiveTradingCallbacks{
        OnMarketData: func(data types.MarketData) error {
            dataCountBeforeError++
            return nil
        },
        OnError: func(err error) {
            errorReceived = err
        },
    }

    mockTrading := testhelper.NewMockTradingProvider(10000.0)
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./simple_strategy.wasm")

    err := s.engine.Run(context.Background(), callbacks)

    // Verify error handling
    s.Equal(50, dataCountBeforeError, "Should process data until error")
    s.NotNil(errorReceived, "OnError callback should be invoked")
    s.Contains(errorReceived.Error(), "connection lost")
}
```

##### Test Scenario 6: Graceful Shutdown

Tests that engine shuts down cleanly when context is cancelled.

```go
func (s *LiveTradingE2ETestSuite) TestGracefulShutdown() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            NumDataPoints: 1000,  // Many data points
            Seed:          42,
        },
    )

    var dataCount int
    var engineStopped bool

    ctx, cancel := context.WithCancel(context.Background())

    callbacks := engine.LiveTradingCallbacks{
        OnMarketData: func(data types.MarketData) error {
            dataCount++
            if dataCount >= 100 {
                cancel()  // Cancel after 100 data points
            }
            return nil
        },
        OnEngineStop: func(err error) {
            engineStopped = true
        },
    }

    mockTrading := testhelper.NewMockTradingProvider(10000.0)
    s.engine.SetMockMarketDataProvider(mockMarketData)
    s.engine.SetMockTradingProvider(mockTrading)
    s.engine.LoadStrategyFromFile("./simple_strategy.wasm")

    err := s.engine.Run(ctx, callbacks)

    // Verify graceful shutdown
    s.Equal(context.Canceled, err, "Should return context.Canceled")
    s.True(engineStopped, "OnEngineStop should be called")
    s.Less(dataCount, 1000, "Should stop before processing all data")
}
```

#### 7.5 Test File Structure

```
e2e/
└── trading/
    ├── testhelper/
    │   ├── mock_market_data_provider.go   # MockMarketDataProvider implementation
    │   ├── mock_trading_provider.go       # MockTradingProvider implementation
    │   └── test_strategies/               # Pre-compiled WASM strategies for testing
    │       ├── simple_strategy.wasm
    │       ├── trend_following_strategy.wasm
    │       ├── volatility_strategy.wasm
    │       └── multi_symbol_strategy.wasm
    └── engine/
        ├── suite_test.go                  # Test suite setup
        ├── basic_strategy_test.go         # Basic execution tests
        ├── market_patterns_test.go        # Pattern-specific tests (increasing/decreasing/volatile)
        ├── multi_symbol_test.go           # Multi-symbol streaming tests
        ├── error_handling_test.go         # Error scenario tests
        └── shutdown_test.go               # Graceful shutdown tests
```

#### 7.6 Example Full E2E Test Suite

```go
package engine_test

import (
    "context"
    "testing"

    "github.com/stretchr/testify/suite"
    "github.com/rxtech-lab/argo-trading/e2e/trading/testhelper"
    "github.com/rxtech-lab/argo-trading/internal/trading/engine"
    "github.com/rxtech-lab/argo-trading/internal/types"
)

type LiveTradingE2ETestSuite struct {
    suite.Suite
    engine engine.LiveTradingEngine
}

func TestLiveTradingE2E(t *testing.T) {
    suite.Run(t, new(LiveTradingE2ETestSuite))
}

func (s *LiveTradingE2ETestSuite) SetupTest() {
    var err error
    s.engine, err = engine.NewLiveTradingEngineV1()
    s.Require().NoError(err)

    // Initialize with default config
    err = s.engine.Initialize(engine.LiveTradingEngineConfig{
        Symbols:             []string{"BTCUSDT"},
        Interval:            "1m",
        MarketDataCacheSize: 100,
        EnableLogging:       false,
    })
    s.Require().NoError(err)
}

func (s *LiveTradingE2ETestSuite) TestBasicStrategyExecution() {
    // Setup mock providers
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            TrendStrength: 0.01,
            NumDataPoints: 100,
            Seed:          42,
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)

    // Configure engine with mocks
    err := s.engine.SetMockMarketDataProvider(mockMarketData)
    s.Require().NoError(err)

    err = s.engine.SetMockTradingProvider(mockTrading)
    s.Require().NoError(err)

    // Load strategy
    err = s.engine.LoadStrategyFromFile("./testhelper/test_strategies/simple_strategy.wasm")
    s.Require().NoError(err)

    // Track execution
    var dataPointsProcessed int
    var engineStarted, engineStopped bool

    onStart := func(symbols []string, interval string, previousDataPath string) error {
        engineStarted = true
        s.Equal([]string{"BTCUSDT"}, symbols)
        s.Equal("1m", interval)
        // previousDataPath will be empty string if persistence is not enabled
        return nil
    }

    onStop := func(err error) {
        engineStopped = true
    }

    onData := func(data types.MarketData) error {
        dataPointsProcessed++
        // Update mock trading provider with current price
        mockTrading.SetCurrentPrice(data.Symbol, data.Close)
        return nil
    }

    callbacks := engine.LiveTradingCallbacks{
        OnEngineStart: &onStart,
        OnEngineStop:  &onStop,
        OnMarketData:  &onData,
    }

    // Run engine
    err = s.engine.Run(context.Background(), callbacks)
    s.Require().NoError(err)

    // Assertions
    s.True(engineStarted, "Engine should have started")
    s.True(engineStopped, "Engine should have stopped")
    s.Equal(100, dataPointsProcessed, "Should process all data points")

    // Verify trading activity
    trades := mockTrading.GetAllTrades()
    s.NotEmpty(trades, "Strategy should have placed trades")

    // Verify final account state
    accountInfo, err := mockTrading.GetAccountInfo()
    s.Require().NoError(err)
    s.NotZero(accountInfo.Balance, "Account should have balance")
}

// Additional test methods for each scenario...
```

#### 7.7 Test Categories Summary

| Category | Description | Mock Providers Used |
|----------|-------------|---------------------|
| **Unit Tests** | Configuration parsing, provider initialization, callback invocation | None (pure unit tests) |
| **E2E Tests** | Full engine execution with mock data and trading | MockMarketDataProvider + MockTradingProvider |
| **Pattern Tests** | Strategy behavior under different market conditions | MockMarketDataProvider with different patterns |
| **Error Tests** | Error injection and handling verification | MockMarketDataProvider with ErrorAfterN |
| **Integration Tests** | Real provider connection tests (optional) | Real Binance testnet providers |

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

    "github.com/rxtech-lab/argo-trading/internal/trading/engine"
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
    onStart := func(symbols []string, interval string, previousDataPath string) error {
        log.Printf("Engine started: symbols=%v, interval=%s\n", symbols, interval)
        if previousDataPath != "" {
            log.Printf("Previous data available at: %s\n", previousDataPath)
        }
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
└── trading/
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
└── trading/
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

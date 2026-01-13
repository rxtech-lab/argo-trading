---
title: Live Trading Engine
description: Real-time strategy execution with streaming market data and live/paper trading
---

# Live Trading Engine

The Live Trading Engine enables real-time strategy execution using streaming market data and live/paper trading providers. It follows patterns established by the backtest engine, allowing the same WASM strategies to work in both backtest and live modes.

## Overview

The Live Trading Engine is designed to:

1. **Load and initialize WASM strategies** - Same as the backtest engine
2. **Connect to streaming market data providers** - Using the `Provider.Stream()` interface
3. **Execute trades via trading providers** - Using the `TradingSystemProvider` interface
4. **Persist session data** - Orders, trades, marks, and statistics saved to disk in real-time
5. **Prefetch historical data** - For accurate indicator calculations from the start

## Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              Live Trading Engine                             │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │                          Engine Configuration                          │ │
│  │  - Market Data Provider Type + Config                                  │ │
│  │  - Trading Provider Type + Config                                      │ │
│  │  - Strategy WASM Path + Config                                         │ │
│  │  - Symbols to trade                                                    │ │
│  │  - Data output path for persistence                                    │ │
│  │  - Prefetch settings                                                   │ │
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
│                              │            │  - ibkr-paper                    │
│                              │            │  - ibkr-live                     │
└──────────────────────────────┘            └──────────────────────────────────┘
```

## Core Interfaces

### LiveTradingEngine Interface

```go
package engine

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
    SetMarketDataProvider(provider provider.Provider) error

    // SetTradingProvider configures the trading provider.
    SetTradingProvider(provider tradingprovider.TradingSystemProvider) error

    // Run starts the live trading engine.
    // Blocks until context is cancelled or a fatal error occurs.
    Run(ctx context.Context, callbacks LiveTradingCallbacks) error

    // GetConfigSchema returns the JSON schema for engine configuration.
    GetConfigSchema() (string, error)
}
```

### LiveTradingEngineConfig

```go
type LiveTradingEngineConfig struct {
    // Symbols to trade/monitor
    Symbols []string `json:"symbols" yaml:"symbols" validate:"required,min=1"`

    // Interval for market data streaming (e.g., "1m", "5m", "1h")
    Interval string `json:"interval" yaml:"interval" validate:"required"`

    // MarketDataCacheSize is the number of historical data points to cache per symbol
    MarketDataCacheSize int `json:"market_data_cache_size" yaml:"market_data_cache_size"`

    // EnableLogging enables strategy log storage
    EnableLogging bool `json:"enable_logging" yaml:"enable_logging"`

    // DataOutputPath is the directory for persisting session data
    DataOutputPath string `json:"data_output_path" yaml:"data_output_path" validate:"required"`

    // Prefetch configuration for historical data
    Prefetch PrefetchConfig `json:"prefetch" yaml:"prefetch"`
}

type PrefetchConfig struct {
    // Enabled enables historical data prefetching
    Enabled bool `json:"enabled" yaml:"enabled"`

    // StartTimeType is either "date" or "days"
    StartTimeType string `json:"start_time_type" yaml:"start_time_type"`

    // StartTime is the absolute start time (used when StartTimeType is "date")
    StartTime time.Time `json:"start_time" yaml:"start_time"`

    // Days is the number of days to prefetch (used when StartTimeType is "days")
    Days int `json:"days" yaml:"days"`
}
```

## Lifecycle Callbacks

```go
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

    // OnStatsUpdate is called periodically with real-time statistics.
    OnStatsUpdate *OnStatsUpdateCallback

    // OnStatusUpdate is called when the engine status changes.
    OnStatusUpdate *OnStatusUpdateCallback
}

// EngineStatus represents the current status of the live trading engine.
type EngineStatus string

const (
    // EngineStatusPrefetching indicates the engine is downloading historical data.
    EngineStatusPrefetching EngineStatus = "prefetching"

    // EngineStatusRunning indicates the engine is processing live market data.
    EngineStatusRunning EngineStatus = "running"

    // EngineStatusStopping indicates the engine is shutting down (cleanup in progress).
    // Note: Not implemented yet - reserved for future use.
    EngineStatusStopping EngineStatus = "stopping"
)

type OnEngineStartCallback func(symbols []string, interval string, dataPath string) error
type OnEngineStopCallback func(err error)
type OnMarketDataCallback func(data types.MarketData) error
type OnOrderPlacedCallback func(order types.ExecuteOrder) error
type OnOrderFilledCallback func(order types.Order) error
type OnErrorCallback func(err error)
type OnStrategyErrorCallback func(data types.MarketData, err error)
type OnStatsUpdateCallback func(stats LiveTradeStats) error
type OnStatusUpdateCallback func(status EngineStatus) error
```

## Market Data Providers

Market data providers implement the `Provider` interface with real-time streaming support:

```go
type Provider interface {
    // Stream returns an iterator of real-time market data
    Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error]

    // Download fetches historical data
    Download(ctx context.Context, params DownloadParams) (string, error)
}
```

### Supported Providers

| Provider | Type | Description |
|----------|------|-------------|
| `binance` | Crypto | Binance WebSocket streaming for cryptocurrency |
| `polygon` | Stocks | Polygon.io WebSocket streaming for US equities |

### Provider Configuration

```yaml
market_data:
  provider_type: binance
  config:
    use_testnet: true
```

## Trading Providers

Trading providers implement the `TradingSystemProvider` interface:

```go
type TradingSystemProvider interface {
    PlaceOrder(order types.ExecuteOrder) error
    PlaceMultipleOrders(orders []types.ExecuteOrder) error
    GetPositions() ([]types.Position, error)
    GetPosition(symbol string) (types.Position, error)
    CancelOrder(orderID string) error
    CancelAllOrders() error
    GetOrderStatus(orderID string) (types.OrderStatus, error)
    GetAccountInfo() (types.AccountInfo, error)
    GetOpenOrders() ([]types.ExecuteOrder, error)
    GetTrades(filter types.TradeFilter) ([]types.Trade, error)
    GetMaxBuyQuantity(symbol string, price float64) (float64, error)
    GetMaxSellQuantity(symbol string) (float64, error)
}
```

### Provider Registry

| Provider | Type | Description |
|----------|------|-------------|
| `binance-paper` | Crypto | Binance testnet for paper trading |
| `binance-live` | Crypto | Binance mainnet for live trading |
| `ibkr-paper` | Stocks | Interactive Brokers paper trading |
| `ibkr-live` | Stocks | Interactive Brokers live trading |

### Trading Provider Configuration

```yaml
trading:
  provider_type: binance-paper
  config:
    api_key: ${BINANCE_TESTNET_API_KEY}
    secret_key: ${BINANCE_TESTNET_SECRET_KEY}
```

## Configuration Examples

### Full YAML Configuration

```yaml
engine:
  symbols:
    - BTCUSDT
    - ETHUSDT
  interval: "1m"
  market_data_cache_size: 1000
  enable_logging: true
  data_output_path: "./data/live-trading"
  prefetch:
    enabled: true
    start_time_type: days
    days: 30

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
    "os"
    "os/signal"
    "syscall"

    "github.com/rxtech-lab/argo-trading/internal/trading/engine"
    "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
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
        EnableLogging:       true,
        DataOutputPath:      "./data/live-trading",
        Prefetch: engine.PrefetchConfig{
            Enabled:       true,
            StartTimeType: "days",
            Days:          30,
        },
    }

    if err := eng.Initialize(config); err != nil {
        log.Fatal(err)
    }

    // Set providers and load strategy...
    // See full example in CLI usage below
}
```

## CLI Usage

### Basic Command

```bash
go run cmd/trading/main.go \
    --strategy-wasm ./strategy.wasm \
    --market-data-provider binance \
    --trading-provider binance-paper \
    --trading-config ./config/trading.json \
    --symbols BTCUSDT,ETHUSDT \
    --interval 1m \
    --data-output ./data/live-trading
```

### All Flags

| Flag | Required | Description |
|------|----------|-------------|
| `--strategy-wasm` | Yes | Path to strategy WASM file |
| `--strategy-config` | No | Path to strategy config |
| `--market-data-provider` | Yes | `binance` or `polygon` |
| `--polygon-api-key` | Polygon only | Polygon API key |
| `--trading-provider` | Yes | `binance-paper`, `binance-live`, etc. |
| `--trading-config` | Yes | Provider config file |
| `--symbols` | Yes | Comma-separated symbols |
| `--interval` | No | Default: `1m` |
| `--cache-size` | No | Default: `1000` |
| `--data-output` | Yes | Data persistence directory |
| `--prefetch-type` | No | `date` or `days` |
| `--prefetch-start` | No | Start time (if type is `date`) |
| `--prefetch-days` | No | Days to prefetch (if type is `days`) |

### Examples

**Paper Trading with Binance:**
```bash
go run cmd/trading/main.go \
    --strategy-wasm ./examples/strategy/strategy.wasm \
    --market-data-provider binance \
    --trading-provider binance-paper \
    --trading-config ./config/binance-testnet.json \
    --symbols BTCUSDT \
    --interval 5m \
    --data-output ./data/live \
    --prefetch-type days \
    --prefetch-days 30
```

**With Polygon Data and IBKR Trading:**
```bash
go run cmd/trading/main.go \
    --strategy-wasm ./examples/strategy/strategy.wasm \
    --market-data-provider polygon \
    --polygon-api-key $POLYGON_API_KEY \
    --trading-provider ibkr-paper \
    --trading-config ./config/ibkr-paper.json \
    --symbols AAPL,GOOGL \
    --interval 1m \
    --data-output ./data/live
```

## Signal Handling

The engine handles graceful shutdown on SIGINT/SIGTERM:

```go
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

sigChan := make(chan os.Signal, 1)
signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
go func() {
    <-sigChan
    log.Println("Shutting down...")
    cancel()
}()

eng.Run(ctx, callbacks)
```

On shutdown:
- Current data is flushed to disk
- `OnEngineStop` callback is invoked
- Exit code 0 on clean shutdown

## Related Documentation

- [Session Management and Persistence](session-and-persistence.md) - Daily statistics, multi-day sessions, data storage
- [Data Prefetch](data-prefetch.md) - Historical data prefetching for indicators
- [Testing](testing.md) - Mock providers and E2E testing

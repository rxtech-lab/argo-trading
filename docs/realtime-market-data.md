---
title: Realtime Market Data Architecture
description: Design plan for adding realtime market data listening capability with DuckDB persistence to the Argo Trading framework
---

# Realtime Market Data Architecture

This document outlines the architecture and implementation plan for adding realtime market data listening capability to the Argo Trading framework. The design supports both Binance and Polygon providers and includes realtime persistence to DuckDB.

## Overview

The realtime market data system extends the existing market data provider architecture to support:

- **WebSocket-based streaming** from Binance and Polygon
- **Event-driven data emission** for strategy consumption
- **Realtime DuckDB persistence** for data durability and historical analysis
- **Seamless integration** with the existing trading system

Key design principles:

- Follow the existing **provider pattern** used in market data and trading providers
- Support both **historical download** and **realtime streaming** through unified interfaces
- Enable **independent configuration** of data source and storage destination
- Maintain **backward compatibility** with existing backtesting workflows

## Architecture Diagram

```
┌─────────────────────────────────────────────────────────────────────────────────┐
│                              Trading System                                      │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                        Strategy (WASM)                                   │    │
│  │  - ProcessData() receives realtime market data                          │    │
│  │  - PlaceOrder() executes trades via trading provider                    │    │
│  └─────────────────────────────────────────────────────────────────────────┘    │
│                                     │                                            │
│                                     │ gRPC (go-plugin)                          │
│                                     ▼                                            │
│  ┌─────────────────────────────────────────────────────────────────────────┐    │
│  │                     Strategy Host Service                                │    │
│  │  ┌────────────────────────┐    ┌────────────────────────────────────┐   │    │
│  │  │   Data Operations      │    │       Trading Operations           │   │    │
│  │  │  - GetRange            │    │  - PlaceOrder                      │   │    │
│  │  │  - ReadLastData        │    │  - GetPositions                    │   │    │
│  │  │  - SubscribeRealtime   │◄───│  - CancelOrder                     │   │    │
│  │  │  - ExecuteSQL          │    │  - GetAccountInfo                  │   │    │
│  │  └────────────┬───────────┘    └────────────────┬───────────────────┘   │    │
│  └───────────────┼──────────────────────────────────┼──────────────────────┘    │
└──────────────────┼──────────────────────────────────┼──────────────────────────┘
                   │                                  │
                   ▼                                  ▼
┌─────────────────────────────────────┐  ┌─────────────────────────────────────────┐
│   Realtime Market Data Provider     │  │     Trading Provider Registry           │
│                                     │  │                                         │
│  ┌────────────────────────────────┐ │  │  ┌─────────────┐ ┌─────────────┐        │
│  │    RealtimeProvider Interface  │ │  │  │binance-paper│ │binance-live │        │
│  │  - Connect()                   │ │  │  └─────────────┘ └─────────────┘        │
│  │  - Subscribe(symbols)          │ │  │  ┌─────────────┐ ┌─────────────┐        │
│  │  - Unsubscribe(symbols)        │ │  │  │ ibkr-paper  │ │  ibkr-live  │        │
│  │  - OnData(callback)            │ │  │  └─────────────┘ └─────────────┘        │
│  │  - Disconnect()                │ │  └─────────────────────────────────────────┘
│  └────────────────────────────────┘ │
│                                     │
│  ┌──────────────┐ ┌──────────────┐  │
│  │   Binance    │ │   Polygon    │  │
│  │  WebSocket   │ │  WebSocket   │  │
│  └──────────────┘ └──────────────┘  │
└─────────────────────────────────────┘
                   │
                   │ Parallel Write
                   ▼
┌─────────────────────────────────────┐
│      Realtime Data Writer           │
│                                     │
│  ┌────────────────────────────────┐ │
│  │  RealtimeWriter Interface      │ │
│  │  - WriteRealtime(data)         │ │
│  │  - Flush()                     │ │
│  │  - GetStats()                  │ │
│  └────────────────────────────────┘ │
│                                     │
│  ┌──────────────────────────────┐   │
│  │    DuckDB Realtime Writer    │   │
│  │  - Buffered writes           │   │
│  │  - Periodic flush            │   │
│  │  - WAL for durability        │   │
│  └──────────────────────────────┘   │
└─────────────────────────────────────┘
```

## Core Interfaces

### RealtimeProvider Interface

```go
package provider

import (
    "context"

    "github.com/rxtech-lab/argo-trading/internal/types"
)

// RealtimeDataCallback is called when new market data arrives.
type RealtimeDataCallback func(data types.MarketData)

// RealtimeErrorCallback is called when an error occurs.
type RealtimeErrorCallback func(err error)

// ConnectionState represents the current connection state.
type ConnectionState string

const (
    ConnectionStateDisconnected ConnectionState = "disconnected"
    ConnectionStateConnecting   ConnectionState = "connecting"
    ConnectionStateConnected    ConnectionState = "connected"
    ConnectionStateReconnecting ConnectionState = "reconnecting"
)

// ConnectionStateCallback is called when connection state changes.
type ConnectionStateCallback func(state ConnectionState)

// RealtimeProvider defines the interface for realtime market data streaming.
type RealtimeProvider interface {
    // Connect establishes the WebSocket connection to the provider.
    // The context can be used to cancel the connection attempt.
    Connect(ctx context.Context) error

    // Subscribe subscribes to realtime data for the given symbols.
    // Returns an error if any subscription fails.
    Subscribe(ctx context.Context, symbols []string, interval string) error

    // Unsubscribe removes subscriptions for the given symbols.
    Unsubscribe(ctx context.Context, symbols []string) error

    // OnData registers a callback for incoming market data.
    // Multiple callbacks can be registered.
    OnData(callback RealtimeDataCallback)

    // OnError registers a callback for error events.
    OnError(callback RealtimeErrorCallback)

    // OnConnectionStateChange registers a callback for connection state changes.
    OnConnectionStateChange(callback ConnectionStateCallback)

    // Disconnect closes the WebSocket connection gracefully.
    Disconnect(ctx context.Context) error

    // GetConnectionState returns the current connection state.
    GetConnectionState() ConnectionState

    // GetSubscribedSymbols returns the list of currently subscribed symbols.
    GetSubscribedSymbols() []string
}
```

### RealtimeWriter Interface

```go
package writer

import (
    "github.com/rxtech-lab/argo-trading/internal/types"
)

// RealtimeWriterStats contains statistics about the realtime writer.
type RealtimeWriterStats struct {
    TotalWrites      int64   // Total number of writes
    BufferedWrites   int     // Current number of buffered writes
    FlushCount       int64   // Number of flush operations
    LastFlushTime    int64   // Unix timestamp of last flush
    AverageFlushTime float64 // Average flush time in milliseconds
    ErrorCount       int64   // Number of write errors
}

// RealtimeWriter defines the interface for realtime market data persistence.
type RealtimeWriter interface {
    // Initialize sets up the writer (creates tables, opens connections).
    Initialize() error

    // WriteRealtime writes a single market data point to the buffer.
    // Data is not immediately persisted - call Flush() or wait for auto-flush.
    WriteRealtime(data types.MarketData) error

    // WriteBatch writes multiple market data points at once.
    WriteBatch(data []types.MarketData) error

    // Flush forces all buffered data to be written to storage.
    Flush() error

    // GetStats returns current writer statistics.
    GetStats() RealtimeWriterStats

    // SetFlushInterval sets the automatic flush interval in milliseconds.
    // Set to 0 to disable automatic flushing.
    SetFlushInterval(intervalMs int)

    // SetBufferSize sets the maximum buffer size before auto-flush.
    SetBufferSize(size int)

    // Close flushes remaining data and closes the writer.
    Close() error
}
```

## Provider Implementations

### Binance Realtime Provider

The Binance provider uses the official Binance WebSocket API for realtime kline/candlestick data.

```go
package provider

// BinanceRealtimeConfig contains configuration for Binance realtime streaming.
type BinanceRealtimeConfig struct {
    // UseTestnet connects to Binance testnet WebSocket if true.
    UseTestnet bool `json:"useTestnet" jsonschema:"description=Use Binance testnet WebSocket"`

    // BufferSize is the size of the internal message buffer.
    BufferSize int `json:"bufferSize" jsonschema:"description=Internal message buffer size,default=100"`

    // ReconnectAttempts is the maximum number of reconnection attempts.
    ReconnectAttempts int `json:"reconnectAttempts" jsonschema:"description=Max reconnection attempts,default=5"`

    // ReconnectDelayMs is the delay between reconnection attempts.
    ReconnectDelayMs int `json:"reconnectDelayMs" jsonschema:"description=Delay between reconnect attempts in ms,default=5000"`
}

// BinanceRealtimeProvider implements RealtimeProvider for Binance.
type BinanceRealtimeProvider struct {
    config     BinanceRealtimeConfig
    conn       *websocket.Conn
    state      ConnectionState
    subscribed map[string]bool
    callbacks  struct {
        data       []RealtimeDataCallback
        error      []RealtimeErrorCallback
        connection []ConnectionStateCallback
    }
    mu sync.RWMutex
}
```

**Binance WebSocket Endpoints:**

| Environment | Endpoint |
|-------------|----------|
| Production | `wss://stream.binance.com:9443/ws` |
| Testnet | `wss://testnet.binance.vision/ws` |

**Subscription Format:**

```json
{
    "method": "SUBSCRIBE",
    "params": ["btcusdt@kline_1m", "ethusdt@kline_1m"],
    "id": 1
}
```

**Message Format (Kline):**

```json
{
    "e": "kline",
    "E": 1638747660000,
    "s": "BTCUSDT",
    "k": {
        "t": 1638747600000,
        "T": 1638747659999,
        "s": "BTCUSDT",
        "i": "1m",
        "o": "48000.00",
        "c": "48100.00",
        "h": "48150.00",
        "l": "47950.00",
        "v": "100.5",
        "x": false
    }
}
```

### Polygon Realtime Provider

The Polygon provider uses the Polygon.io WebSocket API for realtime aggregates.

```go
package provider

// PolygonRealtimeConfig contains configuration for Polygon realtime streaming.
type PolygonRealtimeConfig struct {
    // ApiKey is required for authentication with Polygon WebSocket.
    ApiKey string `json:"apiKey" jsonschema:"description=Polygon API key" validate:"required"`

    // Feed specifies which data feed to connect to.
    // Options: "stocks", "options", "forex", "crypto"
    Feed string `json:"feed" jsonschema:"description=Data feed type,enum=stocks|options|forex|crypto,default=stocks"`

    // BufferSize is the size of the internal message buffer.
    BufferSize int `json:"bufferSize" jsonschema:"description=Internal message buffer size,default=100"`

    // ReconnectAttempts is the maximum number of reconnection attempts.
    ReconnectAttempts int `json:"reconnectAttempts" jsonschema:"description=Max reconnection attempts,default=5"`

    // ReconnectDelayMs is the delay between reconnection attempts.
    ReconnectDelayMs int `json:"reconnectDelayMs" jsonschema:"description=Delay between reconnect attempts in ms,default=5000"`
}

// PolygonRealtimeProvider implements RealtimeProvider for Polygon.io.
type PolygonRealtimeProvider struct {
    config     PolygonRealtimeConfig
    conn       *websocket.Conn
    state      ConnectionState
    subscribed map[string]bool
    callbacks  struct {
        data       []RealtimeDataCallback
        error      []RealtimeErrorCallback
        connection []ConnectionStateCallback
    }
    mu sync.RWMutex
}
```

**Polygon WebSocket Endpoints:**

| Feed | Endpoint |
|------|----------|
| Stocks | `wss://socket.polygon.io/stocks` |
| Options | `wss://socket.polygon.io/options` |
| Forex | `wss://socket.polygon.io/forex` |
| Crypto | `wss://socket.polygon.io/crypto` |

**Authentication:**

```json
{"action": "auth", "params": "YOUR_API_KEY"}
```

**Subscription Format:**

```json
{"action": "subscribe", "params": "AM.AAPL,AM.GOOGL"}
```

**Message Format (Aggregate per Minute):**

```json
{
    "ev": "AM",
    "sym": "AAPL",
    "v": 10000,
    "av": 100000,
    "op": 150.00,
    "vw": 150.50,
    "o": 150.10,
    "c": 150.80,
    "h": 151.00,
    "l": 149.90,
    "s": 1638747600000,
    "e": 1638747660000
}
```

## DuckDB Realtime Writer

The DuckDB realtime writer extends the existing DuckDB writer to support streaming data with buffered writes.

```go
package writer

// DuckDBRealtimeWriterConfig contains configuration for realtime DuckDB writing.
type DuckDBRealtimeWriterConfig struct {
    // OutputPath is the path to the DuckDB database file.
    OutputPath string `json:"outputPath" validate:"required"`

    // TableName is the name of the table to write to.
    TableName string `json:"tableName" jsonschema:"default=realtime_market_data"`

    // FlushIntervalMs is the automatic flush interval in milliseconds.
    FlushIntervalMs int `json:"flushIntervalMs" jsonschema:"default=1000"`

    // BufferSize is the maximum number of records to buffer before auto-flush.
    BufferSize int `json:"bufferSize" jsonschema:"default=100"`

    // EnableWAL enables Write-Ahead Logging for durability.
    EnableWAL bool `json:"enableWal" jsonschema:"default=true"`
}

// DuckDBRealtimeWriter implements RealtimeWriter for DuckDB.
type DuckDBRealtimeWriter struct {
    config     DuckDBRealtimeWriterConfig
    db         *sql.DB
    buffer     []types.MarketData
    stats      RealtimeWriterStats
    flushTimer *time.Timer
    mu         sync.Mutex
}
```

### Table Schema

```sql
CREATE TABLE IF NOT EXISTS realtime_market_data (
    id VARCHAR PRIMARY KEY,
    symbol VARCHAR NOT NULL,
    timestamp TIMESTAMP NOT NULL,
    open DOUBLE NOT NULL,
    high DOUBLE NOT NULL,
    low DOUBLE NOT NULL,
    close DOUBLE NOT NULL,
    volume DOUBLE NOT NULL,
    vwap DOUBLE,
    trade_count INTEGER,
    received_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    
    -- Indexes for efficient querying
    INDEX idx_symbol (symbol),
    INDEX idx_timestamp (timestamp),
    INDEX idx_symbol_timestamp (symbol, timestamp)
);
```

### Write Buffer Strategy

```
┌────────────────────────────────────────────────────────────────────┐
│                     DuckDB Realtime Writer                         │
│                                                                    │
│  ┌──────────────┐     ┌──────────────┐     ┌──────────────────┐   │
│  │  Incoming    │────▶│    Buffer    │────▶│   DuckDB Table   │   │
│  │  MarketData  │     │  (in-memory) │     │   (persisted)    │   │
│  └──────────────┘     └──────────────┘     └──────────────────┘   │
│                              │                                     │
│                              │ Flush Triggers:                     │
│                              │ 1. Buffer full (100 records)        │
│                              │ 2. Timer elapsed (1 second)         │
│                              │ 3. Manual Flush() call              │
│                              │ 4. Close() called                   │
│                              ▼                                     │
│                       ┌──────────────┐                             │
│                       │  Batch INSERT│                             │
│                       │  with WAL    │                             │
│                       └──────────────┘                             │
└────────────────────────────────────────────────────────────────────┘
```

## Provider Registry

Extend the existing provider registry to include realtime providers:

```go
package marketdata

// RealtimeProviderType defines the type of realtime market data provider.
type RealtimeProviderType string

const (
    RealtimeProviderBinance RealtimeProviderType = "binance-realtime"
    RealtimeProviderPolygon RealtimeProviderType = "polygon-realtime"
)

// RealtimeProviderInfo contains metadata about a realtime provider.
type RealtimeProviderInfo struct {
    Name           string   `json:"name"`
    DisplayName    string   `json:"displayName"`
    Description    string   `json:"description"`
    RequiresAuth   bool     `json:"requiresAuth"`
    SupportedFeeds []string `json:"supportedFeeds"`
}

// realtimeProviderRegistry holds metadata about all supported realtime providers.
var realtimeProviderRegistry = map[RealtimeProviderType]RealtimeProviderInfo{
    RealtimeProviderBinance: {
        Name:           string(RealtimeProviderBinance),
        DisplayName:    "Binance Realtime",
        Description:    "Realtime cryptocurrency market data via Binance WebSocket",
        RequiresAuth:   false,
        SupportedFeeds: []string{"spot", "futures"},
    },
    RealtimeProviderPolygon: {
        Name:           string(RealtimeProviderPolygon),
        DisplayName:    "Polygon.io Realtime",
        Description:    "Realtime US stock market data via Polygon WebSocket",
        RequiresAuth:   true,
        SupportedFeeds: []string{"stocks", "options", "forex", "crypto"},
    },
}

// GetSupportedRealtimeProviders returns a list of all supported realtime providers.
func GetSupportedRealtimeProviders() []string {
    providers := make([]string, 0, len(realtimeProviderRegistry))
    for providerType := range realtimeProviderRegistry {
        providers = append(providers, string(providerType))
    }
    return providers
}

// NewRealtimeProvider creates a new realtime provider instance.
func NewRealtimeProvider(providerType RealtimeProviderType, config any) (provider.RealtimeProvider, error) {
    switch providerType {
    case RealtimeProviderBinance:
        cfg, ok := config.(*provider.BinanceRealtimeConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for binance realtime provider")
        }
        return provider.NewBinanceRealtimeProvider(*cfg)

    case RealtimeProviderPolygon:
        cfg, ok := config.(*provider.PolygonRealtimeConfig)
        if !ok {
            return nil, fmt.Errorf("invalid config type for polygon realtime provider")
        }
        return provider.NewPolygonRealtimeProvider(*cfg)

    default:
        return nil, fmt.Errorf("unsupported realtime provider: %s", providerType)
    }
}
```

## Realtime Trading Session

A new `RealtimeTradingSession` orchestrates the realtime data flow:

```go
package trading

// RealtimeTradingSessionConfig defines the configuration for a realtime trading session.
type RealtimeTradingSessionConfig struct {
    // Realtime data provider configuration
    RealtimeProvider       marketdata.RealtimeProviderType `json:"realtimeProvider" validate:"required"`
    RealtimeProviderConfig interface{}                     `json:"realtimeProviderConfig" validate:"required"`

    // Trading provider configuration
    TradingProvider       tradingprovider.ProviderType `json:"tradingProvider" validate:"required"`
    TradingProviderConfig interface{}                  `json:"tradingProviderConfig" validate:"required"`

    // Storage configuration
    StorageConfig writer.DuckDBRealtimeWriterConfig `json:"storageConfig" validate:"required"`

    // Session settings
    Symbols  []string `json:"symbols" validate:"required,min=1"`
    Interval string   `json:"interval" jsonschema:"default=1m"` // e.g., "1m", "5m", "1h"
}

// RealtimeTradingSession manages realtime data streaming and strategy execution.
type RealtimeTradingSession struct {
    config          RealtimeTradingSessionConfig
    realtimeProvider provider.RealtimeProvider
    tradingProvider  tradingprovider.TradingSystemProvider
    writer          writer.RealtimeWriter
    strategy        strategy.TradingStrategy
    isRunning       bool
    mu              sync.RWMutex
}

// Start begins the realtime trading session.
func (s *RealtimeTradingSession) Start(ctx context.Context) error {
    // 1. Connect to realtime provider
    // 2. Subscribe to symbols
    // 3. Register data callback
    // 4. Process incoming data through strategy
    // 5. Persist data to DuckDB
}

// Stop gracefully stops the trading session.
func (s *RealtimeTradingSession) Stop(ctx context.Context) error {
    // 1. Unsubscribe from all symbols
    // 2. Disconnect from provider
    // 3. Flush and close writer
}
```

## Data Flow

```
1. WebSocket Connection
   ┌──────────────┐
   │   Binance/   │
   │   Polygon    │
   │   WebSocket  │
   └──────┬───────┘
          │ Raw JSON Messages
          ▼
2. Message Parsing
   ┌──────────────┐
   │   Provider   │
   │   Parser     │
   └──────┬───────┘
          │ types.MarketData
          ▼
3. Data Distribution (Fan-out)
   ┌──────────────────────────────────────┐
   │                                      │
   ▼                                      ▼
┌──────────────┐                  ┌──────────────┐
│   Strategy   │                  │   DuckDB     │
│  Processor   │                  │   Writer     │
└──────┬───────┘                  └──────┬───────┘
       │                                 │
       │ Trading Signals                 │ Buffered Write
       ▼                                 ▼
┌──────────────┐                  ┌──────────────┐
│   Trading    │                  │   Parquet/   │
│   Provider   │                  │   DuckDB     │
└──────────────┘                  └──────────────┘
```

## Configuration Examples

### Binance Crypto Trading with Realtime Data

```yaml
session:
  realtime_provider: binance-realtime
  realtime_provider_config:
    use_testnet: true
    buffer_size: 100
    reconnect_attempts: 5
    reconnect_delay_ms: 5000

  trading_provider: binance-paper
  trading_provider_config:
    api_key: ${BINANCE_TESTNET_API_KEY}
    secret_key: ${BINANCE_TESTNET_SECRET_KEY}

  storage_config:
    output_path: "./data/realtime.duckdb"
    table_name: "realtime_market_data"
    flush_interval_ms: 1000
    buffer_size: 100
    enable_wal: true

  symbols:
    - BTCUSDT
    - ETHUSDT
  interval: "1m"
```

### Polygon Stock Trading with Realtime Data

```yaml
session:
  realtime_provider: polygon-realtime
  realtime_provider_config:
    api_key: ${POLYGON_API_KEY}
    feed: "stocks"
    buffer_size: 100
    reconnect_attempts: 5
    reconnect_delay_ms: 5000

  trading_provider: ibkr-paper
  trading_provider_config:
    host: "127.0.0.1"
    port: 7497
    client_id: 1
    account_id: "DU123456"

  storage_config:
    output_path: "./data/realtime_stocks.duckdb"
    table_name: "realtime_market_data"
    flush_interval_ms: 1000
    buffer_size: 100
    enable_wal: true

  symbols:
    - AAPL
    - GOOGL
    - MSFT
  interval: "1m"
```

## Error Handling

### Connection Error Recovery

```go
// ConnectionErrorHandler handles WebSocket connection errors.
type ConnectionErrorHandler struct {
    maxAttempts    int
    delayMs        int
    currentAttempt int
}

// HandleError implements exponential backoff for reconnection.
func (h *ConnectionErrorHandler) HandleError(err error) error {
    if h.currentAttempt >= h.maxAttempts {
        return fmt.Errorf("max reconnection attempts reached: %w", err)
    }

    delay := h.delayMs * (1 << h.currentAttempt) // Exponential backoff
    time.Sleep(time.Duration(delay) * time.Millisecond)
    h.currentAttempt++

    return nil // Signal to retry
}
```

### Error Codes

```go
const (
    // Connection errors
    ErrCodeConnectionFailed    = "CONNECTION_FAILED"
    ErrCodeConnectionLost      = "CONNECTION_LOST"
    ErrCodeAuthenticationFailed = "AUTHENTICATION_FAILED"

    // Subscription errors
    ErrCodeSubscriptionFailed  = "SUBSCRIPTION_FAILED"
    ErrCodeInvalidSymbol       = "INVALID_SYMBOL"
    ErrCodeRateLimited         = "RATE_LIMITED"

    // Data errors
    ErrCodeInvalidMessage      = "INVALID_MESSAGE"
    ErrCodeWriteFailed         = "WRITE_FAILED"
    ErrCodeBufferOverflow      = "BUFFER_OVERFLOW"
)
```

## File Structure

```
pkg/
└── marketdata/
    ├── provider/
    │   ├── provider.go              # Existing Provider interface
    │   ├── realtime_provider.go     # NEW: RealtimeProvider interface
    │   ├── binance.go               # Existing Binance download client
    │   ├── binance_realtime.go      # NEW: Binance WebSocket provider
    │   ├── polygon.go               # Existing Polygon download client
    │   └── polygon_realtime.go      # NEW: Polygon WebSocket provider
    ├── writer/
    │   ├── writer.go                # Existing MarketDataWriter interface
    │   ├── realtime_writer.go       # NEW: RealtimeWriter interface
    │   ├── duckdb.go                # Existing DuckDB writer
    │   └── duckdb_realtime.go       # NEW: DuckDB realtime writer
    ├── client.go                    # Existing download client
    ├── realtime_client.go           # NEW: Realtime streaming client
    ├── provider_registry.go         # Existing provider registry
    └── realtime_provider_registry.go # NEW: Realtime provider registry

internal/
└── trading/
    ├── trading_system.go            # Existing TradingSystem
    ├── realtime_session.go          # NEW: RealtimeTradingSession
    └── provider/
        └── ...                      # Existing trading providers

cmd/
└── realtime/                        # NEW: CLI for realtime trading
    └── main.go
```

## Implementation Plan

### Phase 1: Core Interfaces and Infrastructure

1. Define `RealtimeProvider` interface in `pkg/marketdata/provider/realtime_provider.go`
2. Define `RealtimeWriter` interface in `pkg/marketdata/writer/realtime_writer.go`
3. Create realtime provider registry in `pkg/marketdata/realtime_provider_registry.go`

### Phase 2: Binance Realtime Provider

1. Implement `BinanceRealtimeProvider` with WebSocket connection
2. Handle Binance-specific message parsing
3. Implement reconnection logic with exponential backoff
4. Add unit tests with mock WebSocket

### Phase 3: Polygon Realtime Provider

1. Implement `PolygonRealtimeProvider` with WebSocket connection
2. Handle Polygon authentication flow
3. Implement reconnection logic
4. Add unit tests with mock WebSocket

### Phase 4: DuckDB Realtime Writer

1. Implement `DuckDBRealtimeWriter` with buffered writes
2. Add automatic flush timer
3. Implement WAL support for durability
4. Add unit tests with real DuckDB

### Phase 5: Realtime Trading Session

1. Create `RealtimeTradingSession` orchestrator
2. Integrate with existing strategy WASM runtime
3. Connect realtime data to `ProcessData` calls
4. Add CLI entry point in `cmd/realtime/main.go`

### Phase 6: Testing and Documentation

1. Add integration tests with testnet connections
2. Add end-to-end tests with mock providers
3. Update `CLAUDE.md` with realtime commands
4. Add user documentation

## Testing Strategy

### Unit Tests

- Mock WebSocket connections for provider tests
- Test message parsing for all message types
- Test buffer and flush logic for writer
- Test reconnection and error handling

### Integration Tests

- Connect to Binance testnet WebSocket
- Connect to Polygon with test API key
- Verify data persistence to DuckDB
- Test strategy execution with realtime data

### End-to-End Tests

- Run paper trading session with realtime data
- Verify order execution through trading provider
- Validate data consistency between stream and storage

## Performance Considerations

### Memory Management

- Use fixed-size buffers to prevent memory growth
- Implement buffer overflow protection
- Consider using object pools for MarketData structs

### Latency Optimization

- Minimize allocations in hot paths
- Use goroutine-per-symbol for parallel processing
- Consider using lock-free data structures for callbacks

### Storage Optimization

- Batch inserts to reduce DuckDB overhead
- Use prepared statements for repeated inserts
- Consider partitioning by date for large datasets

## Security Considerations

- Store API keys in environment variables, not in config files
- Use TLS for all WebSocket connections
- Validate all incoming data before processing
- Implement rate limiting for outgoing requests

## Related Documents

- [Trading System Provider Architecture](trading-system.md)
- [Implementing Trading Strategies](implementing-strategies.md)
- [Market Data Provider Registry](../pkg/marketdata/provider_registry.go)

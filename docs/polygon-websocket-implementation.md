---
title: Polygon WebSocket API Implementation Plan
description: Design and implementation plan for adding Polygon.io WebSocket streaming support
---

# Polygon WebSocket API Implementation Plan

This document outlines the implementation plan for adding Polygon.io (Massive.com) WebSocket streaming support to the Argo Trading framework.

## Overview

The Polygon.io WebSocket API provides real-time streaming data for US stocks, including trades, quotes, and aggregated bar data. This implementation will extend the existing `PolygonClient` to support the `Stream()` method, enabling live trading with US stocks.

### Current State

- The `Provider` interface already includes a `Stream()` method
- `BinanceClient` has a full WebSocket streaming implementation (reference: `pkg/marketdata/provider/binance.go`)
- `PolygonClient.Stream()` currently returns an error: "streaming is not yet implemented for Polygon provider"
- The `polygon-io/client-go` package (v1.16.10) already includes WebSocket support

## Polygon WebSocket API Overview

### Connection Details

| Feed Type | WebSocket URL | Description |
|-----------|---------------|-------------|
| RealTime | `wss://socket.polygon.io/stocks` | Full real-time data (paid tier) |
| Delayed | `wss://delayed.polygon.io/stocks` | 15-minute delayed data (free tier) |
| Nasdaq | `wss://nasdaqfeed.polygon.io/stocks` | Nasdaq-specific feed |
| LaunchpadFeed | `wss://launchpad.polygon.io/stocks` | Launchpad tier feed |

### Data Channels (Topics)

| Topic | Event Type | Description |
|-------|------------|-------------|
| `StocksSecAggs` | `A.{symbol}` | Per-second aggregates (OHLCV bars) |
| `StocksMinAggs` | `AM.{symbol}` | Per-minute aggregates (OHLCV bars) |
| `StocksTrades` | `T.{symbol}` | Real-time trades |
| `StocksQuotes` | `Q.{symbol}` | Real-time NBBO quotes |
| `StocksImbalances` | `I.{symbol}` | Imbalance events |
| `StocksLULD` | `LULD.{symbol}` | Limit up/limit down events |

### Message Types (from `polygon-io/client-go/websocket/models`)

```go
// EquityAgg is an aggregate for stock tickers (used for A.* and AM.* channels)
type EquityAgg struct {
    EventType         string  `json:"ev,omitempty"`
    Symbol            string  `json:"sym,omitempty"`
    Volume            float64 `json:"v,omitempty"`
    AccumulatedVolume float64 `json:"av,omitempty"`
    OfficialOpenPrice float64 `json:"op,omitempty"`
    VWAP              float64 `json:"vw,omitempty"`
    Open              float64 `json:"o,omitempty"`
    Close             float64 `json:"c,omitempty"`
    High              float64 `json:"h,omitempty"`
    Low               float64 `json:"l,omitempty"`
    AggregateVWAP     float64 `json:"a,omitempty"`
    AverageSize       float64 `json:"z,omitempty"`
    StartTimestamp    int64   `json:"s,omitempty"`  // Unix milliseconds
    EndTimestamp      int64   `json:"e,omitempty"`  // Unix milliseconds
    OTC               bool    `json:"otc,omitempty"`
}

// EquityTrade is trade data for stock tickers (used for T.* channel)
type EquityTrade struct {
    EventType    string  `json:"ev,omitempty"`
    Symbol       string  `json:"sym,omitempty"`
    Exchange     int32   `json:"x,omitempty"`
    ID           string  `json:"i,omitempty"`
    Tape         int32   `json:"z,omitempty"`
    Price        float64 `json:"p,omitempty"`
    Size         int64   `json:"s,omitempty"`
    Conditions   []int32 `json:"c,omitempty"`
    Timestamp    int64   `json:"t,omitempty"`
}
```

## Implementation Design

### Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                            PolygonClient                                     │
│  ┌───────────────────────────────────────────────────────────────────────┐  │
│  │ Stream(ctx, symbols, interval) iter.Seq2[MarketData, error]           │  │
│  └───────────────────────────────────────────────────────────────────────┘  │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                                       ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│                      PolygonWebSocketService (interface)                     │
│  ┌────────────────────────────────────────────────────────────────────────┐ │
│  │ Connect() error                                                        │ │
│  │ Subscribe(topic Topic, tickers ...string) error                        │ │
│  │ Output() <-chan any                                                    │ │
│  │ Error() <-chan error                                                   │ │
│  │ Close()                                                                │ │
│  └────────────────────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────────────────────┘
                                       │
                    ┌──────────────────┴──────────────────┐
                    │                                     │
                    ▼                                     ▼
┌───────────────────────────────────┐   ┌───────────────────────────────────┐
│  polygonWebSocketServiceWrapper   │   │  mockPolygonWebSocketService      │
│  (Production)                     │   │  (Testing)                        │
│                                   │   │                                   │
│  Uses real polygon-io/client-go   │   │  Emits mock events for testing    │
│  websocket.Client                 │   │                                   │
└───────────────────────────────────┘   └───────────────────────────────────┘
                    │
                    ▼
┌───────────────────────────────────┐
│  polygon-io/client-go/websocket   │
│  - websocket.Client               │
│  - websocket.Config               │
│  - websocket.Topic                │
└───────────────────────────────────┘
```

### Interface Design

```go
// PolygonWebSocketService defines the interface for Polygon WebSocket operations.
// This abstraction enables testing with mock implementations.
type PolygonWebSocketService interface {
    // Connect establishes the WebSocket connection to Polygon.
    Connect() error
    
    // Subscribe subscribes to a topic for the given tickers.
    Subscribe(topic websocket.Topic, tickers ...string) error
    
    // Unsubscribe unsubscribes from a topic for the given tickers.
    Unsubscribe(topic websocket.Topic, tickers ...string) error
    
    // Output returns a channel that receives incoming messages.
    // Messages can be of type: models.EquityAgg, models.EquityTrade, etc.
    Output() <-chan any
    
    // Error returns a channel that receives error messages.
    Error() <-chan error
    
    // Close gracefully closes the WebSocket connection.
    Close()
}

// polygonWebSocketServiceWrapper wraps the real Polygon WebSocket client.
type polygonWebSocketServiceWrapper struct {
    client *websocket.Client
}

func newPolygonWebSocketService(apiKey string, feed websocket.Feed) (PolygonWebSocketService, error) {
    config := websocket.Config{
        APIKey: apiKey,
        Feed:   feed,
        Market: websocket.Stocks,
    }
    
    client, err := websocket.New(config)
    if err != nil {
        return nil, fmt.Errorf("failed to create polygon websocket client: %w", err)
    }
    
    return &polygonWebSocketServiceWrapper{client: client}, nil
}

func (w *polygonWebSocketServiceWrapper) Connect() error {
    return w.client.Connect()
}

func (w *polygonWebSocketServiceWrapper) Subscribe(topic websocket.Topic, tickers ...string) error {
    return w.client.Subscribe(topic, tickers...)
}

func (w *polygonWebSocketServiceWrapper) Unsubscribe(topic websocket.Topic, tickers ...string) error {
    return w.client.Unsubscribe(topic, tickers...)
}

func (w *polygonWebSocketServiceWrapper) Output() <-chan any {
    return w.client.Output()
}

func (w *polygonWebSocketServiceWrapper) Error() <-chan error {
    return w.client.Error()
}

func (w *polygonWebSocketServiceWrapper) Close() {
    w.client.Close()
}
```

### Stream() Implementation

```go
// Stream implements Provider.Stream for real-time WebSocket market data from Polygon.
// It subscribes to aggregate streams for all specified symbols and yields data as it arrives.
// The iterator terminates when the context is cancelled or an unrecoverable error occurs.
func (c *PolygonClient) Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error] {
    return func(yield func(types.MarketData, error) bool) {
        // Validate inputs
        if len(symbols) == 0 {
            yield(types.MarketData{}, fmt.Errorf("no symbols provided for streaming"))
            return
        }
        
        topic, err := convertIntervalToPolygonTopic(interval)
        if err != nil {
            yield(types.MarketData{}, err)
            return
        }
        
        // Create WebSocket service
        wsService, err := newPolygonWebSocketService(c.apiKey, websocket.RealTime)
        if err != nil {
            yield(types.MarketData{}, fmt.Errorf("failed to create websocket service: %w", err))
            return
        }
        defer wsService.Close()
        
        // Connect to WebSocket
        if err := wsService.Connect(); err != nil {
            yield(types.MarketData{}, fmt.Errorf("failed to connect to polygon websocket: %w", err))
            return
        }
        
        // Subscribe to aggregate topic for all symbols
        if err := wsService.Subscribe(topic, symbols...); err != nil {
            yield(types.MarketData{}, fmt.Errorf("failed to subscribe to symbols: %w", err))
            return
        }
        
        // Main message loop
        for {
            select {
            case <-ctx.Done():
                return
                
            case err := <-wsService.Error():
                if !yield(types.MarketData{}, fmt.Errorf("websocket error: %w", err)) {
                    return
                }
                
            case msg := <-wsService.Output():
                switch agg := msg.(type) {
                case models.EquityAgg:
                    marketData := convertEquityAggToMarketData(&agg)
                    if !yield(marketData, nil) {
                        return
                    }
                // Ignore other message types (trades, quotes, control messages)
                default:
                    // Skip non-aggregate messages
                }
            }
        }
    }
}

// convertIntervalToPolygonTopic converts an interval string to a Polygon WebSocket topic.
func convertIntervalToPolygonTopic(interval string) (websocket.Topic, error) {
    switch interval {
    case "1s":
        return websocket.StocksSecAggs, nil
    case "1m":
        return websocket.StocksMinAggs, nil
    default:
        // For other intervals, use minute aggregates and aggregate locally
        // This is a limitation of the Polygon WebSocket API which only supports
        // second and minute aggregates natively
        return websocket.StocksMinAggs, nil
    }
}

// convertEquityAggToMarketData converts a Polygon EquityAgg to our internal MarketData type.
func convertEquityAggToMarketData(agg *models.EquityAgg) types.MarketData {
    return types.MarketData{
        Id:     "",
        Symbol: agg.Symbol,
        Time:   time.UnixMilli(agg.StartTimestamp),
        Open:   agg.Open,
        High:   agg.High,
        Low:    agg.Low,
        Close:  agg.Close,
        Volume: agg.Volume,
    }
}
```

## Configuration

### PolygonClient Extension

```go
// PolygonClientConfig holds configuration for the Polygon client.
type PolygonClientConfig struct {
    // APIKey is the Polygon.io API key (required)
    APIKey string `json:"api_key" yaml:"api_key" validate:"required"`
    
    // Feed specifies which data feed to use (default: RealTime)
    // Options: "realtime", "delayed", "nasdaq", "launchpad"
    Feed string `json:"feed" yaml:"feed" jsonschema:"default=realtime,enum=realtime,enum=delayed,enum=nasdaq,enum=launchpad"`
}

// PolygonClient extended fields
type PolygonClient struct {
    apiClient  PolygonAPIClient
    wsService  PolygonWebSocketService  // Added for WebSocket
    apiKey     string                   // Added to store API key for WebSocket
    feed       websocket.Feed           // Added to store feed configuration
    writer     writer.MarketDataWriter
}
```

### Feed Type Mapping

```go
// convertFeedString converts a feed string to a websocket.Feed.
func convertFeedString(feed string) websocket.Feed {
    switch strings.ToLower(feed) {
    case "delayed":
        return websocket.Delayed
    case "nasdaq":
        return websocket.Nasdaq
    case "launchpad":
        return websocket.LaunchpadFeed
    default:
        return websocket.RealTime
    }
}
```

## Interval Support

### Polygon WebSocket Limitations

The Polygon WebSocket API natively supports only:
- **Second aggregates** (`A.{symbol}`) - Updated every second
- **Minute aggregates** (`AM.{symbol}`) - Updated every minute

For other intervals (5m, 15m, 1h, etc.), we have two options:

1. **Use minute aggregates** and let the strategy handle aggregation
2. **Aggregate locally** in the client (more complex)

**Recommended approach**: Use minute aggregates for all intervals and document that strategies should use built-in indicators for longer timeframe analysis.

### Interval Validation

```go
// isValidPolygonInterval validates that the interval is supported.
func isValidPolygonInterval(interval string) bool {
    validIntervals := map[string]bool{
        "1s": true,  // Second aggregates
        "1m": true,  // Minute aggregates
    }
    return validIntervals[interval]
}
```

## Testing Strategy

### Unit Tests

Create `polygon_stream_test.go` with tests mirroring the Binance implementation.

> **Note**: The example code below uses magic number timestamps (e.g., `1704067200000`) for brevity. In the actual implementation, these should be defined as named constants or constructed using `time.Date()` for better readability and maintainability. For example:
> ```go
> const (
>     testTimestamp1 = 1704067200000 // 2024-01-01 00:00:00 UTC
>     testTimestamp2 = 1704067260000 // 2024-01-01 00:01:00 UTC
> )
> ```

```go
package provider

import (
    "context"
    "errors"
    "testing"
    "time"
    
    polygonws "github.com/polygon-io/client-go/websocket"
    "github.com/polygon-io/client-go/websocket/models"
    "github.com/stretchr/testify/suite"
)

// mockPolygonWebSocketService implements PolygonWebSocketService for testing.
type mockPolygonWebSocketService struct {
    events       []any     // Events to emit (models.EquityAgg, etc.)
    errors       []error   // Errors to emit
    connectError error     // Error on Connect() call
    outputChan   chan any
    errorChan    chan error
    closed       bool
}

func newMockPolygonWebSocketService() *mockPolygonWebSocketService {
    return &mockPolygonWebSocketService{
        outputChan: make(chan any, 100),
        errorChan:  make(chan error, 10),
    }
}

func (m *mockPolygonWebSocketService) Connect() error {
    if m.connectError != nil {
        return m.connectError
    }
    
    // Start emitting events in background
    go func() {
        for _, event := range m.events {
            m.outputChan <- event
        }
        for _, err := range m.errors {
            m.errorChan <- err
        }
    }()
    
    return nil
}

func (m *mockPolygonWebSocketService) Subscribe(topic polygonws.Topic, tickers ...string) error {
    return nil
}

func (m *mockPolygonWebSocketService) Unsubscribe(topic polygonws.Topic, tickers ...string) error {
    return nil
}

func (m *mockPolygonWebSocketService) Output() <-chan any {
    return m.outputChan
}

func (m *mockPolygonWebSocketService) Error() <-chan error {
    return m.errorChan
}

func (m *mockPolygonWebSocketService) Close() {
    m.closed = true
    close(m.outputChan)
    close(m.errorChan)
}

type PolygonStreamTestSuite struct {
    suite.Suite
}

func TestPolygonStreamSuite(t *testing.T) {
    suite.Run(t, new(PolygonStreamTestSuite))
}

func (suite *PolygonStreamTestSuite) TestStreamSingleSymbol() {
    events := []any{
        models.EquityAgg{
            EventType:      "AM",
            Symbol:         "AAPL",
            Open:           150.00,
            High:           152.00,
            Low:            149.50,
            Close:          151.50,
            Volume:         1000000,
            StartTimestamp: 1704067200000,
        },
        models.EquityAgg{
            EventType:      "AM",
            Symbol:         "AAPL",
            Open:           151.50,
            High:           153.00,
            Low:            151.00,
            Close:          152.75,
            Volume:         800000,
            StartTimestamp: 1704067260000,
        },
    }
    
    mockWs := newMockPolygonWebSocketService()
    mockWs.events = events
    
    client := NewPolygonClientWithWebSocket("test-api-key", mockWs)
    
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    
    var received []types.MarketData
    for data, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
        if err != nil {
            break
        }
        received = append(received, data)
    }
    
    suite.Len(received, 2)
    suite.Equal("AAPL", received[0].Symbol)
    suite.InDelta(150.00, received[0].Open, 0.01)
    suite.InDelta(151.50, received[0].Close, 0.01)
}

func (suite *PolygonStreamTestSuite) TestStreamMultipleSymbols() {
    events := []any{
        models.EquityAgg{Symbol: "AAPL", Open: 150.00, Close: 151.50, StartTimestamp: 1704067200000},
        models.EquityAgg{Symbol: "GOOGL", Open: 140.00, Close: 141.50, StartTimestamp: 1704067200000},
    }
    
    mockWs := newMockPolygonWebSocketService()
    mockWs.events = events
    
    client := NewPolygonClientWithWebSocket("test-api-key", mockWs)
    
    ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
    defer cancel()
    
    symbolsSeen := make(map[string]bool)
    for data, err := range client.Stream(ctx, []string{"AAPL", "GOOGL"}, "1m") {
        if err != nil {
            break
        }
        symbolsSeen[data.Symbol] = true
    }
    
    suite.True(symbolsSeen["AAPL"])
    suite.True(symbolsSeen["GOOGL"])
}

func (suite *PolygonStreamTestSuite) TestStreamConnectionError() {
    mockWs := newMockPolygonWebSocketService()
    mockWs.connectError = errors.New("authentication failed")
    
    client := NewPolygonClientWithWebSocket("invalid-api-key", mockWs)
    
    ctx := context.Background()
    var gotError bool
    var errorMsg string
    
    for _, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
        if err != nil {
            gotError = true
            errorMsg = err.Error()
            break
        }
    }
    
    suite.True(gotError)
    suite.Contains(errorMsg, "failed to connect")
}

func (suite *PolygonStreamTestSuite) TestStreamEmptySymbols() {
    mockWs := newMockPolygonWebSocketService()
    client := NewPolygonClientWithWebSocket("test-api-key", mockWs)
    
    ctx := context.Background()
    var gotError bool
    
    for _, err := range client.Stream(ctx, []string{}, "1m") {
        if err != nil {
            gotError = true
            break
        }
    }
    
    suite.True(gotError)
}

func (suite *PolygonStreamTestSuite) TestStreamContextCancellation() {
    mockWs := newMockPolygonWebSocketService()
    // Don't add events - let the context cancellation terminate
    
    client := NewPolygonClientWithWebSocket("test-api-key", mockWs)
    
    ctx, cancel := context.WithCancel(context.Background())
    
    go func() {
        time.Sleep(50 * time.Millisecond)
        cancel()
    }()
    
    iterCount := 0
    for range client.Stream(ctx, []string{"AAPL"}, "1m") {
        iterCount++
        if iterCount > 10 {
            break
        }
    }
    
    suite.LessOrEqual(iterCount, 10)
}

func (suite *PolygonStreamTestSuite) TestConvertEquityAggToMarketData() {
    agg := &models.EquityAgg{
        Symbol:         "MSFT",
        Open:           380.50,
        High:           385.00,
        Low:            378.00,
        Close:          383.75,
        Volume:         500000,
        StartTimestamp: 1704067200000,
    }
    
    data := convertEquityAggToMarketData(agg)
    
    suite.Equal("MSFT", data.Symbol)
    suite.Equal(time.UnixMilli(1704067200000), data.Time)
    suite.InDelta(380.50, data.Open, 0.01)
    suite.InDelta(385.00, data.High, 0.01)
    suite.InDelta(378.00, data.Low, 0.01)
    suite.InDelta(383.75, data.Close, 0.01)
    suite.InDelta(500000, data.Volume, 0.01)
}

func (suite *PolygonStreamTestSuite) TestConvertIntervalToPolygonTopic() {
    topic, err := convertIntervalToPolygonTopic("1s")
    suite.NoError(err)
    suite.Equal(polygonws.StocksSecAggs, topic)
    
    topic, err = convertIntervalToPolygonTopic("1m")
    suite.NoError(err)
    suite.Equal(polygonws.StocksMinAggs, topic)
    
    // Other intervals should default to minute aggregates
    topic, err = convertIntervalToPolygonTopic("5m")
    suite.NoError(err)
    suite.Equal(polygonws.StocksMinAggs, topic)
}
```

### Integration Test (Optional)

```go
// +build integration

func TestPolygonStreamIntegration(t *testing.T) {
    apiKey := os.Getenv("POLYGON_API_KEY")
    if apiKey == "" {
        t.Skip("POLYGON_API_KEY not set")
    }
    
    client, err := NewPolygonClient(apiKey)
    require.NoError(t, err)
    
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()
    
    dataReceived := false
    for data, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
        if err != nil {
            t.Logf("Stream error: %v", err)
            break
        }
        
        t.Logf("Received: %s %.2f", data.Symbol, data.Close)
        dataReceived = true
        break // Just verify we can receive one message
    }
    
    assert.True(t, dataReceived, "Should receive at least one data point")
}
```

## Implementation Phases

### Phase 1: Interface and Wrapper
**Files to modify:**
- `pkg/marketdata/provider/polygon.go`

**Tasks:**
1. Add `PolygonWebSocketService` interface
2. Create `polygonWebSocketServiceWrapper` implementation
3. Add `apiKey` field to `PolygonClient`
4. Modify `NewPolygonClient` to store API key

### Phase 2: Stream Implementation
**Files to modify:**
- `pkg/marketdata/provider/polygon.go`

**Tasks:**
1. Implement `Stream()` method
2. Add `convertEquityAggToMarketData()` helper
3. Add `convertIntervalToPolygonTopic()` helper
4. Add constructor for testing: `NewPolygonClientWithWebSocket()`

### Phase 3: Unit Tests
**Files to create:**
- `pkg/marketdata/provider/polygon_stream_test.go`

**Tasks:**
1. Create `mockPolygonWebSocketService`
2. Implement test cases mirroring Binance tests
3. Run tests and verify all pass

### Phase 4: Documentation Update
**Files to modify:**
- `docs/realtime-market-data.md`

**Tasks:**
1. Update Polygon section from "TODO" to implemented
2. Add usage examples
3. Document supported intervals
4. Document authentication requirements

## Dependencies

The implementation uses the existing `polygon-io/client-go` package:

```go
import (
    polygonws "github.com/polygon-io/client-go/websocket"
    "github.com/polygon-io/client-go/websocket/models"
)
```

No new dependencies are required.

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    
    "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

func main() {
    apiKey := os.Getenv("POLYGON_API_KEY")
    if apiKey == "" {
        log.Fatal("POLYGON_API_KEY environment variable required")
    }
    
    // Create Polygon provider
    client, err := provider.NewPolygonClient(apiKey)
    if err != nil {
        log.Fatal(err)
    }
    
    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()
    
    // Stream real-time data for US stocks
    for data, err := range client.Stream(ctx, []string{"AAPL", "GOOGL", "MSFT"}, "1m") {
        if err != nil {
            log.Printf("Stream error: %v", err)
            break
        }
        
        fmt.Printf("%s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.0f\n",
            data.Symbol, data.Open, data.High, data.Low, data.Close, data.Volume)
    }
}
```

## Comparison: Binance vs Polygon Streaming

| Feature | Binance | Polygon |
|---------|---------|---------|
| Market | Crypto | US Stocks |
| Authentication | None required | API key required |
| Intervals | 1m, 3m, 5m, 15m, 30m, 1h, etc. | 1s, 1m (native) |
| WebSocket Library | `go-binance/v2` | `polygon-io/client-go` |
| Data Types | Kline events | EquityAgg events |
| Free Tier | Real-time | 15-minute delayed |

## Security Considerations

1. **API Key Protection**
   - Never log the API key
   - Store in environment variables or secure config
   - Validate API key format before connecting

2. **Rate Limits**
   - Polygon WebSocket has connection limits per API key
   - Handle reconnection with exponential backoff
   - Monitor for rate limit errors

3. **Error Handling**
   - Yield errors through the iterator for caller to handle
   - Distinguish between recoverable and fatal errors
   - Log connection issues for debugging

## Future Enhancements

1. **Trade/Quote Support**: Add optional support for `StocksTrades` and `StocksQuotes` topics
2. **Local Aggregation**: Aggregate minute bars into longer intervals (5m, 15m, 1h)
3. **Feed Selection**: Allow runtime configuration of feed type (RealTime vs Delayed)
4. **Reconnection Logic**: Implement automatic reconnection with backoff
5. **Subscription Management**: Dynamic add/remove symbols without reconnection

## References

- [Polygon.io WebSocket Documentation](https://polygon.io/docs/stocks/ws_getting-started)
- [polygon-io/client-go WebSocket Package](https://pkg.go.dev/github.com/polygon-io/client-go/websocket)
- [Existing Binance Stream Implementation](../pkg/marketdata/provider/binance.go)

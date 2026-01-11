---
title: Realtime Market Data Architecture
description: Design for realtime market data streaming using Go iterators
---

# Realtime Market Data Architecture

This document describes the realtime market data streaming capability in the Argo Trading framework using Go 1.23+ iterators.

## Overview

The realtime market data system extends the existing `Provider` interface to support WebSocket-based streaming. Key design principles:

- **Go Iterator Pattern**: Uses `iter.Seq2[types.MarketData, error]` for streaming
- **Extends Existing Interface**: No new types - just adds `Stream()` to `Provider`
- **Reuses Existing Types**: Uses `types.MarketData` and `writer.MarketDataWriter` as-is

## Provider Interface

The `Provider` interface in [pkg/marketdata/provider/provider.go](../pkg/marketdata/provider/provider.go) includes:

```go
type Provider interface {
    // Existing methods
    ConfigWriter(writer writer.MarketDataWriter)
    Download(ctx context.Context, ticker string, startDate time.Time, endDate time.Time,
             multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error)

    // Stream returns an iterator that yields realtime market data via WebSocket.
    // Uses Go 1.23+ iter.Seq2 pattern for streaming data.
    Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error]
}
```

## Usage Example

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
)

func main() {
    // Create provider
    client, err := provider.NewBinanceClient()
    if err != nil {
        log.Fatal(err)
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    // Stream realtime data using Go iterator
    for data, err := range client.Stream(ctx, []string{"BTCUSDT", "ETHUSDT"}, "1m") {
        if err != nil {
            log.Printf("stream error: %v", err)
            break
        }

        fmt.Printf("%s: O=%.2f H=%.2f L=%.2f C=%.2f V=%.2f\n",
            data.Symbol, data.Open, data.High, data.Low, data.Close, data.Volume)

        // Optionally write to storage
        // writer.Write(data)
    }
}
```

## Supported Providers

### Binance

- **Symbols**: Cryptocurrency pairs (e.g., `BTCUSDT`, `ETHUSDT`)
- **Intervals**: `1m`, `3m`, `5m`, `15m`, `30m`, `1h`, `2h`, `4h`, `6h`, `8h`, `12h`, `1d`, `3d`, `1w`, `1M`
- **WebSocket**: `wss://stream.binance.com:9443/ws`
- **Authentication**: Not required

### Polygon (TODO)

- **Symbols**: US stocks (e.g., `AAPL`, `GOOGL`)
- **WebSocket**: `wss://socket.polygon.io/stocks`
- **Authentication**: API key required

## WebSocket Endpoints

| Provider | Endpoint |
|----------|----------|
| Binance Production | `wss://stream.binance.com:9443/ws` |
| Binance Testnet | `wss://testnet.binance.vision/ws` |
| Polygon Stocks | `wss://socket.polygon.io/stocks` |

## Error Handling

Errors are yielded through the iterator. Handle them in your range loop:

```go
for data, err := range provider.Stream(ctx, symbols, interval) {
    if err != nil {
        // Handle error (connection lost, invalid symbol, etc.)
        log.Printf("error: %v", err)
        // Decide whether to break or continue
        break
    }
    // Process data
}
```

## Stopping the Stream

Cancel the context to stop streaming:

```go
ctx, cancel := context.WithCancel(context.Background())

// Start streaming in a goroutine
go func() {
    for data, err := range provider.Stream(ctx, symbols, interval) {
        // ...
    }
}()

// Later, stop the stream
cancel()
```

## Notes

- Go 1.23+ required for `iter` package
- Context cancellation gracefully closes WebSocket connections
- Each symbol gets its own WebSocket connection (Binance)
- Data is yielded as soon as it arrives (real-time)

---
title: Order Confirmation
description: Configuration and usage of order confirmation for live trading
---

# Order Confirmation

This document describes the order confirmation feature for the Live Trading Engine. Order confirmation allows you to review and approve or reject orders before they are sent to the trading provider, providing an additional safety layer for live trading.

## Overview

When a strategy generates an order via `PlaceOrder()`, the engine can be configured to:

1. **Auto-confirm** — Orders are sent to the trading provider immediately (default behavior).
2. **Manual confirm** — Orders are held and the `OnConfirmOrder` callback is invoked. The order is only placed if the callback returns `true`. If it returns `false`, the order is rejected.

This is particularly useful for:

- Reviewing orders before execution in a live trading environment
- Implementing custom risk checks or approval workflows
- Building mobile/desktop UIs where a user must tap "Confirm" before an order goes through

## Configuration

Order confirmation is configured via the `LiveTradingEngineConfig` struct. The `OrderConfirmation` field controls the confirmation behavior.

### Configuration Options

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `order_confirmation` | `OrderConfirmationMode` | `"auto"` | `"auto"` to auto-confirm, `"manual"` to require callback confirmation |
| `confirmation_timeout` | `duration` | `30s` | Maximum time to wait for a confirmation response. If the timeout expires, the order is rejected. Only used when `order_confirmation` is `"manual"`. |

### Go Configuration

```go
config := engine.LiveTradingEngineConfig{
    MarketDataCacheSize:  1000,
    EnableLogging:        true,
    OrderConfirmation:    engine.OrderConfirmationManual, // require manual confirmation
    ConfirmationTimeout:  60 * time.Second,               // reject if no response within 60s
    Prefetch: engine.PrefetchConfig{
        Enabled:       true,
        StartTimeType: "days",
        Days:          30,
    },
}
```

### YAML Configuration

```yaml
engine:
  market_data_cache_size: 1000
  enable_logging: true
  order_confirmation: manual    # "auto" (default) or "manual"
  confirmation_timeout: 30s     # timeout for manual confirmation (default: 30s)
  prefetch:
    enabled: true
    start_time_type: days
    days: 30
```

### Constants

```go
const (
    // OrderConfirmationAuto sends orders to the trading provider immediately.
    OrderConfirmationAuto OrderConfirmationMode = "auto"

    // OrderConfirmationManual holds orders and invokes the OnConfirmOrder callback.
    OrderConfirmationManual OrderConfirmationMode = "manual"
)
```

## Order Placement Flow

### Auto-Confirm (Default)

When `order_confirmation` is `"auto"` (or unset), the order flow is unchanged from the default behavior:

```
Strategy.ProcessData()
        │
        ▼
  PlaceOrder(order)
        │
        ▼
  Trading Provider
  (order executed)
        │
        ▼
  OnOrderPlaced callback
```

### Manual Confirm

When `order_confirmation` is `"manual"`, the engine intercepts the order and invokes the `OnConfirmOrder` callback. If the callback does not respond within `confirmation_timeout`, the order is automatically rejected:

```
Strategy.ProcessData()
        │
        ▼
  PlaceOrder(order)
        │
        ▼
  OnConfirmOrder(order)
        │
    ┌───┼───────┐
    │   │       │
  true false  timeout
    │   │       │
    ▼   ▼       ▼
  Trading   Order
  Provider  rejected
  (order    (not sent to
  executed) provider)
    │
    ▼
  OnOrderPlaced callback
```

## Callbacks

### Go API

The `OnConfirmOrder` callback is part of the `LiveTradingCallbacks` struct:

```go
// OnConfirmOrderCallback is called when an order requires confirmation.
// Return true to confirm (place the order), or false to reject it.
type OnConfirmOrderCallback func(order types.ExecuteOrder) bool

type LiveTradingCallbacks struct {
    // ... existing callbacks ...

    // OnConfirmOrder is called when the engine is in manual confirmation mode
    // and a strategy wants to place an order.
    // Return true to confirm the order, false to reject it.
    // If nil and mode is "manual", the order is auto-confirmed.
    OnConfirmOrder *OnConfirmOrderCallback
}
```

**Example (Go):**

```go
onConfirm := engine.OnConfirmOrderCallback(func(order types.ExecuteOrder) bool {
    fmt.Printf("Order pending: %s %s %.4f %s @ %.2f\n",
        order.Side, order.Quantity, order.Symbol, order.OrderType, order.Price)

    // Custom risk check
    if order.Price * order.Quantity > 10000 {
        fmt.Println("Order exceeds $10,000 limit — rejected")
        return false
    }

    return true
})

callbacks := engine.LiveTradingCallbacks{
    OnConfirmOrder: &onConfirm,
    // ... other callbacks ...
}

eng.Run(ctx, callbacks)
```

### Swift API

In the Swift API, the `TradingEngineHelper` interface includes an `OnConfirmOrder` method:

```go
// TradingEngineHelper is the callback interface for live trading lifecycle events.
type TradingEngineHelper interface {
    // ... existing methods ...

    // OnConfirmOrder is called when manual confirmation is enabled and a strategy
    // wants to place an order. Return true to confirm, false to reject.
    // orderJSON is the JSON representation of the ExecuteOrder.
    OnConfirmOrder(orderJSON string) bool
}
```

**Example (Swift):**

```swift
class MyTradingHelper: TradingEngineHelper {
    // Called when the engine requests order confirmation
    func onConfirmOrder(_ orderJSON: String) -> Bool {
        // Parse the order JSON
        guard let data = orderJSON.data(using: .utf8),
              let order = try? JSONDecoder().decode(ExecuteOrder.self, from: data) else {
            return false
        }

        // Example: reject orders above a certain size
        if order.price * order.quantity > 10000 {
            print("Order too large, rejecting: \(order.symbol)")
            return false
        }

        print("Confirming order: \(order.side) \(order.quantity) \(order.symbol)")
        return true
    }

    // ... other TradingEngineHelper methods ...
}
```

**Using with a UI confirmation dialog (Swift):**

```swift
class MyTradingHelper: TradingEngineHelper {
    func onConfirmOrder(_ orderJSON: String) -> Bool {
        // Use a semaphore to wait for user input on the main thread
        let semaphore = DispatchSemaphore(value: 0)
        var confirmed = false

        DispatchQueue.main.async {
            // Show confirmation dialog to user
            let alert = UIAlertController(
                title: "Confirm Order",
                message: "Place order: \(orderJSON)?",
                preferredStyle: .alert
            )
            alert.addAction(UIAlertAction(title: "Confirm", style: .default) { _ in
                confirmed = true
                semaphore.signal()
            })
            alert.addAction(UIAlertAction(title: "Reject", style: .cancel) { _ in
                confirmed = false
                semaphore.signal()
            })
            // Present the alert...
        }

        semaphore.wait()
        return confirmed
    }
}
```

> **Note:** The `OnConfirmOrder` callback is invoked on the engine's processing thread. For UI-based confirmations, you must dispatch to the main thread and block the engine thread until the user responds, as shown above.

## Behavior Details

### Default Behavior

If `order_confirmation` is not set or is set to `"auto"`, the engine behaves exactly as before — orders are placed immediately with no confirmation step. The `OnConfirmOrder` callback is never invoked.

### Manual Mode Without Callback

If `order_confirmation` is `"manual"` but the `OnConfirmOrder` callback is `nil` (not provided), the engine falls back to auto-confirm behavior. This ensures backward compatibility.

### Callback Return Value

| Return Value | Behavior |
|-------------|----------|
| `true` | Order is confirmed and sent to the trading provider |
| `false` | Order is rejected and not sent to the trading provider |

### Confirmation Timeout

When `order_confirmation` is `"manual"`, the engine enforces a timeout on the `OnConfirmOrder` callback via the `confirmation_timeout` setting (default: `30s`). If the callback does not return within the timeout period, the order is automatically **rejected** — it will not be sent to the trading provider.

This prevents the engine from stalling indefinitely if the user does not respond (e.g., app in background, UI not visible).

**Go configuration example:**

```go
config := engine.LiveTradingEngineConfig{
    OrderConfirmation:   engine.OrderConfirmationManual,
    ConfirmationTimeout: 60 * time.Second, // 60-second timeout
}
```

**YAML configuration example:**

```yaml
engine:
  order_confirmation: manual
  confirmation_timeout: 60s
```

> **Note:** The timeout is only enforced when `order_confirmation` is `"manual"`. In `"auto"` mode, the timeout setting is ignored.

### Interaction with Other Callbacks

- `OnOrderPlaced` is only called if the order is confirmed (callback returns `true` or mode is `"auto"`).
- `OnOrderFilled` is only called after a confirmed order is filled by the trading provider.
- Rejected orders do not trigger `OnOrderPlaced` or `OnOrderFilled`.

## Full Example

### Go

```go
package main

import (
    "context"
    "fmt"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/rxtech-lab/argo-trading/internal/trading/engine"
    "github.com/rxtech-lab/argo-trading/internal/types"
)

func main() {
    eng, err := engine.NewLiveTradingEngineV1()
    if err != nil {
        log.Fatal(err)
    }

    config := engine.LiveTradingEngineConfig{
        MarketDataCacheSize:  1000,
        EnableLogging:        true,
        OrderConfirmation:    engine.OrderConfirmationManual,
        ConfirmationTimeout:  30 * time.Second,
        Prefetch: engine.PrefetchConfig{
            Enabled:       true,
            StartTimeType: "days",
            Days:          30,
        },
    }

    if err := eng.Initialize(config); err != nil {
        log.Fatal(err)
    }

    // ... set providers, load strategy ...

    // Define confirmation callback
    onConfirm := engine.OnConfirmOrderCallback(func(order types.ExecuteOrder) bool {
        fmt.Printf("Confirm? %s %s %.4f @ %.2f\n",
            order.Side, order.Symbol, order.Quantity, order.Price)
        return true // confirm all orders in this example
    })

    onPlaced := engine.OnOrderPlacedCallback(func(order types.ExecuteOrder) error {
        fmt.Printf("Order placed: %s\n", order.ID)
        return nil
    })

    callbacks := engine.LiveTradingCallbacks{
        OnConfirmOrder: &onConfirm,
        OnOrderPlaced:  &onPlaced,
    }

    ctx, cancel := context.WithCancel(context.Background())
    defer cancel()

    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
    go func() {
        <-sigChan
        cancel()
    }()

    if err := eng.Run(ctx, callbacks); err != nil {
        log.Fatal(err)
    }
}
```

### Swift

```swift
import Argo

class TradingApp: TradingEngineHelper {
    var engine: TradingEngine?

    func start() {
        engine = try? NewTradingEngine(self)

        // Enable manual order confirmation
        let config = """
        {
            "market_data_cache_size": 1000,
            "enable_logging": true,
            "order_confirmation": "manual",
            "confirmation_timeout": "30s",
            "prefetch": {
                "enabled": true,
                "start_time_type": "days",
                "days": 30
            }
        }
        """

        try? engine?.initialize(config)
        // ... set providers, load strategy ...
        try? engine?.run()
    }

    // MARK: - TradingEngineHelper

    func onConfirmOrder(_ orderJSON: String) -> Bool {
        // Implement your confirmation logic here
        print("Order confirmation requested: \(orderJSON)")
        return true
    }

    func onEngineStart(_ symbols: StringCollection, interval: String, previousDataPath: String) throws {
        print("Engine started")
    }

    func onEngineStop(_ err: Error?) {
        print("Engine stopped")
    }

    func onMarketData(_ runId: String, symbol: String, timestamp: Int64,
                       open: Double, high: Double, low: Double,
                       close: Double, volume: Double) throws {
        // Handle market data
    }

    func onOrderPlaced(_ orderJSON: String) throws {
        print("Order placed: \(orderJSON)")
    }

    func onOrderFilled(_ orderJSON: String) throws {
        print("Order filled: \(orderJSON)")
    }

    func onError(_ err: Error) {
        print("Error: \(err)")
    }

    func onStrategyError(_ symbol: String, timestamp: Int64, err: Error) {
        print("Strategy error: \(err)")
    }
}
```

## Related Documentation

- [Live Trading Engine](index.md) — Core engine overview, configuration, and CLI usage
- [Session Management and Persistence](session-and-persistence.md) — Session lifecycle and data storage
- [Data Prefetch](data-prefetch.md) — Historical data prefetching for indicators
- [Testing](testing.md) — Mock providers and E2E testing

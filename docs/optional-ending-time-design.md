# Design Document: Optional Ending Time and Trading Session Cleanup

## Overview

This document outlines the design for implementing an optional ending time for the trading system, along with a lifecycle method to clean up open positions when trading ends. This feature is particularly useful for day trading strategies that need to close all positions at the end of each trading day.

## Requirements

### Functional Requirements

1. **Optional Ending Time Configuration**
   - Add an optional `TradingEndTime` configuration parameter to both backtest and live trading engines
   - The ending time can be specified as:
     - A specific time of day (e.g., "16:00:00" for market close)
     - An absolute datetime for backtest scenarios
   - When the ending time is reached, the engine should trigger the cleanup lifecycle method

2. **Cleanup Lifecycle Method (`OnTradingEnd`)**
   - Add a new RPC method to the `TradingStrategy` protobuf service called `OnTradingEnd`
   - This method is called when:
     - The configured trading end time is reached
     - The trading session ends (e.g., context cancellation, market close)
     - A trading day boundary is crossed
   - The method receives the current market data and should allow the strategy to:
     - Close all open positions
     - Cancel pending orders
     - Perform any final cleanup logic

3. **Engine Support**
   - Backtest engine should call `OnTradingEnd` at the configured end time
   - Live trading engine should call `OnTradingEnd` at the configured end time and/or when the session ends
   - Both engines should provide a callback for external notification of trading end

## Files to be Modified

### 1. Protocol Buffer Definitions

**File: `pkg/strategy/strategy.proto`**

Add new RPC method and messages for the cleanup lifecycle:

```protobuf
// TradingStrategy service - add new method
service TradingStrategy {
  // ... existing methods ...
  
  // OnTradingEnd is called when the trading session ends.
  // This allows strategies to clean up positions and perform end-of-day operations.
  rpc OnTradingEnd(OnTradingEndRequest) returns (google.protobuf.Empty) {}
}

// OnTradingEndRequest contains information about the trading session end
message OnTradingEndRequest {
  // The last market data point before trading ended
  MarketData last_market_data = 1;
  // Reason why trading is ending
  TradingEndReason reason = 2;
}

// TradingEndReason indicates why the trading session is ending
enum TradingEndReason {
  // Trading end time reached (configured end time)
  TRADING_END_REASON_END_TIME_REACHED = 0;
  // Date boundary crossed (new trading day)
  TRADING_END_REASON_DATE_BOUNDARY = 1;
  // Engine shutdown (graceful shutdown)
  TRADING_END_REASON_SHUTDOWN = 2;
  // Manual stop requested
  TRADING_END_REASON_MANUAL_STOP = 3;
}
```

### 2. Strategy Runtime Interface

**File: `internal/runtime/runtime.go`**

Add the new lifecycle method to the `StrategyRuntime` interface:

```go
type StrategyRuntime interface {
    // ... existing methods ...
    
    // OnTradingEnd is called when the trading session ends
    // This allows strategies to clean up positions and perform end-of-day operations
    OnTradingEnd(data types.MarketData, reason types.TradingEndReason) error
}
```

### 3. Types Package

**File: `internal/types/trading_end.go` (new file)**

Add new types for trading end functionality:

```go
package types

import "time"

// TradingEndReason indicates why the trading session is ending
type TradingEndReason int

const (
    TradingEndReasonEndTimeReached TradingEndReason = iota
    TradingEndReasonDateBoundary
    TradingEndReasonShutdown
    TradingEndReasonManualStop
)

// TradingEndConfig holds configuration for trading session end time
type TradingEndConfig struct {
    // Enabled indicates if trading end time is enabled
    Enabled bool `yaml:"enabled" json:"enabled"`
    
    // EndTime is the time of day when trading should end (e.g., "16:00:00")
    // Format: HH:MM:SS in local timezone
    EndTime string `yaml:"end_time" json:"end_time"`
    
    // Timezone for the end time (e.g., "America/New_York")
    // Defaults to UTC if not specified
    Timezone string `yaml:"timezone" json:"timezone"`
    
    // ClosePositionsOnEnd indicates if all positions should be closed when trading ends
    ClosePositionsOnEnd bool `yaml:"close_positions_on_end" json:"close_positions_on_end"`
}

// ParseEndTime parses the EndTime string and returns the next occurrence
func (c *TradingEndConfig) ParseEndTime(currentTime time.Time) (time.Time, error) {
    // Implementation to parse time string and calculate next end time
    // ...
}
```

### 4. Backtest Engine Configuration

**File: `internal/backtest/engine/engine_v1/config.go`**

Add trading end configuration to backtest config:

```go
type BacktestEngineV1Config struct {
    // ... existing fields ...
    
    // TradingEnd configures optional trading session end time behavior
    TradingEnd types.TradingEndConfig `yaml:"trading_end" json:"trading_end" jsonschema:"title=Trading End,description=Configuration for trading session end time and cleanup"`
}
```

### 5. Backtest Engine Implementation

**File: `internal/backtest/engine/engine_v1/backtest_v1.go`**

Modify the `processDataPoints` method to check for trading end time:

```go
func (b *BacktestEngineV1) processDataPoints(params runIterationParams, ...) error {
    // ... existing code ...
    
    for data, err := range b.datasource.ReadAll(b.config.StartTime, b.config.EndTime) {
        // ... existing processing ...
        
        // Check if we've reached the trading end time
        if b.shouldEndTrading(data) {
            if err := b.handleTradingEnd(params.strategy, data, types.TradingEndReasonEndTimeReached); err != nil {
                b.log.Error("Failed to handle trading end", zap.Error(err))
            }
        }
        
        // ... rest of processing ...
    }
    
    return nil
}

// shouldEndTrading checks if the current time has reached the configured end time
func (b *BacktestEngineV1) shouldEndTrading(data types.MarketData) bool {
    if !b.config.TradingEnd.Enabled {
        return false
    }
    // Check if current data time matches the end time
    // ...
}

// handleTradingEnd calls the strategy's OnTradingEnd method
func (b *BacktestEngineV1) handleTradingEnd(strategy runtime.StrategyRuntime, data types.MarketData, reason types.TradingEndReason) error {
    return strategy.OnTradingEnd(data, reason)
}
```

### 6. Live Trading Engine Configuration

**File: `internal/trading/engine/engine.go`**

Add trading end configuration to live trading config:

```go
type LiveTradingEngineConfig struct {
    // ... existing fields ...
    
    // TradingEnd configures optional trading session end time behavior
    TradingEnd types.TradingEndConfig `json:"trading_end" yaml:"trading_end" jsonschema:"description=Configuration for trading session end time and cleanup"`
}
```

### 7. Live Trading Engine Implementation

**File: `internal/trading/engine/engine_v1/live_trading_v1.go`**

Add trading end time handling to the live trading loop:

```go
func (e *LiveTradingEngineV1) Run(ctx context.Context, callbacks engine.LiveTradingCallbacks) error {
    // ... existing code ...
    
    // Create a timer for trading end time if configured
    var endTimeTimer *time.Timer
    if e.config.TradingEnd.Enabled {
        endTime, err := e.config.TradingEnd.ParseEndTime(time.Now())
        if err == nil {
            duration := time.Until(endTime)
            endTimeTimer = time.NewTimer(duration)
        }
    }
    
    for data, err := range stream {
        select {
        case <-ctx.Done():
            e.handleTradingEnd(lastData, types.TradingEndReasonShutdown)
            // ...
            
        case <-endTimeTimer.C:
            e.handleTradingEnd(lastData, types.TradingEndReasonEndTimeReached)
            // Continue processing or stop based on config
            
        default:
            // Process market data
            // ...
        }
    }
}
```

### 8. Live Trading Engine Callbacks

**File: `internal/trading/engine/engine.go`**

Add new callback for trading end notification:

```go
// OnTradingEndCallback is called when the trading session ends
type OnTradingEndCallback func(reason types.TradingEndReason, lastData types.MarketData) error

type LiveTradingCallbacks struct {
    // ... existing callbacks ...
    
    // OnTradingEnd is called when the trading session ends
    OnTradingEnd *OnTradingEndCallback
}
```

### 9. WASM Runtime Implementation

**File: `internal/runtime/wasm/wasm_runtime.go`**

Implement the `OnTradingEnd` method in the WASM runtime:

```go
func (r *StrategyWasmRuntime) OnTradingEnd(data types.MarketData, reason types.TradingEndReason) error {
    // Convert types to protobuf
    req := &strategy.OnTradingEndRequest{
        LastMarketData: convertToProtoMarketData(data),
        Reason:         strategy.TradingEndReason(reason),
    }
    
    // Call the WASM plugin
    _, err := r.plugin.OnTradingEnd(r.ctx, req)
    return err
}
```

## Implementation TODOs

### Phase 1: Core Infrastructure

- [ ] **TODO-1**: Add `OnTradingEnd` RPC method and messages to `strategy.proto`
- [ ] **TODO-2**: Run `make generate` to regenerate protobuf code
- [ ] **TODO-3**: Create `internal/types/trading_end.go` with new types
- [ ] **TODO-4**: Update `StrategyRuntime` interface in `internal/runtime/runtime.go`

### Phase 2: Backtest Engine

- [ ] **TODO-5**: Add `TradingEndConfig` to `BacktestEngineV1Config` in `config.go`
- [ ] **TODO-6**: Implement `shouldEndTrading` method in `backtest_v1.go`
- [ ] **TODO-7**: Implement `handleTradingEnd` method in `backtest_v1.go`
- [ ] **TODO-8**: Modify `processDataPoints` to check for trading end time
- [ ] **TODO-9**: Update config schema generation

### Phase 3: Live Trading Engine

- [ ] **TODO-10**: Add `TradingEndConfig` to `LiveTradingEngineConfig` in `engine.go`
- [ ] **TODO-11**: Add `OnTradingEndCallback` to `LiveTradingCallbacks`
- [ ] **TODO-12**: Implement trading end time timer in `live_trading_v1.go`
- [ ] **TODO-13**: Handle date boundary transitions
- [ ] **TODO-14**: Update config schema generation

### Phase 4: WASM Runtime

- [ ] **TODO-15**: Implement `OnTradingEnd` in WASM runtime (`internal/runtime/wasm/`)
- [ ] **TODO-16**: Add plugin interface support for the new method

### Phase 5: Example Strategy

- [ ] **TODO-17**: Update example strategy to implement `OnTradingEnd`
- [ ] **TODO-18**: Add documentation for the new lifecycle method

## Example Usage

### Strategy Implementation

```go
//go:build wasip1

package main

import (
    "context"
    "github.com/knqyf263/go-plugin/types/known/emptypb"
    "github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type DayTradingStrategy struct{}

func (s *DayTradingStrategy) OnTradingEnd(ctx context.Context, req *strategy.OnTradingEndRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()
    
    // Get all open positions
    positions, err := api.GetPositions(ctx, &emptypb.Empty{})
    if err != nil {
        return nil, err
    }
    
    // Close all open positions
    for _, pos := range positions.Positions {
        if pos.Quantity > 0 {
            // Place market sell order to close position
            api.PlaceOrder(ctx, &strategy.ExecuteOrder{
                Symbol:    pos.Symbol,
                Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
                OrderType: strategy.OrderType_ORDER_TYPE_MARKET,
                Quantity:  pos.Quantity,
                Reason:    &strategy.Reason{
                    Reason:  "trading_end",
                    Message: "Closing position at end of trading day",
                },
            })
        }
    }
    
    // Cancel all pending orders
    api.CancelAllOrders(ctx, &emptypb.Empty{})
    
    return &emptypb.Empty{}, nil
}
```

### Configuration

**Backtest Configuration (YAML):**

```yaml
initial_capital: 100000
broker: interactive_broker

# Optional trading end configuration
trading_end:
  enabled: true
  end_time: "16:00:00"
  timezone: "America/New_York"
  close_positions_on_end: true
```

**Live Trading Configuration (YAML):**

```yaml
symbols:
  - AAPL
  - MSFT
interval: 1m
enable_logging: true
data_output_path: ./output

trading_end:
  enabled: true
  end_time: "15:55:00"  # 5 minutes before market close
  timezone: "America/New_York"
  close_positions_on_end: true
```

## Test Plan

### Unit Tests

#### 1. Types Package Tests
**File: `internal/types/trading_end_test.go`**

```go
func TestTradingEndConfig_ParseEndTime(t *testing.T) {
    tests := []struct {
        name        string
        config      TradingEndConfig
        currentTime time.Time
        wantErr     bool
    }{
        {
            name: "valid end time",
            config: TradingEndConfig{
                Enabled:  true,
                EndTime:  "16:00:00",
                Timezone: "America/New_York",
            },
            currentTime: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
            wantErr:     false,
        },
        {
            name: "invalid time format",
            config: TradingEndConfig{
                Enabled:  true,
                EndTime:  "invalid",
                Timezone: "America/New_York",
            },
            currentTime: time.Now(),
            wantErr:     true,
        },
    }
    // ... test implementation
}
```

#### 2. Backtest Engine Tests
**File: `internal/backtest/engine/engine_v1/backtest_v1_test.go`**

```go
func TestBacktestEngineV1_TradingEndTime(t *testing.T) {
    // Test that OnTradingEnd is called at the configured end time
}

func TestBacktestEngineV1_ShouldEndTrading(t *testing.T) {
    // Test the shouldEndTrading logic with various scenarios
}
```

#### 3. Live Trading Engine Tests
**File: `internal/trading/engine/engine_v1/live_trading_v1_test.go`**

```go
func TestLiveTradingEngineV1_TradingEndTimer(t *testing.T) {
    // Test that the trading end timer triggers correctly
}

func TestLiveTradingEngineV1_TradingEndCallback(t *testing.T) {
    // Test that OnTradingEnd callback is invoked
}
```

#### 4. Strategy Runtime Tests
**File: `internal/runtime/wasm/wasm_runtime_test.go`**

```go
func TestStrategyWasmRuntime_OnTradingEnd(t *testing.T) {
    // Test that OnTradingEnd is properly forwarded to the WASM plugin
}
```

### Integration Tests

#### 1. End-to-End Backtest Test
**File: `e2e/backtest_trading_end_test.go`**

```go
func TestBacktest_TradingEndClosesPositions(t *testing.T) {
    // 1. Set up backtest with trading end time configured
    // 2. Run strategy that opens positions
    // 3. Verify OnTradingEnd is called at configured time
    // 4. Verify all positions are closed after trading end
}
```

#### 2. End-to-End Live Trading Test
**File: `e2e/live_trading_end_test.go`**

```go
func TestLiveTrading_TradingEndTime(t *testing.T) {
    // 1. Set up live trading engine with short trading end time
    // 2. Start trading with mock provider
    // 3. Verify OnTradingEnd is called when timer fires
    // 4. Verify callback is invoked
}
```

### Manual Testing

1. **Backtest with Trading End Time**
   - Configure backtest with end time at "16:00:00"
   - Run with a strategy that opens positions
   - Verify positions are closed at end time
   - Check logs for OnTradingEnd invocation

2. **Live Trading with Trading End Time**
   - Configure live trading with end time 2 minutes from now
   - Start trading with paper trading account
   - Verify trading ends at configured time
   - Verify cleanup callback is invoked

3. **Edge Cases**
   - Test with end time in the past (should use next day)
   - Test with invalid timezone
   - Test with disabled trading end
   - Test strategy that throws error in OnTradingEnd

## Security Considerations

1. **Graceful Shutdown**: Ensure that even if OnTradingEnd fails, the engine shuts down gracefully
2. **Position Cleanup**: Log all position closures for audit trail
3. **Order Validation**: Validate that closing orders don't exceed position quantities

## Performance Considerations

1. **Timer Overhead**: Use efficient timer implementation for live trading
2. **Batch Operations**: Consider batching position closures for better performance
3. **Async Operations**: Handle OnTradingEnd asynchronously to not block the main loop

## Backward Compatibility

1. **Optional Feature**: Trading end time is disabled by default
2. **Strategy Compatibility**: Strategies without OnTradingEnd implementation should continue to work
3. **Configuration Migration**: Existing configurations should work without changes

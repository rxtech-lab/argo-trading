---
title: Implementing Trading Strategies
description: Guide to implementing trading strategies in the Argo Trading framework using WebAssembly plugins
---

# Implementing Trading Strategies

This guide explains how to implement trading strategies in the Argo Trading framework. Strategies are compiled to WebAssembly (WASM) and run in an isolated plugin architecture.

## Prerequisites

- **Go 1.24+** required
- Basic understanding of Go programming and trading concepts
- Familiarity with the Argo Trading framework (see [README.md](../README.md))

## Quick Start

The easiest way to create a new strategy is using the scaffolding tool:

```bash
pnpm create trading-strategy
# or
npx create-trading-strategy
```

This will automatically create a sample strategy on your local machine.

## Strategy Interface

Every strategy must implement the `TradingStrategy` interface with the following methods:

```go
type TradingStrategy interface {
    // Initialize sets up the strategy with a configuration string
    Initialize(ctx context.Context, req *InitializeRequest) (*emptypb.Empty, error)
    
    // ProcessData processes new market data and generates signals
    ProcessData(ctx context.Context, req *ProcessDataRequest) (*emptypb.Empty, error)
    
    // Name returns the name of the strategy
    Name(ctx context.Context, req *NameRequest) (*NameResponse, error)
    
    // GetConfigSchema returns the JSON schema of the strategy configuration
    GetConfigSchema(ctx context.Context, req *GetConfigSchemaRequest) (*GetConfigSchemaResponse, error)
    
    // GetDescription returns a description of the strategy
    GetDescription(ctx context.Context, req *GetDescriptionRequest) (*GetDescriptionResponse, error)
}
```

## Basic Strategy Structure

Here's a minimal strategy template:

```go
//go:build wasip1

package main

import (
    "context"
    "github.com/knqyf263/go-plugin/types/known/emptypb"
    "github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// MyStrategy implements a custom trading strategy
type MyStrategy struct {
    // Strategy is stateless - use cache for state
}

func main() {}

func init() {
    strategy.RegisterTradingStrategy(NewMyStrategy())
}

func NewMyStrategy() strategy.TradingStrategy {
    return &MyStrategy{}
}

// Initialize sets up the strategy
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    // Parse configuration from req.Config (JSON string)
    return &emptypb.Empty{}, nil
}

// ProcessData handles each market data point
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()
    
    // Your trading logic here
    
    return &emptypb.Empty{}, nil
}

// Name returns the strategy name
func (s *MyStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
    return &strategy.NameResponse{Name: "MyStrategy"}, nil
}

// GetConfigSchema returns the configuration schema
func (s *MyStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
    return &strategy.GetConfigSchemaResponse{Schema: ""}, nil
}

// GetDescription returns strategy description
func (s *MyStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
    return &strategy.GetDescriptionResponse{Description: "My custom trading strategy"}, nil
}
```

## Using the Strategy API

The `StrategyApi` provides access to all host functions. Create an instance in your `ProcessData` method:

```go
api := strategy.NewStrategyApi()
```

### Available API Methods

| Category | Method | Description |
|----------|--------|-------------|
| **Data Source** | `GetRange` | Get historical market data for a time range |
| | `ReadLastData` | Get the most recent market data |
| | `ExecuteSQL` | Execute custom SQL queries on DuckDB |
| | `Count` | Count data points in a time range |
| **Indicators** | `ConfigureIndicator` | Configure a technical indicator |
| | `GetSignal` | Get trading signal from an indicator |
| **Cache** | `GetCache` | Retrieve stored state |
| | `SetCache` | Store state (strategies are stateless) |
| **Trading** | `PlaceOrder` | Place a single order |
| | `PlaceMultipleOrders` | Place multiple orders |
| | `GetPositions` | Get all open positions |
| | `GetPosition` | Get position for a specific symbol |
| | `CancelOrder` | Cancel a pending order |
| | `CancelAllOrders` | Cancel all pending orders |
| | `GetOrderStatus` | Get status of an order |
| | `GetAccountInfo` | Get account balance and equity info |
| | `GetOpenOrders` | Get all pending orders |
| | `GetTrades` | Get trade history |
| **Markers** | `Mark` | Create a visual marker on the data |
| | `GetMarkers` | Get all markers |
| **Logging** | `Log` | Log messages with different levels |

## Placing Orders

To place an order, use the `PlaceOrder` method:

```go
api := strategy.NewStrategyApi()

order := &strategy.ExecuteOrder{
    Symbol:       data.Symbol,
    Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,  // or PURCHASE_TYPE_SELL
    OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,     // or ORDER_TYPE_MARKET
    Quantity:     1.0,
    Price:        data.Close,
    StrategyName: "MyStrategy",
    Reason: &strategy.Reason{
        Reason:  "strategy",
        Message: "Buy signal triggered",
    },
    PositionType: strategy.PositionType_POSITION_TYPE_LONG,  // or POSITION_TYPE_SHORT
}

_, err := api.PlaceOrder(ctx, order)
if err != nil {
    return nil, fmt.Errorf("failed to place order: %w", err)
}
```

### Order Types

| Type | Value | Description |
|------|-------|-------------|
| Market | `ORDER_TYPE_MARKET` | Execute immediately at market price |
| Limit | `ORDER_TYPE_LIMIT` | Execute at specified price or better |

### Purchase Types

| Type | Value | Description |
|------|-------|-------------|
| Buy | `PURCHASE_TYPE_BUY` | Buy to open/add to position |
| Sell | `PURCHASE_TYPE_SELL` | Sell to close/reduce position |

### Position Types

| Type | Value | Description |
|------|-------|-------------|
| Long | `POSITION_TYPE_LONG` | Profit from price increase |
| Short | `POSITION_TYPE_SHORT` | Profit from price decrease |

### Order with Take Profit and Stop Loss

```go
order := &strategy.ExecuteOrder{
    Symbol:       data.Symbol,
    Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
    OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
    Quantity:     1.0,
    Price:        data.Close,
    StrategyName: "MyStrategy",
    Reason: &strategy.Reason{
        Reason:  "strategy",
        Message: "Entry signal",
    },
    TakeProfit: &strategy.ExecuteOrderTakeProfitOrStopLoss{
        Symbol:    data.Symbol,
        Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
        OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
    },
    StopLoss: &strategy.ExecuteOrderTakeProfitOrStopLoss{
        Symbol:    data.Symbol,
        Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
        OrderType: strategy.OrderType_ORDER_TYPE_MARKET,
    },
}
```

## Using Technical Indicators

The framework provides built-in technical indicators. First configure them in `Initialize`, then use them in `ProcessData`.

### Available Indicators

| Indicator | Type Constant | Description | Docs |
|-----------|---------------|-------------|------|
| RSI | `INDICATOR_RSI` | Relative Strength Index | [Reference](indicators/rsi.md) |
| MACD | `INDICATOR_MACD` | Moving Average Convergence Divergence | [Reference](indicators/macd.md) |
| Bollinger Bands | `INDICATOR_BOLLINGER_BANDS` | Volatility bands | [Reference](indicators/bollinger-bands.md) |
| EMA | `INDICATOR_EMA` | Exponential Moving Average | [Reference](indicators/ema.md) |
| MA | `INDICATOR_MA` | Simple Moving Average | [Reference](indicators/ma.md) |
| ATR | `INDICATOR_ATR` | Average True Range | [Reference](indicators/atr.md) |
| ADX | `INDICATOR_ADX` | Average Directional Index | Not implemented |
| CCI | `INDICATOR_CCI` | Commodity Channel Index | Not implemented |
| Stochastic | `INDICATOR_STOCHASTIC_OSCILLATOR` | Stochastic Oscillator | Not implemented |
| Williams %R | `INDICATOR_WILLIAMS_R` | Williams Percent Range | Not implemented |
| Range Filter | `INDICATOR_RANGE_FILTER` | Range Filter | [Reference](indicators/range-filter.md) |
| Waddah Attar | `INDICATOR_WADDAH_ATTAR` | Waddah Attar Explosion | [Reference](indicators/waddah-attar.md) |

> **Indicator Reference**: See [docs/indicators/](indicators/) for detailed documentation on each indicator including configuration parameters, raw value outputs, signal generation logic, and usage examples.

### Configuring Indicators

Configure indicators in the `Initialize` method:

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()
    
    // Configure RSI with period 14
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
        Config:        `[14]`,  // JSON array of parameters
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure RSI: %w", err)
    }
    
    // Configure MA with period 20
    _, err = api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_MA,
        Config:        `[20]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure MA: %w", err)
    }
    
    return &emptypb.Empty{}, nil
}
```

> **Note**: Each indicator type has a single configuration. If you need multiple instances of the same indicator (e.g., two MAs with different periods), you should compute the values manually using the `GetRange` or `ExecuteSQL` API methods, or use different indicator types (e.g., `INDICATOR_MA` and `INDICATOR_EMA`).

### Getting Indicator Signals

Use `GetSignal` in `ProcessData` to get trading signals:

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()
    
    // Get RSI signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get RSI signal: %w", err)
    }
    
    // Check signal type
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // RSI indicates oversold - potential buy
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        // RSI indicates overbought - potential sell
    }
    
    // Access raw indicator value
    // signal.RawValue contains JSON with indicator-specific data
    // For RSI: {"rsi": 45.5}
    
    return &emptypb.Empty{}, nil
}
```

### Signal Types

| Type | Description |
|------|-------------|
| `SIGNAL_TYPE_BUY_LONG` | Buy signal for long position |
| `SIGNAL_TYPE_SELL_LONG` | Sell signal for long position |
| `SIGNAL_TYPE_BUY_SHORT` | Buy signal for short position |
| `SIGNAL_TYPE_SELL_SHORT` | Sell signal for short position |
| `SIGNAL_TYPE_NO_ACTION` | No trading action needed |
| `SIGNAL_TYPE_CLOSE_POSITION` | Close current position |
| `SIGNAL_TYPE_WAIT` | Wait for more confirmation |
| `SIGNAL_TYPE_ABORT` | Abort current operation |

## Creating Marks on Data

Marks are visual indicators that appear on charts. Use them to annotate buy/sell signals, important events, or debugging:

```go
api := strategy.NewStrategyApi()

_, err := api.Mark(ctx, &strategy.MarkRequest{
    MarketData: data,
    Mark: &strategy.Mark{
        SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
        Color:      "green",                              // Color name or hex code
        Shape:      strategy.MarkShape_MARK_SHAPE_CIRCLE, // CIRCLE, SQUARE, or TRIANGLE
        Title:      "Buy Signal",
        Message:    "RSI oversold condition detected",
        Category:   "MyStrategy",
    },
})
if err != nil {
    return nil, fmt.Errorf("failed to create mark: %w", err)
}
```

### Mark Shapes

| Shape | Constant |
|-------|----------|
| Circle | `MARK_SHAPE_CIRCLE` |
| Square | `MARK_SHAPE_SQUARE` |
| Triangle | `MARK_SHAPE_TRIANGLE` |

## Storing State in Cache

**Strategies are stateless** - all state must be stored in the cache. The cache persists between `ProcessData` calls.

### Setting Cache

```go
api := strategy.NewStrategyApi()

// Store simple value
_, err := api.SetCache(ctx, &strategy.SetRequest{
    Key:   "last_signal",
    Value: "buy",
})

// Store complex data as JSON
stateData := map[string]interface{}{
    "lastPrice": 150.50,
    "inPosition": true,
}
jsonBytes, _ := json.Marshal(stateData)

_, err = api.SetCache(ctx, &strategy.SetRequest{
    Key:   "strategy_state",
    Value: string(jsonBytes),
})
```

### Getting Cache

```go
api := strategy.NewStrategyApi()

// Get value
resp, err := api.GetCache(ctx, &strategy.GetRequest{Key: "last_signal"})
if err != nil {
    return nil, err
}

// Check if value exists
if resp.Value == "" {
    // No previous state, initialize
}

// Parse complex data
var stateData map[string]interface{}
if err := json.Unmarshal([]byte(resp.Value), &stateData); err != nil {
    return nil, err
}
```

### Cache Key Best Practices

- Include the symbol in cache keys for multi-symbol strategies: `"state_" + data.Symbol`
- Use descriptive prefixes: `"position_state_"`, `"signal_history_"`
- Avoid collision with other strategies by including the strategy name

## Strategy Configuration

Strategies can accept JSON configuration through the `Initialize` method.

### Defining Configuration

```go
// Config represents the strategy configuration
type Config struct {
    FastPeriod int    `yaml:"fastPeriod" jsonschema:"title=Fast Period,description=The period for the fast MA,minimum=1,default=5"`
    SlowPeriod int    `yaml:"slowPeriod" jsonschema:"title=Slow Period,description=The period for the slow MA,minimum=1,default=20"`
    Symbol     string `yaml:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
}
```

### Parsing Configuration

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    var config Config
    if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
        return nil, fmt.Errorf("failed to parse configuration: %w", err)
    }
    
    // Validate configuration
    if config.FastPeriod >= config.SlowPeriod {
        return nil, fmt.Errorf("fast period must be less than slow period")
    }
    
    s.config = config
    return &emptypb.Empty{}, nil
}
```

### Returning Configuration Schema

Use `strategy.ToJSONSchema` to generate a JSON schema from your config struct:

```go
func (s *MyStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
    schema, err := strategy.ToJSONSchema(Config{})
    if err != nil {
        return nil, fmt.Errorf("failed to generate schema: %w", err)
    }
    return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}
```

## Logging

Use the `Log` method to output messages during backtesting:

```go
api := strategy.NewStrategyApi()

// Info level log
_, _ = api.Log(ctx, &strategy.LogRequest{
    Level:   strategy.LogLevel_LOG_LEVEL_INFO,
    Message: "Processing data for " + data.Symbol,
    Fields: map[string]string{
        "price": fmt.Sprintf("%.2f", data.Close),
    },
})

// Debug level log
_, _ = api.Log(ctx, &strategy.LogRequest{
    Level:   strategy.LogLevel_LOG_LEVEL_DEBUG,
    Message: "Indicator values calculated",
})
```

### Log Levels

| Level | Constant | Use Case |
|-------|----------|----------|
| Debug | `LOG_LEVEL_DEBUG` | Detailed debugging info |
| Info | `LOG_LEVEL_INFO` | General information |
| Warn | `LOG_LEVEL_WARN` | Warning conditions |
| Error | `LOG_LEVEL_ERROR` | Error conditions |

## Getting Account and Position Info

### Account Information

```go
api := strategy.NewStrategyApi()

accountInfo, err := api.GetAccountInfo(ctx, &emptypb.Empty{})
if err != nil {
    return nil, err
}

fmt.Printf("Balance: %.2f\n", accountInfo.Balance)
fmt.Printf("Equity: %.2f\n", accountInfo.Equity)
fmt.Printf("Buying Power: %.2f\n", accountInfo.BuyingPower)
fmt.Printf("Unrealized P&L: %.2f\n", accountInfo.UnrealizedPnl)
```

### Positions

```go
api := strategy.NewStrategyApi()

// Get all positions
positions, err := api.GetPositions(ctx, &emptypb.Empty{})
if err != nil {
    return nil, err
}

for _, pos := range positions.Positions {
    fmt.Printf("Symbol: %s, Quantity: %.2f\n", pos.Symbol, pos.Quantity)
}

// Get specific position
position, err := api.GetPosition(ctx, &strategy.GetPositionRequest{
    Symbol: "AAPL",
})
```

## Building and Compiling Strategies

Strategies must be compiled to WebAssembly (WASM):

### Makefile

Create a `Makefile` in your strategy directory:

```makefile
.PHONY: clean build

clean:
	rm -f *.wasm

build:
	GOOS=wasip1 GOARCH=wasm go build -o strategy.wasm -buildmode=c-shared \
		my_strategy.go
```

### Build Command

```bash
cd my-strategy
make build
```

### Running Backtest

```bash
go run cmd/backtest/main.go \
    -strategy-wasm ./my-strategy/strategy.wasm \
    -config ./config/backtest-engine-v1-config.yaml \
    -data "./data/*.parquet"
```

## Complete Example: RSI Strategy

Here's a complete strategy that uses the RSI indicator to detect overbought and oversold conditions:

```go
//go:build wasip1

package main

import (
    "context"
    "encoding/json"
    "fmt"

    "github.com/knqyf263/go-plugin/types/known/emptypb"
    "github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type RSIStrategy struct {
    config Config
}

type Config struct {
    Period         int     `yaml:"period" jsonschema:"title=RSI Period,default=14"`
    OverboughtLevel float64 `yaml:"overboughtLevel" jsonschema:"title=Overbought Level,default=70"`
    OversoldLevel   float64 `yaml:"oversoldLevel" jsonschema:"title=Oversold Level,default=30"`
}

type rsiValue struct {
    RSI float64 `json:"rsi"`
}

func main() {}

func init() {
    strategy.RegisterTradingStrategy(&RSIStrategy{})
}

func (s *RSIStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    // Parse config with defaults
    s.config = Config{Period: 14, OverboughtLevel: 70, OversoldLevel: 30}
    if req.Config != "" {
        json.Unmarshal([]byte(req.Config), &s.config)
    }
    
    api := strategy.NewStrategyApi()
    
    // Configure RSI indicator
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
        Config:        fmt.Sprintf("[%d]", s.config.Period),
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure RSI: %w", err)
    }
    
    return &emptypb.Empty{}, nil
}

func (s *RSIStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()
    cacheKey := "rsi_state_" + data.Symbol
    
    // Get RSI signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get RSI signal: %w", err)
    }
    
    // Parse RSI value
    var rsi rsiValue
    if err := json.Unmarshal([]byte(signal.RawValue), &rsi); err != nil {
        return nil, fmt.Errorf("failed to parse RSI value: %w", err)
    }
    
    // Get previous state to track position
    prevState, _ := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})
    inPosition := prevState.Value == "in_position"
    
    // Check for trading signals
    if rsi.RSI < s.config.OversoldLevel && !inPosition {
        // RSI oversold - buy signal
        _, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
            Symbol:       data.Symbol,
            Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
            OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
            Quantity:     1.0,
            Price:        data.Close,
            StrategyName: "RSIStrategy",
            Reason:       &strategy.Reason{Reason: "strategy", Message: fmt.Sprintf("RSI oversold: %.2f", rsi.RSI)},
        })
        if err != nil {
            return nil, err
        }
        
        // Mark buy signal
        _, _ = api.Mark(ctx, &strategy.MarkRequest{
            MarketData: data,
            Mark: &strategy.Mark{
                SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
                Color:      "green",
                Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
                Title:      "Buy",
                Message:    fmt.Sprintf("RSI oversold: %.2f", rsi.RSI),
                Category:   "RSIStrategy",
            },
        })
        
        // Update state
        _, _ = api.SetCache(ctx, &strategy.SetRequest{Key: cacheKey, Value: "in_position"})
        
    } else if rsi.RSI > s.config.OverboughtLevel && inPosition {
        // RSI overbought - sell signal
        _, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
            Symbol:       data.Symbol,
            Side:         strategy.PurchaseType_PURCHASE_TYPE_SELL,
            OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
            Quantity:     1.0,
            Price:        data.Close,
            StrategyName: "RSIStrategy",
            Reason:       &strategy.Reason{Reason: "strategy", Message: fmt.Sprintf("RSI overbought: %.2f", rsi.RSI)},
        })
        if err != nil {
            return nil, err
        }
        
        // Mark sell signal
        _, _ = api.Mark(ctx, &strategy.MarkRequest{
            MarketData: data,
            Mark: &strategy.Mark{
                SignalType: strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
                Color:      "red",
                Shape:      strategy.MarkShape_MARK_SHAPE_TRIANGLE,
                Title:      "Sell",
                Message:    fmt.Sprintf("RSI overbought: %.2f", rsi.RSI),
                Category:   "RSIStrategy",
            },
        })
        
        // Update state
        _, _ = api.SetCache(ctx, &strategy.SetRequest{Key: cacheKey, Value: ""})
    }
    
    return &emptypb.Empty{}, nil
}

func (s *RSIStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
    return &strategy.NameResponse{Name: "RSIStrategy"}, nil
}

func (s *RSIStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
    schema, _ := strategy.ToJSONSchema(Config{})
    return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

func (s *RSIStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
    return &strategy.GetDescriptionResponse{
        Description: "RSI strategy that buys on oversold and sells on overbought conditions",
    }, nil
}
```

## Best Practices

1. **Stateless Design**: Never store state in struct fields between `ProcessData` calls - always use cache
2. **Error Handling**: Always handle errors from API calls gracefully
3. **Cache Keys**: Use unique, descriptive cache keys including symbol names
4. **Configuration Validation**: Validate all configuration parameters in `Initialize`
5. **Logging**: Use appropriate log levels for debugging and monitoring
6. **Marks**: Add marks for significant events to help visualize strategy behavior
7. **Order Reasons**: Always provide meaningful reasons for orders
8. **Position Management**: Check positions before placing orders to avoid over-leveraging

## Additional Resources

- [Example Strategies](../examples/strategy/)
- [API Reference](../pkg/strategy/strategy.proto)
- [Indicator Reference](./indicators/) - Detailed documentation for all indicators
- [Release Process](./release-process.md)

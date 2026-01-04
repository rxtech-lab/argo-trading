---
title: MACD (Moving Average Convergence Divergence)
description: MACD indicator - configuration, raw values, signal generation, and usage examples
---

# MACD (Moving Average Convergence Divergence)

The Moving Average Convergence Divergence (MACD) is a trend-following momentum indicator that shows the relationship between two exponential moving averages of a security's price.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `fastPeriod` | int | 12 | The period for the fast EMA |
| `slowPeriod` | int | 26 | The period for the slow EMA |
| `signalPeriod` | int | 9 | The period for the signal line EMA |

### Config Format

```json
[fastPeriod, slowPeriod, signalPeriod]
```

**Example:**
```json
[12, 26, 9]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `macd` | float64 | The MACD line value (fast EMA - slow EMA) |

**Example:**
```json
{"macd": 2.35}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| MACD > 0 | `SIGNAL_TYPE_BUY_LONG` | Bullish momentum |
| MACD < 0 | `SIGNAL_TYPE_SELL_SHORT` | Bearish momentum |
| MACD = 0 | `SIGNAL_TYPE_NO_ACTION` | Neutral |

## Calculation Method

1. **Fast EMA**: Calculate EMA with fast period (default 12)
2. **Slow EMA**: Calculate EMA with slow period (default 26)
3. **MACD Line**: Fast EMA - Slow EMA
4. **Signal Line**: EMA of MACD line with signal period (default 9)
5. **Histogram**: MACD Line - Signal Line

The indicator uses the EMA indicator internally for calculations.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure MACD with default parameters
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_MACD,
        Config:        `[12, 26, 9]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure MACD: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting MACD Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get MACD signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_MACD,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get MACD signal: %w", err)
    }

    // Parse raw value
    var macdValue struct {
        MACD float64 `json:"macd"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &macdValue); err != nil {
        return nil, fmt.Errorf("failed to parse MACD value: %w", err)
    }

    // Use MACD signal and value
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // MACD positive - bullish momentum
        fmt.Printf("MACD bullish: %.4f\n", macdValue.MACD)
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        // MACD negative - bearish momentum
        fmt.Printf("MACD bearish: %.4f\n", macdValue.MACD)
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Zero Line Crossover**: MACD crossing above/below zero indicates momentum change
2. **Signal Line Crossover**: MACD crossing above/below signal line for entry/exit
3. **Divergence**: Price making new highs/lows while MACD doesn't - reversal signal
4. **Histogram Analysis**: Increasing/decreasing histogram shows momentum strength

## Interpretation Guide

| Signal | Interpretation |
|--------|----------------|
| MACD crosses above 0 | Bullish - fast EMA > slow EMA |
| MACD crosses below 0 | Bearish - fast EMA < slow EMA |
| MACD crosses above signal line | Bullish entry signal |
| MACD crosses below signal line | Bearish exit signal |
| Rising histogram | Increasing bullish momentum |
| Falling histogram | Increasing bearish momentum |

## Common Parameter Sets

| Style | Fast | Slow | Signal | Use Case |
|-------|------|------|--------|----------|
| Standard | 12 | 26 | 9 | General purpose |
| Fast | 8 | 17 | 9 | More responsive, short-term |
| Slow | 19 | 39 | 9 | Less noise, long-term |

## Related Indicators

- [EMA](ema.md) - Component of MACD calculation
- [RSI](rsi.md) - Another momentum indicator
- [Waddah Attar](waddah-attar.md) - Uses MACD internally

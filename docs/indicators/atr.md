---
title: ATR (Average True Range)
description: Average True Range volatility indicator - configuration, raw values, and usage examples
---

# ATR (Average True Range)

The Average True Range (ATR) is a volatility indicator that measures market volatility by decomposing the entire range of an asset price for a given period.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 14 | The number of periods for the ATR calculation |

### Config Format

```json
[period]
```

**Example:**
```json
[14]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `atr` | float64 | The calculated ATR value (volatility measure in price units) |

**Example:**
```json
{"atr": 2.45}
```

## Signal Generation

The ATR indicator returns `SIGNAL_TYPE_NO_ACTION` by default. ATR is a volatility indicator and does not generate buy/sell signals directly. It is used to:
- Set stop-loss distances
- Determine position sizes
- Filter trades based on volatility
- Identify volatility expansions/contractions

## Calculation Method

1. **True Range (TR)** is the greatest of:
   - Current High - Current Low
   - |Current High - Previous Close|
   - |Current Low - Previous Close|

2. **ATR** is the exponential moving average of True Range values over the specified period

The implementation uses EMA for smoothing the true range values.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure ATR with period 14
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_ATR,
        Config:        `[14]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure ATR: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting ATR Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get ATR signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_ATR,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get ATR signal: %w", err)
    }

    // Parse raw value
    var atrValue struct {
        ATR float64 `json:"atr"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &atrValue); err != nil {
        return nil, fmt.Errorf("failed to parse ATR value: %w", err)
    }

    // Use ATR for risk management
    stopLossDistance := atrValue.ATR * 2  // 2x ATR stop loss
    stopLossPrice := data.Close - stopLossDistance

    // Use ATR for position sizing
    riskPerTrade := 1000.0  // $1000 risk per trade
    positionSize := riskPerTrade / atrValue.ATR

    fmt.Printf("ATR: %.4f, Stop Loss: %.2f, Position Size: %.2f\n",
        atrValue.ATR, stopLossPrice, positionSize)

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Stop-Loss Placement**: Set stops at 1.5x-3x ATR from entry
2. **Position Sizing**: Risk fixed amount per trade based on ATR
3. **Volatility Filter**: Only trade when ATR is above/below threshold
4. **Trailing Stops**: Use ATR-based trailing stops (e.g., Chandelier Exit)
5. **Breakout Confirmation**: Large ATR expansion confirms breakout validity

## Interpretation Guide

| ATR Behavior | Interpretation |
|--------------|----------------|
| Rising ATR | Increasing volatility, stronger moves |
| Falling ATR | Decreasing volatility, consolidation |
| High ATR | Wide price swings, need wider stops |
| Low ATR | Tight trading range, potential breakout coming |

## ATR-Based Risk Management

### Stop-Loss Multipliers

| Multiplier | Use Case |
|------------|----------|
| 1.0x ATR | Tight stop, high chance of being stopped out |
| 1.5x ATR | Conservative, suits trending markets |
| 2.0x ATR | Standard, balances risk and reward |
| 3.0x ATR | Wide stop, suits volatile markets |

### Position Sizing Formula

```
Position Size = Risk Amount / (ATR * Multiplier)
```

**Example:**
- Risk per trade: $500
- ATR: 2.50
- Multiplier: 2
- Position Size = $500 / (2.50 * 2) = 100 shares

## Related Indicators

- [Bollinger Bands](bollinger-bands.md) - Another volatility indicator
- [Range Filter](range-filter.md) - Uses ATR-like calculations
- [Waddah Attar](waddah-attar.md) - Uses ATR internally

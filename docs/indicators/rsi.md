---
title: RSI (Relative Strength Index)
description: Relative Strength Index indicator - configuration, raw values, signal generation, and usage examples
---

# RSI (Relative Strength Index)

The Relative Strength Index (RSI) is a momentum oscillator that measures the speed and magnitude of price changes. It oscillates between 0 and 100 and is used to identify overbought or oversold conditions.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 14 | The number of periods for RSI calculation |
| `rsiLowerThreshold` | float64 | 30 | Threshold below which the asset is considered oversold |
| `rsiUpperThreshold` | float64 | 70 | Threshold above which the asset is considered overbought |

### Config Format

```json
[period, rsiLowerThreshold?, rsiUpperThreshold?]
```

**Examples:**
```json
[14]           // Just period, uses default thresholds (30, 70)
[14, 30, 70]   // Period with custom thresholds
[21, 25, 75]   // Custom period and tighter thresholds
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `rsi` | float64 | The calculated RSI value (0-100 scale) |

**Example:**
```json
{"rsi": 45.5}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| RSI < lower threshold | `SIGNAL_TYPE_BUY_LONG` | Asset is oversold, potential buy opportunity |
| RSI > upper threshold | `SIGNAL_TYPE_SELL_SHORT` | Asset is overbought, potential sell opportunity |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | RSI is in neutral zone |

## Calculation Method

1. Calculate price changes between consecutive periods
2. Separate gains (positive changes) and losses (negative changes)
3. Calculate average gain and average loss over the period
4. **Relative Strength (RS)** = Average Gain / Average Loss
5. **RSI** = 100 - (100 / (1 + RS))

The implementation uses exponential smoothing for the averages after the initial calculation.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure RSI with custom thresholds
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
        Config:        `[14, 30, 70]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure RSI: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting RSI Signal

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

    // Parse raw value
    var rsiValue struct {
        RSI float64 `json:"rsi"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &rsiValue); err != nil {
        return nil, fmt.Errorf("failed to parse RSI value: %w", err)
    }

    // Use RSI signal and value
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // RSI indicates oversold - consider buying
        fmt.Printf("RSI oversold: %.2f\n", rsiValue.RSI)
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        // RSI indicates overbought - consider selling
        fmt.Printf("RSI overbought: %.2f\n", rsiValue.RSI)
    default:
        // Neutral zone
        fmt.Printf("RSI neutral: %.2f\n", rsiValue.RSI)
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Overbought/Oversold**: Traditional use - buy when RSI < 30, sell when RSI > 70
2. **Divergence**: When price makes new highs/lows but RSI doesn't, indicating potential reversal
3. **Centerline Crossover**: RSI crossing above/below 50 as trend confirmation
4. **Failure Swings**: RSI pattern where it fails to make new high/low before reversing

## Interpretation Guide

| RSI Range | Interpretation |
|-----------|----------------|
| 0-30 | Oversold - potential buying opportunity |
| 30-50 | Bearish momentum, but not oversold |
| 50 | Neutral point |
| 50-70 | Bullish momentum, but not overbought |
| 70-100 | Overbought - potential selling opportunity |

## Related Indicators

- [MACD](macd.md) - Another momentum indicator
- [Stochastic Oscillator](stochastic.md) - Similar oscillator concept (not implemented)
- [Williams %R](williams-r.md) - Related momentum oscillator (not implemented)

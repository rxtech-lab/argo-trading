---
title: PSY (Psychological Line)
description: Psychological Line sentiment indicator - configuration, raw values, signal generation, and usage examples
---

# PSY (Psychological Line)

The Psychological Line (PSY) is a sentiment indicator that measures the percentage of periods within a lookback window that closed higher than the previous period. It reflects the buying vs. selling pressure on a percentage scale from 0 to 100 and is often used to identify overbought and oversold market conditions driven by trader psychology.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 12 | The number of periods used to count up-days |
| `upperThreshold` | float64 | 75 | Threshold above which the market is considered overbought |
| `lowerThreshold` | float64 | 25 | Threshold below which the market is considered oversold |

### Config Format

```json
[period, upperThreshold?, lowerThreshold?]
```

**Examples:**
```json
[12]            // Just period, uses default thresholds (75, 25)
[12, 75, 25]    // Period with standard thresholds
[24, 80, 20]    // Longer period with tighter thresholds
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `psy` | float64 | The calculated PSY value (0-100 scale, percent of up days) |

**Example:**
```json
{"psy": 58.33}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| PSY < lower threshold | `SIGNAL_TYPE_BUY_LONG` | Market is oversold, potential buy opportunity |
| PSY > upper threshold | `SIGNAL_TYPE_SELL_SHORT` | Market is overbought, potential sell opportunity |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | PSY is in neutral zone |

## Calculation Method

For a lookback `period`:

1. Look at the last `period + 1` closing prices.
2. Count the number of periods where the close is strictly greater than the previous period's close (an "up day").
3. **PSY** = (up-day count / period) * 100

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_PSY,
        Config:        `[12, 75, 25]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure PSY: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting the PSY Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_PSY,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get PSY signal: %w", err)
    }

    var raw struct {
        PSY float64 `json:"psy"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &raw); err != nil {
        return nil, fmt.Errorf("failed to parse PSY value: %w", err)
    }

    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        fmt.Printf("PSY oversold: %.2f\n", raw.PSY)
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        fmt.Printf("PSY overbought: %.2f\n", raw.PSY)
    default:
        fmt.Printf("PSY neutral: %.2f\n", raw.PSY)
    }

    return &emptypb.Empty{}, nil
}
```

## Interpretation Guide

| PSY Range | Interpretation |
|-----------|----------------|
| 0-25 | Oversold - sustained selling pressure |
| 25-50 | Bearish sentiment, but not oversold |
| 50 | Balanced sentiment |
| 50-75 | Bullish sentiment, but not overbought |
| 75-100 | Overbought - sustained buying pressure |

## Common Use Cases

1. **Overbought/Oversold**: Buy when PSY falls below the lower threshold, sell when PSY rises above the upper threshold.
2. **Sentiment Confirmation**: Use PSY to confirm signals from price-based indicators like RSI or Williams %R.
3. **Divergence**: When price makes new highs/lows but PSY does not, signaling potential trend exhaustion.

## Related Indicators

- [RSI](rsi.md) - Price-change momentum oscillator
- [Williams %R](williams-r.md) - Close vs. high/low range oscillator

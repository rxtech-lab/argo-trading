---
slug: indicators/williams-r
title: Williams %R (WR)
description: Williams %R momentum oscillator - configuration, raw values, signal generation, and usage examples
---
# Williams %R (WR)

Williams %R is a momentum oscillator, developed by Larry Williams, that measures the level of the current close relative to the highest high over a lookback period. It oscillates between -100 and 0 and is used to identify overbought and oversold conditions.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 14 | Number of periods used to compute the highest high and lowest low |
| `overboughtThreshold` | float64 | -20 | Threshold above which the market is considered overbought |
| `oversoldThreshold` | float64 | -80 | Threshold below which the market is considered oversold |

### Config Format

```json
[period, overboughtThreshold?, oversoldThreshold?]
```

**Examples:**
```json
[14]              // Just period, uses default thresholds (-20, -80)
[14, -20, -80]    // Period with standard thresholds
[21, -10, -90]    // Longer period with tighter thresholds
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `wr` | float64 | The calculated Williams %R value (-100 to 0 scale) |

**Example:**
```json
{"wr": -55.25}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| WR < oversold threshold | `SIGNAL_TYPE_BUY_LONG` | Market is oversold, potential buy opportunity |
| WR > overbought threshold | `SIGNAL_TYPE_SELL_SHORT` | Market is overbought, potential sell opportunity |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | WR is in neutral zone |

## Calculation Method

For a lookback `period`:

1. **Highest High (HH)** = maximum of `high` over the last `period` bars
2. **Lowest Low (LL)** = minimum of `low` over the last `period` bars
3. **Williams %R** = ((HH - Close) / (HH - LL)) * -100

When the range `HH - LL` is zero (flat market), Williams %R is reported as 0.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_WILLIAMS_R,
        Config:        `[14, -20, -80]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure Williams %%R: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting the Williams %R Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_WILLIAMS_R,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get Williams %%R signal: %w", err)
    }

    var raw struct {
        WR float64 `json:"wr"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &raw); err != nil {
        return nil, fmt.Errorf("failed to parse Williams %%R value: %w", err)
    }

    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        fmt.Printf("Williams %%R oversold: %.2f\n", raw.WR)
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        fmt.Printf("Williams %%R overbought: %.2f\n", raw.WR)
    default:
        fmt.Printf("Williams %%R neutral: %.2f\n", raw.WR)
    }

    return &emptypb.Empty{}, nil
}
```

## Interpretation Guide

| WR Range | Interpretation |
|----------|----------------|
| -100 to -80 | Oversold - potential buying opportunity |
| -80 to -50 | Bearish momentum, but not oversold |
| -50 | Neutral point |
| -50 to -20 | Bullish momentum, but not overbought |
| -20 to 0 | Overbought - potential selling opportunity |

## Related Indicators

- [RSI](rsi.md) - Another momentum oscillator (0 to 100 scale)
- [MACD](macd.md) - Trend-following momentum indicator
- [PSY](psy.md) - Psychological Line, sentiment oscillator based on up-days

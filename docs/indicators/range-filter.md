---
title: Range Filter
description: Range Filter trend-following indicator - configuration, raw values, signal generation, and usage examples
---

# Range Filter

The Range Filter is a trend-following indicator that filters out market noise by using a dynamic range calculation. It helps identify trend direction and generates signals when price breaks through the filter levels.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 100 | The number of periods for range calculation |
| `multiplier` | float64 | 3.0 | Multiplier for the smoothed range |

### Config Format

```json
[period, multiplier]
```

**Example:**
```json
[100, 3.0]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `filter` | float64 | The current filter value (trend line) |
| `smooth_range` | float64 | The smoothed average range |
| `upward_count` | float64 | Count of consecutive upward trend bars |
| `downward_count` | float64 | Count of consecutive downward trend bars |

**Example:**
```json
{
  "filter": 150.25,
  "smooth_range": 2.15,
  "upward_count": 5,
  "downward_count": 0
}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| Upward trend detected | `SIGNAL_TYPE_BUY_LONG` | Price breaking above filter |
| Downward trend detected | `SIGNAL_TYPE_SELL_SHORT` | Price breaking below filter |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | No clear trend |

## Calculation Method

1. **Smooth Range**: Calculate range (high - low) and smooth it using EMA
2. **Filter Calculation**:
   - If price > filter + smooth_range: Filter moves up
   - If price < filter - smooth_range: Filter moves down
   - Otherwise: Filter stays at previous level
3. **Trend Direction**:
   - Upward: Filter is increasing
   - Downward: Filter is decreasing

The indicator maintains state in cache to track filter values and trend counts across data points.

### Dependencies
- Uses EMA indicator internally (short EMA with period, long EMA with period*2-1)

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure Range Filter
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RANGE_FILTER,
        Config:        `[100, 3.0]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure Range Filter: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting Range Filter Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get Range Filter signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_RANGE_FILTER,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get Range Filter signal: %w", err)
    }

    // Parse raw value
    var rfValue struct {
        Filter        float64 `json:"filter"`
        SmoothRange   float64 `json:"smooth_range"`
        UpwardCount   float64 `json:"upward_count"`
        DownwardCount float64 `json:"downward_count"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &rfValue); err != nil {
        return nil, fmt.Errorf("failed to parse Range Filter value: %w", err)
    }

    fmt.Printf("Filter: %.2f, Range: %.2f, Up: %.0f, Down: %.0f\n",
        rfValue.Filter, rfValue.SmoothRange,
        rfValue.UpwardCount, rfValue.DownwardCount)

    // Use signal type for trading decisions
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // Upward trend - consider buying
        if rfValue.UpwardCount >= 3 {
            // Strong upward trend confirmed
        }
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        // Downward trend - consider selling
        if rfValue.DownwardCount >= 3 {
            // Strong downward trend confirmed
        }
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Trend Following**: Enter positions in the direction of the filter
2. **Noise Filtering**: Avoid false signals during choppy markets
3. **Stop-Loss Placement**: Use filter level as dynamic stop-loss
4. **Trend Strength**: Use upward/downward counts to gauge trend strength

## Interpretation Guide

| Indicator State | Interpretation |
|-----------------|----------------|
| Filter rising, high upward count | Strong uptrend |
| Filter falling, high downward count | Strong downtrend |
| Filter flat | Consolidation / ranging market |
| Price above filter | Bullish bias |
| Price below filter | Bearish bias |

## Parameter Tuning

| Parameter | Effect of Increasing | Effect of Decreasing |
|-----------|---------------------|---------------------|
| Period | Smoother filter, fewer signals | More responsive, more signals |
| Multiplier | Wider range, fewer breakouts | Tighter range, more breakouts |

### Suggested Settings

| Market Type | Period | Multiplier | Description |
|-------------|--------|------------|-------------|
| Trending | 100 | 3.0 | Standard settings |
| Volatile | 50 | 4.0 | More responsive, wider range |
| Calm | 150 | 2.5 | Smoother filter |

## Related Indicators

- [EMA](ema.md) - Used internally for calculations
- [ATR](atr.md) - Similar volatility-based approach
- [Bollinger Bands](bollinger-bands.md) - Another range-based indicator

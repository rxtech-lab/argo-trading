---
title: EMA (Exponential Moving Average)
description: Exponential Moving Average indicator - configuration, raw values, and usage examples
---

# EMA (Exponential Moving Average)

The Exponential Moving Average (EMA) is a type of moving average that places greater weight on recent data points, making it more responsive to new information compared to a Simple Moving Average (SMA).

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 20 | The number of periods to use for the EMA calculation |

### Config Format

```json
[period]
```

**Example:**
```json
[20]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `ema` | float64 | The calculated EMA value |

**Example:**
```json
{"ema": 150.25}
```

## Signal Generation

The EMA indicator returns `SIGNAL_TYPE_NO_ACTION` by default. It is primarily used as a calculation indicator for other strategies or indicators rather than generating trading signals directly.

## Calculation Method

1. **Initial EMA**: Uses Simple Moving Average (SMA) for the first EMA value
2. **Multiplier (Alpha)**: `2 / (period + 1)` - matches pandas ewm implementation with `adjust=False`
3. **EMA Formula**: `EMA = price * alpha + EMA_prev * (1 - alpha)`

If there are fewer data points than the specified period, the indicator falls back to calculating a simple average of available data.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure EMA with period 20
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_EMA,
        Config:        `[20]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure EMA: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting EMA Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get EMA signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_EMA,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get EMA signal: %w", err)
    }

    // Parse raw value
    var emaValue struct {
        EMA float64 `json:"ema"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &emaValue); err != nil {
        return nil, fmt.Errorf("failed to parse EMA value: %w", err)
    }

    // Use EMA value in your strategy logic
    if data.Close > emaValue.EMA {
        // Price is above EMA - potential uptrend
    } else {
        // Price is below EMA - potential downtrend
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Trend Identification**: Price above EMA indicates uptrend, below indicates downtrend
2. **Support/Resistance**: EMA can act as dynamic support in uptrends or resistance in downtrends
3. **Crossover Strategies**: Combined with other EMAs or MAs for crossover signals
4. **Component for Other Indicators**: Used internally by MACD, ATR, Range Filter, etc.

## Related Indicators

- [MA (Simple Moving Average)](ma.md) - Simpler alternative with equal weighting
- [MACD](macd.md) - Uses EMA internally for calculations
- [Bollinger Bands](bollinger-bands.md) - Uses SMA as the middle band

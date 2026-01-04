---
title: MA (Simple Moving Average)
description: Simple Moving Average indicator - configuration, raw values, and usage examples
---

# MA (Simple Moving Average)

The Simple Moving Average (MA/SMA) calculates the arithmetic mean of a given set of prices over a specified number of periods. It provides equal weight to all data points in the calculation.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 20 | The number of periods to use for the MA calculation |

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
| `ma` | float64 | The calculated simple moving average value |

**Example:**
```json
{"ma": 148.75}
```

## Signal Generation

The MA indicator returns `SIGNAL_TYPE_NO_ACTION` by default. It is primarily used as a calculation indicator for other strategies or indicators rather than generating trading signals directly.

## Calculation Method

```
MA = (P1 + P2 + ... + Pn) / n
```

Where:
- `P1, P2, ... Pn` are the closing prices for the last n periods
- `n` is the period

If there are fewer data points than the specified period, the indicator calculates the average of all available data.

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure MA with period 50
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_MA,
        Config:        `[50]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure MA: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting MA Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get MA signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_MA,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get MA signal: %w", err)
    }

    // Parse raw value
    var maValue struct {
        MA float64 `json:"ma"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &maValue); err != nil {
        return nil, fmt.Errorf("failed to parse MA value: %w", err)
    }

    // Use MA value in your strategy logic
    if data.Close > maValue.MA {
        // Price is above MA - potential bullish signal
    } else {
        // Price is below MA - potential bearish signal
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Trend Identification**: Price above MA indicates uptrend, below indicates downtrend
2. **Support/Resistance Levels**: MA often acts as dynamic support or resistance
3. **Golden/Death Cross**: 50-day MA crossing 200-day MA for long-term trend signals
4. **Mean Reversion**: Trading when price deviates significantly from MA

## MA vs EMA

| Aspect | MA (Simple) | EMA (Exponential) |
|--------|-------------|-------------------|
| Weighting | Equal weight to all periods | More weight to recent data |
| Responsiveness | Slower to react | Faster to react |
| Lag | More lag | Less lag |
| Noise | Smoother, less noise | More responsive, potentially more noise |
| Use Case | Long-term trends | Short-term trends, faster signals |

## Related Indicators

- [EMA (Exponential Moving Average)](ema.md) - More responsive alternative
- [Bollinger Bands](bollinger-bands.md) - Uses SMA as the middle band
- [MACD](macd.md) - Uses EMA for calculations

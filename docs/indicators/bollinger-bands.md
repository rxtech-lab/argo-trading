---
title: Bollinger Bands
description: Bollinger Bands volatility indicator - configuration, raw values, signal generation, and usage examples
---

# Bollinger Bands

Bollinger Bands are a volatility indicator consisting of a middle band (simple moving average) and two outer bands set at standard deviations above and below the middle band.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `period` | int | 20 | The number of periods for the SMA calculation |
| `stdDev` | float64 | 2.0 | Number of standard deviations for the bands |
| `lookback` | duration | "24h" | Lookback period for data retrieval |

### Config Format

```json
[period, stdDev, lookback]
```

**Example:**
```json
[20, 2.0, "24h"]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `upper` | float64 | Upper band (middle + stdDev * standard_deviation) |
| `middle` | float64 | Middle band (Simple Moving Average) |
| `lower` | float64 | Lower band (middle - stdDev * standard_deviation) |

**Example:**
```json
{"upper": 155.50, "middle": 150.00, "lower": 144.50}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| Price < Lower Band | `SIGNAL_TYPE_BUY_LONG` | Price below lower band - oversold |
| Price > Upper Band | `SIGNAL_TYPE_SELL_LONG` | Price above upper band - overbought |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | Price within bands |

If insufficient data (fewer than period points), returns `SIGNAL_TYPE_NO_ACTION`.

## Calculation Method

1. **Middle Band (SMA)**: Sum of closing prices / period
2. **Standard Deviation**: Calculate standard deviation of closing prices
3. **Upper Band**: Middle Band + (stdDev multiplier * Standard Deviation)
4. **Lower Band**: Middle Band - (stdDev multiplier * Standard Deviation)

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure Bollinger Bands
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_BOLLINGER_BANDS,
        Config:        `[20, 2.0, "24h"]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure Bollinger Bands: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting Bollinger Bands Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get Bollinger Bands signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_BOLLINGER_BANDS,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get Bollinger Bands signal: %w", err)
    }

    // Parse raw value
    var bbValue struct {
        Upper  float64 `json:"upper"`
        Middle float64 `json:"middle"`
        Lower  float64 `json:"lower"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &bbValue); err != nil {
        return nil, fmt.Errorf("failed to parse Bollinger Bands value: %w", err)
    }

    // Calculate bandwidth and %B
    bandwidth := (bbValue.Upper - bbValue.Lower) / bbValue.Middle * 100
    percentB := (data.Close - bbValue.Lower) / (bbValue.Upper - bbValue.Lower)

    fmt.Printf("BB Upper: %.2f, Middle: %.2f, Lower: %.2f\n",
        bbValue.Upper, bbValue.Middle, bbValue.Lower)
    fmt.Printf("Bandwidth: %.2f%%, %%B: %.2f\n", bandwidth, percentB)

    // Use signal type
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // Price below lower band - potential buy
    case strategy.SignalType_SIGNAL_TYPE_SELL_LONG:
        // Price above upper band - potential sell
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Mean Reversion**: Buy at lower band, sell at upper band
2. **Breakout Trading**: Price breaking above upper band with high volume signals breakout
3. **Squeeze Strategy**: Narrow bands (low volatility) often precede large price moves
4. **Trend Following**: Riding the upper band in uptrends, lower band in downtrends

## Interpretation Guide

| Condition | Interpretation |
|-----------|----------------|
| Price at upper band | Potentially overbought / strong uptrend |
| Price at lower band | Potentially oversold / strong downtrend |
| Bands widening | Increasing volatility |
| Bands narrowing (squeeze) | Decreasing volatility, potential breakout coming |
| Price bouncing off middle band | Middle band acting as support/resistance |

## Additional Metrics

You can calculate additional metrics from the raw values:

| Metric | Formula | Description |
|--------|---------|-------------|
| Bandwidth | (Upper - Lower) / Middle * 100 | Measures volatility as percentage |
| %B | (Price - Lower) / (Upper - Lower) | Where price is within the bands (0-1 range) |

## Common Parameter Sets

| Style | Period | Std Dev | Use Case |
|-------|--------|---------|----------|
| Standard | 20 | 2.0 | General purpose |
| Tight | 20 | 1.5 | More frequent signals |
| Wide | 20 | 2.5 | Fewer, stronger signals |
| Short-term | 10 | 1.5 | Day trading |
| Long-term | 50 | 2.0 | Position trading |

## Related Indicators

- [MA](ma.md) - Middle band is a simple moving average
- [ATR](atr.md) - Another volatility indicator
- [Range Filter](range-filter.md) - Alternative volatility-based indicator

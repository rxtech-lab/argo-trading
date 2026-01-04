---
title: Waddah Attar Explosion
description: Waddah Attar Explosion momentum/volatility indicator - configuration, raw values, signal generation, and usage examples
---

# Waddah Attar Explosion

The Waddah Attar Explosion is a momentum/volatility indicator that combines MACD and ATR to identify explosive price movements and trend direction.

## Configuration

| Parameter | Type | Default | Description |
|-----------|------|---------|-------------|
| `fastPeriod` | int | 20 | Fast EMA period for MACD calculation |
| `slowPeriod` | int | 40 | Slow EMA period for MACD calculation |
| `signalPeriod` | int | 9 | Signal line EMA period |
| `atrPeriod` | int | 14 | Period for ATR calculation |
| `multiplier` | float64 | 150.0 | Multiplier for trend and explosion values |

### Config Format

```json
[fastPeriod, slowPeriod, signalPeriod, atrPeriod, multiplier]
```

**Example:**
```json
[20, 40, 9, 14, 150.0]
```

## Raw Value Output

The signal's `RawValue` field contains a JSON object with the following keys:

| Key | Type | Description |
|-----|------|-------------|
| `macd` | float64 | MACD line value |
| `signal` | float64 | Signal line value |
| `histogram` | float64 | MACD histogram (macd - signal) |
| `atr` | float64 | ATR value |
| `trend` | float64 | Trend value (macd * multiplier) |
| `explosion` | float64 | Explosion value (atr * multiplier) |

**Example:**
```json
{
  "macd": 0.015,
  "signal": 0.012,
  "histogram": 0.003,
  "atr": 2.45,
  "trend": 2.25,
  "explosion": 367.5
}
```

## Signal Generation

| Condition | Signal Type | Description |
|-----------|-------------|-------------|
| explosion > 0 AND trend > 0 | `SIGNAL_TYPE_BUY_LONG` | Bullish explosion |
| explosion > 0 AND trend < 0 | `SIGNAL_TYPE_SELL_SHORT` | Bearish explosion |
| Otherwise | `SIGNAL_TYPE_NO_ACTION` | No explosion or trend |

## Calculation Method

1. **MACD Calculation**: Uses fast and slow EMAs to calculate MACD line
2. **Signal Line**: EMA of MACD line
3. **Histogram**: MACD - Signal
4. **ATR Calculation**: Average True Range over specified period
5. **Trend**: MACD * multiplier (determines direction)
6. **Explosion**: ATR * multiplier (determines volatility threshold)

The indicator maintains state in cache to track MACD, signal, histogram, and ATR values.

### Dependencies
- Uses MACD indicator internally
- Uses ATR indicator internally

## Usage Example

### Configuring the Indicator

```go
func (s *MyStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
    api := strategy.NewStrategyApi()

    // Configure Waddah Attar
    _, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_WADDAH_ATTAR,
        Config:        `[20, 40, 9, 14, 150.0]`,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to configure Waddah Attar: %w", err)
    }

    return &emptypb.Empty{}, nil
}
```

### Getting Waddah Attar Signal

```go
func (s *MyStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
    data := req.Data
    api := strategy.NewStrategyApi()

    // Get Waddah Attar signal
    signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
        IndicatorType: strategy.IndicatorType_INDICATOR_WADDAH_ATTAR,
        MarketData:    data,
    })
    if err != nil {
        return nil, fmt.Errorf("failed to get Waddah Attar signal: %w", err)
    }

    // Parse raw value
    var waValue struct {
        MACD      float64 `json:"macd"`
        Signal    float64 `json:"signal"`
        Histogram float64 `json:"histogram"`
        ATR       float64 `json:"atr"`
        Trend     float64 `json:"trend"`
        Explosion float64 `json:"explosion"`
    }
    if err := json.Unmarshal([]byte(signal.RawValue), &waValue); err != nil {
        return nil, fmt.Errorf("failed to parse Waddah Attar value: %w", err)
    }

    fmt.Printf("Trend: %.2f, Explosion: %.2f, Histogram: %.4f\n",
        waValue.Trend, waValue.Explosion, waValue.Histogram)

    // Trading logic
    switch signal.Type {
    case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
        // Bullish explosion detected
        if waValue.Trend > waValue.Explosion {
            // Strong bullish signal - trend exceeds explosion threshold
        }
    case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
        // Bearish explosion detected
        if math.Abs(waValue.Trend) > waValue.Explosion {
            // Strong bearish signal - trend exceeds explosion threshold
        }
    }

    return &emptypb.Empty{}, nil
}
```

## Common Use Cases

1. **Explosion Detection**: Identify periods of high volatility with clear direction
2. **Trend Confirmation**: Use trend value to confirm direction before entry
3. **Entry Timing**: Enter when explosion threshold is exceeded
4. **Exit Signal**: Exit when trend reverses or explosion diminishes

## Interpretation Guide

| Condition | Interpretation |
|-----------|----------------|
| Trend > 0, Explosion > 0 | Bullish momentum with volatility |
| Trend < 0, Explosion > 0 | Bearish momentum with volatility |
| Trend crossing zero | Potential trend reversal |
| Rising histogram | Increasing momentum |
| Falling histogram | Decreasing momentum |
| Trend > Explosion | Very strong momentum |

## Visual Interpretation

The indicator is typically displayed as:
- **Green bars** (Trend > 0): Bullish trend
- **Red bars** (Trend < 0): Bearish trend
- **Explosion line**: Volatility threshold
- When bars exceed the explosion line, it signals a strong move

## Parameter Tuning

| Parameter | Effect of Increasing | Effect of Decreasing |
|-----------|---------------------|---------------------|
| Fast Period | Slower MACD response | Faster MACD response |
| Slow Period | Smoother MACD | More responsive MACD |
| Signal Period | Smoother signal line | Faster signal crosses |
| ATR Period | Smoother explosion line | More volatile explosion |
| Multiplier | Larger trend/explosion values | Smaller values |

### Suggested Settings

| Market | Fast | Slow | Signal | ATR | Multiplier |
|--------|------|------|--------|-----|------------|
| Forex | 20 | 40 | 9 | 14 | 150 |
| Stocks | 12 | 26 | 9 | 14 | 100 |
| Crypto | 20 | 40 | 9 | 14 | 200 |

## Related Indicators

- [MACD](macd.md) - Used internally for trend direction
- [ATR](atr.md) - Used internally for volatility
- [RSI](rsi.md) - Can be combined for confirmation

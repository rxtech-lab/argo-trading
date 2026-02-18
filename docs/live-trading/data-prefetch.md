---
title: Data Prefetch
description: Historical data prefetching for accurate indicator calculations in live trading
---

# Data Prefetch

This document describes the data prefetch feature that ensures indicators have sufficient historical data for accurate calculations from the moment live trading begins.

## Overview

### The Problem

Technical indicators require historical data for calculation:
- RSI needs 14 periods of price history
- MACD needs 26 periods for the slow EMA
- Bollinger Bands need 20 periods for the moving average

Without historical data, indicators produce incorrect or no values at the start of live trading, leading to poor trading decisions.

### The Solution

The prefetch feature:
1. Downloads historical data before starting the live stream
2. Stores it in the same parquet format as live data
3. Detects and fills any gaps between historical and live data
4. Ensures indicators have complete history from the first live candle

## Configuration

### YAML Configuration

```yaml
engine:
  symbols:
    - BTCUSDT
  interval: "1m"
  prefetch:
    enabled: true
    start_time_type: days    # "date" or "days"
    days: 30                 # Prefetch 30 days of history
```

Note: `data_output_path` is set via `SetDataOutputPath()`, not in the config.

Or with absolute date:

```yaml
engine:
  prefetch:
    enabled: true
    start_time_type: date
    start_time: "2025-01-01T00:00:00Z"
```

### Go Configuration

```go
config := engine.LiveTradingEngineConfig{
    Prefetch: engine.PrefetchConfig{
        Enabled:       true,
        StartTimeType: "days",
        Days:          30,
    },
}
eng.Initialize(config)

// Set data output path separately
eng.SetDataOutputPath("./data/live-trading")
```

### Configuration Fields

| Field | Type | Description |
|-------|------|-------------|
| `enabled` | bool | Enable/disable prefetch |
| `start_time_type` | string | `"date"` or `"days"` |
| `start_time` | time.Time | Absolute start time (when type is `"date"`) |
| `days` | int | Number of days to prefetch (when type is `"days"`) |

## Data Flow

The prefetch process consists of four phases:

```
┌─────────────────────────────────────────────────────────────────┐
│                     Phase 1: Prefetch                            │
│                                                                 │
│  ┌─────────────┐    ┌──────────────┐    ┌──────────────────┐   │
│  │ Calculate   │───▶│   Download   │───▶│ Store to Parquet │   │
│  │ Start Time  │    │ via REST API │    │                  │   │
│  └─────────────┘    └──────────────┘    └──────────────────┘   │
│                                                                 │
│  - If type="days": start = now - N days                         │
│  - If type="date": start = configured date                      │
│  - Download from start to now                                   │
│  - Store in market_data.parquet                                 │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                   Phase 2: Gap Detection                         │
│                                                                 │
│  ┌──────────────────┐    ┌─────────────────┐    ┌───────────┐  │
│  │ Get last stored  │───▶│ Connect to      │───▶│ Calculate │  │
│  │ timestamp        │    │ live stream     │    │ gap range │  │
│  └──────────────────┘    └─────────────────┘    └───────────┘  │
│                                                                 │
│  - Query: SELECT MAX(time) FROM market_data.parquet             │
│  - Note first stream timestamp                                  │
│  - Gap = stream_start - last_stored                             │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Phase 3: Gap Fill                             │
│                                                                 │
│  ┌────────────────┐    ┌─────────────────┐                     │
│  │ Fetch gap data │───▶│ Store gap data  │                     │
│  │ via REST API   │    │ to parquet      │                     │
│  └────────────────┘    └─────────────────┘                     │
│                                                                 │
│  - Pause live stream consumption during gap fill                │
│  - REST API fetches missing candles                             │
│  - Store to parquet                                             │
│  - Some live data points may be missed (acceptable)             │
│  - Resume live stream after gap fill completes                  │
└─────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────────┐
│                    Phase 4: Live Trading                         │
│                                                                 │
│  ┌──────────────┐    ┌──────────────┐    ┌──────────────────┐  │
│  │ Receive live │───▶│ Store to     │───▶│ Feed to          │  │
│  │ stream data  │    │ parquet      │    │ strategy         │  │
│  └──────────────┘    └──────────────┘    └──────────────────┘  │
│                                                                 │
│  - Normal live trading mode                                     │
│  - Indicators have full history                                 │
│  - All data persisted for recovery                              │
└─────────────────────────────────────────────────────────────────┘
```

## Phase Details

### Phase 1: Prefetch

**Status:** `EngineStatusPrefetching`

The engine downloads historical data using the market data provider's `Download()` method:

```go
// Calculate start time
var startTime time.Time
if config.Prefetch.StartTimeType == "days" {
    startTime = time.Now().AddDate(0, 0, -config.Prefetch.Days)
} else {
    startTime = config.Prefetch.StartTime
}

// Download historical data
for _, symbol := range config.Symbols {
    provider.Download(ctx, DownloadParams{
        Ticker:    symbol,
        StartDate: startTime,
        EndDate:   time.Now(),
        Timespan:  intervalToTimespan(config.Interval),
    })
}
```

Data is stored to `market_data.parquet` in the session folder.

### Phase 2: Gap Detection

After prefetch completes, the engine:

1. Queries the last stored timestamp:
```sql
SELECT MAX(time) as last_time FROM read_parquet('market_data.parquet')
```

2. Connects to the live stream and notes the first received timestamp

3. Calculates the gap:
```go
gap := firstStreamTime.Sub(lastStoredTime)
if gap > tolerance {
    // Need to fill gap
}
```

**Gap Tolerance:**
- Small gaps (< 2 intervals) are acceptable and skipped
- This accounts for network latency and provider delays

### Phase 3: Gap Fill

If a significant gap is detected:

1. **Pause stream**: Stop consuming live stream data temporarily
2. **Fetch gap data**: REST API downloads the missing candles
3. **Store gap data**: Append to parquet file
4. **Resume stream**: Continue with live data (some data points may be missed, which is acceptable)

```go
// Pseudocode for gap fill

// Fetch and store gap data (blocks until complete)
gapData := provider.Download(ctx, DownloadParams{
    StartDate: lastStoredTime,
    EndDate:   time.Now(),
})
storeToParquet(gapData)

// Resume live stream - some candles during gap fill may be missed
for data := range stream {
    storeToParquet(data)
    strategy.ProcessData(data)
}
```

### Phase 4: Live Trading

**Status:** `EngineStatusRunning`

Normal live trading proceeds:
- Each candle is stored to parquet
- Each candle is fed to the strategy
- Indicators use `GetPreviousNumberOfDataPoints()` which queries parquet

## Error Handling

### Prefetch Failure

If historical data download fails:

```go
// Retry with exponential backoff
for attempt := 0; attempt < maxRetries; attempt++ {
    err := provider.Download(ctx, params)
    if err == nil {
        break
    }

    backoff := time.Second * time.Duration(math.Pow(2, float64(attempt)))
    time.Sleep(backoff)

    if attempt == maxRetries-1 {
        // Log warning, continue without full history
        log.Warn("Prefetch failed, starting with limited history")
    }
}
```

**Behavior on failure:**
- Engine continues with available data
- Indicators may have insufficient data initially
- Warning logged for operator awareness

### Gap Fill Failure

If gap fill fails:

```go
err := fillGap(lastStored, firstStream)
if err != nil {
    log.Warn("Gap fill failed, some indicator values may be inaccurate",
        "gap", firstStream.Sub(lastStored))
    // Continue with live data only
}
```

**Behavior:**
- Live trading continues
- Indicators may have a gap in their history
- Warning logged

### Provider Rate Limits

When providers rate limit requests:

```go
if isRateLimitError(err) {
    wait := parseRetryAfter(err)
    log.Info("Rate limited, waiting", "duration", wait)
    time.Sleep(wait)
    // Retry
}
```

## Edge Cases

### Data Points Missed During Gap Fill

During gap fill, live stream data is not consumed. This means some candles may be missed:

```
Timeline:
  T+0s:   Gap fill starts
  T+5s:   Gap fill completes
  T+5s:   Resume live stream

  Missed: Any candles that closed between T+0s and T+5s
```

This is acceptable because:
- Gap fill ensures historical continuity for indicators
- Missing a few recent candles has minimal impact
- The next candle will be processed normally

### Clock Skew

When local time differs from provider time:

```go
// Use provider timestamps, not local time
const tolerance = 2 * interval

if abs(localTime - providerTime) > tolerance {
    log.Warn("Clock skew detected", "local", localTime, "provider", providerTime)
}

// Gap calculation uses provider timestamps only
gap := firstStreamProviderTime.Sub(lastStoredProviderTime)
```

### Restart Recovery

When engine restarts with existing data:

1. Scan existing parquet file for last timestamp
2. Calculate prefetch range from last stored to now
3. Skip already-stored data during download
4. Continue with gap fill and live trading

```go
// Check for existing data
lastStored, err := getLastStoredTime(parquetPath)
if err == nil {
    // Have existing data, adjust prefetch range
    prefetchStart = lastStored
    log.Info("Resuming from existing data", "lastStored", lastStored)
}
```

### Empty or Corrupt Parquet

If the parquet file is empty or corrupt:

```go
_, err := getLastStoredTime(parquetPath)
if err != nil {
    // Start fresh prefetch
    log.Warn("Could not read existing data, starting fresh prefetch")
    os.Remove(parquetPath)
    // Full prefetch from configured start time
}
```

## Indicator Accuracy

### How Indicators Use Prefetched Data

The `PersistentStreamingDataSource` queries the parquet file:

```go
func (d *PersistentStreamingDataSource) GetPreviousNumberOfDataPoints(
    symbol string,
    timestamp time.Time,
    count int,
) ([]types.MarketData, error) {
    query := fmt.Sprintf(`
        SELECT * FROM read_parquet('%s')
        WHERE symbol = '%s' AND time < '%s'
        ORDER BY time DESC
        LIMIT %d
    `, d.parquetPath, symbol, timestamp.Format(time.RFC3339), count)

    return executeQuery(query)
}
```

### Example: RSI Calculation

For a 14-period RSI on the first live candle:

```
Prefetched data: 30 days = ~43,200 candles (1m interval)
RSI needs: 14 candles minimum

First live candle at T:
  - Strategy calls GetPreviousNumberOfDataPoints(symbol, T, 14)
  - Returns candles T-1, T-2, ..., T-14 from parquet
  - RSI calculated correctly
```

### Verifying Indicator Accuracy

To verify indicators are accurate after prefetch:

1. Run backtest on same date range
2. Compare indicator values at first few live candles
3. Values should match within floating-point tolerance

```go
// Test helper
func TestIndicatorAccuracyAfterPrefetch(t *testing.T) {
    // Run backtest
    backtestRSI := runBacktest(startDate, endDate)

    // Run live with prefetch from same startDate
    liveRSI := runLiveWithPrefetch(startDate)

    // Compare at first live candle
    assert.InDelta(t, backtestRSI[0], liveRSI[0], 0.001)
}
```

## Performance Considerations

### Prefetch Duration

Approximate prefetch times (depends on provider and network):

| Data Range | Candles (1m) | Time Estimate |
|------------|--------------|---------------|
| 1 day | 1,440 | 2-5 seconds |
| 7 days | 10,080 | 10-20 seconds |
| 30 days | 43,200 | 30-60 seconds |
| 90 days | 129,600 | 1-3 minutes |

### Memory Usage

During gap fill, stream data is buffered in memory:
- Each candle ~100 bytes
- 1 hour of 1m data = 60 candles = ~6 KB
- Typical gap fill < 5 minutes = ~500 bytes

### Disk Usage

Parquet files are compressed:
- 30 days of 1m data ≈ 2-5 MB per symbol
- 90 days of 1m data ≈ 6-15 MB per symbol

## Related Documentation

- [Live Trading Engine Overview](README.md)
- [Session Management and Persistence](session-and-persistence.md)
- [Testing](testing.md) - Includes prefetch testing scenarios

---
title: Session Management and Persistence
description: Daily statistics, multi-day sessions, and real-time data storage for live trading
---

# Session Management and Persistence

This document describes how the Live Trading Engine manages trading sessions, persists data in real-time, and handles multi-day operations.

## Overview

Live trading sessions need to:

1. **Track statistics** similar to backtest runs (orders, trades, PnL, etc.)
2. **Persist data in real-time** so it survives crashes and can be queried externally
3. **Handle multi-day sessions** with per-day record storage
4. **Support multiple sessions per day** stored separately

## Session Lifecycle

### Session Start

When the engine starts:

1. Generate a unique run identifier (incrementing `run_1`, `run_2`, etc.)
2. Create the output folder structure
3. Initialize DuckDB tables for orders, trades, marks, logs
4. Call `OnEngineStart` callback with session info
5. Begin streaming market data

### During Session

For each market data point:

1. Store market data to parquet (for prefetch recovery)
2. Execute strategy's `ProcessData()`
3. Record any orders placed
4. Update statistics in real-time
5. Persist stats.yaml to disk
6. Call `OnMarketData` and `OnStatsUpdate` callbacks

### Session End

When the engine stops (graceful or crash):

1. Flush all pending data to disk
2. Calculate final statistics
3. Write final stats.yaml
4. Call `OnEngineStop` callback
5. Clean up resources

## Data Folder Structure

All session data is stored under the configured `data_output_path`:

```
{data_output_path}/
└── {YYYY-MM-DD}/
    └── run_1/
        ├── stats.yaml           # Real-time updated statistics
        ├── orders.parquet       # All orders placed
        ├── trades.parquet       # All executed trades
        ├── marks.parquet        # Strategy markers/annotations
        ├── logs.parquet         # Strategy logs
        └── market_data.parquet  # Stored market data
```

### Field Descriptions

| Field | Description |
|-------|-------------|
| `{YYYY-MM-DD}` | Date in ISO format (e.g., `2025-10-01`) |
| `run_1`, `run_2`, ... | Incrementing run number for the day |

## Multi-Day Sessions

When a session spans multiple days, data is stored per-day with consistent run numbers.

### Example: Session from Oct 1 to Oct 3

A session starting on 2025-10-01 and running until 2025-10-03 produces:

```
{data_output_path}/
├── 2025-10-01/
│   └── run_1/
│       ├── stats.yaml       # Stats for Oct 1
│       ├── orders.parquet   # Orders placed on Oct 1
│       ├── trades.parquet   # Trades executed on Oct 1
│       ├── marks.parquet
│       ├── logs.parquet
│       └── market_data.parquet
├── 2025-10-02/
│   └── run_1/               # Same session continues
│       ├── stats.yaml       # Stats for Oct 2
│       ├── orders.parquet   # Orders placed on Oct 2
│       └── ...
└── 2025-10-03/
    └── run_1/               # Same session continues
        ├── stats.yaml       # Stats for Oct 3
        └── ...
```

### Date Boundary Detection

The engine detects date boundaries by comparing the current market data timestamp with the previous one. When the date changes:

1. Finalize current day's data (flush to disk)
2. Create new day's folder with same run number
3. Initialize new day's DuckDB tables
4. Continue processing

### Cumulative vs Daily Stats

Each day's `stats.yaml` contains:

- **Daily stats**: Statistics for that specific day only
- **Cumulative stats**: Running totals from session start

```yaml
# stats.yaml structure
id: "run_1"
date: "2025-10-02"
session_start: "2025-10-01T09:00:00Z"

# Daily statistics (this day only)
daily:
  number_of_trades: 15
  realized_pnl: 250.50
  win_rate: 0.67

# Cumulative statistics (from session start)
cumulative:
  number_of_trades: 42
  realized_pnl: 780.25
  win_rate: 0.62
```

## Multiple Sessions on Same Day

If multiple trading sessions run on the same day, each gets its own run folder:

```
{data_output_path}/
└── 2025-10-03/
    ├── run_1/          # First session (morning)
    │   ├── stats.yaml
    │   └── ...
    ├── run_2/          # Second session (afternoon)
    │   ├── stats.yaml
    │   └── ...
    └── run_3/          # Third session (evening)
        ├── stats.yaml
        └── ...
```

### Run Number Determination

On startup, the engine:

1. Scans `{data_output_path}/{today}/` for existing `run_*` folders
2. Finds the highest run number
3. Increments by 1 for the new session

## Session Statistics

### LiveTradeStats Structure

Live trading statistics mirror the backtest `TradeStats` structure:

```go
type LiveTradeStats struct {
    // Session identification
    ID            string    `yaml:"id" json:"id"`
    Date          string    `yaml:"date" json:"date"`
    SessionStart  time.Time `yaml:"session_start" json:"session_start"`
    LastUpdated   time.Time `yaml:"last_updated" json:"last_updated"`

    // Symbols being traded
    Symbols []string `yaml:"symbols" json:"symbols"`

    // Trade results
    TradeResult TradeResult `yaml:"trade_result" json:"trade_result"`

    // PnL breakdown
    TradePnl TradePnl `yaml:"trade_pnl" json:"trade_pnl"`

    // Holding time statistics
    TradeHoldingTime TradeHoldingTime `yaml:"trade_holding_time" json:"trade_holding_time"`

    // Total fees paid
    TotalFees float64 `yaml:"total_fees" json:"total_fees"`

    // File paths for related data
    OrdersFilePath     string `yaml:"orders_file_path" json:"orders_file_path"`
    TradesFilePath     string `yaml:"trades_file_path" json:"trades_file_path"`
    MarksFilePath      string `yaml:"marks_file_path" json:"marks_file_path"`
    LogsFilePath       string `yaml:"logs_file_path" json:"logs_file_path"`
    MarketDataFilePath string `yaml:"market_data_file_path" json:"market_data_file_path"`

    // Strategy information
    Strategy StrategyInfo `yaml:"strategy" json:"strategy"`
}

type TradeResult struct {
    NumberOfTrades        int     `yaml:"number_of_trades"`
    NumberOfWinningTrades int     `yaml:"number_of_winning_trades"`
    NumberOfLosingTrades  int     `yaml:"number_of_losing_trades"`
    WinRate               float64 `yaml:"win_rate"`
    MaxDrawdown           float64 `yaml:"max_drawdown"`
}

type TradePnl struct {
    RealizedPnL   float64 `yaml:"realized_pnl"`
    UnrealizedPnL float64 `yaml:"unrealized_pnl"`
    TotalPnL      float64 `yaml:"total_pnl"`
    MaximumLoss   float64 `yaml:"maximum_loss"`
    MaximumProfit float64 `yaml:"maximum_profit"`
}
```

### Real-Time Statistics Emission

Statistics are emitted via the `OnStatsUpdate` callback:

```go
onStats := engine.OnStatsUpdateCallback(func(stats LiveTradeStats) error {
    // Push to dashboard, log, alert, etc.
    fmt.Printf("PnL: %.2f, Trades: %d\n", stats.TradePnl.TotalPnL, stats.TradeResult.NumberOfTrades)
    return nil
})

callbacks := engine.LiveTradingCallbacks{
    OnStatsUpdate: &onStats,
}
```

Statistics are updated:
- After each trade execution
- Periodically (every N market data points)
- On session end

## Real-Time Persistence

### Data Folder Parameter

The data output path is passed when initializing the engine:

```go
config := engine.LiveTradingEngineConfig{
    DataOutputPath: "./data/live-trading",
    // ... other config
}
eng.Initialize(config)
```

### Persistence Timing

| Data Type | When Persisted |
|-----------|----------------|
| Market data | Each finalized candle |
| Orders | When placed |
| Trades | When executed |
| Marks | When created |
| Logs | When logged |
| stats.yaml | After each trade, periodically |

### DuckDB to Parquet

Data is stored in-memory using DuckDB during the session, then exported to Parquet:

```go
// In-memory storage during session
db, _ := sql.Open("duckdb", ":memory:")

// Periodic export to Parquet
_, _ = db.Exec(`COPY orders TO 'orders.parquet' (FORMAT PARQUET)`)
```

### Query Interface

Since data is stored in Parquet, external tools can query it in real-time:

```sql
-- Query trades from a running session
SELECT * FROM read_parquet('./data/live-trading/2025-10-03/run_1/trades.parquet')
WHERE pnl > 0;

-- Aggregate statistics
SELECT
    COUNT(*) as total_trades,
    SUM(pnl) as total_pnl,
    AVG(pnl) as avg_pnl
FROM read_parquet('./data/live-trading/2025-10-03/run_1/trades.parquet');
```

## Crash Recovery

### Data Preservation

On unexpected exit:

1. **Market data**: Already persisted to parquet
2. **Orders/Trades**: Most recent state in parquet (may miss last few)
3. **Logs**: Preserved up to last flush
4. **stats.yaml**: Preserved up to last update

### Recovery on Restart

When the engine restarts:

1. Scans existing parquet files in data output path
2. Loads market data to avoid re-downloading
3. Starts new run with incremented run number
4. Prefetch fills any gaps (see [Data Prefetch](data-prefetch.md))

### Example Recovery Scenario

```
Session crashes at 2025-10-03 14:30

Before crash:
./data/live-trading/2025-10-03/run_1/
├── stats.yaml           (updated at 14:29)
├── orders.parquet       (up to 14:28)
├── trades.parquet       (up to 14:28)
└── market_data.parquet  (up to 14:30)

After restart (new session):
./data/live-trading/2025-10-03/run_2/   # New run number
├── stats.yaml
└── ...
```

## File Formats

### stats.yaml

Full example:

```yaml
id: "run_1"
date: "2025-10-03"
session_start: "2025-10-03T09:00:00Z"
last_updated: "2025-10-03T14:30:00Z"

symbols:
  - BTCUSDT
  - ETHUSDT

trade_result:
  number_of_trades: 42
  number_of_winning_trades: 26
  number_of_losing_trades: 16
  win_rate: 0.619
  max_drawdown: 0.045

trade_pnl:
  realized_pnl: 1250.75
  unrealized_pnl: 150.25
  total_pnl: 1401.00
  maximum_loss: -125.50
  maximum_profit: 350.00

trade_holding_time:
  min: 120
  max: 3600
  avg: 840

total_fees: 42.50

orders_file_path: "./data/live-trading/2025-10-03/run_1/orders.parquet"
trades_file_path: "./data/live-trading/2025-10-03/run_1/trades.parquet"
marks_file_path: "./data/live-trading/2025-10-03/run_1/marks.parquet"
logs_file_path: "./data/live-trading/2025-10-03/run_1/logs.parquet"
market_data_file_path: "./data/live-trading/2025-10-03/run_1/market_data.parquet"

strategy:
  id: "com.example.strategy.momentum"
  version: "1.0.0"
  name: "Momentum Strategy"
```

### Parquet Schemas

**orders.parquet:**
| Column | Type |
|--------|------|
| order_id | TEXT |
| symbol | TEXT |
| side | TEXT |
| order_type | TEXT |
| quantity | DOUBLE |
| price | DOUBLE |
| timestamp | TIMESTAMP |
| status | TEXT |
| strategy_name | TEXT |
| position_type | TEXT |

**trades.parquet:**
| Column | Type |
|--------|------|
| order_id | TEXT |
| symbol | TEXT |
| side | TEXT |
| quantity | DOUBLE |
| price | DOUBLE |
| timestamp | TIMESTAMP |
| executed_at | TIMESTAMP |
| executed_qty | DOUBLE |
| executed_price | DOUBLE |
| commission | DOUBLE |
| pnl | DOUBLE |
| position_type | TEXT |

## Related Documentation

- [Live Trading Engine Overview](README.md)
- [Data Prefetch](data-prefetch.md)
- [Testing](testing.md)

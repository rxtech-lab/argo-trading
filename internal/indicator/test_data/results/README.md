# Backtest engine comparison results

This folder contains the reference outputs produced by `backtesting.py`
running an RSI(14) long-only single-share strategy on `../comparison_data.parquet`.

The committed files are the source of truth for the Go end-to-end test
under `e2e/backtest/wasm/rsi_comparison`, which runs the same strategy in
argo's backtest engine and asserts that every trade, order and PnL value
matches the values here within `1e-6`.

## Files

| File          | Description                                                                   |
| ------------- | ----------------------------------------------------------------------------- |
| `trades.csv`  | One row per executed buy or sell, in chronological order. Columns: `executed_at, side, quantity, price, pnl, cumulative_pnl, symbol`. |
| `orders.csv`  | One row per filled order. Columns: `timestamp, symbol, side, quantity, price`. |
| `equity.csv`  | Per-bar equity curve from `backtesting.py`. Columns: `time, equity, drawdown_pct, drawdown_duration, pnl`. |
| `summary.json`| Aggregate statistics (counts, realised PnL, final equity).                   |

## Regenerating

From `internal/indicator/test_data` run:

```bash
poetry install         # first time only, installs backtesting / duckdb / pandas
poetry run python -m comparison.main
```

This regenerates `../comparison_data.parquet` and overwrites every file
in this directory. After regenerating, run the e2e test to make sure argo
still matches:

```bash
make -C e2e/backtest/wasm build
go test ./e2e/backtest/wasm/rsi_comparison/...
```

## Strategy

* RSI(14), simple-rolling-mean variant (matches `internal/indicator/rsi.go`).
* When RSI < 30 and flat: BUY 1 share at the bar's close.
* When RSI > 70 and long: SELL 1 share at the bar's close.
* `cash = 10_000`, `commission = 0`, `trade_on_close = True`.

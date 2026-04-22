"""Run an RSI(14) strategy with `backtesting.py` and export the reference results.

Strategy semantics (intentionally identical to the WASM strategy in
`e2e/backtest/wasm/rsi_comparison`):

* Single symbol, single 1-share long position.
* When RSI(14) < 30 and we are flat, place a market BUY of 1 share.
* When RSI(14) > 70 and we are long, place a market SELL of 1 share.
* `commission = 0`, `cash = 10_000`, `trade_on_close = True` so market
  orders fill at the current bar's close. Argo's limit orders priced at
  `data.Close` also fill at the same bar (because Low <= price <= High when
  open == high == low == close), so fill prices match exactly.

Outputs (CSV / JSON, all timestamps in ISO-8601 UTC):

* `trades.csv`   - one row per executed buy or sell.
* `orders.csv`   - one row per placed order (buys + sells, all filled).
* `equity.csv`   - bar-by-bar equity curve / mark-to-market PnL.
* `summary.json` - aggregate stats (final equity, realised PnL, counts...).
"""

from __future__ import annotations

import json
import os
from dataclasses import dataclass

import pandas as pd
from backtesting import Backtest, Strategy

from .indicator import rsi

INITIAL_CASH = 10_000.0
RSI_PERIOD = 14
RSI_LOWER = 30.0
RSI_UPPER = 70.0


@dataclass
class RunResult:
    trades: pd.DataFrame
    orders: pd.DataFrame
    equity: pd.DataFrame
    summary: dict


class RSIComparisonStrategy(Strategy):
    """RSI(14) long-only single-share strategy used for comparison."""

    rsi_period = RSI_PERIOD
    rsi_lower = RSI_LOWER
    rsi_upper = RSI_UPPER

    def init(self) -> None:
        self.rsi = self.I(rsi, self.data.Close, self.rsi_period, name="rsi")

    def next(self) -> None:
        current_rsi = self.rsi[-1]
        if pd.isna(current_rsi):
            return

        if current_rsi < self.rsi_lower and not self.position:
            # Buy exactly one share at the current close.
            self.buy(size=1)
        elif current_rsi > self.rsi_upper and self.position.is_long:
            # Close the long position by selling exactly one share.
            self.sell(size=1)


def _prepare_dataframe(parquet_path: str) -> pd.DataFrame:
    df = pd.read_parquet(parquet_path)
    df = df.sort_values("time").reset_index(drop=True)
    bt_df = pd.DataFrame(
        {
            "Open": df["open"].astype(float).to_numpy(),
            "High": df["high"].astype(float).to_numpy(),
            "Low": df["low"].astype(float).to_numpy(),
            "Close": df["close"].astype(float).to_numpy(),
            "Volume": df["volume"].astype(float).to_numpy(),
        },
        index=pd.DatetimeIndex(pd.to_datetime(df["time"], utc=False), name="time"),
    )
    return bt_df


def run_backtest(parquet_path: str) -> RunResult:
    """Run the RSI strategy and return all artefacts as DataFrames / dict."""

    bt_df = _prepare_dataframe(parquet_path)
    bt = Backtest(
        bt_df,
        RSIComparisonStrategy,
        cash=INITIAL_CASH,
        commission=0.0,
        trade_on_close=True,
        exclusive_orders=False,
        finalize_trades=False,
    )
    stats = bt.run()

    raw_trades = stats["_trades"]

    # Build the per-execution trades and orders frames. Each closed trade in
    # backtesting.py corresponds to one BUY entry and one SELL exit, so we
    # explode them into individual executions to match argo's per-trade output.
    rows = []
    for trade in raw_trades.itertuples():
        rows.append(
            {
                "executed_at": trade.EntryTime,
                "side": "BUY",
                "quantity": float(trade.Size),
                "price": float(trade.EntryPrice),
                "pnl": 0.0,  # Buys realize no PnL (FIFO match happens on exit).
            }
        )
        rows.append(
            {
                "executed_at": trade.ExitTime,
                "side": "SELL",
                "quantity": float(trade.Size),
                "price": float(trade.ExitPrice),
                "pnl": float(trade.PnL),
            }
        )

    trades_df = pd.DataFrame(rows).sort_values("executed_at").reset_index(drop=True)
    if not trades_df.empty:
        trades_df["cumulative_pnl"] = trades_df["pnl"].cumsum()
    else:
        trades_df["cumulative_pnl"] = pd.Series(dtype=float)
    trades_df["symbol"] = "TESTSTOCK"

    orders_df = trades_df[["executed_at", "symbol", "side", "quantity", "price"]].copy()
    orders_df = orders_df.rename(columns={"executed_at": "timestamp"})

    equity_df = stats["_equity_curve"].copy()
    equity_df.index.name = "time"
    equity_df = equity_df.reset_index()
    equity_df = equity_df.rename(
        columns={"Equity": "equity", "DrawdownPct": "drawdown_pct", "DrawdownDuration": "drawdown_duration"}
    )
    equity_df["pnl"] = equity_df["equity"] - INITIAL_CASH

    realized_pnl = float(trades_df["pnl"].sum()) if not trades_df.empty else 0.0
    summary = {
        "symbol": "TESTSTOCK",
        "rsi_period": RSI_PERIOD,
        "rsi_lower": RSI_LOWER,
        "rsi_upper": RSI_UPPER,
        "initial_cash": INITIAL_CASH,
        "final_equity": float(stats["Equity Final [$]"]),
        "realized_pnl": realized_pnl,
        "num_trades": int(len(trades_df)),
        "num_buys": int((trades_df["side"] == "BUY").sum()) if not trades_df.empty else 0,
        "num_sells": int((trades_df["side"] == "SELL").sum()) if not trades_df.empty else 0,
        "num_closed_positions": int(len(raw_trades)),
        "first_trade_time": trades_df["executed_at"].iloc[0].isoformat() if not trades_df.empty else None,
        "last_trade_time": trades_df["executed_at"].iloc[-1].isoformat() if not trades_df.empty else None,
    }

    return RunResult(trades=trades_df, orders=orders_df, equity=equity_df, summary=summary)


def write_results(result: RunResult, results_dir: str) -> None:
    """Persist the run artefacts to disk."""

    os.makedirs(results_dir, exist_ok=True)

    # Use ISO-8601 strings so Go can parse them deterministically.
    trades = result.trades.copy()
    if not trades.empty:
        trades["executed_at"] = trades["executed_at"].apply(lambda t: pd.Timestamp(t).isoformat())
    trades.to_csv(os.path.join(results_dir, "trades.csv"), index=False)

    orders = result.orders.copy()
    if not orders.empty:
        orders["timestamp"] = orders["timestamp"].apply(lambda t: pd.Timestamp(t).isoformat())
    orders.to_csv(os.path.join(results_dir, "orders.csv"), index=False)

    equity = result.equity.copy()
    equity["time"] = equity["time"].apply(lambda t: pd.Timestamp(t).isoformat())
    equity.to_csv(os.path.join(results_dir, "equity.csv"), index=False)

    with open(os.path.join(results_dir, "summary.json"), "w", encoding="utf-8") as fh:
        json.dump(result.summary, fh, indent=2, sort_keys=True)
        fh.write("\n")

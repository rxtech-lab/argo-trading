"""Deterministic OHLCV data generator for the comparison test.

Generates a 10,000-row dataset with `open == high == low == close` so that
the argo backtest engine and `backtesting.py` produce unambiguous and
identical fill prices for limit / trade-on-close orders.

The price series is a seeded random-walk wrapped around a slow sine wave,
which guarantees a healthy number of RSI(14) crossings of the 30 / 70
thresholds without depending on the system RNG.
"""

from __future__ import annotations

import math
from collections.abc import Callable
from datetime import datetime, timedelta, timezone

import duckdb
import pandas as pd

# Generator constants. Changing any of these invalidates the committed dataset.
SYMBOL = "TESTSTOCK"
NUM_ROWS = 10_000
START_TIME = datetime(2024, 1, 1, 9, 0, 0, tzinfo=timezone.utc)
INTERVAL = timedelta(minutes=1)
INITIAL_PRICE = 100.0
SEED = 20240422
# Sine-wave + random-walk parameters tuned to produce frequent RSI crossings.
WAVE_AMPLITUDE = 8.0
WAVE_PERIOD = 250  # bars per full sine cycle
NOISE_STDDEV = 0.35
VOLUME_BASE = 1_000.0


def _lcg(seed: int) -> Callable[[], float]:
    """Return a deterministic uniform[0, 1) generator (LCG) for portability."""

    state = seed & 0xFFFFFFFF

    def next_value() -> float:
        nonlocal state
        # Numerical Recipes constants.
        state = (state * 1664525 + 1013904223) & 0xFFFFFFFF
        return state / 0x100000000

    return next_value


def _gauss(uniform: Callable[[], float]) -> Callable[[], float]:
    """Box-Muller transform on top of the LCG to produce N(0, 1) samples."""

    cached: list[float] = []

    def next_value() -> float:
        if cached:
            return cached.pop()
        u1 = max(uniform(), 1e-12)
        u2 = uniform()
        mag = math.sqrt(-2.0 * math.log(u1))
        z0 = mag * math.cos(2 * math.pi * u2)
        z1 = mag * math.sin(2 * math.pi * u2)
        cached.append(z1)
        return z0

    return next_value


def generate_dataframe() -> pd.DataFrame:
    """Generate the deterministic OHLCV DataFrame."""

    uniform = _lcg(SEED)
    gauss = _gauss(uniform)

    times: list[datetime] = []
    prices: list[float] = []

    price = INITIAL_PRICE
    for i in range(NUM_ROWS):
        # Sine component drives medium-term trend cycles.
        wave = WAVE_AMPLITUDE * math.sin(2 * math.pi * i / WAVE_PERIOD)
        # Random walk noise on top of the wave.
        price = INITIAL_PRICE + wave + NOISE_STDDEV * gauss() + (price - INITIAL_PRICE - wave) * 0.85
        # Floor to avoid non-positive prices.
        if price < 1.0:
            price = 1.0
        times.append(START_TIME + i * INTERVAL)
        prices.append(round(price, 4))

    df = pd.DataFrame(
        {
            "time": times,
            "symbol": SYMBOL,
            "open": prices,
            "high": prices,
            "low": prices,
            "close": prices,
            "volume": [VOLUME_BASE] * NUM_ROWS,
        }
    )
    # Drop tz info: argo writes naive timestamps and DuckDB compares them as such.
    df["time"] = df["time"].dt.tz_convert(None)
    return df


def write_parquet(df: pd.DataFrame, path: str) -> None:
    """Write the DataFrame to a parquet file via DuckDB so the schema matches argo."""

    duckdb.from_df(df).write_parquet(path)

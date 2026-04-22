"""SMA-based RSI implementation that matches argo's `internal/indicator/rsi.go`.

Argo fetches `period + 1` previous close prices (inclusive of the current bar)
and computes the RSI from a simple average of gains and losses over the
`period` differences. The `for i := period; i < len(gains)` Wilder smoothing
loop in argo is a no-op because `len(gains) == period`, so the result is a
plain rolling-mean RSI. We replicate that exactly here.
"""

from __future__ import annotations

import numpy as np
import pandas as pd


def rsi(close: pd.Series | np.ndarray, period: int = 14) -> np.ndarray:
    """Compute the rolling-mean RSI used by argo.

    Returns an array aligned with `close` where positions before enough history
    is available are filled with NaN.
    """

    series = pd.Series(np.asarray(close, dtype=float))
    delta = series.diff()
    gain = delta.where(delta > 0, 0.0)
    loss = (-delta).where(delta < 0, 0.0)

    avg_gain = gain.rolling(window=period, min_periods=period).mean()
    avg_loss = loss.rolling(window=period, min_periods=period).mean()

    rs = avg_gain / avg_loss
    rsi_values = 100.0 - (100.0 / (1.0 + rs))
    # When avg_loss == 0 and avg_gain > 0, argo returns 100 (perfect uptrend).
    rsi_values = rsi_values.where(avg_loss != 0, 100.0)
    # When both averages are zero the RSI is undefined; argo would also return 100.
    rsi_values = rsi_values.where(~((avg_gain == 0) & (avg_loss == 0)), 100.0)
    return rsi_values.to_numpy()

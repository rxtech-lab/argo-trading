"""CLI entrypoint for regenerating the comparison dataset and reference results.

Run from the `internal/indicator/test_data` folder:

    poetry run python -m comparison.main
"""

from __future__ import annotations

import os
import sys

from . import generate_data, runner

# Resolve the test_data directory regardless of where the script is invoked from.
TEST_DATA_DIR = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
PARQUET_PATH = os.path.join(TEST_DATA_DIR, "comparison_data.parquet")
RESULTS_DIR = os.path.join(TEST_DATA_DIR, "results")


def main() -> int:
    print(f"Generating {generate_data.NUM_ROWS} rows of OHLCV data ...")
    df = generate_data.generate_dataframe()
    generate_data.write_parquet(df, PARQUET_PATH)
    print(f"  wrote {PARQUET_PATH}")

    print("Running backtesting.py RSI strategy ...")
    result = runner.run_backtest(PARQUET_PATH)
    runner.write_results(result, RESULTS_DIR)
    print(f"  wrote results to {RESULTS_DIR}")

    summary = result.summary
    print("Summary:")
    for key in (
        "num_trades",
        "num_buys",
        "num_sells",
        "num_closed_positions",
        "realized_pnl",
        "final_equity",
    ):
        print(f"  {key}: {summary[key]}")
    return 0


if __name__ == "__main__":
    sys.exit(main())

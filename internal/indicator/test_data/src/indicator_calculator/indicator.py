from .duckdb import read_parquet_to_df
import pandas as pd


def calculate_rsi(data: pd.DataFrame) -> pd.DataFrame:
    """
    Calculate the Relative Strength Index (RSI) for the given market data.
    Args:
        data (pd.DataFrame): A dataframe containing market data with a 'close' column.
    Returns:
        pd.DataFrame: A dataframe with an additional 'rsi' column containing the RSI values.
    """
    # Create a copy of the input dataframe to avoid modifying the original
    result = data.copy()

    # Default RSI period
    period = 14

    # Calculate price changes
    delta = result['close'].diff()

    # Separate gains and losses
    gain = delta.where(delta > 0, 0)
    loss = -delta.where(delta < 0, 0)

    # Calculate average gains and losses over the specified period
    avg_gain = gain.rolling(window=period).mean()
    avg_loss = loss.rolling(window=period).mean()

    # Calculate relative strength (RS)
    rs = avg_gain / avg_loss

    # Calculate RSI
    result['rsi'] = 100 - (100 / (1 + rs))

    return result
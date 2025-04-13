import pandas as pd


def calculate_rsi(data: pd.DataFrame, period: int = 14) -> pd.DataFrame:
    """
    Calculate the Relative Strength Index (RSI) for the given market data.
    Args:
        data (pd.DataFrame): A dataframe containing market data with a 'close' column.
        period (int): The number of periods to use for the RSI calculation. Default is 14.
    Returns:
        pd.DataFrame: A dataframe with an additional 'rsi' column containing the RSI values.
    """
    # Create a copy of the input dataframe to avoid modifying the original
    result = data.copy()

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
    column_name = f"rsi_{period}"
    result[column_name] = 100 - (100 / (1 + rs))

    return result


def calculate_ema(data: pd.DataFrame, period: int = 14) -> pd.DataFrame:
    """
    Calculate the Exponential Moving Average (EMA) for the given market data.

    Args:
        data (pd.DataFrame): A dataframe containing market data with a specified column.
        period (int): The number of periods to use for the EMA calculation. Default is 14.
        column (str): The name of the column to calculate the EMA on. Default is 'close'.
    Returns:
        pd.DataFrame: A dataframe with an additional column containing the EMA values.
    """
    # Create a copy of the input dataframe to avoid modifying the original
    result = data.copy()

    # Calculate the EMA
    # First value of EMA is the SMA (Simple Moving Average) for the specified period
    result[f'ema_{period}'] = result["close"].ewm(span=period, adjust=False).mean()
    return result


def calculate_macd(data: pd.DataFrame,
                   fast_period: int = 12,
                   slow_period: int = 26,
                   signal_period: int = 9,
                   column: str = 'close') -> pd.DataFrame:
    """
    Calculate the Moving Average Convergence Divergence (MACD) for the given market data.

    Args:
        data (pd.DataFrame): A dataframe containing market data with a specified column.
        fast_period (int): The number of periods to use for the fast EMA calculation. Default is 12.
        slow_period (int): The number of periods to use for the slow EMA calculation. Default is 26.
        signal_period (int): The number of periods to use for the signal line calculation. Default is 9.
        column (str): The name of the column to calculate the MACD on. Default is 'close'.
    Returns:
        pd.DataFrame: A dataframe with additional columns for the MACD line, signal line, and histogram.
    """
    # Create a copy of the input dataframe to avoid modifying the original
    result = data.copy()

    # Create necessary columns filled with NaN
    result['ema_fast'] = pd.NA
    result['ema_slow'] = pd.NA
    result['macd_line'] = pd.NA
    result['signal_line'] = pd.NA
    result['macd_histogram'] = pd.NA

    # Calculate the fast EMA
    fast_multiplier = 2 / (fast_period + 1)
    # First fast EMA is SMA of first fast_period values
    result.loc[fast_period - 1, 'ema_fast'] = result[column].iloc[:fast_period].mean()
    for i in range(fast_period, len(result)):
        current_price = result.loc[i, column]
        previous_ema = result.loc[i - 1, 'ema_fast']
        result.loc[i, 'ema_fast'] = (current_price * fast_multiplier) + (previous_ema * (1 - fast_multiplier))

    # Calculate the slow EMA
    slow_multiplier = 2 / (slow_period + 1)
    # First slow EMA is SMA of first slow_period values
    result.loc[slow_period - 1, 'ema_slow'] = result[column].iloc[:slow_period].mean()
    for i in range(slow_period, len(result)):
        current_price = result.loc[i, column]
        previous_ema = result.loc[i - 1, 'ema_slow']
        result.loc[i, 'ema_slow'] = (current_price * slow_multiplier) + (previous_ema * (1 - slow_multiplier))

    # Calculate the MACD line (fast EMA - slow EMA)
    # MACD line will start from the slow period (since we need both EMAs)
    for i in range(slow_period - 1, len(result)):
        result.loc[i, 'macd_line'] = result.loc[i, 'ema_fast'] - result.loc[i, 'ema_slow']

    # Calculate the signal line (EMA of MACD line)
    signal_multiplier = 2 / (signal_period + 1)

    # Determine the first valid MACD index (needs both fast and slow EMAs)
    first_valid_macd = max(fast_period, slow_period) - 1

    # First signal line value is SMA of first signal_period MACD values
    first_signal_index = first_valid_macd + signal_period - 1
    if first_signal_index < len(result):
        valid_macd_values = result['macd_line'].iloc[first_valid_macd:first_signal_index + 1].dropna()
        if len(valid_macd_values) > 0:
            result.loc[first_signal_index, 'signal_line'] = valid_macd_values.mean()

            # Calculate signal line for the remaining data points
            for i in range(first_signal_index + 1, len(result)):
                current_macd = result.loc[i, 'macd_line']
                previous_signal = result.loc[i - 1, 'signal_line']
                result.loc[i, 'signal_line'] = (current_macd * signal_multiplier) + (
                            previous_signal * (1 - signal_multiplier))

    # Calculate the MACD histogram (MACD line - signal line)
    for i in range(first_signal_index, len(result)):
        result.loc[i, 'macd_histogram'] = result.loc[i, 'macd_line'] - result.loc[i, 'signal_line']

    # Drop the intermediate columns if you don't want them in the final result
    result = result.drop(['ema_fast', 'ema_slow'], axis=1)

    return result


def calculate_sma(data: pd.DataFrame, period: int = 20, column: str = 'close') -> pd.DataFrame:
    """
    Calculate the Simple Moving Average (SMA) for the given market data.

    Args:
        data (pd.DataFrame): A dataframe containing market data with a specified column.
        period (int): The number of periods to use for the moving average calculation. Default is 20.
        column (str): The name of the column to calculate the moving average on. Default is 'close'.
    Returns:
        pd.DataFrame: A dataframe with an additional column containing the moving average values.
    """
    # Create a copy of the input dataframe to avoid modifying the original
    result = data.copy()

    # Calculate the simple moving average
    # The first (period-1) values will be NaN
    result[f'sma_{period}'] = result[column].rolling(window=period).mean()

    return result
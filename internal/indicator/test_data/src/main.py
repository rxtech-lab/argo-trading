from indicator_calculator import calculate_rsi, calculate_ema, read_parquet_to_df, write_df_to_parquet, calculate_macd, calculate_sma


def main():
    # Path to the input parquet file
    input_path = '../input.parquet'

    # Path to the output parquet file
    output_path = '../test_data.parquet'

    # Read the input data
    data = read_parquet_to_df(input_path)

    # Calculate RSI 7, 14, 21
    for period in [7, 14, 21]:
        data = calculate_rsi(data, period=period)
        data = calculate_ema(data, period=period)
        data = calculate_sma(data, period=period)

    data = calculate_macd(data)
    # Only first 500 rows
    data = data.head(500)

    # Write the result to a new parquet file
    write_df_to_parquet(data, output_path)

    print(f"Indicators are calculated and saved to {output_path}")


if __name__ == "__main__":
    main()
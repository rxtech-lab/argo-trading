from indicator_calculator import calculate_rsi, read_parquet_to_df, write_df_to_parquet


def main():
    # Path to the input parquet file
    input_path = '../apple.parquet'

    # Path to the output parquet file
    output_path = '../test_data.parquet'

    # Read the input data
    data = read_parquet_to_df(input_path)

    # Calculate RSI
    result = calculate_rsi(data)

    # Write the result to a new parquet file
    write_df_to_parquet(result, output_path)

    print(f"RSI calculated and saved to {output_path}")


if __name__ == "__main__":
    main()
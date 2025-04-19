# Market CLI

This CLI tool allows you to download historical market data for a specified stock ticker from a chosen provider and save it using a selected writer format.

## Usage

```bash
# Run directly
go run cmd/market/main.go download [flags]

# Or, build the binary first (assuming output name is 'market')
# go build -o market cmd/market/main.go
# ./market download [flags]
```

## Flags

The `download` command accepts the following flags:

- `--ticker`, `-t` (**Required**): The stock ticker symbol to download data for (e.g., `AAPL`).
- `--start`, `-s` (**Required**): The start date for the data download. The format should be `YYYY-MM-DD` or any other format compatible with RFC3339 (e.g., `2023-01-01`).
- `--end`, `-e` (_Optional_): The end date for the data download. Format is the same as `--start`. Defaults to the current date.
- `--provider`, `-p` (_Optional_): The data provider to use.
  - Available: `polygon`
  - Planned: `binance`
  - Defaults to `polygon`.
  - **Note**: The Polygon provider likely requires the `POLYGON_API_KEY` environment variable to be set.
- `--writer`, `-w` (_Optional_): The format/writer to use for saving the data.
  - Available: `duckdb` (writes to Parquet format readable by DuckDB).
  - Defaults to `duckdb`.
- `--data`, `-d` (_Optional_): The directory where the output data file will be saved.
  - Defaults to `./data`.

## Data Providers

Currently supported providers:

- **Polygon.io**: Fetches data using the official Polygon REST client. Requires an API key set via the `POLYGON_API_KEY` environment variable.

Planned providers:

- **Binance**

## Data Writers

Currently supported writers:

- **DuckDB**: Writes the downloaded data to a `.parquet` file.

## Output

By default, the data is saved in the `data/` directory relative to where the command is run.

The output filename follows this pattern:

```
<TICKER>_<START_DATE>_<END_DATE>_<MULTIPLIER>_<TIMESPAN>.parquet
```

Where:

- `<TICKER>` is the ticker symbol.
- `<START_DATE>` is the start date in `YYYY-MM-DD` format.
- `<END_DATE>` is the end date in `YYYY-MM-DD` format.
- `<MULTIPLIER>` is currently hardcoded to `1`.
- `<TIMESPAN>` is currently hardcoded to `minute`.

**Example Filename:** `AAPL_2023-01-01_2023-12-31_1_minute.parquet`

## Example Command

Download Apple (AAPL) stock data for the year 2023 using the Polygon provider and saving it to the default `data` directory:

```bash
go run cmd/market/main.go download --ticker AAPL --start 2023-01-01 --end 2023-12-31
```

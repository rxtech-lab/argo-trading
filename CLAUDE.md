# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Argo Trading is a Go-based algorithmic trading framework for developing, testing, and deploying trading strategies. Strategies compile to WebAssembly (WASM) and run in an isolated plugin architecture. The framework supports backtesting with real market data from Polygon.io and Binance.

## Common Commands

```bash
# Setup and code generation
go mod download
make generate          # Generate protobuf code and run go generate

# Testing
go test ./...                      # Run all tests
go test -v ./internal/indicator    # Run tests in specific package
go test -v -race -cover ./...      # Full test with race detection

# Linting
./scripts/setup.sh     # Install linting tools (first time)
./scripts/lint.sh      # Run nilaway
golangci-lint run      # Run golangci-lint

# Building strategies (WASM)
cd examples/strategy && make build

# Building Swift framework
make build-swift-argo  # Builds macOS xcframework

# Market data download
go run cmd/market/main.go -ticker SPY -start 2024-01-01 -end 2024-12-31 -provider polygon -writer duckdb -data ./data

# Running backtests
go run cmd/backtest/main.go -strategy-wasm ./examples/strategy/strategy.wasm -config ./config/backtest-engine-v1-config.yaml -data "./data/*.parquet"
```

## Architecture

```
cmd/                    # CLI entry points (backtest, market, generate)
pkg/                    # Public API packages
  strategy/             # Protobuf-generated strategy interfaces (go-plugin RPC)
  marketdata/           # Market data client with provider/writer abstraction
  swift-argo/           # Swift bindings (gomobile)
internal/               # Private packages
  backtest/engine/      # BacktestEngineV1 - orchestrates simulations
  indicator/            # Technical indicators (RSI, MACD, EMA, Bollinger, ATR, etc.)
  runtime/wasm/         # WASM runtime using wazero
  types/                # Core types (Order, Signal, Trade, Position, MarketData)
  trading/              # TradingSystem interface for order simulation
examples/strategy/      # Example WASM strategy implementation
```

**Strategy-Host Communication**: Strategies run as WASM plugins communicating via gRPC (go-plugin). The host provides:
- Data access: `GetRange`, `ReadLastData`, `ExecuteSQL`
- Indicators: `ConfigureIndicator`, `GetSignal`
- Cache: `GetCache`, `SetCache` (strategies are stateless, store state here)
- Trading: `PlaceOrder`, `GetPositions`, `CancelOrder`

## Key Development Patterns

**Testing**: Use table-driven tests with `testify/suite`. For indicator and DB tests, use real datasources (not mocks). Reference: `internal/indicator/rsi_test.go`

**Mocking**: Generate mocks with `mockgen`. Mocks are in `/mocks/` directory.

**Strategies are stateless**: All strategy state must be stored in the cache via `SetCache`/`GetCache`.

**WASM compilation**: Strategies use `GOOS=wasip1 GOARCH=wasm` and register via `strategy.RegisterTradingStrategy()`.

**Database**: DuckDB handles all data storage and queries. Market data stored as Parquet files.

## Key Types

Located in `internal/types/`:
- `Signal`: Trading signals (BUY_LONG, SELL_LONG, CLOSE_POSITION, WAIT, ABORT)
- `Order`: Trading orders with validation
- `ExecuteOrder`: Order with optional take-profit/stop-loss
- `MarketData`: OHLCV + timestamp

## Configuration

- Engine config: `config/backtest-engine-v1-config.yaml`
- Strategy configs: `config/strategy/*.yaml`
- Environment: `.env` (contains POLYGON_API_KEY)

## CI/CD

GitHub Actions runs: generate → lint → compile WASM → test with coverage → Codecov upload

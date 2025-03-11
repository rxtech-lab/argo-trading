# Argo Trading Backtest Engine

This is a backtesting engine for trading strategies. It allows you to test trading strategies against historical market data and evaluate their performance.

## Features

- Backtest trading strategies against historical market data
- Track positions, orders, and trades
- Calculate performance statistics (win rate, P&L, Sharpe ratio, max drawdown)
- Compare strategy performance with buy-and-hold
- Support for multiple strategies

## Usage

```go
package main

import (
    "fmt"
    "log"

    "github.com/sirily11/argo-trading-go/src/engine"
    datasource "github.com/sirily11/argo-trading-go/src/engine/data_source"
    "github.com/sirily11/argo-trading-go/src/strategy"
)

func main() {
    // Create CSV data source
    csvSource := &datasource.CSVIterator{
        FilePath: "path/to/your/data.csv",
    }

    // Create backtest engine
    backtester := engine.NewBacktestEngineV1()

    // Set initial capital
    err := backtester.SetInitialCapital(10000.0)
    if err != nil {
        log.Fatalf("Failed to set initial capital: %v", err)
    }

    // Add market data source
    err = backtester.AddMarketDataSource(csvSource)
    if err != nil {
        log.Fatalf("Failed to add market data source: %v", err)
    }

    // Create and add strategy
    strategy := &YourStrategy{}
    err = backtester.AddStrategy(strategy, "")
    if err != nil {
        log.Fatalf("Failed to add strategy: %v", err)
    }

    // Run backtest
    err = backtester.Run()
    if err != nil {
        log.Fatalf("Failed to run backtest: %v", err)
    }

    // Print results
    trades := backtester.GetTrades()
    fmt.Printf("Total trades: %d\n", len(trades))

    stats := backtester.GetTradeStats()
    fmt.Printf("Win rate: %.2f%%\n", stats.WinRate*100)
    fmt.Printf("Total P&L: $%.2f\n", stats.TotalPnL)
    fmt.Printf("Sharpe ratio: %.2f\n", stats.SharpeRatio)
    fmt.Printf("Max drawdown: %.2f%%\n", stats.MaxDrawdown*100)
}
```

## Creating a Strategy

To create a trading strategy, implement the `strategy.TradingStrategy` interface:

```go
type TradingStrategy interface {
    // Initialize sets up the strategy with a configuration string and initial context
    Initialize(config string, initialContext StrategyContext) error

    // ProcessData processes new market data and generates signals
    ProcessData(ctx StrategyContext, data types.MarketData) ([]types.Order, error)

    // Name returns the name of the strategy
    Name() string
}
```

The `StrategyContext` provides access to historical data, current positions, pending orders, executed trades, and account balance.

## Example Strategy

The repository includes a simple moving average crossover strategy as an example:

```go
type SimpleMovingAverageCrossover struct {
    shortPeriod int
    longPeriod  int
    symbol      string
}

func (s *SimpleMovingAverageCrossover) Name() string {
    return fmt.Sprintf("SMA_Cross_%d_%d", s.shortPeriod, s.longPeriod)
}

func (s *SimpleMovingAverageCrossover) Initialize(config string, initialContext strategy.StrategyContext) error {
    // Parse config or use default values
    s.shortPeriod = 5
    s.longPeriod = 20
    s.symbol = "AAPL"
    return nil
}

func (s *SimpleMovingAverageCrossover) ProcessData(ctx strategy.StrategyContext, data types.MarketData) ([]types.Order, error) {
    // Implementation details...
}
```

## Data Format

The backtest engine expects market data in the following format:

```go
type MarketData struct {
    Time   time.Time `csv:"time"`
    Open   float64   `csv:"open"`
    High   float64   `csv:"high"`
    Low    float64   `csv:"low"`
    Close  float64   `csv:"close"`
    Volume float64   `csv:"volume"`
}
```

## Performance Statistics

The backtest engine calculates the following performance statistics:

- Total trades
- Winning trades
- Losing trades
- Win rate
- Average profit/loss
- Total P&L
- Sharpe ratio
- Maximum drawdown

It also compares the strategy performance with a buy-and-hold strategy.

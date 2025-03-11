package main

import (
	"log"
	"os"

	datasource "github.com/sirily11/argo-trading-go/src/engine/data_source"
	engine "github.com/sirily11/argo-trading-go/src/engine/engine_v1"
	"github.com/sirily11/argo-trading-go/src/strategy"
)

func main() {
	// Create CSV data source
	csvSource := &datasource.CSVIterator{
		FilePath: "./data/AAPL_2025-01-01_2025-01-31.csv",
	}

	config, err := os.ReadFile("./config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	// Create backtest engine
	backtester := engine.NewBacktestEngineV1()

	// Initialize backtest engine
	err = backtester.Initialize(string(config))
	if err != nil {
		log.Fatalf("Failed to initialize backtest engine: %v", err)
	}

	// Add market data source
	err = backtester.AddMarketDataSource(csvSource)
	if err != nil {
		log.Fatalf("Failed to add market data source: %v", err)
	}

	// Create and add strategy
	smaStrategy := strategy.NewExampleIndicatorStrategy("AAPL")
	err = backtester.AddStrategy(smaStrategy, "")
	if err != nil {
		log.Fatalf("Failed to add strategy: %v", err)
	}

	// Run backtest
	err = backtester.Run()
	if err != nil {
		log.Fatalf("Failed to run backtest: %v", err)
	}
}

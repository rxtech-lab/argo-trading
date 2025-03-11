package main

import (
	"log"
	"os"
	"path/filepath"

	datasource "github.com/sirily11/argo-trading-go/src/engine/data_source/csv"
	engine "github.com/sirily11/argo-trading-go/src/engine/engine_v1"
	"github.com/sirily11/argo-trading-go/src/strategy"
)

func listCsvFiles() ([]string, error) {
	files, err := filepath.Glob("./data/*.csv")
	if err != nil {
		return nil, err
	}
	return files, nil
}

func main() {
	config, err := os.ReadFile("./config/backtest_config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	files, err := listCsvFiles()
	if err != nil {
		log.Fatalf("Failed to list CSV files: %v", err)
	}

	for _, file := range files {
		log.Printf("Processing file: %s", file)
		// Create CSV data source
		csvSource := &datasource.CSVIterator{
			FilePath: file,
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
}

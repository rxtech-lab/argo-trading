package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/moznion/go-optional"
	engine_types "github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	engine "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/schollz/progressbar/v3"
)

func main() {
	// Define command-line flags
	configFlag := flag.String("config", "config/backtest-engine-v1-config.yaml", "Path to backtest engine configuration file")
	resultsFlag := flag.String("results", "results", "Path to results folder")
	dataPathFlag := flag.String("data", "data/*.parquet", "Path pattern to data files")
	strategyConfigFlag := flag.String("strategy-config", "config/strategy/*.yaml", "Path pattern to strategy configuration files")
	strategyWasmFlag := flag.String("strategy-wasm", "", "Path to strategy WASM file (required)")
	dbPathFlag := flag.String("db", ":memory:", "Path to database file")

	// Parse command-line flags
	flag.Parse()

	// Validate required parameters
	if *strategyWasmFlag == "" {
		fmt.Println("Error: -strategy-wasm flag is required")
		flag.Usage()
		os.Exit(1)
	}

	engine := engine.NewBacktestEngineV1()
	var progressBar *progressbar.ProgressBar

	// read config from the provided path
	config, err := os.ReadFile(*configFlag)
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	if err := engine.Initialize(string(config)); err != nil {
		log.Fatalf("Failed to initialize engine: %v", err)
	}

	// set the results folder
	engine.SetResultsFolder(*resultsFlag)

	// set the data path
	engine.SetDataPath(*dataPathFlag)

	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	engine.SetConfigPath(*strategyConfigFlag)

	datasource, err := datasource.NewDataSource(*dbPathFlag, logger)
	if err != nil {
		log.Fatalf("Failed to create datasource: %v", err)
	}
	engine.SetDataSource(datasource)

	// set strategy
	strategy_runtime, err := wasm.NewStrategyWasmRuntime(*strategyWasmFlag)
	if err != nil {
		log.Fatalf("Failed to create strategy runtime: %v", err)
	}
	engine.LoadStrategy(strategy_runtime)

	onProcessDataCallback := func(currentCount int, totalCount int) error {
		if progressBar == nil {
			progressBar = progressbar.New(totalCount)
			progressBar.Add(currentCount)
		}
		progressBar.Add(1)
		return nil
	}

	// run the engine
	err = engine.Run(optional.Some[engine_types.OnProcessDataCallback](onProcessDataCallback))
	if err != nil {
		log.Fatalf("Failed to run engine: %v", err)
	}
}

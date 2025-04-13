package main

import (
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
	engine := engine.NewBacktestEngineV1()
	var progressBar *progressbar.ProgressBar

	// read config from config/backtest_config.yaml
	config, err := os.ReadFile("config/backtest-engine-v1-config.yaml")
	if err != nil {
		log.Fatalf("Failed to read config: %v", err)
	}

	if err := engine.Initialize(string(config)); err != nil {
		log.Fatalf("Failed to initialize engine: %v", err)
	}

	// set the results folder
	engine.SetResultsFolder("results")

	// set the data path
	engine.SetDataPath("data/*.parquet")

	logger, err := logger.NewLogger()
	if err != nil {
		log.Fatalf("Failed to create logger: %v", err)
	}

	engine.SetConfigPath("config/strategy/*.yaml")

	datasource, err := datasource.NewDataSource(":memory:", logger)
	if err != nil {
		log.Fatalf("Failed to create datasource: %v", err)
	}
	engine.SetDataSource(datasource)

	// set strategy
	strategy_runtime, err := wasm.NewStrategyWasmRuntime("e2e/backtest/wasm/place_order/place_order_plugin.wasm")
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

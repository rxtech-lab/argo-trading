package main

import (
	"log"
	"os"

	engine "github.com/sirily11/argo-trading-go/src/engine/engine_v1"
	"github.com/sirily11/argo-trading-go/src/strategy"
)

func main() {
	engine := engine.NewBacktestEngineV1()

	// read config from config/backtest_config.yaml
	config, err := os.ReadFile("config/backtest_config.yaml")
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

	engine.SetConfigPath("config")

	// set strategy
	engine.LoadStrategy(strategy.NewExampleIndicatorStrategy("APPL"))

	// run the engine
	engine.Run()
}

package engine

import (
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
)

type OnProcessDataCallback func(current int, total int) error

type StrategyType string

const (
	StrategyTypeWASM StrategyType = "wasm"
)

type Engine interface {
	// Initialize the engine with the given configuration file.
	Initialize(config string) error
	// SetConfigPath sets the path to the strategy configuration file.
	SetConfigPath(path string) error
	// SetDataPath sets the path to the market data file. Supports loading data from:
	// 1. Multiple files for a single stock (e.g., AAPL_2020.parquet, AAPL_2021.parquet)
	// 2. Multiple stocks in separate files (e.g., AAPL_2020.parquet, GOOGL_2020.parquet)
	// Accepts glob patterns for batch loading (e.g., "data/*.parquet")
	SetDataPath(path string) error
	// SetResultsFolder sets the output directory for saving backtest results.
	// The results folder will be structured as: <symbol>_<year>_<strategy_name>_<config_name>
	// Example: AAPL_2020_MovingAverageCrossover_Default
	SetResultsFolder(folder string) error
	// LoadStrategy loads the trading strategy from the given strategy. Could be called multiple times to load multiple strategies.
	LoadStrategy(strategy runtime.StrategyRuntime) error
	// LoadStrategyFromFile loads the trading strategy from the given strategy file.
	LoadStrategyFromFile(strategyPath string) error
	// LoadStrategyFromBytes loads the trading strategy from the given strategy bytes.
	LoadStrategyFromBytes(strategyBytes []byte, strategyType StrategyType) error
	// Run runs the engine and executes the trading strategy
	Run(onProcessDataCallback optional.Option[OnProcessDataCallback]) error
	// SetDataSource sets the data source for the engine.
	SetDataSource(dataSource datasource.DataSource) error
	// GetConfigSchema returns the schema of the engine configuration
	GetConfigSchema() (string, error)
}

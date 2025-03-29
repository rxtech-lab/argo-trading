package engine

import (
	"github.com/sirily11/argo-trading-go/src/strategy"
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
	LoadStrategy(strategy strategy.TradingStrategy) error
	// Run runs the engine and executes the trading strategy
	Run() error
}

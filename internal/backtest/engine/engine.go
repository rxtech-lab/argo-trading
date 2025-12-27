package engine

import (
	"context"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
)

// Lifecycle callback types for backtest phases
// All callbacks with error return can abort execution if they return an error

// OnBacktestStartCallback is called when the entire backtest begins.
type OnBacktestStartCallback func(totalStrategies int, totalConfigs int, totalDataFiles int) error

// OnBacktestEndCallback is called when the entire backtest completes (always called via defer).
type OnBacktestEndCallback func(err error)

// OnStrategyStartCallback is called when a strategy iteration begins.
type OnStrategyStartCallback func(strategyIndex int, strategyName string, totalStrategies int) error

// OnStrategyEndCallback is called when a strategy iteration ends.
type OnStrategyEndCallback func(strategyIndex int, strategyName string)

// OnRunStartCallback is called when processing of a config+data file combination begins.
// runID is a unique identifier for this run, generated before processing starts.
type OnRunStartCallback func(runID string, configIndex int, configName string, dataFileIndex int, dataFilePath string, totalDataPoints int) error

// OnRunEndCallback is called when processing of a config+data file combination ends.
type OnRunEndCallback func(configIndex int, configName string, dataFileIndex int, dataFilePath string, resultFolderPath string)

// OnProcessDataCallback is called for each data point processed.
type OnProcessDataCallback func(current int, total int) error

// LifecycleCallbacks holds all lifecycle callback functions for the backtest engine.
// All fields are pointers - nil means no callback will be invoked.
type LifecycleCallbacks struct {
	OnBacktestStart *OnBacktestStartCallback
	OnBacktestEnd   *OnBacktestEndCallback
	OnStrategyStart *OnStrategyStartCallback
	OnStrategyEnd   *OnStrategyEndCallback
	OnRunStart      *OnRunStartCallback
	OnRunEnd        *OnRunEndCallback
	OnProcessData   *OnProcessDataCallback
}

type StrategyType string

const (
	StrategyTypeWASM StrategyType = "wasm"
)

//nolint:interfacebloat // Engine is a core interface that naturally requires multiple methods
type Engine interface {
	// Initialize the engine with the given configuration file.
	Initialize(config string) error
	// SetConfigPath sets the path to the strategy configuration file.
	SetConfigPath(path string) error
	// SetConfigContent sets strategy configurations directly from string content.
	// This is an alternative to SetConfigPath for programmatic API usage.
	SetConfigContent(configs []string) error
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
	// Run runs the engine and executes the trading strategy.
	// The context can be used to cancel the backtest operation.
	// Use LifecycleCallbacks to receive notifications at different phases of the backtest.
	Run(ctx context.Context, callbacks LifecycleCallbacks) error
	// SetDataSource sets the data source for the engine.
	SetDataSource(dataSource datasource.DataSource) error
	// GetConfigSchema returns the schema of the engine configuration
	GetConfigSchema() (string, error)
}

package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/indicator"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/marker"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/trading"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1 struct {
	config              BacktestEngineV1Config
	strategies          []runtime.StrategyRuntime
	strategyConfigPaths []string
	dataPaths           []string
	resultsFolder       string
	log                 *logger.Logger
	indicatorRegistry   indicator.IndicatorRegistry
	marker              marker.Marker
	tradingSystem       trading.TradingSystem
	state               *BacktestState
	datasource          datasource.DataSource
	balance             float64
	cache               cache.Cache
}

func NewBacktestEngineV1() engine.Engine {
	return &BacktestEngineV1{
		cache: cache.NewCacheV1(),
	}
}

// Initialize implements engine.Engine.
func (b *BacktestEngineV1) Initialize(config string) error {
	// parse the config
	err := yaml.Unmarshal([]byte(config), &b.config)
	if err != nil {
		return err
	}

	// initialize the logger
	var loggerError error
	b.log, loggerError = logger.NewLogger()
	if loggerError != nil {
		return loggerError
	}

	b.log.Debug("Backtest engine initialized",
		zap.String("config", config),
	)

	// initialize the indicator registry
	b.indicatorRegistry = indicator.NewIndicatorRegistry()
	b.indicatorRegistry.RegisterIndicator(indicator.NewBollingerBands())
	b.indicatorRegistry.RegisterIndicator(indicator.NewEMA())
	b.indicatorRegistry.RegisterIndicator(indicator.NewMACD())
	b.indicatorRegistry.RegisterIndicator(indicator.NewATR())
	b.indicatorRegistry.RegisterIndicator(indicator.NewWaddahAttar())
	b.indicatorRegistry.RegisterIndicator(indicator.NewRSI())
	b.indicatorRegistry.RegisterIndicator(indicator.NewMA())

	// initialize the state
	b.state = NewBacktestState(b.log)
	if err := b.state.Initialize(); err != nil {
		return fmt.Errorf("failed to initialize state: %w", err)
	}
	b.balance = b.config.InitialCapital
	b.tradingSystem = NewBacktestTrading(b.state, b.config.InitialCapital, commission_fee.NewInteractiveBrokerCommissionFee())
	return nil
}

// LoadStrategy implements engine.Engine.
func (b *BacktestEngineV1) LoadStrategy(strategy runtime.StrategyRuntime) error {
	b.strategies = append(b.strategies, strategy)
	b.log.Debug("Strategy loaded",
		zap.Int("total_strategies", len(b.strategies)),
	)
	return nil
}

// SetConfigPath implements engine.Engine.
func (b *BacktestEngineV1) SetConfigPath(path string) error {
	// use glob to get all the files that match the path
	files, err := filepath.Glob(path)
	if err != nil {
		b.log.Error("Failed to set config path",
			zap.String("path", path),
			zap.Error(err),
		)
		return err
	}

	b.strategyConfigPaths = files
	b.log.Debug("Config paths set",
		zap.Strings("files", files),
	)
	return nil
}

// SetDataPath implements engine.Engine.
func (b *BacktestEngineV1) SetDataPath(path string) error {
	// use glob to get all the files that match the path
	files, err := filepath.Glob(path)
	if err != nil {
		b.log.Error("Failed to set data path",
			zap.String("path", path),
			zap.Error(err),
		)
		return err
	}

	// Convert all paths to absolute paths
	absolutePaths := make([]string, len(files))
	for i, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			b.log.Error("Failed to get absolute path",
				zap.String("path", file),
				zap.Error(err),
			)
			return err
		}
		absolutePaths[i] = absPath
	}

	b.dataPaths = absolutePaths
	b.log.Debug("Data paths set",
		zap.Strings("files", absolutePaths),
	)
	return nil
}

// SetResultsFolder implements engine.Engine.
func (b *BacktestEngineV1) SetResultsFolder(folder string) error {
	b.resultsFolder = folder
	b.log.Debug("Results folder set",
		zap.String("folder", folder),
	)
	return nil
}

func (b *BacktestEngineV1) SetDataSource(datasource datasource.DataSource) error {
	b.datasource = datasource
	return nil
}

func (b *BacktestEngineV1) preRunCheck() error {
	if len(b.strategies) == 0 {
		b.log.Error("No strategies loaded")
		return errors.New("no strategies loaded")
	}

	if len(b.strategyConfigPaths) == 0 {
		b.log.Error("No strategy config paths loaded")
		return errors.New("no strategy config paths loaded")
	}

	if len(b.dataPaths) == 0 {
		b.log.Error("No data paths loaded")
		return errors.New("no data paths loaded")
	}

	if b.resultsFolder == "" {
		b.log.Error("No results folder set")
		return errors.New("no results folder set")
	}

	if b.datasource == nil {
		b.log.Error("No datasource set")
		return errors.New("no datasource set")
	}
	return nil
}

// ParallelRunState holds the state for a single parallel run
type ParallelRunState struct {
	state      *BacktestState
	balance    float64
	datasource datasource.DataSource
}

// Run implements engine.Engine.
func (b *BacktestEngineV1) Run(onProcessDataCallback optional.Option[engine.OnProcessDataCallback]) error {
	if err := b.preRunCheck(); err != nil {
		return err
	}

	// clean the results folder
	// remove results folder if it exists
	if _, err := os.Stat(b.resultsFolder); err == nil {
		os.RemoveAll(b.resultsFolder)
	}
	// create results folder
	os.MkdirAll(b.resultsFolder, 0755)

	// Run strategies sequentially
	for _, strategy := range b.strategies {
		for _, configPath := range b.strategyConfigPaths {
			config, err := os.ReadFile(configPath)
			if err != nil {
				b.log.Error("Failed to read config",
					zap.String("config", configPath),
					zap.Error(err),
				)
				return err
			}
			for _, dataPath := range b.dataPaths {
				// Initialize the state
				if err := b.state.Initialize(); err != nil {
					return fmt.Errorf("failed to initialize state: %w", err)
				}

				strategyContext := runtime.RuntimeContext{
					DataSource:        b.datasource,
					IndicatorRegistry: b.indicatorRegistry,
					Marker:            b.marker,
					TradingSystem:     b.tradingSystem,
					Cache:             b.cache,
				}

				// need to initialize the strategy api first since there is no wasm plugin available before this line
				err = strategy.InitializeApi(wasm.NewWasmStrategyApi(&strategyContext))
				if err != nil {
					return fmt.Errorf("failed to initialize strategy api: %w", err)
				}
				err = strategy.Initialize(string(config))
				if err != nil {
					return fmt.Errorf("failed to initialize strategy: %w", err)
				}
				resultFolderPath := getResultFolder(configPath, dataPath, b, strategy)

				b.log.Debug("Running strategy",
					zap.String("strategy", strategy.Name()),
					zap.String("config", configPath),
					zap.String("data", dataPath),
					zap.String("result", resultFolderPath),
				)

				// Initialize the data source with the given data path
				if err := b.datasource.Initialize(dataPath); err != nil {
					return fmt.Errorf("failed to initialize data source: %w", err)
				}

				// create a progress bar
				count, err := b.datasource.Count(b.config.StartTime, b.config.EndTime)
				if err != nil {
					return fmt.Errorf("failed to get data count: %w", err)
				}

				currentCount := 0
				for data, err := range b.datasource.ReadAll(b.config.StartTime, b.config.EndTime) {
					if err != nil {
						return fmt.Errorf("failed to read data: %w", err)
					}
					// run the strategy
					if backtestTrading, ok := b.tradingSystem.(*BacktestTrading); ok {
						backtestTrading.UpdateCurrentMarketData(data)
					}
					err = strategy.ProcessData(data)
					if err != nil {
						return fmt.Errorf("failed to process data: %w", err)
					}
					// Update progress bar
					onProcessDataCallback.IfSome(func(callback engine.OnProcessDataCallback) {
						callback(currentCount, count)
					})
					currentCount++
				}

				// Write results and cleanup
				if err := b.state.Write(resultFolderPath); err != nil {
					return fmt.Errorf("failed to write results: %w", err)
				}

				stats, err := b.state.GetStats(strategyContext)
				if err != nil {
					return fmt.Errorf("failed to get stats: %w", err)
				}
				if err := types.WriteTradeStats(filepath.Join(resultFolderPath, "stats.yaml"), stats); err != nil {
					return fmt.Errorf("failed to write stats: %w", err)
				}

				if err := b.state.Cleanup(); err != nil {
					return fmt.Errorf("failed to cleanup state: %w", err)
				}
			}
		}
	}
	return nil
}

func (b *BacktestEngineV1) GetStrategyApi() (strategy.StrategyApi, error) {
	strategyApi := wasm.NewWasmStrategyApi(&runtime.RuntimeContext{
		DataSource:        b.datasource,
		IndicatorRegistry: b.indicatorRegistry,
		Marker:            b.marker,
		TradingSystem:     b.tradingSystem,
		Cache:             b.cache,
	})

	return strategyApi, nil
}

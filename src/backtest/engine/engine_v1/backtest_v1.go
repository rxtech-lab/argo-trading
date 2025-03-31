package engine

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/sirily11/argo-trading-go/src/backtest/engine"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/indicator"
	"github.com/sirily11/argo-trading-go/src/logger"
	s "github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/vbauerster/mpb/v8"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1 struct {
	config              BacktestEngineV1Config
	strategies          []s.TradingStrategy
	strategyConfigPaths []string
	dataPaths           []string
	resultsFolder       string
	log                 *logger.Logger
	indicatorRegistry   *indicator.IndicatorRegistry
	state               *BacktestState
	balance             float64
}

func NewBacktestEngineV1() engine.Engine {
	return &BacktestEngineV1{}
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
	b.indicatorRegistry.RegisterIndicator(indicator.NewBollingerBands(20, 2, time.Hour*24*30))
	b.state = NewBacktestState(b.log)
	b.balance = b.config.InitialCapital
	return nil
}

// LoadStrategy implements engine.Engine.
func (b *BacktestEngineV1) LoadStrategy(strategy s.TradingStrategy) error {
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
	return nil
}

// ParallelRunState holds the state for a single parallel run
type ParallelRunState struct {
	state      *BacktestState
	balance    float64
	datasource datasource.DataSource
}

// Run implements engine.Engine.
func (b *BacktestEngineV1) Run() error {
	if err := b.preRunCheck(); err != nil {
		return err
	}

	// Create a channel to collect errors from goroutines
	errChan := make(chan error, len(b.strategies)*len(b.strategyConfigPaths)*len(b.dataPaths))
	var wg sync.WaitGroup

	// Create progress container
	p := mpb.New(mpb.WithWaitGroup(&wg))
	// Create progress bars
	bars, err := createProgressBars(ProgressBarConfig{
		Progress:    p,
		Strategies:  b.strategies,
		ConfigPaths: b.strategyConfigPaths,
		DataPaths:   b.dataPaths,
		Logger:      b.log,
		StartTime:   b.config.StartTime,
		EndTime:     b.config.EndTime,
	})
	if err != nil {
		return fmt.Errorf("failed to create progress bars: %w", err)
	}

	// Start the goroutines
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
			err = strategy.Initialize(string(config))
			if err != nil {
				b.log.Error("Failed to initialize strategy",
					zap.String("strategy", strategy.Name()),
					zap.Error(err),
				)
			}
			for _, dataPath := range b.dataPaths {
				wg.Add(1)
				go func(strategy s.TradingStrategy, configPath, dataPath string) {
					defer wg.Done()

					key := runKey{
						strategy:   strategy.Name(),
						configPath: configPath,
						dataPath:   dataPath,
					}
					bar := bars[key]

					// Create a new state for this parallel run
					runState := &ParallelRunState{
						state:   NewBacktestState(b.log),
						balance: b.config.InitialCapital,
					}

					// Initialize the state
					if err := runState.state.Initialize(); err != nil {
						errChan <- fmt.Errorf("failed to initialize state: %w", err)
						return
					}

					resultFolderPath := getResultFolder(configPath, dataPath, b, strategy)

					b.log.Debug("Running strategy",
						zap.String("strategy", strategy.Name()),
						zap.String("config", configPath),
						zap.String("data", dataPath),
						zap.String("result", resultFolderPath),
					)

					// Initialize the data source with in-memory database
					datasource, err := datasource.NewDataSource(":memory:", b.log)
					if err != nil {
						errChan <- fmt.Errorf("failed to create data source: %w", err)
						return
					}
					runState.datasource = datasource

					strategyContext := s.StrategyContext{
						DataSource:        datasource,
						IndicatorRegistry: b.indicatorRegistry,
						GetPosition:       runState.state.GetPosition,
					}

					// Initialize the data source with the given data path
					if err := datasource.Initialize(dataPath); err != nil {
						errChan <- fmt.Errorf("failed to initialize data source: %w", err)
						return
					}

					for data, err := range datasource.ReadAll(b.config.StartTime, b.config.EndTime) {
						if err != nil {
							errChan <- fmt.Errorf("failed to read data: %w", err)
							return
						}
						// run the strategy
						executeOrders, err := strategy.ProcessData(strategyContext, data, data.Symbol)
						if err != nil {
							errChan <- fmt.Errorf("failed to process data: %w", err)
							return
						}
						_, err = b.executeOrdersWithState(data, strategy, executeOrders, runState)
						if err != nil {
							errChan <- fmt.Errorf("failed to execute orders: %w", err)
							return
						}
						// Update progress bar
						bar.Increment()
					}

					// Write results and cleanup
					if err := runState.state.Write(resultFolderPath); err != nil {
						errChan <- fmt.Errorf("failed to write results: %w", err)
						return
					}

					stats, err := runState.state.GetStats(strategyContext)
					if err != nil {
						errChan <- fmt.Errorf("failed to get stats: %w", err)
						return
					}
					if err := types.WriteTradeStats(filepath.Join(resultFolderPath, "stats.yaml"), stats); err != nil {
						errChan <- fmt.Errorf("failed to write stats: %w", err)
						return
					}

					if err := runState.state.Cleanup(); err != nil {
						errChan <- fmt.Errorf("failed to cleanup state: %w", err)
						return
					}
				}(strategy, configPath, dataPath)
			}
		}
	}

	// Wait for all progress bars to complete
	p.Wait()
	close(errChan)

	// Check for any errors
	var errors []error
	for err := range errChan {
		errors = append(errors, err)
	}

	if len(errors) > 0 {
		return fmt.Errorf("encountered %d errors during parallel execution: %v", len(errors), errors)
	}

	return nil
}

// executeOrdersWithState executes the orders using the provided state
func (b *BacktestEngineV1) executeOrdersWithState(marketData types.MarketData, strategy s.TradingStrategy, executeOrders []types.ExecuteOrder, runState *ParallelRunState) ([]types.Order, error) {
	orders := []types.Order{}
	pendingOrders := []types.Order{}
	totalCost := 0.0

	for _, executeOrder := range executeOrders {
		position, err := runState.state.GetPosition(executeOrder.Symbol)
		price := (marketData.High + marketData.Low) / 2
		commissionFeeHandler := commission_fee.GetCommissionFeeHandler(b.config.Broker)
		if err != nil {
			b.log.Error("Failed to get position",
				zap.String("symbol", executeOrder.Symbol),
				zap.Error(err),
			)
			return nil, err
		}

		quantity := 0.0
		if executeOrder.OrderType == types.OrderTypeBuy {
			// calculate the quantity by the current balance / price
			quantity = float64(CalculateMaxQuantity(runState.balance, price, commissionFeeHandler))
		} else {
			quantity = position.Quantity
		}
		totalCost += quantity*price + commissionFeeHandler.Calculate(quantity)

		if position.Quantity == 0 {
			pendingOrder := types.Order{
				Symbol:       executeOrder.Symbol,
				OrderType:    executeOrder.OrderType,
				Quantity:     quantity,
				Price:        (marketData.High + marketData.Low) / 2,
				Timestamp:    marketData.Time,
				StrategyName: strategy.Name(),
				Reason: types.Reason{
					Reason:  executeOrder.Reason.Reason,
					Message: executeOrder.Reason.Message,
				},
			}
			pendingOrders = append(pendingOrders, pendingOrder)
		}
	}

	results, err := runState.state.Update(pendingOrders)
	if err != nil {
		b.log.Error("Failed to update state",
			zap.Error(err),
		)
		return nil, err
	}

	for _, result := range results {
		orders = append(orders, result.Order)
	}
	runState.balance -= totalCost

	return orders, nil
}

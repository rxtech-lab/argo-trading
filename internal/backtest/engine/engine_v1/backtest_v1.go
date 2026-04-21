package engine

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
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
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1 struct {
	config              BacktestEngineV1Config
	strategies          []runtime.StrategyRuntime
	strategyPaths       []string
	strategyBytes       [][]byte
	strategyConfigPaths []string
	strategyConfigs     []string
	dataPaths           []string
	resultsFolder       string
	log                 *logger.Logger
	indicatorRegistry   indicator.IndicatorRegistry
	marker              marker.Marker
	tradingSystem       tradingprovider.TradingSystemProvider
	state               *BacktestState
	datasource          datasource.DataSource
	balance             float64
	cache               cache.Cache
	logStorage          *BacktestLog
}

func NewBacktestEngineV1() (engine.Engine, error) {
	log, err := logger.NewLogger()
	if err != nil {
		return nil, err
	}

	return &BacktestEngineV1{
		config:              EmptyConfig(),
		strategies:          nil,
		strategyPaths:       nil,
		strategyBytes:       nil,
		strategyConfigPaths: nil,
		strategyConfigs:     nil,
		dataPaths:           nil,
		resultsFolder:       "",
		log:                 log,
		indicatorRegistry:   nil,
		marker:              nil,
		tradingSystem:       nil,
		state:               nil,
		datasource:          nil,
		balance:             0,
		cache:               cache.NewCacheV1(),
		logStorage:          nil,
	}, nil
}

// Initialize implements engine.Engine.
func (b *BacktestEngineV1) Initialize(config string) error {
	// parse the config
	err := yaml.Unmarshal([]byte(config), &b.config)
	if err != nil {
		return err
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
	b.state, err = NewBacktestState(b.log)
	if err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to create backtest state", err)
	}

	if err := b.state.Initialize(); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize state", err)
	}

	b.state.SetInitialBalance(b.config.InitialCapital)
	b.balance = b.config.InitialCapital
	// Use the configured broker for the commission fee and decimal precision for quantity precision
	var commissionFee commission_fee.CommissionFee

	switch b.config.Broker {
	case commission_fee.BrokerInteractiveBroker:
		commissionFee = commission_fee.NewInteractiveBrokerCommissionFee()
	case commission_fee.BrokerZero:
		commissionFee = commission_fee.NewZeroCommissionFee()
	default:
		commissionFee = commission_fee.NewInteractiveBrokerCommissionFee()
	}

	b.tradingSystem = NewBacktestTrading(b.state, b.config.InitialCapital, commissionFee, b.config.DecimalPrecision)

	return nil
}

// LoadStrategy implements engine.Engine.
func (b *BacktestEngineV1) LoadStrategy(strategy runtime.StrategyRuntime) error {
	b.strategies = append(b.strategies, strategy)
	b.strategyPaths = append(b.strategyPaths, "")
	b.strategyBytes = append(b.strategyBytes, nil)
	b.log.Debug("Strategy loaded",
		zap.Int("total_strategies", len(b.strategies)),
	)

	return nil
}

func (b *BacktestEngineV1) LoadStrategyFromFile(strategyPath string) error {
	// get extension
	extension := filepath.Ext(strategyPath)

	var strategy runtime.StrategyRuntime

	var err error

	switch extension {
	case ".wasm":
		strategy, err = wasm.NewStrategyWasmRuntime(strategyPath)
		if err != nil {
			return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to create strategy runtime", err)
		}
	default:
		return errors.Newf(errors.ErrCodeUnsupportedStrategy, "unsupported strategy type: %s", extension)
	}

	b.strategies = append(b.strategies, strategy)
	b.strategyPaths = append(b.strategyPaths, strategyPath)
	b.strategyBytes = append(b.strategyBytes, nil)
	b.log.Debug("Strategy loaded",
		zap.Int("total_strategies", len(b.strategies)),
	)

	return nil
}

func (b *BacktestEngineV1) LoadStrategyFromBytes(strategyBytes []byte, strategyType engine.StrategyType) error {
	var strategy runtime.StrategyRuntime

	var err error

	switch strategyType {
	case engine.StrategyTypeWASM:
		strategy, err = wasm.NewStrategyWasmRuntimeFromBytes(strategyBytes)
		if err != nil {
			return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to create strategy runtime", err)
		}
	default:
		return errors.Newf(errors.ErrCodeUnsupportedStrategy, "unsupported strategy type: %s", strategyType)
	}

	b.strategies = append(b.strategies, strategy)
	b.strategyPaths = append(b.strategyPaths, "")
	// Store a defensive copy of the bytes so that parallel workers can clone the
	// strategy runtime later without sharing slice memory with the caller.
	bytesCopy := make([]byte, len(strategyBytes))
	copy(bytesCopy, strategyBytes)
	b.strategyBytes = append(b.strategyBytes, bytesCopy)
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

// SetConfigContent implements engine.Engine.
func (b *BacktestEngineV1) SetConfigContent(configs []string) error {
	b.strategyConfigs = configs
	b.strategyConfigPaths = nil
	b.log.Debug("Config content set",
		zap.Int("count", len(configs)),
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

	// Create and set the datasource (in-memory)
	ds, err := datasource.NewDataSource(":memory:", b.log)
	if err != nil {
		b.log.Error("Failed to create datasource",
			zap.Error(err),
		)

		return err
	}

	b.datasource = ds

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

// iterResources groups all per-iteration resources used while running a single
// (config, dataPath) combination. When the engine runs sequentially it points at
// the engine-level fields. When the engine runs in parallel mode each worker
// owns its own iterResources so that workers don't share mutable state.
type iterResources struct {
	state             *BacktestState
	tradingSystem     tradingprovider.TradingSystemProvider
	marker            marker.Marker
	logStorage        *BacktestLog
	cache             cache.Cache
	datasource        datasource.DataSource
	strategy          runtime.StrategyRuntime
	indicatorRegistry indicator.IndicatorRegistry
}

// configItem is a strategy configuration alongside its identifying name.
type configItem struct {
	name    string
	content string
}

// runIterationParams holds parameters for a single run iteration.
type runIterationParams struct {
	ctx              context.Context
	res              *iterResources
	strategyPath     string
	runID            string
	configIdx        int
	configName       string
	configContent    string
	dataIdx          int
	dataPath         string
	callbacks        engine.LifecycleCallbacks
	resultFolderPath string
}

// Run implements engine.Engine.
func (b *BacktestEngineV1) Run(ctx context.Context, callbacks engine.LifecycleCallbacks) error {
	if err := b.preRunCheck(); err != nil {
		return err
	}

	// Create timestamped subfolder for this backtest session
	timestamp := time.Now().Format("20060102_150405")
	sessionFolder := filepath.Join(b.resultsFolder, timestamp)
	os.MkdirAll(sessionFolder, 0755)
	b.resultsFolder = sessionFolder

	// Build config list from either file paths or direct content
	var configs []configItem

	if len(b.strategyConfigs) > 0 {
		for i, content := range b.strategyConfigs {
			configs = append(configs, configItem{
				name:    fmt.Sprintf("config_%d", i),
				content: content,
			})
		}
	} else {
		for _, configPath := range b.strategyConfigPaths {
			content, err := os.ReadFile(configPath)
			if err != nil {
				b.log.Error("Failed to read config",
					zap.String("config", configPath),
					zap.Error(err),
				)

				return err
			}

			configs = append(configs, configItem{
				name:    configPath,
				content: string(content),
			})
		}
	}

	// Track any error for OnBacktestEnd callback
	var runErr error

	// Ensure OnBacktestEnd is always called
	defer func() {
		if callbacks.OnBacktestEnd != nil {
			(*callbacks.OnBacktestEnd)(runErr)
		}
	}()

	// Invoke OnBacktestStart callback
	if callbacks.OnBacktestStart != nil {
		if err := (*callbacks.OnBacktestStart)(len(b.strategies), len(configs), len(b.dataPaths)); err != nil {
			runErr = errors.Wrap(errors.ErrCodeCallbackFailed, "OnBacktestStart callback failed", err)

			return runErr
		}
	}

	// Run strategies sequentially
	for strategyIdx, strategy := range b.strategies {
		strategyPath := b.strategyPaths[strategyIdx]

		// Invoke OnStrategyStart callback
		if callbacks.OnStrategyStart != nil {
			if err := (*callbacks.OnStrategyStart)(strategyIdx, strategy.Name(), len(b.strategies)); err != nil {
				runErr = errors.Wrap(errors.ErrCodeCallbackFailed, "OnStrategyStart callback failed", err)

				return runErr
			}
		}

		// Decide between sequential and parallel execution.
		// Parallel mode kicks in when MaxConcurrency > 1 AND there is more than
		// one (config, data) iteration to distribute. With only a single
		// iteration there is no benefit to parallelism, so the sequential path
		// is preferred to preserve existing behavior and avoid the cost of
		// cloning per-worker resources.
		totalIterations := len(configs) * len(b.dataPaths)
		concurrency := b.config.MaxConcurrency
		if concurrency < 1 {
			concurrency = 1
		}

		if concurrency > 1 && totalIterations > 1 {
			if err := b.runStrategyParallel(ctx, strategyIdx, strategy, strategyPath, configs, callbacks, concurrency); err != nil {
				runErr = err

				return runErr
			}
		} else {
			// Sequential path - reuse the engine-level resources for compatibility
			// with existing tests and dependency injection patterns.
			res := &iterResources{
				state:             b.state,
				tradingSystem:     b.tradingSystem,
				marker:            b.marker,
				logStorage:        b.logStorage,
				cache:             b.cache,
				datasource:        b.datasource,
				strategy:          strategy,
				indicatorRegistry: b.indicatorRegistry,
			}

			for configIdx, cfg := range configs {
				for dataIdx, dataPath := range b.dataPaths {
					resultFolderPath := getResultFolder(cfg.name, dataPath, b, strategy)
					runID := uuid.New().String()

					params := runIterationParams{
						ctx:              ctx,
						res:              res,
						strategyPath:     strategyPath,
						runID:            runID,
						configIdx:        configIdx,
						configName:       cfg.name,
						configContent:    cfg.content,
						dataIdx:          dataIdx,
						dataPath:         dataPath,
						callbacks:        callbacks,
						resultFolderPath: resultFolderPath,
					}

					if err := b.runSingleIteration(params); err != nil {
						runErr = err

						return runErr
					}
				}
			}

			// Reflect any newly-created marker/log storage back onto the engine
			// so that callers depending on b.marker / b.logStorage observe them.
			b.marker = res.marker
			b.logStorage = res.logStorage
		}

		// Invoke OnStrategyEnd callback
		if callbacks.OnStrategyEnd != nil {
			(*callbacks.OnStrategyEnd)(strategyIdx, strategy.Name())
		}
	}

	return nil
}

func (b *BacktestEngineV1) GetConfigSchema() (string, error) {
	config := b.config

	schema, err := config.GenerateSchemaJSON()
	if err != nil {
		return "", errors.Wrap(errors.ErrCodeBacktestConfigError, "failed to generate schema", err)
	}

	return schema, nil
}

// runSingleIteration processes a single config+data combination using the
// per-iteration resources supplied via params.res.
func (b *BacktestEngineV1) runSingleIteration(params runIterationParams) error {
	res := params.res
	if res == nil {
		return errors.New(errors.ErrCodeBacktestStateNil, "iteration resources are nil")
	}

	if res.state == nil {
		return errors.New(errors.ErrCodeBacktestStateNil, "backtest state is nil")
	}

	var err error

	// Only create a new marker if one isn't already set (allows dependency injection in tests)
	if res.marker == nil {
		res.marker, err = NewBacktestMarker(b.log)
		if err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to create backtest marker", err)
		}
	}

	// Only create a new log storage if one isn't already set (allows dependency injection in tests)
	if res.logStorage == nil {
		res.logStorage, err = NewBacktestLog(b.log)
		if err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to create backtest log storage", err)
		}
	}

	if err := res.state.Initialize(); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize state", err)
	}

	// Wrap the datasource with a sliding window cache to improve performance
	// when multiple indicators query similar historical data within the same bar.
	// The sliding window cache stores market data as it's processed, allowing
	// strategies to access recent data without hitting DuckDB.
	slidingWindowDS := datasource.NewSlidingWindowDataSource(res.datasource, b.config.MarketDataCacheSize)

	strategyContext := runtime.RuntimeContext{
		DataSource:        slidingWindowDS,
		IndicatorRegistry: res.indicatorRegistry,
		Marker:            res.marker,
		TradingSystem:     res.tradingSystem,
		Cache:             res.cache,
		Logger:            b.log,
		LogStorage:        res.logStorage,
		CurrentMarketData: nil,
	}

	// need to initialize the strategy api first since there is no wasm plugin available before this line
	err = res.strategy.InitializeApi(wasm.NewWasmStrategyApi(&strategyContext))
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to initialize strategy api", err)
	}

	// Check version compatibility between engine and strategy
	strategyRuntimeVersion, err := res.strategy.GetRuntimeEngineVersion()
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to get strategy runtime version", err)
	}

	if err := version.CheckVersionCompatibility(version.Version, strategyRuntimeVersion); err != nil {
		return errors.Wrapf(errors.ErrCodeVersionMismatch, err, "version mismatch: engine version %s is incompatible with strategy compiled for version %s",
			version.Version, strategyRuntimeVersion)
	}

	err = res.strategy.Initialize(params.configContent)
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to initialize strategy", err)
	}

	b.log.Debug("Running strategy",
		zap.String("strategy", res.strategy.Name()),
		zap.String("config", params.configName),
		zap.String("data", params.dataPath),
		zap.String("result", params.resultFolderPath),
	)

	// Initialize the data source with the given data path
	if err := res.datasource.Initialize(params.dataPath); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestDataPathError, "failed to initialize data source", err)
	}

	// create a progress bar
	count, err := res.datasource.Count(b.config.StartTime, b.config.EndTime)
	if err != nil {
		return errors.Wrap(errors.ErrCodeQueryFailed, "failed to get data count", err)
	}

	// Invoke OnRunStart callback
	if params.callbacks.OnRunStart != nil {
		if err := (*params.callbacks.OnRunStart)(params.runID, params.configIdx, params.configName, params.dataIdx, params.dataPath, count); err != nil {
			return errors.Wrap(errors.ErrCodeCallbackFailed, "OnRunStart callback failed", err)
		}
	}

	// Process data points
	if err := b.processDataPoints(params, &strategyContext, slidingWindowDS, count); err != nil {
		return err
	}

	// Create result folder
	os.MkdirAll(params.resultFolderPath, 0755)

	// Get stats
	if res.state == nil {
		return errors.New(errors.ErrCodeBacktestStateNil, "backtest state is nil")
	}

	if err := b.writeResults(strategyContext, res, params.runID, params.resultFolderPath, params.strategyPath, params.dataPath); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to write results", err)
	}

	// Invoke OnRunEnd callback
	if params.callbacks.OnRunEnd != nil {
		(*params.callbacks.OnRunEnd)(params.configIdx, params.configName, params.dataIdx, params.dataPath, params.resultFolderPath)
	}

	// Cleanup state so the same resources can be reused for a subsequent iteration.
	if err := b.cleanUpRun(res); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to cleanup run", err)
	}

	return nil
}

// processDataPoints processes all data points for a single run iteration.
func (b *BacktestEngineV1) processDataPoints(params runIterationParams, strategyContext *runtime.RuntimeContext, slidingWindowDS *datasource.SlidingWindowDataSource, count int) error {
	res := params.res
	currentCount := 0

	// Track insufficient data error state for marker boundaries
	var (
		inInsufficientDataError bool
		lastInsufficientData    types.MarketData
	)

	for data, err := range res.datasource.ReadAll(b.config.StartTime, b.config.EndTime) {
		// Check for context cancellation
		select {
		case <-params.ctx.Done():
			if cleanupErr := b.cleanUpRun(res); cleanupErr != nil {
				b.log.Error("Failed to cleanup run after cancellation",
					zap.Error(cleanupErr),
				)
			}

			return params.ctx.Err()
		default:
		}

		if err != nil {
			return errors.Wrap(errors.ErrCodeDataNotFound, "failed to read data", err)
		}

		// Add market data to the sliding window cache for future lookups
		slidingWindowDS.AddToCache(data)

		// run the strategy
		if backtestTrading, ok := res.tradingSystem.(*BacktestTrading); ok {
			backtestTrading.UpdateCurrentMarketData(data)
		}

		// Set current market data in strategy context for implicit log context
		strategyContext.CurrentMarketData = &data

		// Process data and track insufficient data errors for markers
		processErr := res.strategy.ProcessData(data)

		if errors.IsInsufficientDataError(processErr) {
			if !inInsufficientDataError {
				// Transition: OK → Insufficient - mark beginning
				b.markInsufficientDataStart(res.marker, data)

				inInsufficientDataError = true
			}
			// Track the last data point with insufficient error for end marker
			lastInsufficientData = data
		} else {
			if inInsufficientDataError {
				// Transition: Insufficient → OK - mark end at last insufficient data point
				b.markInsufficientDataEnd(res.marker, lastInsufficientData)

				inInsufficientDataError = false
			}

			// Add error marker for non-insufficient errors (continue processing)
			if processErr != nil {
				b.markStrategyError(res.marker, data, processErr)
			}
		}

		// Update progress bar
		currentCount++

		// Invoke OnProcessData callback
		if params.callbacks.OnProcessData != nil {
			if err := (*params.callbacks.OnProcessData)(currentCount, count); err != nil {
				return err
			}
		}
	}

	// If we ended in an insufficient data error state, mark the end
	if inInsufficientDataError {
		b.markInsufficientDataEnd(res.marker, lastInsufficientData)
	}

	return nil
}

// markInsufficientDataStart adds a warning marker at the start of an insufficient data error sequence.
func (b *BacktestEngineV1) markInsufficientDataStart(mk marker.Marker, data types.MarketData) {
	if mk == nil {
		return
	}

	mark := types.Mark{
		MarketDataId: data.Id,
		Color:        types.MarkColorYellow,
		Shape:        types.MarkShapeTriangle,
		Level:        types.MarkLevelWarning,
		Title:        "Insufficient Data",
		Message:      "Insufficient data error started",
		Category:     "InsufficientData",
		Signal: optional.Some(types.Signal{
			Time:      data.Time,
			Symbol:    data.Symbol,
			Type:      types.SignalTypeNoAction,
			Name:      "Insufficient Data",
			Reason:    "",
			RawValue:  nil,
			Indicator: "",
		}),
	}

	if err := mk.Mark(data, mark); err != nil {
		b.log.Error("Failed to mark insufficient data start",
			zap.Error(err),
		)
	}
}

// markInsufficientDataEnd adds a warning marker at the end of an insufficient data error sequence.
func (b *BacktestEngineV1) markInsufficientDataEnd(mk marker.Marker, data types.MarketData) {
	if mk == nil {
		return
	}

	mark := types.Mark{
		MarketDataId: data.Id,
		Color:        types.MarkColorYellow,
		Shape:        types.MarkShapeTriangle,
		Level:        types.MarkLevelWarning,
		Title:        "Insufficient Data",
		Message:      "Insufficient data error ended",
		Category:     "InsufficientData",
		Signal: optional.Some(types.Signal{
			Time:      data.Time,
			Symbol:    data.Symbol,
			Type:      types.SignalTypeNoAction,
			Name:      "Insufficient Data",
			Reason:    "",
			RawValue:  nil,
			Indicator: "",
		}),
	}

	if err := mk.Mark(data, mark); err != nil {
		b.log.Error("Failed to mark insufficient data end",
			zap.Error(err),
		)
	}
}

// markStrategyError adds an error marker for strategy errors (non-insufficient data errors).
func (b *BacktestEngineV1) markStrategyError(mk marker.Marker, data types.MarketData, strategyErr error) {
	if mk == nil {
		return
	}

	mark := types.Mark{
		MarketDataId: data.Id,
		Color:        types.MarkColorRed,
		Shape:        types.MarkShapeCircle,
		Level:        types.MarkLevelError,
		Title:        "Strategy Error",
		Message:      strategyErr.Error(),
		Category:     "StrategyError",
		Signal: optional.Some(types.Signal{
			Time:      data.Time,
			Symbol:    data.Symbol,
			Type:      types.SignalTypeAbort,
			Name:      "Strategy Error",
			Reason:    "",
			RawValue:  nil,
			Indicator: "",
		}),
	}

	if err := mk.Mark(data, mark); err != nil {
		b.log.Error("Failed to mark strategy error",
			zap.Error(err),
		)
	}
}

func (b *BacktestEngineV1) writeResults(strategyContext runtime.RuntimeContext, res *iterResources, runID string, resultFolderPath string, strategyPath string, dataPath string) error {
	if res.state == nil {
		return errors.New(errors.ErrCodeBacktestStateNil, "backtest state is nil")
	}

	// Calculate file paths
	stateDBPath := filepath.Join(resultFolderPath, "state.db")
	tradesFilePath := filepath.Join(stateDBPath, "trades.parquet")
	ordersFilePath := filepath.Join(stateDBPath, "orders.parquet")
	marksFilePath := filepath.Join(resultFolderPath, "marks.parquet")
	logsFilePath := filepath.Join(resultFolderPath, "logs.parquet")

	stats, err := res.state.GetStats(strategyContext, res.strategy, runID, tradesFilePath, ordersFilePath, marksFilePath, logsFilePath, strategyPath, dataPath)
	if err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to get stats", err)
	}

	// Write stats to file
	if err := types.WriteTradeStats(filepath.Join(resultFolderPath, "stats.yaml"), stats); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to write stats", err)
	}

	// Write state to disk
	if err := res.state.Write(stateDBPath); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to write state", err)
	}

	// write the marker to disk
	if mk, ok := res.marker.(*BacktestMarker); ok {
		if err := mk.Write(resultFolderPath); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to write marker", err)
		}
	}

	// write the logs to disk
	if res.logStorage != nil {
		if err := res.logStorage.Write(resultFolderPath); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to write logs", err)
		}
	}

	return nil
}

func (b *BacktestEngineV1) cleanUpRun(res *iterResources) error {
	if res == nil || res.state == nil {
		return errors.New(errors.ErrCodeBacktestStateNil, "backtest state is nil")
	}

	if err := res.state.Cleanup(); err != nil {
		return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to cleanup state", err)
	}

	// Cleanup the cache
	if res.cache != nil {
		res.cache.Reset()
	}

	// clean up the trading system
	if backtestTrading, ok := res.tradingSystem.(*BacktestTrading); ok {
		backtestTrading.Reset(b.config.InitialCapital)
	}

	// Cleanup the marker
	if mk, ok := res.marker.(*BacktestMarker); ok {
		if err := mk.Cleanup(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to cleanup marker", err)
		}
	}

	// Cleanup the log storage
	if res.logStorage != nil {
		if err := res.logStorage.Cleanup(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to cleanup log storage", err)
		}
	}

	return nil
}

func (b *BacktestEngineV1) preRunCheck() error {
	if len(b.strategies) == 0 {
		b.log.Error("No strategies loaded")

		return errors.New(errors.ErrCodeBacktestNoStrategies, "no strategies loaded")
	}

	if len(b.strategyConfigPaths) == 0 && len(b.strategyConfigs) == 0 {
		b.log.Error("No strategy configs loaded")

		return errors.New(errors.ErrCodeBacktestNoConfigs, "no strategy configs loaded")
	}

	if len(b.dataPaths) == 0 {
		b.log.Error("No data paths loaded")

		return errors.New(errors.ErrCodeBacktestNoDataPaths, "no data paths loaded")
	}

	if b.resultsFolder == "" {
		b.log.Error("No results folder set")

		return errors.New(errors.ErrCodeBacktestNoResultsDir, "no results folder set")
	}

	if b.datasource == nil {
		b.log.Error("No datasource set")

		return errors.New(errors.ErrCodeBacktestNoDatasource, "no datasource set")
	}

	return nil
}

package engine_v1

import (
	"context"
	"fmt"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/indicator"
	internalLog "github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/marker"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
	"go.uber.org/zap"
)

// Default configuration values.
const (
	DefaultMarketDataCacheSize = 1000
)

// LiveTradingEngineV1 implements the LiveTradingEngine interface for real-time trading.
type LiveTradingEngineV1 struct {
	config              engine.LiveTradingEngineConfig
	strategy            runtime.StrategyRuntime
	strategyConfig      string
	marketDataProvider  provider.Provider
	tradingProvider     tradingprovider.TradingSystemProvider
	streamingDataSource *StreamingDataSource
	indicatorRegistry   indicator.IndicatorRegistry
	cache               cache.Cache
	marker              marker.Marker
	log                 *logger.Logger
	logStorage          internalLog.Log
	initialized         bool

	// Persistence fields for streaming data
	dataDir              string                         // Directory for storing parquet files
	providerName         string                         // Name of the data provider (e.g., "binance")
	streamingWriter      *writer.StreamingDuckDBWriter  // Writes finalized candles to parquet
	persistentDataSource *PersistentStreamingDataSource // Reads from parquet for indicator calculations
}

// NewLiveTradingEngineV1 creates a new LiveTradingEngineV1 instance without persistence.
func NewLiveTradingEngineV1() (engine.LiveTradingEngine, error) {
	log, err := logger.NewLogger()
	if err != nil {
		return nil, err
	}

	return &LiveTradingEngineV1{
		config:               engine.LiveTradingEngineConfig{}, //nolint:exhaustruct // initialized via Initialize()
		strategy:             nil,
		strategyConfig:       "",
		marketDataProvider:   nil,
		tradingProvider:      nil,
		streamingDataSource:  nil,
		indicatorRegistry:    nil,
		cache:                cache.NewCacheV1(),
		marker:               nil,
		log:                  log,
		logStorage:           nil,
		initialized:          false,
		dataDir:              "",
		providerName:         "",
		streamingWriter:      nil,
		persistentDataSource: nil,
	}, nil
}

// NewLiveTradingEngineV1WithPersistence creates a new LiveTradingEngineV1 instance with data persistence.
// dataDir: directory where parquet files will be stored
// providerName: name of the data provider (e.g., "binance").
func NewLiveTradingEngineV1WithPersistence(dataDir, providerName string) (engine.LiveTradingEngine, error) {
	log, err := logger.NewLogger()
	if err != nil {
		return nil, err
	}

	return &LiveTradingEngineV1{
		config:               engine.LiveTradingEngineConfig{}, //nolint:exhaustruct // initialized via Initialize()
		strategy:             nil,
		strategyConfig:       "",
		marketDataProvider:   nil,
		tradingProvider:      nil,
		streamingDataSource:  nil,
		indicatorRegistry:    nil,
		cache:                cache.NewCacheV1(),
		marker:               nil,
		log:                  log,
		logStorage:           nil,
		initialized:          false,
		dataDir:              dataDir,
		providerName:         providerName,
		streamingWriter:      nil,
		persistentDataSource: nil,
	}, nil
}

// Initialize implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) Initialize(config engine.LiveTradingEngineConfig) error {
	// Set default values
	if config.MarketDataCacheSize <= 0 {
		config.MarketDataCacheSize = DefaultMarketDataCacheSize
	}

	e.config = config

	// Initialize indicator registry with standard indicators
	e.indicatorRegistry = indicator.NewIndicatorRegistry()
	e.indicatorRegistry.RegisterIndicator(indicator.NewBollingerBands())
	e.indicatorRegistry.RegisterIndicator(indicator.NewEMA())
	e.indicatorRegistry.RegisterIndicator(indicator.NewMACD())
	e.indicatorRegistry.RegisterIndicator(indicator.NewATR())
	e.indicatorRegistry.RegisterIndicator(indicator.NewWaddahAttar())
	e.indicatorRegistry.RegisterIndicator(indicator.NewRSI())
	e.indicatorRegistry.RegisterIndicator(indicator.NewMA())

	// Create streaming data source with configured cache size (used as fallback without persistence)
	e.streamingDataSource = NewStreamingDataSource(config.MarketDataCacheSize)

	// Initialize persistence components if dataDir and providerName are set
	if e.dataDir != "" && e.providerName != "" {
		// Create streaming writer for persisting finalized candles
		e.streamingWriter = writer.NewStreamingDuckDBWriter(e.dataDir, e.providerName, config.Interval)
		if err := e.streamingWriter.Initialize(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize streaming writer", err)
		}

		// Create persistent data source for indicator calculations
		parquetPath := e.streamingWriter.GetOutputPath()
		e.persistentDataSource = NewPersistentStreamingDataSource(parquetPath, config.Interval)
		if err := e.persistentDataSource.Initialize(""); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize persistent datasource", err)
		}

		e.log.Info("Data persistence enabled",
			zap.String("parquet_path", parquetPath),
			zap.String("provider", e.providerName),
			zap.String("interval", config.Interval),
		)
	}

	// Create marker and log storage if logging is enabled
	if config.EnableLogging {
		var err error

		e.marker, err = NewLiveTradingMarker(e.log)
		if err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to create marker", err)
		}

		e.logStorage, err = NewLiveTradingLog(e.log)
		if err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to create log storage", err)
		}
	}

	e.initialized = true

	e.log.Debug("Live trading engine initialized",
		zap.Strings("symbols", config.Symbols),
		zap.String("interval", config.Interval),
		zap.Int("cache_size", config.MarketDataCacheSize),
	)

	return nil
}

// LoadStrategyFromFile implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) LoadStrategyFromFile(strategyPath string) error {
	strategy, err := wasm.NewStrategyWasmRuntime(strategyPath)
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to create strategy runtime", err)
	}

	e.strategy = strategy
	e.log.Debug("Strategy loaded from file",
		zap.String("path", strategyPath),
	)

	return nil
}

// LoadStrategyFromBytes implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) LoadStrategyFromBytes(strategyBytes []byte) error {
	strategy, err := wasm.NewStrategyWasmRuntimeFromBytes(strategyBytes)
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to create strategy runtime", err)
	}

	e.strategy = strategy
	e.log.Debug("Strategy loaded from bytes")

	return nil
}

// LoadStrategy implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) LoadStrategy(strategy runtime.StrategyRuntime) error {
	e.strategy = strategy
	e.log.Debug("Strategy loaded",
		zap.String("name", strategy.Name()),
	)

	return nil
}

// SetStrategyConfig implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) SetStrategyConfig(config string) error {
	e.strategyConfig = config
	e.log.Debug("Strategy config set")

	return nil
}

// SetMarketDataProvider implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) SetMarketDataProvider(marketProvider provider.Provider) error {
	e.marketDataProvider = marketProvider
	e.log.Debug("Market data provider set")

	return nil
}

// SetTradingProvider implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) SetTradingProvider(tradingProvider tradingprovider.TradingSystemProvider) error {
	e.tradingProvider = tradingProvider
	e.log.Debug("Trading provider set")

	return nil
}

// Run implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) Run(ctx context.Context, callbacks engine.LiveTradingCallbacks) error {
	var runErr error

	// Always call OnEngineStop and cleanup when Run exits
	defer func() {
		// Cleanup persistence components
		if e.streamingWriter != nil {
			if err := e.streamingWriter.Flush(); err != nil {
				e.log.Warn("Failed to flush streaming writer", zap.Error(err))
			}

			if err := e.streamingWriter.Close(); err != nil {
				e.log.Warn("Failed to close streaming writer", zap.Error(err))
			}
		}

		if e.persistentDataSource != nil {
			if err := e.persistentDataSource.Close(); err != nil {
				e.log.Warn("Failed to close persistent datasource", zap.Error(err))
			}
		}

		if callbacks.OnEngineStop != nil {
			(*callbacks.OnEngineStop)(runErr)
		}
	}()

	// Pre-run validation
	if err := e.preRunCheck(); err != nil {
		runErr = err

		return err
	}

	// Initialize strategy
	if err := e.initializeStrategy(); err != nil {
		runErr = err

		return err
	}

	// Call OnEngineStart callback
	if callbacks.OnEngineStart != nil {
		// Determine previousDataPath - if persistence is enabled, provide the parquet file path
		previousDataPath := ""
		if e.streamingWriter != nil {
			previousDataPath = e.streamingWriter.GetOutputPath()
		}

		if err := (*callbacks.OnEngineStart)(e.config.Symbols, e.config.Interval, previousDataPath); err != nil {
			runErr = errors.Wrap(errors.ErrCodeCallbackFailed, "OnEngineStart callback failed", err)

			return runErr
		}
	}

	// Start streaming market data
	stream := e.marketDataProvider.Stream(ctx, e.config.Symbols, e.config.Interval)

	// Determine which datasource to use for indicator calculations
	// Use persistent datasource if available (queries parquet directly), otherwise use in-memory streaming datasource
	var dataSource datasource.DataSource = e.streamingDataSource
	if e.persistentDataSource != nil {
		dataSource = e.persistentDataSource
	}

	// Create RuntimeContext for strategy - the context will be updated with current market data
	strategyContext := runtime.RuntimeContext{
		DataSource:        dataSource,
		IndicatorRegistry: e.indicatorRegistry,
		Marker:            e.marker,
		TradingSystem:     e.tradingProvider,
		Cache:             e.cache,
		Logger:            e.log,
		LogStorage:        e.logStorage,
		CurrentMarketData: nil,
	}

	// Process each market data point from the stream
	for data, err := range stream {
		// Check for context cancellation
		select {
		case <-ctx.Done():
			runErr = ctx.Err()

			return runErr
		default:
		}

		// Handle stream errors
		if err != nil {
			if callbacks.OnError != nil {
				(*callbacks.OnError)(err)
			}

			e.log.Warn("Stream error received",
				zap.Error(err),
			)
			// Continue processing - don't abort on non-fatal stream errors
			continue
		}

		// Persist finalized candle to parquet file if persistence is enabled
		if e.streamingWriter != nil {
			if writeErr := e.streamingWriter.Write(data); writeErr != nil {
				e.log.Error("Failed to persist market data",
					zap.String("symbol", data.Symbol),
					zap.Time("time", data.Time),
					zap.Error(writeErr),
				)
				// Continue processing - don't abort on write errors
			}
		}

		// Add to in-memory cache as well (used when persistence is not enabled)
		e.streamingDataSource.AddToCache(data)

		// Update current market data in strategy context
		strategyContext.CurrentMarketData = &data

		// Invoke OnMarketData callback
		if callbacks.OnMarketData != nil {
			if err := (*callbacks.OnMarketData)(data); err != nil {
				runErr = errors.Wrap(errors.ErrCodeCallbackFailed, "OnMarketData callback failed", err)

				return runErr
			}
		}

		// Execute strategy
		if err := e.strategy.ProcessData(data); err != nil {
			if callbacks.OnStrategyError != nil {
				(*callbacks.OnStrategyError)(data, err)
			}

			e.log.Warn("Strategy error",
				zap.String("symbol", data.Symbol),
				zap.Error(err),
			)
			// Continue processing - don't abort on strategy errors
		}
	}

	// After stream ends, check if it was due to context cancellation
	if ctx.Err() != nil {
		runErr = ctx.Err()

		return runErr
	}

	return nil
}

// GetConfigSchema implements engine.LiveTradingEngine.
func (e *LiveTradingEngineV1) GetConfigSchema() (string, error) {
	return engine.GetConfigSchema()
}

// preRunCheck validates that all required components are configured before running.
func (e *LiveTradingEngineV1) preRunCheck() error {
	if !e.initialized {
		return errors.New(errors.ErrCodeBacktestInitFailed, "engine not initialized - call Initialize() first")
	}

	if e.strategy == nil {
		return errors.New(errors.ErrCodeBacktestInitFailed, "strategy not loaded - call LoadStrategy*() first")
	}

	if e.marketDataProvider == nil {
		return errors.New(errors.ErrCodeBacktestInitFailed, "market data provider not set - call SetMarketDataProvider() first")
	}

	if e.tradingProvider == nil {
		return errors.New(errors.ErrCodeBacktestInitFailed, "trading provider not set - call SetTradingProvider() first")
	}

	if len(e.config.Symbols) == 0 {
		return errors.New(errors.ErrCodeBacktestInitFailed, "no symbols configured")
	}

	if e.config.Interval == "" {
		return errors.New(errors.ErrCodeBacktestInitFailed, "no interval configured")
	}

	return nil
}

// initializeStrategy sets up the strategy with the RuntimeContext and configuration.
func (e *LiveTradingEngineV1) initializeStrategy() error {
	// Determine which datasource to use for indicator calculations
	var dataSource datasource.DataSource = e.streamingDataSource
	if e.persistentDataSource != nil {
		dataSource = e.persistentDataSource
	}

	// Create RuntimeContext for strategy initialization
	strategyContext := runtime.RuntimeContext{
		DataSource:        dataSource,
		IndicatorRegistry: e.indicatorRegistry,
		Marker:            e.marker,
		TradingSystem:     e.tradingProvider,
		Cache:             e.cache,
		Logger:            e.log,
		LogStorage:        e.logStorage,
		CurrentMarketData: nil,
	}

	// Initialize strategy API first
	err := e.strategy.InitializeApi(wasm.NewWasmStrategyApi(&strategyContext))
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to initialize strategy API", err)
	}

	// Check version compatibility between engine and strategy
	strategyRuntimeVersion, err := e.strategy.GetRuntimeEngineVersion()
	if err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to get strategy runtime version", err)
	}

	if err := version.CheckVersionCompatibility(version.Version, strategyRuntimeVersion); err != nil {
		return errors.Wrapf(errors.ErrCodeVersionMismatch, err,
			"version mismatch: engine version %s is incompatible with strategy compiled for version %s",
			version.Version, strategyRuntimeVersion)
	}

	// Initialize strategy with config
	if err := e.strategy.Initialize(e.strategyConfig); err != nil {
		return errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to initialize strategy", err)
	}

	e.log.Info("Strategy initialized",
		zap.String("name", e.strategy.Name()),
	)

	return nil
}

// Verify LiveTradingEngineV1 implements engine.LiveTradingEngine interface.
var _ engine.LiveTradingEngine = (*LiveTradingEngineV1)(nil)

// LiveTradingMarker implements marker.Marker for live trading.
// It stores marks in memory since we don't have file-based persistence in live mode.
type LiveTradingMarker struct {
	marks []types.Mark
	log   *logger.Logger
}

// NewLiveTradingMarker creates a new LiveTradingMarker instance.
func NewLiveTradingMarker(log *logger.Logger) (marker.Marker, error) {
	return &LiveTradingMarker{
		marks: make([]types.Mark, 0),
		log:   log,
	}, nil
}

// Mark implements marker.Marker.
func (m *LiveTradingMarker) Mark(marketData types.MarketData, mark types.Mark) error {
	m.marks = append(m.marks, mark)
	m.log.Debug("Mark added",
		zap.String("symbol", marketData.Symbol),
		zap.String("signal", fmt.Sprintf("%v", mark.Signal)),
	)

	return nil
}

// GetMarks implements marker.Marker.
func (m *LiveTradingMarker) GetMarks() ([]types.Mark, error) {
	return m.marks, nil
}

// LiveTradingLog implements internalLog.Log for live trading.
// It stores logs in memory since we don't have file-based persistence in live mode.
type LiveTradingLog struct {
	logs []internalLog.LogEntry
	log  *logger.Logger
}

// NewLiveTradingLog creates a new LiveTradingLog instance.
func NewLiveTradingLog(log *logger.Logger) (internalLog.Log, error) {
	return &LiveTradingLog{
		logs: make([]internalLog.LogEntry, 0),
		log:  log,
	}, nil
}

// Log implements internalLog.Log.
func (l *LiveTradingLog) Log(entry internalLog.LogEntry) error {
	l.logs = append(l.logs, entry)

	return nil
}

// GetLogs implements internalLog.Log.
func (l *LiveTradingLog) GetLogs() ([]internalLog.LogEntry, error) {
	return l.logs, nil
}

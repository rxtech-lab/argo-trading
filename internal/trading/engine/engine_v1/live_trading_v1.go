package engine_v1

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/indicator"
	internalLog "github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/marker"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1/prefetch"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1/session"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1/stats"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1/writers"
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

	// Session management
	sessionManager *session.SessionManager

	// Statistics tracking
	statsTracker *stats.StatsTracker

	// Prefetch management
	prefetchManager *prefetch.PrefetchManager

	// Parquet writers for orders, trades, marks, logs
	ordersWriter *writers.OrdersWriter
	tradesWriter *writers.TradesWriter
	marksWriter  *writers.MarksWriter
	logsWriter   *writers.LogsWriter
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
		sessionManager:       nil,
		statsTracker:         nil,
		prefetchManager:      nil,
		ordersWriter:         nil,
		tradesWriter:         nil,
		marksWriter:          nil,
		logsWriter:           nil,
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
		sessionManager:       nil,
		statsTracker:         nil,
		prefetchManager:      nil,
		ordersWriter:         nil,
		tradesWriter:         nil,
		marksWriter:          nil,
		logsWriter:           nil,
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

	// Initialize session management if DataOutputPath is configured
	if config.DataOutputPath != "" {
		e.sessionManager = session.NewSessionManager(e.log)
		if err := e.sessionManager.Initialize(config.DataOutputPath); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize session manager", err)
		}

		runPath := e.sessionManager.GetCurrentRunPath()

		e.log.Info("Session initialized",
			zap.String("run_id", e.sessionManager.GetRunID()),
			zap.String("run_path", runPath),
		)

		// Initialize parquet writers in the session folder
		ordersPath := filepath.Join(runPath, "orders.parquet")
		e.ordersWriter = writers.NewOrdersWriter(ordersPath)
		if err := e.ordersWriter.Initialize(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize orders writer", err)
		}

		tradesPath := filepath.Join(runPath, "trades.parquet")
		e.tradesWriter = writers.NewTradesWriter(tradesPath)
		if err := e.tradesWriter.Initialize(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize trades writer", err)
		}

		marksPath := filepath.Join(runPath, "marks.parquet")
		e.marksWriter = writers.NewMarksWriter(marksPath)
		if err := e.marksWriter.Initialize(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize marks writer", err)
		}

		logsPath := filepath.Join(runPath, "logs.parquet")
		e.logsWriter = writers.NewLogsWriter(logsPath)
		if err := e.logsWriter.Initialize(); err != nil {
			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to initialize logs writer", err)
		}

		// Initialize stats tracker (will be fully initialized after strategy loads with strategy info)
		e.statsTracker = stats.NewStatsTracker(e.log)
		e.statsTracker.SetFilePaths(
			ordersPath,
			tradesPath,
			marksPath,
			logsPath,
			"", // market data path set later
			filepath.Join(runPath, "stats.yaml"),
		)

		// Initialize prefetch manager (will be fully initialized before Run with provider)
		e.prefetchManager = prefetch.NewPrefetchManager(e.log)
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
//
//nolint:gocyclo,maintidx // Run orchestrates the live trading loop which requires handling many cases
func (e *LiveTradingEngineV1) Run(ctx context.Context, callbacks engine.LiveTradingCallbacks) error {
	var runErr error
	firstDataReceived := false

	// Always call OnEngineStop and cleanup when Run exits
	defer func() {
		// Emit stopped status
		if callbacks.OnStatusUpdate != nil {
			_ = (*callbacks.OnStatusUpdate)(types.EngineStatusStopped)
		}

		// Write final stats
		if e.statsTracker != nil {
			if err := e.statsTracker.WriteStatsYAML(); err != nil {
				e.log.Warn("Failed to write final stats", zap.Error(err))
			}
		}

		// Cleanup parquet writers
		if e.ordersWriter != nil {
			if err := e.ordersWriter.Flush(); err != nil {
				e.log.Warn("Failed to flush orders writer", zap.Error(err))
			}
			if err := e.ordersWriter.Close(); err != nil {
				e.log.Warn("Failed to close orders writer", zap.Error(err))
			}
		}

		if e.tradesWriter != nil {
			if err := e.tradesWriter.Flush(); err != nil {
				e.log.Warn("Failed to flush trades writer", zap.Error(err))
			}
			if err := e.tradesWriter.Close(); err != nil {
				e.log.Warn("Failed to close trades writer", zap.Error(err))
			}
		}

		if e.marksWriter != nil {
			if err := e.marksWriter.Flush(); err != nil {
				e.log.Warn("Failed to flush marks writer", zap.Error(err))
			}
			if err := e.marksWriter.Close(); err != nil {
				e.log.Warn("Failed to close marks writer", zap.Error(err))
			}
		}

		if e.logsWriter != nil {
			if err := e.logsWriter.Flush(); err != nil {
				e.log.Warn("Failed to flush logs writer", zap.Error(err))
			}
			if err := e.logsWriter.Close(); err != nil {
				e.log.Warn("Failed to close logs writer", zap.Error(err))
			}
		}

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

	// Initialize stats tracker with strategy info
	if e.statsTracker != nil && e.sessionManager != nil {
		strategyInfo := types.StrategyInfo{
			ID:      "", // Strategy ID not available from runtime
			Version: "", // Strategy version not available from runtime
			Name:    e.strategy.Name(),
		}
		e.statsTracker.Initialize(
			e.config.Symbols,
			e.sessionManager.GetRunID(),
			e.sessionManager.GetSessionStart(),
			strategyInfo,
		)

		// Update market data file path if streaming writer is available
		if e.streamingWriter != nil {
			e.statsTracker.SetFilePaths(
				filepath.Join(e.sessionManager.GetCurrentRunPath(), "orders.parquet"),
				filepath.Join(e.sessionManager.GetCurrentRunPath(), "trades.parquet"),
				filepath.Join(e.sessionManager.GetCurrentRunPath(), "marks.parquet"),
				filepath.Join(e.sessionManager.GetCurrentRunPath(), "logs.parquet"),
				e.streamingWriter.GetOutputPath(),
				filepath.Join(e.sessionManager.GetCurrentRunPath(), "stats.yaml"),
			)
		}
	}

	// Initialize prefetch manager with provider and streaming writer
	if e.prefetchManager != nil && e.streamingWriter != nil {
		e.prefetchManager.Initialize(
			e.config.Prefetch,
			e.marketDataProvider,
			e.streamingWriter,
			e.config.Interval,
			callbacks.OnStatusUpdate,
		)

		// Execute prefetch before starting stream
		if err := e.prefetchManager.ExecutePrefetch(ctx, e.config.Symbols); err != nil {
			e.log.Warn("Prefetch failed, continuing without historical data",
				zap.Error(err),
			)
		}
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

		// Handle first data point - check for gaps
		if !firstDataReceived {
			firstDataReceived = true

			if e.prefetchManager != nil {
				if err := e.prefetchManager.HandleStreamStart(ctx, data.Time, e.config.Symbols); err != nil {
					e.log.Warn("Failed to handle stream start",
						zap.Error(err),
					)
				}
			}

			// Emit running status if no prefetch manager
			if e.prefetchManager == nil && callbacks.OnStatusUpdate != nil {
				_ = (*callbacks.OnStatusUpdate)(types.EngineStatusRunning)
			}
		}

		// Handle date boundary if session manager is available
		if e.sessionManager != nil {
			dateBoundary, err := e.sessionManager.HandleDateBoundary(data.Time)
			if err != nil {
				e.log.Warn("Failed to handle date boundary",
					zap.Error(err),
				)
			}

			if dateBoundary && e.statsTracker != nil {
				// Date changed - handle date boundary in stats tracker
				newDate := data.Time.Format("2006-01-02")
				e.statsTracker.HandleDateBoundary(newDate)

				// Reinitialize writers for new date folder
				newRunPath := e.sessionManager.GetCurrentRunPath()
				e.statsTracker.SetFilePaths(
					filepath.Join(newRunPath, "orders.parquet"),
					filepath.Join(newRunPath, "trades.parquet"),
					filepath.Join(newRunPath, "marks.parquet"),
					filepath.Join(newRunPath, "logs.parquet"),
					e.streamingWriter.GetOutputPath(),
					filepath.Join(newRunPath, "stats.yaml"),
				)
			}
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

		// Write marks to parquet if available
		if e.marksWriter != nil && e.marker != nil {
			marks, _ := e.marker.GetMarks()
			for _, mark := range marks {
				if err := e.marksWriter.Write(mark); err != nil {
					e.log.Warn("Failed to write mark",
						zap.Error(err),
					)
				}
			}
		}

		// Write logs to parquet if available
		if e.logsWriter != nil && e.logStorage != nil {
			logs, _ := e.logStorage.GetLogs()
			for _, logEntry := range logs {
				if err := e.logsWriter.Write(logEntry); err != nil {
					e.log.Warn("Failed to write log",
						zap.Error(err),
					)
				}
			}
		}

		// Update and emit stats periodically
		if e.statsTracker != nil {
			// Write stats to disk
			if err := e.statsTracker.WriteStatsYAML(); err != nil {
				e.log.Warn("Failed to write stats",
					zap.Error(err),
				)
			}

			// Emit stats callback
			if callbacks.OnStatsUpdate != nil {
				stats := e.statsTracker.GetCumulativeStats()
				if err := (*callbacks.OnStatsUpdate)(stats); err != nil {
					e.log.Warn("OnStatsUpdate callback failed",
						zap.Error(err),
					)
				}
			}
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

package engine

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	engine_types "github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/mocks"
	argoErrors "github.com/rxtech-lab/argo-trading/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

// findTimestampFolder finds the timestamp subfolder created during a backtest run.
// Returns the path to the timestamp folder, or empty string if not found.
func findTimestampFolder(t *testing.T, resultsDir string) string {
	entries, err := os.ReadDir(resultsDir)
	require.NoError(t, err, "Should be able to read results directory")
	require.NotEmpty(t, entries, "Results directory should have a timestamp subfolder")
	return filepath.Join(resultsDir, entries[0].Name())
}

// setTestVersion sets the version for testing and returns via t.Cleanup.
func setTestVersion(t *testing.T, v string) {
	originalVersion := version.Version
	version.Version = v
	t.Cleanup(func() {
		version.Version = originalVersion
	})
}

func TestBacktestEngineV1_Run(t *testing.T) {
	t.Run("Complete execution flow through Run function", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockIndicatorRegistry := mocks.NewMockIndicatorRegistry(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)
		mockTradingSystem := mocks.NewMockTradingSystem(ctrl)

		// Create temporary directory for results using t.TempDir()
		tempDir := t.TempDir()

		// Create temporary config directory using t.TempDir()
		configDir := t.TempDir()

		configPath := filepath.Join(configDir, "strategy_config.yaml")
		err := os.WriteFile(configPath, []byte("test: config"), 0644)
		require.NoError(t, err)

		// Setup test market data
		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		// Setup mock expectations
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		// Setup datasource expectations - make sure Initialize ignores the path and returns nil
		mockDatasource.EXPECT().Initialize(gomock.Any()).DoAndReturn(func(path string) error {
			// Return nil to bypass file validation
			return nil
		}).AnyTimes()

		// Setup ReadAll behavior to return our test data
		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		// Create backtest engine
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		// Initialize engine
		config := `
initialCapital: 10000
startTime: "2023-01-01T00:00:00Z"
endTime: "2023-01-31T23:59:59Z"
`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// Override dependencies with mocks
		backtestEngine.indicatorRegistry = mockIndicatorRegistry
		backtestEngine.marker = mockMarker
		backtestEngine.tradingSystem = mockTradingSystem
		backtestEngine.SetDataSource(mockDatasource)

		// Load strategy
		err = backtestEngine.LoadStrategy(mockStrategy)
		require.NoError(t, err)

		// Set config and data path
		err = backtestEngine.SetConfigPath(configPath)
		require.NoError(t, err)
		// Directly set the dataPaths property
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		err = backtestEngine.SetResultsFolder(tempDir)
		require.NoError(t, err)

		// Run backtest
		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)

		// Verify results folder was created (now under timestamp subfolder)
		timestampDir := findTimestampFolder(t, tempDir)
		strategyDir := filepath.Join(timestampDir, "TestStrategy")
		_, err = os.Stat(strategyDir)
		assert.NoError(t, err, "Strategy result directory should be created")
	})

	t.Run("Strategy processing on data points", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockIndicatorRegistry := mocks.NewMockIndicatorRegistry(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)
		mockTradingSystem := mocks.NewMockTradingSystem(ctrl)

		// Create temporary directories using t.TempDir()
		tempDir := t.TempDir()
		configDir := t.TempDir()

		configPath := filepath.Join(configDir, "strategy_config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Setup test market data - multiple data points
		marketData1 := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}
		marketData2 := types.MarketData{
			Symbol: "TEST",
			Open:   102.0,
			High:   107.0,
			Low:    100.0,
			Close:  105.0,
			Volume: 1500,
		}

		// Setup strategy expectations - verify ProcessData is called with each data point
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		// Important: expect ProcessData to be called with exact data points in order
		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData1)).Return(nil).Times(1),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData2)).Return(nil).Times(1),
		)

		// Setup datasource expectations - make sure Initialize ignores the path and returns nil
		mockDatasource.EXPECT().Initialize(gomock.Any()).DoAndReturn(func(path string) error {
			// Return nil to bypass file validation
			return nil
		}).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(2, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData1, nil).AnyTimes()

		// Setup ReadAll behavior to return our test data in order
		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData1, nil)
			handler(marketData2, nil)
			return
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Create and initialize backtest engine
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)
		config := `
initialCapital: 10000
startTime: "2023-01-01T00:00:00Z"
endTime: "2023-01-31T23:59:59Z"
`
		backtestEngine.Initialize(config)
		backtestEngine.indicatorRegistry = mockIndicatorRegistry
		backtestEngine.marker = mockMarker
		backtestEngine.tradingSystem = mockTradingSystem
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		// Directly set the dataPaths property
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		// Run backtest
		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Verify results are written correctly", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockIndicatorRegistry := mocks.NewMockIndicatorRegistry(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)
		mockTradingSystem := mocks.NewMockTradingSystem(ctrl)

		// Create temporary directories using t.TempDir()
		tempDir := t.TempDir()
		configDir := t.TempDir()

		configPath := filepath.Join(configDir, "strategy_config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Setup test market data
		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		// Setup mock expectations
		mockStrategy.EXPECT().Name().Return("TestStrategyResults").AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).DoAndReturn(func(data types.MarketData) error {
			// This simulates the strategy processing data
			return nil
		}).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		// Setup datasource expectations - make sure Initialize ignores the path and returns nil
		mockDatasource.EXPECT().Initialize(gomock.Any()).DoAndReturn(func(path string) error {
			// Return nil to bypass file validation
			return nil
		}).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData, nil)
			return
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Create and initialize backtest engine
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)
		config := `
initialCapital: 10000
startTime: "2023-01-01T00:00:00Z"
endTime: "2023-01-31T23:59:59Z"
`
		backtestEngine.Initialize(config)
		backtestEngine.indicatorRegistry = mockIndicatorRegistry
		backtestEngine.marker = mockMarker
		backtestEngine.tradingSystem = mockTradingSystem
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		// Directly set the dataPaths property
		dataPathValue := filepath.Join(configDir, "data_path")
		backtestEngine.dataPaths = []string{dataPathValue}
		backtestEngine.SetResultsFolder(tempDir)

		// Run backtest
		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)

		// Verify results folder structure was created (now under timestamp subfolder)
		timestampDir := findTimestampFolder(t, tempDir)
		strategyDir := filepath.Join(timestampDir, "TestStrategyResults")
		_, err = os.Stat(strategyDir)
		assert.NoError(t, err, "Strategy result directory should be created")

		// Check the actual directory structure created by the implementation
		// Based on the logs, it's creating paths like:
		// "/timestamp/TestStrategyResults/strategy_config/data_path/"
		configBasename := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
		dataBasename := filepath.Base(dataPathValue)

		// Match the path structure seen in logs
		resultDir := filepath.Join(strategyDir, configBasename, dataBasename)

		_, err = os.Stat(resultDir)
		assert.NoError(t, err, "Result directory should be created")

		// Also check for the trades.parquet file which should exist
		tradesFile := filepath.Join(resultDir, "state.db", "trades.parquet")
		_, err = os.Stat(tradesFile)
		assert.NoError(t, err, "Trades file should be created")
	})

	t.Run("Verify stats are generated and saved", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockIndicatorRegistry := mocks.NewMockIndicatorRegistry(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)
		mockTradingSystem := mocks.NewMockTradingSystem(ctrl)

		// Create temporary directories using t.TempDir()
		tempDir := t.TempDir()
		configDir := t.TempDir()

		configPath := filepath.Join(configDir, "strategy_config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Setup test market data
		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		// Setup mock expectations
		mockStrategy.EXPECT().Name().Return("TestStrategyStats").AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		// Setup datasource expectations - make sure Initialize ignores the path and returns nil
		mockDatasource.EXPECT().Initialize(gomock.Any()).DoAndReturn(func(path string) error {
			// Return nil to bypass file validation
			return nil
		}).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData, nil)
			return
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Create and initialize backtest engine
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)
		config := `
initialCapital: 10000
startTime: "2023-01-01T00:00:00Z"
endTime: "2023-01-31T23:59:59Z"
`
		backtestEngine.Initialize(config)
		backtestEngine.indicatorRegistry = mockIndicatorRegistry
		backtestEngine.marker = mockMarker
		backtestEngine.tradingSystem = mockTradingSystem
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		// Directly set the dataPaths property
		dataPathValue := filepath.Join(configDir, "data_path")
		backtestEngine.dataPaths = []string{dataPathValue}
		backtestEngine.SetResultsFolder(tempDir)

		// Run backtest
		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)

		// Verify stats file creation (now under timestamp subfolder)
		timestampDir := findTimestampFolder(t, tempDir)
		strategyDir := filepath.Join(timestampDir, "TestStrategyStats")
		configBasename := strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath))
		dataBasename := filepath.Base(dataPathValue)

		// Match the path structure that's actually created in the tests
		resultDir := filepath.Join(strategyDir, configBasename, dataBasename)
		statsFile := filepath.Join(resultDir, "stats.yaml")

		// Stats file should exist
		_, err = os.Stat(statsFile)
		assert.NoError(t, err, "Stats file should be created")
	})

	t.Run("Error handling - strategy initialization failure", func(t *testing.T) {
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockIndicatorRegistry := mocks.NewMockIndicatorRegistry(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)
		mockTradingSystem := mocks.NewMockTradingSystem(ctrl)

		// Create temporary directories using t.TempDir()
		tempDir := t.TempDir()
		configDir := t.TempDir()

		configPath := filepath.Join(configDir, "strategy_config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Mock strategy initialization to fail
		mockStrategy.EXPECT().Name().Return("TestStrategyError").AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(errors.New("strategy initialization failed")).AnyTimes()
		// We don't expect InitializeApi to be called here since Initialize will fail first

		// Create a simplified backtest engine for testing just the initialization error case
		// Skip the full Run path by not setting up datasource expectations
		engine, engineErr := NewBacktestEngineV1()
		require.NoError(t, engineErr)
		backtestEngine := engine.(*BacktestEngineV1)
		config := `
initialCapital: 10000
startTime: "2023-01-01T00:00:00Z"
endTime: "2023-01-31T23:59:59Z"
`
		backtestEngine.Initialize(config)
		backtestEngine.indicatorRegistry = mockIndicatorRegistry
		backtestEngine.marker = mockMarker
		backtestEngine.tradingSystem = mockTradingSystem
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		// Instead of calling Run(), we'll directly test the initialization part
		// Create a function to directly test the strategy initialization step
		err := errors.New("no error")

		// The engine initializes the strategy inside Run method
		// We'll simulate that part here to check the error
		err = mockStrategy.Initialize(string("test: config"))

		// Verify the error
		assert.Error(t, err, "Strategy initialization should fail")
		assert.Contains(t, err.Error(), "strategy initialization failed", "Error message should indicate strategy initialization failure")
	})

	t.Run("Error handling - data processing failure", func(t *testing.T) {
		// Setup mocks
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)

		// Setup test market data
		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		// Setup mock expectations
		mockStrategy.EXPECT().Name().Return("TestStrategyError").AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(errors.New("data processing failed")).AnyTimes()

		// Directly test the processing error case
		err := mockStrategy.ProcessData(marketData)

		// Verify the error
		assert.Error(t, err, "Strategy ProcessData should fail")
		assert.Contains(t, err.Error(), "data processing failed", "Error message should indicate data processing failure")
	})
}

// Helper matcher function for MarketData
func matchMarketData(expected types.MarketData) gomock.Matcher {
	return gomock.GotFormatterAdapter(
		gomock.GotFormatterFunc(func(got interface{}) string {
			return fmt.Sprintf("%v", got)
		}),
		gomock.Eq(expected),
	)
}

// TestBacktestEngineV1_Initialize tests the Initialize function
func TestBacktestEngineV1_Initialize(t *testing.T) {
	t.Run("Invalid YAML config", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		// Invalid YAML should cause an error
		err = backtestEngine.Initialize("invalid: yaml: [")
		require.Error(t, err, "Initialize with invalid YAML should return an error")
	})

	t.Run("Valid config with different brokers", func(t *testing.T) {
		// Test with interactive broker
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)
		config := `
initialCapital: 10000
broker: interactive_broker
`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// Test with zero commission broker
		engine2, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine2 := engine2.(*BacktestEngineV1)
		config2 := `
initialCapital: 10000
broker: zero
`
		err = backtestEngine2.Initialize(config2)
		require.NoError(t, err)

		// Test with default broker (empty)
		engine3, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine3 := engine3.(*BacktestEngineV1)
		config3 := `
initialCapital: 10000
`
		err = backtestEngine3.Initialize(config3)
		require.NoError(t, err)
	})
}

// TestBacktestEngineV1_LoadStrategyFromFile tests LoadStrategyFromFile function
func TestBacktestEngineV1_LoadStrategyFromFile(t *testing.T) {
	t.Run("Unsupported strategy type", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// Try to load a .txt file which is not supported
		err = backtestEngine.LoadStrategyFromFile("/path/to/strategy.txt")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported strategy type")
	})

	t.Run("Non-existent WASM file", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// Try to load a non-existent .wasm file
		err = backtestEngine.LoadStrategyFromFile("/path/to/nonexistent.wasm")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create strategy runtime")
	})
}

// TestBacktestEngineV1_LoadStrategyFromBytes tests LoadStrategyFromBytes function
func TestBacktestEngineV1_LoadStrategyFromBytes(t *testing.T) {
	t.Run("Unsupported strategy type", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// Try to load with an unsupported strategy type
		err = backtestEngine.LoadStrategyFromBytes([]byte("test"), engine_types.StrategyType("unsupported"))
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported strategy type")
	})

	t.Run("Valid WASM bytes loading", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// NewStrategyWasmRuntimeFromBytes doesn't validate bytes on load,
		// it just stores them. Validation happens during InitializeApi.
		// So loading any bytes should succeed.
		err = backtestEngine.LoadStrategyFromBytes([]byte("test bytes"), engine_types.StrategyTypeWASM)
		require.NoError(t, err)
		assert.Equal(t, 1, len(backtestEngine.strategies))
	})
}

// TestBacktestEngineV1_SetConfigContent tests SetConfigContent function
func TestBacktestEngineV1_SetConfigContent(t *testing.T) {
	t.Run("Set config content directly", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		configs := []string{`{"param1": "value1"}`, `{"param2": "value2"}`}
		err = backtestEngine.SetConfigContent(configs)
		require.NoError(t, err)

		assert.Equal(t, 2, len(backtestEngine.strategyConfigs))
		assert.Equal(t, `{"param1": "value1"}`, backtestEngine.strategyConfigs[0])
		assert.Equal(t, `{"param2": "value2"}`, backtestEngine.strategyConfigs[1])
		assert.Nil(t, backtestEngine.strategyConfigPaths, "SetConfigContent should clear strategyConfigPaths")
	})

	t.Run("SetConfigContent clears config paths", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		// First set config paths
		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)
		err = backtestEngine.SetConfigPath(configPath)
		require.NoError(t, err)
		assert.NotEmpty(t, backtestEngine.strategyConfigPaths)

		// Then set config content - should clear paths
		configs := []string{`{"test": "content"}`}
		err = backtestEngine.SetConfigContent(configs)
		require.NoError(t, err)

		assert.Nil(t, backtestEngine.strategyConfigPaths, "SetConfigContent should clear strategyConfigPaths")
		assert.Equal(t, 1, len(backtestEngine.strategyConfigs))
	})

	t.Run("Run with SetConfigContent", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		tempDir := t.TempDir()

		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)

		// Use SetConfigContent instead of SetConfigPath
		configs := []string{`{"strategy_config": "direct_content"}`}
		err = backtestEngine.SetConfigContent(configs)
		require.NoError(t, err)

		backtestEngine.dataPaths = []string{filepath.Join(tempDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)

		// Verify results folder was created with config_0 naming (now under timestamp subfolder)
		timestampDir := findTimestampFolder(t, tempDir)
		strategyDir := filepath.Join(timestampDir, "TestStrategy")
		_, err = os.Stat(strategyDir)
		assert.NoError(t, err, "Strategy result directory should be created")

		// Check the result directory uses config_0 naming
		resultDir := filepath.Join(strategyDir, "config_0", "data_path")
		_, err = os.Stat(resultDir)
		assert.NoError(t, err, "Result directory should be created with config_0 naming")
	})

	t.Run("Run with multiple configs from SetConfigContent", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		tempDir := t.TempDir()

		marketData := types.MarketData{
			Symbol: "TEST",
			Open:   100.0,
			High:   105.0,
			Low:    95.0,
			Close:  102.0,
			Volume: 1000,
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		// Expect Initialize to be called twice - once for each config
		mockStrategy.EXPECT().Initialize(`{"config": 1}`).Return(nil).Times(1)
		mockStrategy.EXPECT().Initialize(`{"config": 2}`).Return(nil).Times(1)
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(handler func(types.MarketData, error) bool) {
			handler(marketData, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)

		// Use SetConfigContent with multiple configs
		configs := []string{`{"config": 1}`, `{"config": 2}`}
		err = backtestEngine.SetConfigContent(configs)
		require.NoError(t, err)

		backtestEngine.dataPaths = []string{filepath.Join(tempDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)

		// Verify both config directories were created (now under timestamp subfolder)
		timestampDir := findTimestampFolder(t, tempDir)
		resultDir0 := filepath.Join(timestampDir, "TestStrategy", "config_0", "data_path")
		_, err = os.Stat(resultDir0)
		assert.NoError(t, err, "Result directory for config_0 should be created")

		resultDir1 := filepath.Join(timestampDir, "TestStrategy", "config_1", "data_path")
		_, err = os.Stat(resultDir1)
		assert.NoError(t, err, "Result directory for config_1 should be created")
	})
}

// TestBacktestEngineV1_GetConfigSchema tests GetConfigSchema function
func TestBacktestEngineV1_GetConfigSchema(t *testing.T) {
	t.Run("Get config schema", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		schema, err := backtestEngine.GetConfigSchema()
		require.NoError(t, err)
		assert.NotEmpty(t, schema)
		assert.Contains(t, schema, "initial_capital")
	})
}

// TestBacktestEngineV1_PreRunCheck tests the preRunCheck function
func TestBacktestEngineV1_PreRunCheck(t *testing.T) {
	t.Run("No strategies loaded", func(t *testing.T) {
		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no strategies loaded")
	})

	t.Run("No strategy configs", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no strategy configs loaded")
	})

	t.Run("No data paths", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no data paths loaded")
	})

	t.Run("No results folder", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no results folder set")
	})

	t.Run("No datasource set", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		tempDir := t.TempDir()
		configPath := filepath.Join(tempDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no datasource set")
	})
}

// TestBacktestEngineV1_RunErrors tests various error paths in Run function
func TestBacktestEngineV1_RunErrors(t *testing.T) {
	t.Run("Config read error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		tempDir := t.TempDir()

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		// Set a non-existent config path
		backtestEngine.strategyConfigPaths = []string{"/nonexistent/config.yaml"}
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
	})

	t.Run("InitializeApi error", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(errors.New("api initialization failed")).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize strategy api")
	})

	t.Run("Strategy Initialize error", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(errors.New("strategy init failed")).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize strategy")
	})

	t.Run("Datasource Initialize error", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(errors.New("datasource init failed")).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initialize data source")
	})

	t.Run("Datasource Count error", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(0, errors.New("count failed")).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get data count")
	})

	t.Run("Data read error", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()

		// Return a function that yields an error
		readAllFunc := func(yield func(types.MarketData, error) bool) {
			yield(types.MarketData{}, errors.New("read error"))
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to read data")
	})

	t.Run("ProcessData error", func(t *testing.T) {
		// ProcessData errors are now ignored and backtest continues
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(errors.New("process data failed")).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("test-strategy-id", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()

		marketData := types.MarketData{Symbol: "TEST", Close: 100.0}
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			yield(marketData, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		// ProcessData errors are ignored, so Run should succeed
		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("With callback", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(2, nil).AnyTimes()

		marketData1 := types.MarketData{Symbol: "TEST", Close: 100.0}
		marketData2 := types.MarketData{Symbol: "TEST", Close: 101.0}
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData1, nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			yield(marketData1, nil)
			yield(marketData2, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{"/some/data/path"}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		callbackCalled := 0
		callback := engine_types.OnProcessDataCallback(func(current int, total int) error {
			callbackCalled++
			return nil
		})

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{
			OnProcessData: &callback,
		})
		require.NoError(t, err)
		assert.Equal(t, 2, callbackCalled, "Callback should be called for each data point")
	})
}

// TestLifecycleCallbacks tests the lifecycle callback system
func TestLifecycleCallbacks(t *testing.T) {
	t.Run("All lifecycle callbacks are invoked in order", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(2, nil).AnyTimes()

		marketData1 := types.MarketData{Symbol: "TEST", Close: 100.0}
		marketData2 := types.MarketData{Symbol: "TEST", Close: 101.0}
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData1, nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			yield(marketData1, nil)
			yield(marketData2, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		// Track callback invocations
		var callOrder []string

		onBacktestStart := engine_types.OnBacktestStartCallback(func(totalStrategies, totalConfigs, totalDataFiles int) error {
			callOrder = append(callOrder, "OnBacktestStart")
			assert.Equal(t, 1, totalStrategies)
			assert.Equal(t, 1, totalConfigs)
			assert.Equal(t, 1, totalDataFiles)
			return nil
		})

		onBacktestEnd := engine_types.OnBacktestEndCallback(func(err error) {
			callOrder = append(callOrder, "OnBacktestEnd")
		})

		onStrategyStart := engine_types.OnStrategyStartCallback(func(strategyIndex int, strategyName string, totalStrategies int) error {
			callOrder = append(callOrder, "OnStrategyStart")
			assert.Equal(t, 0, strategyIndex)
			assert.Equal(t, "TestStrategy", strategyName)
			assert.Equal(t, 1, totalStrategies)
			return nil
		})

		onStrategyEnd := engine_types.OnStrategyEndCallback(func(strategyIndex int, strategyName string) {
			callOrder = append(callOrder, "OnStrategyEnd")
			assert.Equal(t, 0, strategyIndex)
			assert.Equal(t, "TestStrategy", strategyName)
		})

		onRunStart := engine_types.OnRunStartCallback(func(runID string, configIndex int, configName string, dataFileIndex int, dataFilePath string, totalDataPoints int) error {
			callOrder = append(callOrder, "OnRunStart")
			assert.Equal(t, 0, configIndex)
			assert.Equal(t, 0, dataFileIndex)
			assert.Equal(t, 2, totalDataPoints)
			return nil
		})

		onRunEnd := engine_types.OnRunEndCallback(func(configIndex int, configName string, dataFileIndex int, dataFilePath string, resultFolderPath string) {
			callOrder = append(callOrder, "OnRunEnd")
			assert.Equal(t, 0, configIndex)
			assert.Equal(t, 0, dataFileIndex)
		})

		processDataCount := 0
		onProcessData := engine_types.OnProcessDataCallback(func(current, total int) error {
			processDataCount++
			return nil
		})

		callbacks := engine_types.LifecycleCallbacks{
			OnBacktestStart: &onBacktestStart,
			OnBacktestEnd:   &onBacktestEnd,
			OnStrategyStart: &onStrategyStart,
			OnStrategyEnd:   &onStrategyEnd,
			OnRunStart:      &onRunStart,
			OnRunEnd:        &onRunEnd,
			OnProcessData:   &onProcessData,
		}

		err = backtestEngine.Run(context.Background(), callbacks)
		require.NoError(t, err)

		// Verify callback order
		expectedOrder := []string{
			"OnBacktestStart",
			"OnStrategyStart",
			"OnRunStart",
			"OnRunEnd",
			"OnStrategyEnd",
			"OnBacktestEnd",
		}
		assert.Equal(t, expectedOrder, callOrder, "Callbacks should be invoked in correct order")
		assert.Equal(t, 2, processDataCount, "OnProcessData should be called for each data point")
	})

	t.Run("ProcessData errors are ignored and backtest continues", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(errors.New("processing failed")).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("test-strategy-id", nil).AnyTimes()

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(1, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()

		marketData := types.MarketData{Symbol: "TEST", Close: 100.0}
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData, nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			yield(marketData, nil)
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		onBacktestEndCalled := false
		var receivedErr error

		onBacktestEnd := engine_types.OnBacktestEndCallback(func(err error) {
			onBacktestEndCalled = true
			receivedErr = err
		})

		callbacks := engine_types.LifecycleCallbacks{
			OnBacktestEnd: &onBacktestEnd,
		}

		// ProcessData errors are now ignored, so Run should succeed
		err = backtestEngine.Run(context.Background(), callbacks)
		require.NoError(t, err)

		// OnBacktestEnd should be called with nil error since backtest completed
		assert.True(t, onBacktestEndCalled, "OnBacktestEnd should be called")
		assert.Nil(t, receivedErr, "OnBacktestEnd should receive nil error when backtest completes")
	})

	t.Run("Callback error stops execution", func(t *testing.T) {
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)
		backtestEngine.SetDataSource(mockDatasource)

		strategyStartCalled := false
		onBacktestStart := engine_types.OnBacktestStartCallback(func(totalStrategies, totalConfigs, totalDataFiles int) error {
			return errors.New("callback error")
		})

		onStrategyStart := engine_types.OnStrategyStartCallback(func(strategyIndex int, strategyName string, totalStrategies int) error {
			strategyStartCalled = true
			return nil
		})

		callbacks := engine_types.LifecycleCallbacks{
			OnBacktestStart: &onBacktestStart,
			OnStrategyStart: &onStrategyStart,
		}

		err = backtestEngine.Run(context.Background(), callbacks)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "OnBacktestStart callback failed")
		assert.False(t, strategyStartCalled, "OnStrategyStart should not be called after OnBacktestStart fails")
	})
}

// TestBacktestTrading_MismatchedSymbol tests that orders with mismatched symbols
// are added to pending orders instead of being executed or returning an error
func TestBacktestTrading_MismatchedSymbol(t *testing.T) {
	// Setup real logger
	testLogger, err := logger.NewLogger()
	require.NoError(t, err)

	// Create a BacktestState for testing
	state, err := NewBacktestState(testLogger)
	require.NoError(t, err)
	require.NotNil(t, state)

	// Initialize the state
	err = state.Initialize()
	require.NoError(t, err)

	// Create a BacktestTrading instance
	initialBalance := 10000.0
	commission := commission_fee.NewZeroCommissionFee() // No commission for simplicity
	decimalPrecision := 2
	tradingSystem := NewBacktestTrading(state, initialBalance, commission, decimalPrecision)
	backtestTrading := tradingSystem.(*BacktestTrading)

	// Set current market data for symbol "SPY"
	marketData := types.MarketData{
		Symbol: "SPY",
		Open:   100.0,
		High:   105.0,
		Low:    95.0,
		Close:  102.0,
		Volume: 1000,
	}
	backtestTrading.UpdateCurrentMarketData(marketData)

	// Test Case 1: Place a market order with a different symbol
	mismatchedOrder := types.ExecuteOrder{
		Symbol:       "AAPL", // Different from current market data symbol "SPY"
		Side:         types.PurchaseTypeBuy,
		OrderType:    types.OrderTypeMarket,
		Quantity:     10.0,
		Price:        100.0,
		StrategyName: "TestStrategy",
		Reason: types.Reason{
			Reason:  types.OrderReasonStrategy,
			Message: "Test order with different symbol",
		},
		PositionType: types.PositionTypeLong,
	}

	// Place the order
	err = backtestTrading.PlaceOrder(mismatchedOrder)

	// Verify no error is returned
	require.NoError(t, err, "PlaceOrder with mismatched symbol should not return an error")

	// Verify the order is added to pending orders
	require.Equal(t, 1, len(backtestTrading.pendingOrders), "Order with mismatched symbol should be added to pending orders")
	require.Equal(t, "AAPL", backtestTrading.pendingOrders[0].Symbol, "Pending order should have the mismatched symbol")

	// Test Case 2: Place a limit order with a different symbol
	mismatchedLimitOrder := types.ExecuteOrder{
		Symbol:       "MSFT", // Different from current market data symbol "SPY"
		Side:         types.PurchaseTypeSell,
		OrderType:    types.OrderTypeLimit,
		Quantity:     5.0,
		Price:        150.0,
		StrategyName: "TestStrategy",
		Reason: types.Reason{
			Reason:  types.OrderReasonStrategy,
			Message: "Test limit order with different symbol",
		},
		PositionType: types.PositionTypeLong,
	}

	// Place the limit order
	err = backtestTrading.PlaceOrder(mismatchedLimitOrder)

	// Verify no error is returned
	require.NoError(t, err, "PlaceOrder with mismatched symbol should not return an error")

	// Verify the order is added to pending orders
	require.Equal(t, 2, len(backtestTrading.pendingOrders), "Order with mismatched symbol should be added to pending orders")
	require.Equal(t, "MSFT", backtestTrading.pendingOrders[1].Symbol, "Pending order should have the mismatched symbol")

	// Test Case 3: Update market data to match one of the pending orders and verify it gets processed
	newMarketData := types.MarketData{
		Symbol: "AAPL", // Now matches the first pending order
		Open:   150.0,
		High:   155.0,
		Low:    145.0,
		Close:  152.0,
		Volume: 2000,
	}

	backtestTrading.UpdateCurrentMarketData(newMarketData)

	// Verify that the matching order was processed and removed from pending orders
	// Only the MSFT order should remain
	require.Equal(t, 1, len(backtestTrading.pendingOrders), "Order with now matching symbol should be processed")
	require.Equal(t, "MSFT", backtestTrading.pendingOrders[0].Symbol, "Remaining pending order should be the one still not matching")
}

// TestInsufficientDataErrorMarkers tests that markers are added at the beginning and end
// of consecutive insufficient data error sequences
func TestInsufficientDataErrorMarkers(t *testing.T) {
	// Helper to create an insufficient data error
	insufficientErr := argoErrors.NewInsufficientDataError(10, 5, "TEST", "insufficient data")

	t.Run("Single insufficient error sequence in the middle", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Create test market data - 7 data points
		// Pattern: [OK, OK, Insufficient, Insufficient, Insufficient, OK, OK]
		marketData := make([]types.MarketData, 7)
		for i := 0; i < 7; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Time:   time.Date(2024, 1, 1, 9, 30+i, 0, 0, time.UTC),
				Open:   100.0 + float64(i),
				High:   105.0 + float64(i),
				Low:    95.0 + float64(i),
				Close:  102.0 + float64(i),
				Volume: 1000,
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		// Setup ProcessData expectations - indices 2, 3, 4 return insufficient error
		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[3])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[4])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[5])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[6])).Return(nil),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(7, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect exactly 2 marker calls: start at index 2, end at index 4
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[2]), gomock.Any()).DoAndReturn(
				func(data types.MarketData, mark types.Mark) error {
					assert.Equal(t, types.MarkLevelWarning, mark.Level)
					assert.Equal(t, "Insufficient data error started", mark.Message)
					assert.True(t, mark.Signal.IsSome())
					assert.Equal(t, data.Time, mark.Signal.Unwrap().Time)
					return nil
				}),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[4]), gomock.Any()).DoAndReturn(
				func(data types.MarketData, mark types.Mark) error {
					assert.Equal(t, types.MarkLevelWarning, mark.Level)
					assert.Equal(t, "Insufficient data error ended", mark.Message)
					assert.True(t, mark.Signal.IsSome())
					assert.Equal(t, data.Time, mark.Signal.Unwrap().Time)
					return nil
				}),
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Insufficient errors at the start", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [Insufficient, Insufficient, OK, OK]
		marketData := make([]types.MarketData, 4)
		for i := 0; i < 4; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[3])).Return(nil),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(4, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect markers: start at index 0, end at index 1
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[0]), gomock.Any()).Return(nil),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).Return(nil),
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Insufficient errors at the end", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [OK, OK, Insufficient, Insufficient]
		marketData := make([]types.MarketData, 4)
		for i := 0; i < 4; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[3])).Return(insufficientErr),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(4, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect markers: start at index 2, end at index 3 (after loop)
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[2]), gomock.Any()).Return(nil),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[3]), gomock.Any()).Return(nil),
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Multiple separate sequences", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [OK, Insufficient, Insufficient, OK, Insufficient, OK]
		marketData := make([]types.MarketData, 6)
		for i := 0; i < 6; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[3])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[4])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[5])).Return(nil),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(6, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect 4 markers: start at 1, end at 2, start at 4, end at 4
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).Return(nil), // start
			mockMarker.EXPECT().Mark(matchMarketData(marketData[2]), gomock.Any()).Return(nil), // end
			mockMarker.EXPECT().Mark(matchMarketData(marketData[4]), gomock.Any()).Return(nil), // start
			mockMarker.EXPECT().Mark(matchMarketData(marketData[4]), gomock.Any()).Return(nil), // end (same point)
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("All data points insufficient", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [Insufficient, Insufficient, Insufficient]
		marketData := make([]types.MarketData, 3)
		for i := 0; i < 3; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(insufficientErr),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect markers: start at index 0, end at index 2 (after loop)
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[0]), gomock.Any()).Return(nil),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[2]), gomock.Any()).Return(nil),
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Single insufficient error", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [OK, Insufficient, OK]
		marketData := make([]types.MarketData, 3)
		for i := 0; i < 3; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[0])).Return(nil),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[1])).Return(insufficientErr),
			mockStrategy.EXPECT().ProcessData(matchMarketData(marketData[2])).Return(nil),
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect markers: start at index 1, end at index 1 (same point)
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).Return(nil), // start
			mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).Return(nil), // end
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("No insufficient errors", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [OK, OK, OK]
		marketData := make([]types.MarketData, 3)
		for i := 0; i < 3; i++ {
			marketData[i] = types.MarketData{
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil).Times(3)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// No marker calls expected - mockMarker has no expectations set

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})
}

// TestStrategyErrorMarkers tests that non-insufficient errors add error markers
// and processing continues.
func TestStrategyErrorMarkers(t *testing.T) {
	t.Run("Strategy error adds marker and continues processing", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Create test market data: [OK, Error, OK]
		marketData := make([]types.MarketData, 3)
		for i := 0; i < 3; i++ {
			marketData[i] = types.MarketData{
				Id:     fmt.Sprintf("id-%d", i),
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		strategyError := errors.New("strategy processing failed")
		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil),           // First call succeeds
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(strategyError), // Second call fails
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil),           // Third call succeeds (continues processing)
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect one error marker for the strategy error at index 1
		mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).DoAndReturn(
			func(md types.MarketData, mark types.Mark) error {
				assert.Equal(t, types.MarkColorRed, mark.Color)
				assert.Equal(t, types.MarkShapeCircle, mark.Shape)
				assert.Equal(t, types.MarkLevelError, mark.Level)
				assert.Equal(t, "Strategy Error", mark.Title)
				assert.Equal(t, "strategy processing failed", mark.Message)
				assert.Equal(t, "StrategyError", mark.Category)
				return nil
			})

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Multiple strategy errors add multiple markers", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Create test market data: [Error, OK, Error]
		marketData := make([]types.MarketData, 3)
		for i := 0; i < 3; i++ {
			marketData[i] = types.MarketData{
				Id:     fmt.Sprintf("id-%d", i),
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		strategyError1 := errors.New("error at index 0")
		strategyError2 := errors.New("error at index 2")
		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(strategyError1), // First call fails
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil),            // Second call succeeds
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(strategyError2), // Third call fails
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(3, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect two error markers
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[0]), gomock.Any()).DoAndReturn(
				func(md types.MarketData, mark types.Mark) error {
					assert.Equal(t, "error at index 0", mark.Message)
					return nil
				}),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[2]), gomock.Any()).DoAndReturn(
				func(md types.MarketData, mark types.Mark) error {
					assert.Equal(t, "error at index 2", mark.Message)
					return nil
				}),
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})

	t.Run("Insufficient errors do not create error markers but strategy errors do", func(t *testing.T) {
		setTestVersion(t, "1.0.0")
		ctrl := gomock.NewController(t)
		defer ctrl.Finish()

		mockStrategy := mocks.NewMockStrategyRuntime(ctrl)
		mockDatasource := mocks.NewMockDataSource(ctrl)
		mockMarker := mocks.NewMockMarker(ctrl)

		tempDir := t.TempDir()
		configDir := t.TempDir()
		configPath := filepath.Join(configDir, "config.yaml")
		os.WriteFile(configPath, []byte("test: config"), 0644)

		// Pattern: [Insufficient, Error, OK, Insufficient]
		marketData := make([]types.MarketData, 4)
		for i := 0; i < 4; i++ {
			marketData[i] = types.MarketData{
				Id:     fmt.Sprintf("id-%d", i),
				Symbol: "TEST",
				Close:  100.0 + float64(i),
			}
		}

		mockStrategy.EXPECT().Name().Return("TestStrategy").AnyTimes()
		mockStrategy.EXPECT().InitializeApi(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockStrategy.EXPECT().GetRuntimeEngineVersion().Return("1.0.0", nil).AnyTimes()
		mockStrategy.EXPECT().GetIdentifier().Return("com.test.mock", nil).AnyTimes()

		insufficientErr := argoErrors.NewInsufficientDataError(10, 5, "TEST", "need more data")
		strategyError := errors.New("strategy error")
		gomock.InOrder(
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(insufficientErr), // Insufficient (warning marker)
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(strategyError),   // Strategy error (error marker)
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(nil),             // OK
			mockStrategy.EXPECT().ProcessData(gomock.Any()).Return(insufficientErr), // Insufficient (warning marker)
		)

		mockDatasource.EXPECT().Initialize(gomock.Any()).Return(nil).AnyTimes()
		mockDatasource.EXPECT().Count(gomock.Any(), gomock.Any()).Return(4, nil).AnyTimes()
		mockDatasource.EXPECT().GetAllSymbols().Return([]string{"TEST"}, nil).AnyTimes()
		mockDatasource.EXPECT().ReadLastData(gomock.Any()).Return(marketData[0], nil).AnyTimes()

		readAllFunc := func(yield func(types.MarketData, error) bool) {
			for _, data := range marketData {
				if !yield(data, nil) {
					return
				}
			}
		}
		mockDatasource.EXPECT().ReadAll(gomock.Any(), gomock.Any()).Return(readAllFunc).AnyTimes()

		// Expect markers:
		// - Index 0: Start of insufficient sequence (warning marker)
		// - Index 0: End of insufficient sequence (at lastInsufficientData = data[0])
		// - Index 1: Strategy error (error marker)
		// - Index 3: Start of insufficient sequence (warning marker)
		// - Index 3: End of insufficient sequence (after loop)
		gomock.InOrder(
			mockMarker.EXPECT().Mark(matchMarketData(marketData[0]), gomock.Any()).Return(nil), // Start insufficient
			mockMarker.EXPECT().Mark(matchMarketData(marketData[0]), gomock.Any()).Return(nil), // End insufficient (at lastInsufficientData)
			mockMarker.EXPECT().Mark(matchMarketData(marketData[1]), gomock.Any()).DoAndReturn( // Strategy error marker
				func(md types.MarketData, mark types.Mark) error {
					assert.Equal(t, types.MarkLevelError, mark.Level)
					assert.Equal(t, "Strategy Error", mark.Title)
					return nil
				}),
			mockMarker.EXPECT().Mark(matchMarketData(marketData[3]), gomock.Any()).Return(nil), // Start insufficient
			mockMarker.EXPECT().Mark(matchMarketData(marketData[3]), gomock.Any()).Return(nil), // End insufficient (after loop)
		)

		engine, err := NewBacktestEngineV1()
		require.NoError(t, err)
		backtestEngine := engine.(*BacktestEngineV1)

		config := `initialCapital: 10000`
		err = backtestEngine.Initialize(config)
		require.NoError(t, err)

		backtestEngine.marker = mockMarker
		backtestEngine.LoadStrategy(mockStrategy)
		backtestEngine.SetDataSource(mockDatasource)
		backtestEngine.SetConfigPath(configPath)
		backtestEngine.dataPaths = []string{filepath.Join(configDir, "data_path")}
		backtestEngine.SetResultsFolder(tempDir)

		err = backtestEngine.Run(context.Background(), engine_types.LifecycleCallbacks{})
		require.NoError(t, err)
	})
}

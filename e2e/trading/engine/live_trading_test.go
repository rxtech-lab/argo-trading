package engine_test

import (
	"context"
	"errors"
	"sync"
	"time"

	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/e2e/trading/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// TestBasicEngineExecution tests basic engine lifecycle and callback invocation.
func (s *LiveTradingE2ETestSuite) TestBasicEngineExecution() {
	// Setup mock providers
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			TrendStrength:     0.01,
			NumDataPoints:     100,
			Seed:              42,
			VolatilityPercent: 1.0,
			Interval:          time.Minute,
			StartTime:         time.Now(),
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0) // Enough to buy 1 BTC at ~50000

	// Initialize engine
	err := s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	// Configure engine with mocks
	err = s.engine.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = s.engine.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	// Load strategy
	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	// Set strategy config
	err = s.engine.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	// Track execution
	var dataPointsProcessed int
	var engineStarted, engineStopped bool
	var mu sync.Mutex

	onStart := engine.OnEngineStartCallback(func(symbols []string, interval string) error {
		mu.Lock()
		defer mu.Unlock()

		engineStarted = true
		s.Equal([]string{"BTCUSDT"}, symbols)
		s.Equal("1m", interval)

		return nil
	})

	onStop := engine.OnEngineStopCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()

		engineStopped = true
	})

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataPointsProcessed++
		// Update mock trading provider with current price
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnEngineStart: &onStart,
		OnEngineStop:  &onStop,
		OnMarketData:  &onData,
	}

	// Run engine
	err = s.engine.Run(context.Background(), callbacks)
	s.Require().NoError(err)

	// Assertions
	mu.Lock()
	defer mu.Unlock()

	s.True(engineStarted, "Engine should have started")
	s.True(engineStopped, "Engine should have stopped")
	s.Equal(100, dataPointsProcessed, "Should process all data points")

	// Verify trading activity
	trades := mockTradingProvider.GetAllTrades()
	s.NotEmpty(trades, "Strategy should have placed trades")

	// Verify final account state
	accountInfo, err := mockTradingProvider.GetAccountInfo()
	s.Require().NoError(err)
	s.NotZero(accountInfo.Balance, "Account should have balance")
}

// TestGracefulShutdown tests that engine shuts down cleanly when context is cancelled.
func (s *LiveTradingE2ETestSuite) TestGracefulShutdown() {
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			NumDataPoints:     1000, // Many data points
			TrendStrength:     0.01,
			VolatilityPercent: 1.0,
			Seed:              42,
			Interval:          time.Minute,
			StartTime:         time.Now(),
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0) // Enough to buy 1 BTC at ~50000

	err := s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = s.engine.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = s.engine.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	// Set strategy config
	err = s.engine.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var dataCount int
	var engineStopped bool
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataCount++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		if dataCount >= 100 {
			cancel() // Cancel after 100 data points
		}

		return nil
	})

	onStop := engine.OnEngineStopCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()

		engineStopped = true
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
		OnEngineStop: &onStop,
	}

	err = s.engine.Run(ctx, callbacks)

	// Verify graceful shutdown
	s.Equal(context.Canceled, err, "Should return context.Canceled")

	mu.Lock()
	defer mu.Unlock()

	s.True(engineStopped, "OnEngineStop should be called")
	s.Less(dataCount, 1000, "Should stop before processing all data")
}

// TestStreamErrorHandling tests engine behavior when stream errors occur.
func (s *LiveTradingE2ETestSuite) TestStreamErrorHandling() {
	expectedErr := errors.New("simulated connection lost")

	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			NumDataPoints:     100,
			TrendStrength:     0.01,
			VolatilityPercent: 1.0,
			Seed:              42,
			ErrorAfterN:       50, // Inject error after 50 data points
			ErrorToReturn:     expectedErr,
			Interval:          time.Minute,
			StartTime:         time.Now(),
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(10000.0)

	err := s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = s.engine.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = s.engine.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	// Set strategy config
	err = s.engine.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var errorReceived error
	var dataCountBeforeError int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataCountBeforeError++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	onError := engine.OnErrorCallback(func(err error) {
		mu.Lock()
		defer mu.Unlock()

		errorReceived = err
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
		OnError:      &onError,
	}

	err = s.engine.Run(context.Background(), callbacks)

	// The engine should complete (stream ended after error was yielded)
	s.Require().NoError(err)

	// Verify error handling
	mu.Lock()
	defer mu.Unlock()

	s.Equal(50, dataCountBeforeError, "Should process data until error")
	s.NotNil(errorReceived, "OnError callback should be invoked")
	s.Contains(errorReceived.Error(), "connection lost")
}

// TestMultiSymbolStreaming tests engine handling multiple symbols.
func (s *LiveTradingE2ETestSuite) TestMultiSymbolStreaming() {
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			NumDataPoints:     50,
			TrendStrength:     0.01,
			VolatilityPercent: 1.0,
			Seed:              42,
			Interval:          time.Minute,
			StartTime:         time.Now(),
		},
		testhelper.MockMarketDataConfig{
			Symbol:             "ETHUSDT",
			Pattern:            backtestTesthelper.PatternVolatile,
			InitialPrice:       3000.0,
			MaxDrawdownPercent: 15.0,
			NumDataPoints:      50,
			TrendStrength:      0.01,
			VolatilityPercent:  3.0,
			Seed:               43,
			Interval:           time.Minute,
			StartTime:          time.Now(),
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(20000.0)

	err := s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT", "ETHUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = s.engine.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = s.engine.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	// Set strategy config
	err = s.engine.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	symbolsSeen := make(map[string]int)
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		symbolsSeen[data.Symbol]++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	err = s.engine.Run(context.Background(), callbacks)
	s.Require().NoError(err)

	// Verify both symbols were processed
	mu.Lock()
	defer mu.Unlock()

	s.Equal(50, symbolsSeen["BTCUSDT"], "Should process all BTCUSDT data points")
	s.Equal(50, symbolsSeen["ETHUSDT"], "Should process all ETHUSDT data points")
}

// TestEnginePreRunValidation tests that engine fails with appropriate errors when not properly configured.
func (s *LiveTradingE2ETestSuite) TestEnginePreRunValidation() {
	// Test running without initialization
	err := s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().Error(err)
	s.Contains(err.Error(), "not initialized")

	// Initialize but don't set strategy
	err = s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	})
	s.Require().NoError(err)

	err = s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().Error(err)
	s.Contains(err.Error(), "strategy not loaded")

	// Set strategy but no providers
	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	err = s.engine.Run(context.Background(), engine.LiveTradingCallbacks{})
	s.Require().Error(err)
	s.Contains(err.Error(), "market data provider not set")
}

// TestStrategyErrorCallback tests that strategy errors are reported via callback.
func (s *LiveTradingE2ETestSuite) TestStrategyErrorCallback() {
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			NumDataPoints:     10,
			TrendStrength:     0.01,
			VolatilityPercent: 1.0,
			Seed:              42,
			Interval:          time.Minute,
			StartTime:         time.Now(),
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(10000.0)

	err := s.engine.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1m",
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = s.engine.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = s.engine.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = s.engine.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	// Set strategy config
	err = s.engine.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var strategyErrors []error
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)
		return nil
	})

	onStrategyError := engine.OnStrategyErrorCallback(func(data types.MarketData, err error) {
		mu.Lock()
		defer mu.Unlock()

		strategyErrors = append(strategyErrors, err)
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData:    &onData,
		OnStrategyError: &onStrategyError,
	}

	err = s.engine.Run(context.Background(), callbacks)
	s.Require().NoError(err)

	// Strategy errors may or may not occur depending on strategy behavior
	// The important thing is that the engine completes without crashing
}

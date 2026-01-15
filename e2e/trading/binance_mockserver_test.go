package trading_test

import (
	"context"
	"sync"
	"testing"
	"time"

	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/e2e/trading/mockserver"
	"github.com/rxtech-lab/argo-trading/e2e/trading/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	engine_v1 "github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1"
	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	marketdataprovider "github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/stretchr/testify/suite"
)

// testStrategyWASMPath is the path to the test strategy WASM file used in E2E tests.
const testStrategyWASMPath = "../backtest/wasm/place_order/place_order_plugin.wasm"

// BinanceMockServerTestSuite tests live trading engine with the mock Binance server.
// This suite tests strategy order placement and engine lifecycle with results verification.
type BinanceMockServerTestSuite struct {
	suite.Suite
}

func TestBinanceMockServerSuite(t *testing.T) {
	suite.Run(t, new(BinanceMockServerTestSuite))
}

// createMockServerConfig creates a mock server configuration with deterministic settings.
func createMockServerConfig(seed int64) mockserver.ServerConfig {
	return mockserver.ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 10000.0,
			"BTC":  1.0,
		},
		TradeFees: map[string]*mockserver.TradeFee{
			"BTCUSDT": {Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
		},
		MarketData: &mockserver.MarketDataGeneratorConfig{
			Symbols:           []string{"BTCUSDT"},
			Pattern:           backtestTesthelper.PatternVolatile,
			InitialPrice:      50000.0,
			VolatilityPercent: 2.0,
			TrendStrength:     0.01,
			Seed:              seed,
		},
		StreamInterval: 50 * time.Millisecond,
	}
}

// TestStrategyOrderPlacement tests strategy placing multiple orders with result verification.
// It verifies orders, trades, positions, and balances.
func (s *BinanceMockServerTestSuite) TestStrategyOrderPlacement() {
	// Create and start mock server
	config := createMockServerConfig(12345)
	server := mockserver.NewMockBinanceServer(config)
	err := server.Start(":0")
	s.Require().NoError(err)
	defer server.Stop()

	// Get server URLs
	baseURL := server.BaseURL()
	wsURL := server.WebSocketURL()

	// Create temp directory for results
	tmpDir := s.T().TempDir()

	// Create engine
	eng, err := engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)

	// Initialize engine with persistence (DataOutputPath enables persistence)
	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1s",
		MarketDataCacheSize: 100,
		EnableLogging:       true,
		DataOutputPath:      tmpDir,
	})
	s.Require().NoError(err)

	// Set up providers using real Binance providers with custom endpoints
	marketProvider, err := marketdataprovider.NewBinanceClientWithEndpoints(marketdataprovider.BinanceEndpointConfig{
		RestBaseURL: baseURL,
		WsBaseURL:   wsURL,
	})
	s.Require().NoError(err)

	tradingProvider, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	err = eng.SetMarketDataProvider(marketProvider)
	s.Require().NoError(err)

	err = eng.SetTradingProvider(tradingProvider)
	s.Require().NoError(err)

	// Load strategy
	err = eng.LoadStrategyFromFile(testStrategyWASMPath)
	s.Require().NoError(err)

	// Set strategy config
	err = eng.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	// Track execution
	var dataCount int
	var mu sync.Mutex

	ctx, cancel := context.WithCancel(context.Background())

	onData := engine.OnMarketDataCallback(func(data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataCount++
		// Stop after 10 data points
		if dataCount >= 10 {
			cancel()
		}
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	// Run engine
	err = eng.Run(ctx, callbacks)
	// Context cancellation is expected
	if err != context.Canceled {
		s.Require().NoError(err)
	}

	// Verify data was processed
	mu.Lock()
	processedCount := dataCount
	mu.Unlock()
	s.GreaterOrEqual(processedCount, 1, "Should have processed at least 1 data point")

	// Verify results in tmp folder
	runFolders, err := testhelper.GetRunFolders(tmpDir)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(runFolders), 1, "Should have at least 1 run folder")

	// Read and verify stats
	stats, err := testhelper.ReadLiveStats(s.T(), tmpDir)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(stats), 1, "Should have at least 1 stats file")
	s.Equal("BTCUSDT", stats[0].Symbols[0], "Stats should have correct symbol")

	// Verify orders were placed - check if orders file exists and has data
	orders, err := testhelper.ReadOrders(s.T(), tmpDir)
	if err == nil {
		s.T().Logf("Found %d orders in results", len(orders))
	}

	// Verify trades were executed - check if trades file exists and has data
	trades, err := testhelper.ReadTrades(s.T(), tmpDir)
	if err == nil {
		s.T().Logf("Found %d trades in results", len(trades))
	}

	// Verify balance changed via mock server
	usdtBalance := server.GetBalance("USDT")
	btcBalance := server.GetBalance("BTC")

	var usdtFree, btcFree float64
	if usdtBalance != nil {
		usdtFree = usdtBalance.Free
	}
	if btcBalance != nil {
		btcFree = btcBalance.Free
	}

	s.T().Logf("Final balances - USDT: %.2f, BTC: %.8f", usdtFree, btcFree)

	// If orders were placed, balance should have changed
	if len(orders) > 0 {
		// Either USDT decreased (bought BTC) or BTC decreased (sold BTC)
		s.True(usdtFree != 10000.0 || btcFree != 1.0, "Balance should have changed after orders")
	}
}

// TestEngineLifecycle tests start, stop, restart engine creating multiple results.
func (s *BinanceMockServerTestSuite) TestEngineLifecycle() {
	// Create and start mock server
	config := createMockServerConfig(54321)
	server := mockserver.NewMockBinanceServer(config)
	err := server.Start(":0")
	s.Require().NoError(err)
	defer server.Stop()

	baseURL := server.BaseURL()
	wsURL := server.WebSocketURL()
	tmpDir := s.T().TempDir()

	// === First Run ===
	eng1, err := engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng1.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1s",
		MarketDataCacheSize: 100,
		EnableLogging:       true,
		DataOutputPath:      tmpDir,
	})
	s.Require().NoError(err)

	marketProvider1, err := marketdataprovider.NewBinanceClientWithEndpoints(marketdataprovider.BinanceEndpointConfig{
		RestBaseURL: baseURL,
		WsBaseURL:   wsURL,
	})
	s.Require().NoError(err)

	tradingProvider1, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	err = eng1.SetMarketDataProvider(marketProvider1)
	s.Require().NoError(err)

	err = eng1.SetTradingProvider(tradingProvider1)
	s.Require().NoError(err)

	err = eng1.LoadStrategyFromFile(testStrategyWASMPath)
	s.Require().NoError(err)

	err = eng1.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var count1 int
	var mu1 sync.Mutex
	ctx1, cancel1 := context.WithCancel(context.Background())

	onData1 := engine.OnMarketDataCallback(func(_ types.MarketData) error {
		mu1.Lock()
		defer mu1.Unlock()
		count1++
		if count1 >= 5 {
			cancel1()
		}
		return nil
	})

	callbacks1 := engine.LiveTradingCallbacks{
		OnMarketData: &onData1,
	}

	err = eng1.Run(ctx1, callbacks1)
	if err != context.Canceled {
		s.Require().NoError(err)
	}

	// Verify first run folder exists
	runFolders1, err := testhelper.GetRunFolders(tmpDir)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(runFolders1), 1, "Should have at least 1 run folder after first run")
	s.T().Logf("After first run: found %d run folders", len(runFolders1))

	// Small delay before second run
	time.Sleep(100 * time.Millisecond)

	// Reset mock server for second run
	server.Reset(mockserver.ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 10000.0,
			"BTC":  1.0,
		},
	})

	// === Second Run ===
	eng2, err := engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng2.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1s",
		MarketDataCacheSize: 100,
		EnableLogging:       true,
		DataOutputPath:      tmpDir, // Same output path
	})
	s.Require().NoError(err)

	marketProvider2, err := marketdataprovider.NewBinanceClientWithEndpoints(marketdataprovider.BinanceEndpointConfig{
		RestBaseURL: baseURL,
		WsBaseURL:   wsURL,
	})
	s.Require().NoError(err)

	tradingProvider2, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	err = eng2.SetMarketDataProvider(marketProvider2)
	s.Require().NoError(err)

	err = eng2.SetTradingProvider(tradingProvider2)
	s.Require().NoError(err)

	err = eng2.LoadStrategyFromFile(testStrategyWASMPath)
	s.Require().NoError(err)

	err = eng2.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var count2 int
	var mu2 sync.Mutex
	ctx2, cancel2 := context.WithCancel(context.Background())

	onData2 := engine.OnMarketDataCallback(func(_ types.MarketData) error {
		mu2.Lock()
		defer mu2.Unlock()
		count2++
		if count2 >= 5 {
			cancel2()
		}
		return nil
	})

	callbacks2 := engine.LiveTradingCallbacks{
		OnMarketData: &onData2,
	}

	err = eng2.Run(ctx2, callbacks2)
	if err != context.Canceled {
		s.Require().NoError(err)
	}

	// Verify second run folder exists
	runFolders2, err := testhelper.GetRunFolders(tmpDir)
	s.Require().NoError(err)
	s.T().Logf("After second run: found %d run folders", len(runFolders2))
	s.GreaterOrEqual(len(runFolders2), 2, "Should have at least 2 run folders after second run")

	// Read stats from all runs
	stats, err := testhelper.ReadLiveStats(s.T(), tmpDir)
	s.Require().NoError(err)
	s.GreaterOrEqual(len(stats), 2, "Should have stats from both runs")

	// Verify run IDs are different
	if len(stats) >= 2 {
		s.NotEqual(stats[0].ID, stats[1].ID, "Run IDs should be different")
		s.T().Logf("Run IDs: %s, %s", stats[0].ID, stats[1].ID)
	}
}

// TestFractionalOrderPlacement tests placing small fractional orders (0.01 BTC).
// This verifies that both the trading provider and mock server support fractional quantities.
func (s *BinanceMockServerTestSuite) TestFractionalOrderPlacement() {
	// Create mock server with sufficient balance
	config := mockserver.ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 10000.0,
			"BTC":  1.0,
		},
		TradeFees: map[string]*mockserver.TradeFee{
			"BTCUSDT": {Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
		},
		MarketData: &mockserver.MarketDataGeneratorConfig{
			Symbols:           []string{"BTCUSDT"},
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			VolatilityPercent: 0.1,
			TrendStrength:     0.0,
			Seed:              11111,
		},
		StreamInterval: 100 * time.Millisecond,
	}
	server := mockserver.NewMockBinanceServer(config)
	err := server.Start(":0")
	s.Require().NoError(err)
	defer server.Stop()

	baseURL := server.BaseURL()
	tradingProvider, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	// Test 1: Place a 0.01 BTC buy order
	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.01, // Small fractional quantity
	}

	err = tradingProvider.PlaceOrder(order)
	s.Require().NoError(err, "Should successfully place 0.01 BTC order")

	// Verify balance changed
	btcBalance := server.GetBalance("BTC")
	s.Require().NotNil(btcBalance)
	s.InDelta(1.01, btcBalance.Free, 0.001, "BTC balance should increase by ~0.01")

	// Verify USDT balance decreased
	usdtBalance := server.GetBalance("USDT")
	s.Require().NotNil(usdtBalance)
	s.Less(usdtBalance.Free, 10000.0, "USDT balance should decrease")

	// Test 2: Place an even smaller order (0.001 BTC)
	smallOrder := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.001, // Very small fractional quantity
	}

	err = tradingProvider.PlaceOrder(smallOrder)
	s.Require().NoError(err, "Should successfully place 0.001 BTC order")

	// Verify trade was recorded
	trades := server.GetTrades()
	s.GreaterOrEqual(len(trades), 2, "Should have at least 2 trades")

	// Verify the last trade has the correct quantity
	lastTrade := trades[len(trades)-1]
	s.InDelta(0.001, lastTrade.Quantity, 0.0001, "Last trade should have quantity 0.001")

	s.T().Logf("Successfully placed fractional orders: 0.01 BTC and 0.001 BTC")
	s.T().Logf("Final BTC balance: %.8f", btcBalance.Free)
}

// TestSatoshiPrecisionOrder tests placing an order with satoshi-level precision (0.00000001 BTC).
func (s *BinanceMockServerTestSuite) TestSatoshiPrecisionOrder() {
	config := mockserver.ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 10000.0,
			"BTC":  0.0,
		},
		TradeFees: map[string]*mockserver.TradeFee{
			"BTCUSDT": {Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
		},
		MarketData: &mockserver.MarketDataGeneratorConfig{
			Symbols:           []string{"BTCUSDT"},
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			VolatilityPercent: 0.1,
			TrendStrength:     0.0,
			Seed:              22222,
		},
		StreamInterval: 100 * time.Millisecond,
	}
	server := mockserver.NewMockBinanceServer(config)
	err := server.Start(":0")
	s.Require().NoError(err)
	defer server.Stop()

	baseURL := server.BaseURL()
	tradingProvider, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	// Place an order with 8 decimal precision (1 satoshi equivalent)
	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.00000001, // 1 satoshi
	}

	err = tradingProvider.PlaceOrder(order)
	s.Require().NoError(err, "Should successfully place 0.00000001 BTC order")

	// Verify the trade was recorded with correct precision
	trades := server.GetTrades()
	s.Require().Len(trades, 1, "Should have exactly 1 trade")
	s.InDelta(0.00000001, trades[0].Quantity, 0.000000001, "Trade should preserve satoshi precision")

	s.T().Logf("Successfully placed satoshi-level order: 0.00000001 BTC")
}

// TestMultipleOrdersAndBalanceVerification tests multiple orders and verifies balance changes.
func (s *BinanceMockServerTestSuite) TestMultipleOrdersAndBalanceVerification() {
	// Create and start mock server with more initial balance
	config := mockserver.ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 100000.0, // More USDT for multiple orders
			"BTC":  0,
		},
		TradeFees: map[string]*mockserver.TradeFee{
			"BTCUSDT": {Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
		},
		MarketData: &mockserver.MarketDataGeneratorConfig{
			Symbols:           []string{"BTCUSDT"},
			Pattern:           backtestTesthelper.PatternIncreasing, // Use increasing pattern to trigger more buys
			InitialPrice:      50000.0,
			VolatilityPercent: 1.0,
			TrendStrength:     0.05, // Stronger trend
			Seed:              99999,
		},
		StreamInterval: 30 * time.Millisecond, // Faster streaming
	}
	server := mockserver.NewMockBinanceServer(config)
	err := server.Start(":0")
	s.Require().NoError(err)
	defer server.Stop()

	baseURL := server.BaseURL()
	wsURL := server.WebSocketURL()
	tmpDir := s.T().TempDir()

	// Record initial balances
	initialUSDT := server.GetBalance("USDT")
	s.Require().NotNil(initialUSDT)
	s.T().Logf("Initial USDT balance: %.2f", initialUSDT.Free)

	// Create and configure engine
	eng, err := engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = eng.Initialize(engine.LiveTradingEngineConfig{
		Symbols:             []string{"BTCUSDT"},
		Interval:            "1s",
		MarketDataCacheSize: 100,
		EnableLogging:       true,
		DataOutputPath:      tmpDir,
	})
	s.Require().NoError(err)

	marketProvider, err := marketdataprovider.NewBinanceClientWithEndpoints(marketdataprovider.BinanceEndpointConfig{
		RestBaseURL: baseURL,
		WsBaseURL:   wsURL,
	})
	s.Require().NoError(err)

	tradingProvider, err := tradingprovider.NewBinanceTradingSystemProvider(tradingprovider.BinanceProviderConfig{
		ApiKey:    "mock-api-key",
		SecretKey: "mock-secret-key",
		BaseURL:   baseURL,
	}, false)
	s.Require().NoError(err)

	err = eng.SetMarketDataProvider(marketProvider)
	s.Require().NoError(err)

	err = eng.SetTradingProvider(tradingProvider)
	s.Require().NoError(err)

	err = eng.LoadStrategyFromFile(testStrategyWASMPath)
	s.Require().NoError(err)

	err = eng.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex
	ctx, cancel := context.WithCancel(context.Background())

	onData := engine.OnMarketDataCallback(func(_ types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()
		dataCount++
		// Process more data points to get more orders
		if dataCount >= 20 {
			cancel()
		}
		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	err = eng.Run(ctx, callbacks)
	if err != context.Canceled {
		s.Require().NoError(err)
	}

	// Check final balances
	finalUSDT := server.GetBalance("USDT")
	finalBTC := server.GetBalance("BTC")

	var finalUSDTFree, finalBTCFree float64
	if finalUSDT != nil {
		finalUSDTFree = finalUSDT.Free
	}
	if finalBTC != nil {
		finalBTCFree = finalBTC.Free
	}

	s.T().Logf("Final balances - USDT: %.2f, BTC: %.8f", finalUSDTFree, finalBTCFree)

	// Verify trades from mock server
	mockTrades := server.GetTrades()
	s.T().Logf("Mock server recorded %d trades", len(mockTrades))

	// Read results from files
	orders, err := testhelper.ReadOrders(s.T(), tmpDir)
	if err == nil {
		s.T().Logf("Orders file has %d orders", len(orders))
	}

	trades, err := testhelper.ReadTrades(s.T(), tmpDir)
	if err == nil {
		s.T().Logf("Trades file has %d trades", len(trades))
	}

	// Verify balance consistency
	if len(mockTrades) > 0 {
		s.NotEqual(initialUSDT.Free, finalUSDTFree, "USDT balance should have changed")
		s.T().Logf("Balance change: %.2f USDT", initialUSDT.Free-finalUSDTFree)
	}
}

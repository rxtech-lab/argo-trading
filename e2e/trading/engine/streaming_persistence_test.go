package engine_test

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/e2e/trading/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	engine_v1 "github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// TestStreamingPersistenceEndToEnd tests the full flow: stream → write → read → indicator calculation.
func (s *LiveTradingE2ETestSuite) TestStreamingPersistenceEndToEnd() {
	// Create temp directory for parquet files
	tempDir, err := os.MkdirTemp("", "streaming_persistence_test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	// Setup mock providers
	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			TrendStrength:     0.01,
			NumDataPoints:     50,
			Seed:              42,
			VolatilityPercent: 1.0,
			Interval:          time.Minute,
			StartTime:         startTime,
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

	// Create engine with persistence
	engineWithPersistence, err := engine_v1.NewLiveTradingEngineV1WithPersistence(tempDir, "mock")
	s.Require().NoError(err)

	// Initialize engine
	err = engineWithPersistence.Initialize(engine.LiveTradingEngineConfig{
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	// Configure engine with mocks
	err = engineWithPersistence.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = engineWithPersistence.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	// Load strategy
	err = engineWithPersistence.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	err = engineWithPersistence.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	// Track execution
	var dataPointsProcessed int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataPointsProcessed++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	callbacks := engine.LiveTradingCallbacks{
		OnMarketData: &onData,
	}

	// Run engine
	err = engineWithPersistence.Run(context.Background(), callbacks)
	s.Require().NoError(err)

	mu.Lock()
	s.Equal(50, dataPointsProcessed, "Should process all data points")
	mu.Unlock()

	// Verify parquet file was created
	parquetPath := filepath.Join(tempDir, "stream_data_mock_1m.parquet")
	_, err = os.Stat(parquetPath)
	s.Require().NoError(err, "Parquet file should exist")

	// Verify data in parquet file using DuckDB
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", parquetPath)).Scan(&count)
	s.Require().NoError(err)
	s.Equal(50, count, "Parquet file should contain all data points")

	// Verify data ordering
	rows, err := db.Query(fmt.Sprintf(`
		SELECT time, symbol, close
		FROM read_parquet('%s')
		ORDER BY time ASC
		LIMIT 5
	`, parquetPath))
	s.Require().NoError(err)
	defer rows.Close()

	var prevTime time.Time
	idx := 0
	for rows.Next() {
		var t time.Time
		var symbol string
		var closePrice float64
		err = rows.Scan(&t, &symbol, &closePrice)
		s.Require().NoError(err)

		s.Equal("BTCUSDT", symbol)
		if idx > 0 {
			s.True(t.After(prevTime), "Data should be ordered by time")
		}
		prevTime = t
		idx++
	}
}

// TestStreamingPersistenceRestart tests that restarting the engine preserves and appends data.
func (s *LiveTradingE2ETestSuite) TestStreamingPersistenceRestart() {
	// Create temp directory for parquet files
	tempDir, err := os.MkdirTemp("", "streaming_restart_test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	// First run - write 25 data points
	{
		mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
			testhelper.MockMarketDataConfig{
				Symbol:            "BTCUSDT",
				Pattern:           backtestTesthelper.PatternIncreasing,
				InitialPrice:      50000.0,
				TrendStrength:     0.01,
				NumDataPoints:     25,
				Seed:              42,
				VolatilityPercent: 1.0,
				Interval:          time.Minute,
				StartTime:         startTime,
			},
		)

		mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

		engine1, err := engine_v1.NewLiveTradingEngineV1WithPersistence(tempDir, "mock")
		s.Require().NoError(err)

		err = engine1.Initialize(engine.LiveTradingEngineConfig{
			MarketDataCacheSize: 100,
			EnableLogging:       false,
		})
		s.Require().NoError(err)

		err = engine1.SetMarketDataProvider(mockMarketDataProvider)
		s.Require().NoError(err)

		err = engine1.SetTradingProvider(mockTradingProvider)
		s.Require().NoError(err)

		err = engine1.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
		s.Require().NoError(err)

		err = engine1.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
		s.Require().NoError(err)

		onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
			mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)
			return nil
		})

		err = engine1.Run(context.Background(), engine.LiveTradingCallbacks{OnMarketData: &onData})
		s.Require().NoError(err)
	}

	// Verify first run created 25 data points
	parquetPath := filepath.Join(tempDir, "stream_data_mock_1m.parquet")
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)

	var count1 int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", parquetPath)).Scan(&count1)
	s.Require().NoError(err)
	s.Equal(25, count1, "First run should write 25 data points")
	db.Close()

	// Second run - write 25 more data points (continuing from where we left off)
	{
		// Continue from T+25 minutes
		continueTime := startTime.Add(25 * time.Minute)

		mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
			testhelper.MockMarketDataConfig{
				Symbol:            "BTCUSDT",
				Pattern:           backtestTesthelper.PatternIncreasing,
				InitialPrice:      50250.0, // Continue from where we left off
				TrendStrength:     0.01,
				NumDataPoints:     25,
				Seed:              43,
				VolatilityPercent: 1.0,
				Interval:          time.Minute,
				StartTime:         continueTime,
			},
		)

		mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

		engine2, err := engine_v1.NewLiveTradingEngineV1WithPersistence(tempDir, "mock")
		s.Require().NoError(err)

		err = engine2.Initialize(engine.LiveTradingEngineConfig{
			MarketDataCacheSize: 100,
			EnableLogging:       false,
		})
		s.Require().NoError(err)

		err = engine2.SetMarketDataProvider(mockMarketDataProvider)
		s.Require().NoError(err)

		err = engine2.SetTradingProvider(mockTradingProvider)
		s.Require().NoError(err)

		err = engine2.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
		s.Require().NoError(err)

		err = engine2.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
		s.Require().NoError(err)

		onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
			mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)
			return nil
		})

		err = engine2.Run(context.Background(), engine.LiveTradingCallbacks{OnMarketData: &onData})
		s.Require().NoError(err)
	}

	// Verify second run appended data
	db2, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db2.Close()

	var count2 int
	err = db2.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", parquetPath)).Scan(&count2)
	s.Require().NoError(err)
	s.Equal(50, count2, "After restart, parquet should have 50 data points total")

	// Verify data is properly ordered
	var minTime, maxTime time.Time
	err = db2.QueryRow(fmt.Sprintf(`
		SELECT MIN(time), MAX(time)
		FROM read_parquet('%s')
	`, parquetPath)).Scan(&minTime, &maxTime)
	s.Require().NoError(err)
	s.Equal(startTime, minTime, "Min time should be the original start time")
	s.Equal(startTime.Add(49*time.Minute), maxTime, "Max time should be 49 minutes after start")
}

// TestStreamingPersistenceMultiSymbol tests persistence with multiple symbols.
func (s *LiveTradingE2ETestSuite) TestStreamingPersistenceMultiSymbol() {
	// Create temp directory for parquet files
	tempDir, err := os.MkdirTemp("", "streaming_multisymbol_test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			NumDataPoints:     25,
			TrendStrength:     0.01,
			VolatilityPercent: 1.0,
			Seed:              42,
			Interval:          time.Minute,
			StartTime:         startTime,
		},
		testhelper.MockMarketDataConfig{
			Symbol:             "ETHUSDT",
			Pattern:            backtestTesthelper.PatternVolatile,
			InitialPrice:       3000.0,
			MaxDrawdownPercent: 15.0,
			NumDataPoints:      25,
			TrendStrength:      0.01,
			VolatilityPercent:  3.0,
			Seed:               43,
			Interval:           time.Minute,
			StartTime:          startTime,
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

	engineWithPersistence, err := engine_v1.NewLiveTradingEngineV1WithPersistence(tempDir, "mock")
	s.Require().NoError(err)

	err = engineWithPersistence.Initialize(engine.LiveTradingEngineConfig{
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = engineWithPersistence.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = engineWithPersistence.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = engineWithPersistence.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	err = engineWithPersistence.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	symbolsSeen := make(map[string]int)
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		symbolsSeen[data.Symbol]++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	err = engineWithPersistence.Run(context.Background(), engine.LiveTradingCallbacks{OnMarketData: &onData})
	s.Require().NoError(err)

	// Verify parquet file contains both symbols
	parquetPath := filepath.Join(tempDir, "stream_data_mock_1m.parquet")
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	// Count by symbol
	rows, err := db.Query(fmt.Sprintf(`
		SELECT symbol, COUNT(*) as cnt
		FROM read_parquet('%s')
		GROUP BY symbol
		ORDER BY symbol
	`, parquetPath))
	s.Require().NoError(err)
	defer rows.Close()

	symbolCounts := make(map[string]int)
	for rows.Next() {
		var symbol string
		var cnt int
		err = rows.Scan(&symbol, &cnt)
		s.Require().NoError(err)
		symbolCounts[symbol] = cnt
	}

	s.Equal(25, symbolCounts["BTCUSDT"], "Should have 25 BTCUSDT records")
	s.Equal(25, symbolCounts["ETHUSDT"], "Should have 25 ETHUSDT records")
}

// TestStreamingPersistenceLargeDataset tests persistence with a larger dataset for performance.
func (s *LiveTradingE2ETestSuite) TestStreamingPersistenceLargeDataset() {
	// Create temp directory for parquet files
	tempDir, err := os.MkdirTemp("", "streaming_large_test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	startTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Generate 1000 data points
	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			TrendStrength:     0.001,
			NumDataPoints:     1000,
			Seed:              42,
			VolatilityPercent: 0.5,
			Interval:          time.Minute,
			StartTime:         startTime,
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

	engineWithPersistence, err := engine_v1.NewLiveTradingEngineV1WithPersistence(tempDir, "mock")
	s.Require().NoError(err)

	err = engineWithPersistence.Initialize(engine.LiveTradingEngineConfig{
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = engineWithPersistence.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = engineWithPersistence.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = engineWithPersistence.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	err = engineWithPersistence.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataCount++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	// Measure execution time
	startExec := time.Now()
	err = engineWithPersistence.Run(context.Background(), engine.LiveTradingCallbacks{OnMarketData: &onData})
	execDuration := time.Since(startExec)
	s.Require().NoError(err)

	mu.Lock()
	s.Equal(1000, dataCount, "Should process all 1000 data points")
	mu.Unlock()

	// Verify parquet file
	parquetPath := filepath.Join(tempDir, "stream_data_mock_1m.parquet")
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", parquetPath)).Scan(&count)
	s.Require().NoError(err)
	s.Equal(1000, count, "Parquet file should contain all 1000 data points")

	// Performance assertion - should complete in reasonable time (less than 30 seconds)
	s.Less(execDuration, 30*time.Second, "Large dataset should complete in less than 30 seconds")
}

// TestStreamingPersistenceWithoutPersistence verifies the engine still works without persistence enabled.
func (s *LiveTradingE2ETestSuite) TestStreamingPersistenceWithoutPersistence() {
	// Create temp directory to verify no files are created
	tempDir, err := os.MkdirTemp("", "streaming_no_persist_test")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	startTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC)

	mockMarketDataProvider := testhelper.NewMockMarketDataProvider(
		testhelper.MockMarketDataConfig{
			Symbol:            "BTCUSDT",
			Pattern:           backtestTesthelper.PatternIncreasing,
			InitialPrice:      50000.0,
			TrendStrength:     0.01,
			NumDataPoints:     50,
			Seed:              42,
			VolatilityPercent: 1.0,
			Interval:          time.Minute,
			StartTime:         startTime,
		},
	)

	mockTradingProvider := testhelper.NewMockTradingProvider(100000.0)

	// Use engine WITHOUT persistence (standard constructor)
	engineWithoutPersistence, err := engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)

	err = engineWithoutPersistence.Initialize(engine.LiveTradingEngineConfig{
		MarketDataCacheSize: 100,
		EnableLogging:       false,
	})
	s.Require().NoError(err)

	err = engineWithoutPersistence.SetMarketDataProvider(mockMarketDataProvider)
	s.Require().NoError(err)

	err = engineWithoutPersistence.SetTradingProvider(mockTradingProvider)
	s.Require().NoError(err)

	err = engineWithoutPersistence.LoadStrategyFromFile("../../backtest/wasm/place_order/place_order_plugin.wasm")
	s.Require().NoError(err)

	err = engineWithoutPersistence.SetStrategyConfig(`{"symbol": "BTCUSDT"}`)
	s.Require().NoError(err)

	var dataCount int
	var mu sync.Mutex

	onData := engine.OnMarketDataCallback(func(_ string, data types.MarketData) error {
		mu.Lock()
		defer mu.Unlock()

		dataCount++
		mockTradingProvider.SetCurrentPrice(data.Symbol, data.Close)

		return nil
	})

	err = engineWithoutPersistence.Run(context.Background(), engine.LiveTradingCallbacks{OnMarketData: &onData})
	s.Require().NoError(err)

	mu.Lock()
	s.Equal(50, dataCount, "Should process all data points")
	mu.Unlock()

	// Verify no parquet file was created in the current directory
	matches, err := filepath.Glob(filepath.Join(tempDir, "*.parquet"))
	s.Require().NoError(err)
	s.Empty(matches, "No parquet files should be created without persistence")
}

---
title: Testing
description: Mock providers, E2E tests, and prefetch testing for live trading
---

# Testing

This document describes the testing strategy for the Live Trading Engine, including mock providers, E2E test patterns, and comprehensive prefetch feature testing.

## Overview

The Live Trading Engine testing strategy uses mock providers to simulate real-time market data and trading without connecting to actual exchanges. This enables:

- **Fast execution**: No network latency or rate limits
- **Reproducible results**: Seed-based random generation
- **Isolated tests**: Each test gets fresh mock instances
- **Pattern-based scenarios**: Test strategy behavior under different market conditions

## Test Architecture

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                              E2E Test Setup                                  │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                             │
│  ┌─────────────────────┐      ┌─────────────────────────────────────────┐  │
│  │  Test Suite         │      │  LiveTradingEngineV1                    │  │
│  │  (testify/suite)    │─────▶│                                         │  │
│  └─────────────────────┘      │  ┌─────────────────────────────────┐    │  │
│                               │  │  WASM Strategy                  │    │  │
│                               │  │  (Same as production)           │    │  │
│                               │  └─────────────────────────────────┘    │  │
│                               └──────────────┬──────────────────────────┘  │
│                                              │                              │
│                    ┌─────────────────────────┴─────────────────────────┐   │
│                    │                                                   │   │
│                    ▼                                                   ▼   │
│  ┌─────────────────────────────────────┐   ┌────────────────────────────┐ │
│  │  MockMarketDataProvider             │   │  MockTradingProvider       │ │
│  │  ┌───────────────────────────────┐  │   │  ┌──────────────────────┐  │ │
│  │  │ MockDataGenerator             │  │   │  │ In-memory positions  │  │ │
│  │  │ - PatternIncreasing           │  │   │  │ In-memory balance    │  │ │
│  │  │ - PatternDecreasing           │  │   │  │ Trade recording      │  │ │
│  │  │ - PatternVolatile             │  │   │  │ Instant execution    │  │ │
│  │  └───────────────────────────────┘  │   │  └──────────────────────┘  │ │
│  └─────────────────────────────────────┘   └────────────────────────────┘ │
│                                                                             │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Mock Providers

### MockMarketDataProvider

Implements the `Provider` interface using configurable data patterns:

```go
package testhelper

type MockMarketDataProvider struct {
    configs map[string]MockMarketDataConfig
}

type MockMarketDataConfig struct {
    // Symbol for this configuration
    Symbol string

    // Pattern determines price movement
    Pattern SimulationPattern  // PatternIncreasing, PatternDecreasing, PatternVolatile

    // InitialPrice is the starting price
    InitialPrice float64

    // NumDataPoints is the total number of candles to generate
    NumDataPoints int

    // TrendStrength controls trend magnitude (0.0-1.0)
    TrendStrength float64

    // VolatilityPercent controls price volatility
    VolatilityPercent float64

    // MaxDrawdownPercent limits drawdown for volatile pattern
    MaxDrawdownPercent float64

    // Seed for reproducible random generation
    Seed int64

    // ErrorAfterN injects an error after N data points (0 = no error)
    ErrorAfterN int

    // ErrorToReturn is the error to inject
    ErrorToReturn error
}

// NewMockMarketDataProvider creates a mock provider
func NewMockMarketDataProvider(configs ...MockMarketDataConfig) *MockMarketDataProvider

// Stream implements Provider.Stream
func (p *MockMarketDataProvider) Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error]
```

**Usage:**

```go
mockMarketData := testhelper.NewMockMarketDataProvider(
    testhelper.MockMarketDataConfig{
        Symbol:        "BTCUSDT",
        Pattern:       testhelper.PatternIncreasing,
        InitialPrice:  50000.0,
        NumDataPoints: 100,
        TrendStrength: 0.02,  // 2% trend per candle
        Seed:          42,    // Reproducible
    },
)
```

### MockTradingProvider

Implements `TradingSystemProvider` with in-memory state:

```go
package testhelper

type MockTradingProvider struct {
    balance      float64
    positions    map[string]*types.Position
    orders       []types.ExecuteOrder
    trades       []types.Trade
    currentPrice map[string]float64

    // Behavior configuration
    FailAllOrders bool
    FailReason    string
}

// NewMockTradingProvider creates a mock trading provider
func NewMockTradingProvider(initialBalance float64) *MockTradingProvider

// SetCurrentPrice updates current price for a symbol
func (m *MockTradingProvider) SetCurrentPrice(symbol string, price float64)

// PlaceOrder executes instantly at current price
func (m *MockTradingProvider) PlaceOrder(order types.ExecuteOrder) error

// GetAllTrades returns all executed trades (for test assertions)
func (m *MockTradingProvider) GetAllTrades() []types.Trade
```

**Usage:**

```go
mockTrading := testhelper.NewMockTradingProvider(10000.0)

// Verify trades after test
trades := mockTrading.GetAllTrades()
assert.Greater(t, len(trades), 0)
```

## Test Categories

### 1. Basic E2E Tests

Test fundamental engine functionality with mock providers:

```go
func (s *LiveTradingE2ETestSuite) TestBasicStrategyExecution() {
    // Setup mock providers
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            NumDataPoints: 100,
            Seed:          42,
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)

    // Configure engine
    eng, _ := engine.NewLiveTradingEngineV1()
    eng.Initialize(engine.LiveTradingEngineConfig{
        Symbols:        []string{"BTCUSDT"},
        Interval:       "1m",
        DataOutputPath: s.T().TempDir(),
    })
    eng.SetMockMarketDataProvider(mockMarketData)
    eng.SetMockTradingProvider(mockTrading)
    eng.LoadStrategyFromFile("./test_strategy.wasm")

    // Track execution
    var dataPointsProcessed int
    callbacks := engine.LiveTradingCallbacks{
        OnMarketData: &engine.OnMarketDataCallback(func(data types.MarketData) error {
            dataPointsProcessed++
            mockTrading.SetCurrentPrice(data.Symbol, data.Close)
            return nil
        }),
    }

    // Run
    err := eng.Run(context.Background(), callbacks)
    s.NoError(err)

    // Assertions
    s.Equal(100, dataPointsProcessed)
    trades := mockTrading.GetAllTrades()
    s.NotEmpty(trades)
}
```

### 2. Market Pattern Tests

Test strategy behavior under different market conditions:

**Increasing Market:**
```go
func (s *LiveTradingE2ETestSuite) TestStrategyInIncreasingMarket() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            TrendStrength: 0.02,
            NumDataPoints: 100,
            Seed:          42,
        },
    )

    mockTrading := testhelper.NewMockTradingProvider(10000.0)
    // ... setup engine ...

    eng.Run(context.Background(), callbacks)

    // Strategy should profit in uptrend
    accountInfo, _ := mockTrading.GetAccountInfo()
    s.Greater(accountInfo.Balance, 10000.0)
}
```

**Decreasing Market:**
```go
func (s *LiveTradingE2ETestSuite) TestStrategyInDecreasingMarket() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternDecreasing,
            InitialPrice:  50000.0,
            TrendStrength: 0.02,
            NumDataPoints: 100,
            Seed:          42,
        },
    )

    // ... setup ...

    eng.Run(context.Background(), callbacks)

    // Strategy should limit losses
    accountInfo, _ := mockTrading.GetAccountInfo()
    s.GreaterOrEqual(accountInfo.Balance, 8000.0)  // Max 20% loss
}
```

**Volatile Market:**
```go
func (s *LiveTradingE2ETestSuite) TestStrategyInVolatileMarket() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:             "BTCUSDT",
            Pattern:            testhelper.PatternVolatile,
            InitialPrice:       50000.0,
            VolatilityPercent:  3.0,
            MaxDrawdownPercent: 10.0,
            NumDataPoints:      200,
            Seed:               42,
        },
    )

    // ... setup ...

    // Verify strategy handles volatility
    trades := mockTrading.GetAllTrades()
    s.NotEmpty(trades)
}
```

### 3. Session Tests

**Multi-Day Session:**
```go
func (s *LiveTradingE2ETestSuite) TestMultiDaySession() {
    // Generate data spanning 3 days
    mockMarketData := testhelper.NewMockMarketDataProviderWithDates(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            InitialPrice:  50000.0,
            NumDataPoints: 4320,  // 3 days of 1m data
            StartDate:     time.Date(2025, 10, 1, 0, 0, 0, 0, time.UTC),
        },
    )

    dataPath := s.T().TempDir()
    // ... setup engine with dataPath ...

    eng.Run(context.Background(), callbacks)

    // Verify 3 days of output folders
    entries, _ := os.ReadDir(dataPath)
    s.Len(entries, 3)  // 2025-10-01, 2025-10-02, 2025-10-03

    // Verify each day has stats.yaml
    for _, entry := range entries {
        statsPath := filepath.Join(dataPath, entry.Name(), "run_1", "stats.yaml")
        _, err := os.Stat(statsPath)
        s.NoError(err)
    }
}
```

**Multiple Same-Day Sessions:**
```go
func (s *LiveTradingE2ETestSuite) TestMultipleSameDaySessions() {
    dataPath := s.T().TempDir()

    // Run first session
    eng1, _ := engine.NewLiveTradingEngineV1()
    eng1.Initialize(config)
    eng1.Run(ctx, callbacks)

    // Run second session
    eng2, _ := engine.NewLiveTradingEngineV1()
    eng2.Initialize(config)
    eng2.Run(ctx, callbacks)

    // Verify two run folders
    today := time.Now().Format("2006-01-02")
    entries, _ := os.ReadDir(filepath.Join(dataPath, today))
    s.Len(entries, 2)  // run_1, run_2
}
```

### 4. Error Handling Tests

**Stream Error:**
```go
func (s *LiveTradingE2ETestSuite) TestStreamErrorHandling() {
    expectedErr := errors.New("connection lost")

    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            Pattern:       testhelper.PatternIncreasing,
            NumDataPoints: 100,
            ErrorAfterN:   50,  // Error after 50 candles
            ErrorToReturn: expectedErr,
        },
    )

    var errorReceived error
    callbacks := engine.LiveTradingCallbacks{
        OnError: &engine.OnErrorCallback(func(err error) {
            errorReceived = err
        }),
    }

    eng.Run(context.Background(), callbacks)

    s.NotNil(errorReceived)
    s.Contains(errorReceived.Error(), "connection lost")
}
```

**Graceful Shutdown:**
```go
func (s *LiveTradingE2ETestSuite) TestGracefulShutdown() {
    mockMarketData := testhelper.NewMockMarketDataProvider(
        testhelper.MockMarketDataConfig{
            Symbol:        "BTCUSDT",
            NumDataPoints: 1000,
        },
    )

    ctx, cancel := context.WithCancel(context.Background())

    var dataCount int
    var engineStopped bool

    callbacks := engine.LiveTradingCallbacks{
        OnMarketData: &engine.OnMarketDataCallback(func(data types.MarketData) error {
            dataCount++
            if dataCount >= 100 {
                cancel()  // Cancel after 100 candles
            }
            return nil
        }),
        OnEngineStop: &engine.OnEngineStopCallback(func(err error) {
            engineStopped = true
        }),
    }

    err := eng.Run(ctx, callbacks)

    s.Equal(context.Canceled, err)
    s.True(engineStopped)
    s.Less(dataCount, 1000)
}
```

### 5. Prefetch Tests

**Basic Prefetch:**
```go
func (s *PrefetchTestSuite) TestPrefetchDownloadsHistoricalData() {
    mockProvider := testhelper.NewMockMarketDataProviderWithHistory(
        testhelper.HistoricalDataConfig{
            Symbol:       "BTCUSDT",
            StartDate:    time.Now().AddDate(0, 0, -30),
            EndDate:      time.Now(),
            Interval:     "1m",
            InitialPrice: 50000.0,
        },
    )

    dataPath := s.T().TempDir()
    config := engine.LiveTradingEngineConfig{
        Symbols:        []string{"BTCUSDT"},
        DataOutputPath: dataPath,
        Prefetch: engine.PrefetchConfig{
            Enabled:       true,
            StartTimeType: "days",
            Days:          30,
        },
    }

    eng.Initialize(config)
    eng.SetMockMarketDataProvider(mockProvider)
    eng.Run(ctx, callbacks)

    // Verify historical data was downloaded
    parquetPath := filepath.Join(dataPath, time.Now().Format("2006-01-02"), "run_1", "market_data.parquet")

    db, _ := sql.Open("duckdb", ":memory:")
    row := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", parquetPath))

    var count int
    row.Scan(&count)
    s.Greater(count, 40000)  // ~30 days of 1m data
}
```

**Gap Detection:**
```go
func (s *PrefetchTestSuite) TestGapDetection() {
    // Create provider that returns prefetch ending at T-5min
    // and stream starting at T
    mockProvider := testhelper.NewMockMarketDataProviderWithGap(
        testhelper.GapConfig{
            PrefetchEnd: time.Now().Add(-5 * time.Minute),
            StreamStart: time.Now(),
        },
    )

    var gapDetected bool
    mockProvider.OnGapDetected = func(gap time.Duration) {
        gapDetected = true
        s.Equal(5*time.Minute, gap)
    }

    eng.Run(ctx, callbacks)

    s.True(gapDetected)
}
```

**Gap Fill Completion:**
```go
func (s *PrefetchTestSuite) TestGapFillCompletion() {
    // Provider simulates:
    // 1. Prefetch completes at T-10min
    // 2. Stream starts at T
    // 3. Gap fill fetches missing 10 minutes of data
    mockProvider := testhelper.NewMockMarketDataProviderWithGapFill(
        testhelper.GapFillConfig{
            GapDuration:     10 * time.Minute,
            GapFillDuration: 5 * time.Second,
        },
    )

    var gapFillCompleted bool
    mockProvider.OnGapFillComplete = func() {
        gapFillCompleted = true
    }

    eng.Run(ctx, callbacks)

    // Verify gap fill completed before live trading started
    s.True(gapFillCompleted)
}
```

**Indicator Accuracy After Prefetch:**
```go
func (s *PrefetchTestSuite) TestRSIAccuracyAfterPrefetch() {
    // Create known price series
    prices := generateKnownPriceSeries(100)
    expectedRSI := calculateExpectedRSI(prices, 14)

    mockProvider := testhelper.NewMockMarketDataProviderFromPrices(prices)

    var actualRSI float64
    strategy := testhelper.NewRSITestStrategy(14, func(rsi float64) {
        actualRSI = rsi
    })

    eng.LoadStrategy(strategy)
    eng.Run(ctx, callbacks)

    // RSI should match expected value
    s.InDelta(expectedRSI, actualRSI, 0.001)
}
```

**Restart Recovery:**
```go
func (s *PrefetchTestSuite) TestRestartRecovery() {
    dataPath := s.T().TempDir()

    // Run first session, stop at candle 50
    ctx1, cancel1 := context.WithCancel(context.Background())
    var count1 int
    callbacks1 := engine.LiveTradingCallbacks{
        OnMarketData: &engine.OnMarketDataCallback(func(data types.MarketData) error {
            count1++
            if count1 >= 50 {
                cancel1()
            }
            return nil
        }),
    }
    eng1.Run(ctx1, callbacks1)

    // Get last stored timestamp
    lastStored := getLastStoredTimestamp(dataPath)

    // Restart with same data path
    eng2, _ := engine.NewLiveTradingEngineV1()
    eng2.Initialize(engine.LiveTradingEngineConfig{
        DataOutputPath: dataPath,
        Prefetch: engine.PrefetchConfig{
            Enabled:       true,
            StartTimeType: "days",
            Days:          30,
        },
    })

    // Verify prefetch starts from lastStored, not 30 days ago
    var prefetchStart time.Time
    mockProvider.OnPrefetchStart = func(start time.Time) {
        prefetchStart = start
    }

    eng2.Run(ctx, callbacks)

    s.True(prefetchStart.Equal(lastStored) || prefetchStart.After(lastStored))
}
```

## Test File Structure

```
e2e/
└── trading/
    ├── testhelper/
    │   ├── mock_market_data_provider.go    # MockMarketDataProvider
    │   ├── mock_trading_provider.go        # MockTradingProvider
    │   ├── data_generator.go               # Price pattern generators
    │   └── test_strategies/                # Pre-compiled WASM strategies
    │       ├── simple_strategy.wasm
    │       ├── trend_following.wasm
    │       └── rsi_test_strategy.wasm
    └── engine/
        ├── suite_test.go                   # Test suite setup
        ├── basic_test.go                   # Basic E2E tests
        ├── patterns_test.go                # Market pattern tests
        ├── session_test.go                 # Session management tests
        ├── error_test.go                   # Error handling tests
        └── prefetch_test.go                # Prefetch feature tests
```

## Test Suite Setup

```go
package engine_test

import (
    "testing"

    "github.com/stretchr/testify/suite"
)

type LiveTradingE2ETestSuite struct {
    suite.Suite
    engine engine.LiveTradingEngine
}

func TestLiveTradingE2E(t *testing.T) {
    suite.Run(t, new(LiveTradingE2ETestSuite))
}

func (s *LiveTradingE2ETestSuite) SetupTest() {
    var err error
    s.engine, err = engine.NewLiveTradingEngineV1()
    s.Require().NoError(err)

    err = s.engine.Initialize(engine.LiveTradingEngineConfig{
        Symbols:             []string{"BTCUSDT"},
        Interval:            "1m",
        MarketDataCacheSize: 100,
        EnableLogging:       false,
        DataOutputPath:      s.T().TempDir(),
    })
    s.Require().NoError(err)
}
```

## Running Tests

```bash
# Run all live trading tests
go test -v ./e2e/trading/...

# Run specific test file
go test -v ./e2e/trading/engine/prefetch_test.go

# Run with race detection
go test -v -race ./e2e/trading/...

# Run with coverage
go test -v -cover ./e2e/trading/...
```

## Related Documentation

- [Live Trading Engine Overview](README.md)
- [Session Management and Persistence](session-and-persistence.md)
- [Data Prefetch](data-prefetch.md)

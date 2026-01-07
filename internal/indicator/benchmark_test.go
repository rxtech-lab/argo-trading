package indicator

import (
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// setupBenchmarkEnvironment creates a test environment for benchmarking
func setupBenchmarkEnvironment(b *testing.B) (datasource.DataSource, IndicatorRegistry, cache.Cache) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, err := loggerConfig.Build()
	if err != nil {
		b.Fatal(err)
	}
	log := &logger.Logger{Logger: zapLogger}

	ds, err := datasource.NewDataSource(":memory:", log)
	if err != nil {
		b.Fatal(err)
	}

	err = ds.Initialize("./test_data/test_data.parquet")
	if err != nil {
		b.Fatal(err)
	}

	registry := NewIndicatorRegistry()
	registry.RegisterIndicator(NewEMA())
	registry.RegisterIndicator(NewRSI())
	registry.RegisterIndicator(NewMACD())
	registry.RegisterIndicator(NewATR())
	registry.RegisterIndicator(NewBollingerBands())
	registry.RegisterIndicator(NewMA())

	cacheInstance := cache.NewCacheV1()

	return ds, registry, cacheInstance
}

// BenchmarkMultipleIndicatorsWithoutCaching benchmarks multiple indicator calls without caching.
// This simulates the old behavior where each indicator call makes a separate DB query.
func BenchmarkMultipleIndicatorsWithoutCaching(b *testing.B) {
	ds, registry, cacheInstance := setupBenchmarkEnvironment(b)
	defer ds.Close()

	// Simulate a time point where indicators would be calculated
	endTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	symbol := "AAPL"

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: registry,
		Cache:             cacheInstance,
	}

	ema, err := registry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		b.Fatal(err)
	}
	rsi, err := registry.GetIndicator(types.IndicatorTypeRSI)
	if err != nil {
		b.Fatal(err)
	}
	ma, err := registry.GetIndicator(types.IndicatorTypeMA)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate calling multiple indicators on the same bar (without caching)
		// Each call makes a separate database query
		ema.RawValue(symbol, endTime, ctx, 20)
		ema.RawValue(symbol, endTime, ctx, 12) // Fast EMA for MACD
		ema.RawValue(symbol, endTime, ctx, 26) // Slow EMA for MACD
		rsi.RawValue(symbol, endTime, ctx, 14) // RSI 14
		ma.RawValue(symbol, endTime, ctx, 20)  // MA 20
	}
}

// BenchmarkMultipleIndicatorsWithCaching benchmarks multiple indicator calls with caching.
// This simulates the new behavior where repeated calls to the same data are cached.
func BenchmarkMultipleIndicatorsWithCaching(b *testing.B) {
	ds, registry, cacheInstance := setupBenchmarkEnvironment(b)
	defer ds.Close()

	// Wrap with cached datasource
	cachedDS := datasource.NewCachedDataSource(ds)

	// Simulate a time point where indicators would be calculated
	endTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	symbol := "AAPL"

	ctx := IndicatorContext{
		DataSource:        cachedDS,
		IndicatorRegistry: registry,
		Cache:             cacheInstance,
	}

	ema, err := registry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		b.Fatal(err)
	}
	rsi, err := registry.GetIndicator(types.IndicatorTypeRSI)
	if err != nil {
		b.Fatal(err)
	}
	ma, err := registry.GetIndicator(types.IndicatorTypeMA)
	if err != nil {
		b.Fatal(err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Simulate calling multiple indicators on the same bar (with caching)
		// Duplicate queries are served from cache
		ema.RawValue(symbol, endTime, ctx, 20)
		ema.RawValue(symbol, endTime, ctx, 12) // Fast EMA for MACD
		ema.RawValue(symbol, endTime, ctx, 26) // Slow EMA for MACD
		rsi.RawValue(symbol, endTime, ctx, 14) // RSI 14
		ma.RawValue(symbol, endTime, ctx, 20)  // MA 20

		// Clear cache at end of bar (simulating bar transition)
		cachedDS.ClearCache()
	}
}

// BenchmarkCacheHitVsDBQuery directly compares cache hit vs DB query performance.
func BenchmarkCacheHitVsDBQuery(b *testing.B) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, _ := loggerConfig.Build()
	log := &logger.Logger{Logger: zapLogger}

	ds, err := datasource.NewDataSource(":memory:", log)
	if err != nil {
		b.Fatal(err)
	}
	defer ds.Close()
	err = ds.Initialize("./test_data/test_data.parquet")
	if err != nil {
		b.Fatal(err)
	}

	cachedDS := datasource.NewCachedDataSource(ds)

	endTime := time.Date(2024, 1, 5, 10, 0, 0, 0, time.UTC)
	symbol := "AAPL"

	// Warm up cache
	cachedDS.GetPreviousNumberOfDataPoints(endTime, symbol, 20)

	b.Run("CacheHit", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// This should be served from cache
			cachedDS.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
		}
	})

	b.Run("DBQuery", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			// This always hits the database
			ds.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
		}
	})
}

// TestCachePerformanceImprovement verifies that caching provides a measurable speedup.
// This test runs both cached and uncached queries and asserts cache is faster.
func TestCachePerformanceImprovement(t *testing.T) {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, err := loggerConfig.Build()
	if err != nil {
		t.Fatal(err)
	}
	log := &logger.Logger{Logger: zapLogger}

	ds, err := datasource.NewDataSource(":memory:", log)
	if err != nil {
		t.Fatal(err)
	}
	defer ds.Close()

	err = ds.Initialize("./test_data/test_data.parquet")
	if err != nil {
		t.Fatal(err)
	}

	cachedDS := datasource.NewCachedDataSource(ds)
	endTime := time.Date(2025, 1, 2, 10, 0, 0, 0, time.UTC)
	symbol := "AAPL"
	iterations := 100

	// Warm up cache
	_, err = cachedDS.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
	if err != nil {
		t.Fatal(err)
	}

	// Measure uncached queries
	uncachedStart := time.Now()
	for i := 0; i < iterations; i++ {
		ds.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
	}
	uncachedDuration := time.Since(uncachedStart)

	// Measure cached queries
	cachedStart := time.Now()
	for i := 0; i < iterations; i++ {
		cachedDS.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
	}
	cachedDuration := time.Since(cachedStart)

	// Log results
	t.Logf("Uncached: %v, Cached: %v, Speedup: %.2fx",
		uncachedDuration, cachedDuration,
		float64(uncachedDuration)/float64(cachedDuration))

	// Assert cache is faster
	if cachedDuration >= uncachedDuration {
		t.Errorf("Cache should be faster: cached=%v, uncached=%v",
			cachedDuration, uncachedDuration)
	}
}

package datasource

import (
	"fmt"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MockSlowDataSource simulates a slow datasource (like DuckDB) for benchmarking
type MockSlowDataSource struct {
	data map[string][]types.MarketData
}

func NewMockSlowDataSource() *MockSlowDataSource {
	return &MockSlowDataSource{
		data: make(map[string][]types.MarketData),
	}
}

func (m *MockSlowDataSource) Initialize(path string) error {
	return nil
}

func (m *MockSlowDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		for _, symbolData := range m.data {
			for _, d := range symbolData {
				if !yield(d, nil) {
					return
				}
			}
		}
	}
}

func (m *MockSlowDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	// Simulate slow query by iterating through all data
	var result []types.MarketData
	for _, symbolData := range m.data {
		for _, d := range symbolData {
			if !d.Time.Before(start) && !d.Time.After(end) {
				result = append(result, d)
			}
		}
	}
	return result, nil
}

func (m *MockSlowDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	// Simulate slow query by searching through data
	symbolData, ok := m.data[symbol]
	if !ok {
		return nil, nil
	}

	var result []types.MarketData
	for i := len(symbolData) - 1; i >= 0 && len(result) < count; i-- {
		if !symbolData[i].Time.After(end) {
			result = append([]types.MarketData{symbolData[i]}, result...)
		}
	}
	return result, nil
}

func (m *MockSlowDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	symbolData, ok := m.data[symbol]
	if !ok {
		return types.MarketData{}, fmt.Errorf("symbol not found")
	}

	for _, d := range symbolData {
		if d.Time.Equal(timestamp) {
			return d, nil
		}
	}
	return types.MarketData{}, fmt.Errorf("data not found")
}

func (m *MockSlowDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	symbolData, ok := m.data[symbol]
	if !ok || len(symbolData) == 0 {
		return types.MarketData{}, fmt.Errorf("no data")
	}
	return symbolData[len(symbolData)-1], nil
}

func (m *MockSlowDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	return nil, nil
}

func (m *MockSlowDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	count := 0
	for _, symbolData := range m.data {
		count += len(symbolData)
	}
	return count, nil
}

func (m *MockSlowDataSource) Close() error {
	return nil
}

func (m *MockSlowDataSource) GetAllSymbols() ([]string, error) {
	symbols := make([]string, 0, len(m.data))
	for s := range m.data {
		symbols = append(symbols, s)
	}
	return symbols, nil
}

func (m *MockSlowDataSource) AddData(data types.MarketData) {
	if _, ok := m.data[data.Symbol]; !ok {
		m.data[data.Symbol] = make([]types.MarketData, 0)
	}
	m.data[data.Symbol] = append(m.data[data.Symbol], data)
}

// generateMockData generates n data points for testing
func generateMockData(symbol string, n int) []types.MarketData {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := make([]types.MarketData, n)

	for i := 0; i < n; i++ {
		data[i] = types.MarketData{
			Id:     fmt.Sprintf("%s_%d", symbol, i),
			Symbol: symbol,
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   100 + float64(i)*0.1,
			High:   101 + float64(i)*0.1,
			Low:    99 + float64(i)*0.1,
			Close:  100.5 + float64(i)*0.1,
			Volume: 1000 + float64(i),
		}
	}

	return data
}

// BenchmarkWithoutCache benchmarks queries without the sliding window cache
func BenchmarkWithoutCache(b *testing.B) {
	mockDS := NewMockSlowDataSource()
	data := generateMockData("SPY", 10000)

	// Populate the mock datasource
	for _, d := range data {
		mockDS.AddData(d)
	}

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	b.Run("GetPreviousDataPoints_50", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50)
		}
	})

	b.Run("GetPreviousDataPoints_100", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 100)
		}
	})

	b.Run("GetPreviousDataPoints_200", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 200)
		}
	})

	b.Run("GetMarketData", func(b *testing.B) {
		targetTime := baseTime.Add(5000 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockDS.GetMarketData("SPY", targetTime)
		}
	})

	b.Run("GetRange_100", func(b *testing.B) {
		start := baseTime.Add(9800 * time.Minute)
		end := baseTime.Add(9900 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = mockDS.GetRange(start, end, optional.None[Interval]())
		}
	})
}

// BenchmarkWithCache benchmarks queries with the sliding window cache
func BenchmarkWithCache(b *testing.B) {
	mockDS := NewMockSlowDataSource()
	data := generateMockData("SPY", 10000)

	// Populate the mock datasource
	for _, d := range data {
		mockDS.AddData(d)
	}

	// Create sliding window datasource with cache size of 1000
	slidingDS := NewSlidingWindowDataSource(mockDS, 1000)

	// Simulate backtest: populate cache as data is "processed"
	// Only cache the last 1000 data points (simulating processing)
	for i := 9000; i < 10000; i++ {
		slidingDS.AddToCache(data[i])
	}

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	b.Run("GetPreviousDataPoints_50_CacheHit", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50)
		}
	})

	b.Run("GetPreviousDataPoints_100_CacheHit", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 100)
		}
	})

	b.Run("GetPreviousDataPoints_200_CacheHit", func(b *testing.B) {
		endTime := baseTime.Add(9999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 200)
		}
	})

	b.Run("GetMarketData_CacheHit", func(b *testing.B) {
		targetTime := baseTime.Add(9500 * time.Minute) // Within cache range
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetMarketData("SPY", targetTime)
		}
	})

	b.Run("GetRange_100_CacheHit", func(b *testing.B) {
		start := baseTime.Add(9800 * time.Minute)
		end := baseTime.Add(9900 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetRange(start, end, optional.None[Interval]())
		}
	})

	// Cache miss scenarios (data before cache window)
	b.Run("GetPreviousDataPoints_50_CacheMiss", func(b *testing.B) {
		endTime := baseTime.Add(5000 * time.Minute) // Before cache window
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50)
		}
	})

	b.Run("GetMarketData_CacheMiss", func(b *testing.B) {
		targetTime := baseTime.Add(5000 * time.Minute) // Before cache window
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = slidingDS.GetMarketData("SPY", targetTime)
		}
	})
}

// BenchmarkCacheOperations benchmarks the cache operations directly
func BenchmarkCacheOperations(b *testing.B) {
	b.Run("Add_Sequential", func(b *testing.B) {
		cache := NewSlidingWindowCache(1000)
		data := generateMockData("SPY", b.N)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			cache.Add(data[i%len(data)])
		}
	})

	b.Run("GetPreviousDataPoints_50", func(b *testing.B) {
		cache := NewSlidingWindowCache(1000)
		data := generateMockData("SPY", 1000)
		for _, d := range data {
			cache.Add(d)
		}
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endTime := baseTime.Add(999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetPreviousDataPoints(endTime, "SPY", 50)
		}
	})

	b.Run("GetPreviousDataPoints_200", func(b *testing.B) {
		cache := NewSlidingWindowCache(1000)
		data := generateMockData("SPY", 1000)
		for _, d := range data {
			cache.Add(d)
		}
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		endTime := baseTime.Add(999 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetPreviousDataPoints(endTime, "SPY", 200)
		}
	})

	b.Run("GetMarketData", func(b *testing.B) {
		cache := NewSlidingWindowCache(1000)
		data := generateMockData("SPY", 1000)
		for _, d := range data {
			cache.Add(d)
		}
		baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
		targetTime := baseTime.Add(500 * time.Minute)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetMarketData("SPY", targetTime)
		}
	})

	b.Run("GetLastData", func(b *testing.B) {
		cache := NewSlidingWindowCache(1000)
		data := generateMockData("SPY", 1000)
		for _, d := range data {
			cache.Add(d)
		}
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_, _ = cache.GetLastData("SPY")
		}
	})
}

// BenchmarkBacktestSimulation simulates a backtest scenario
func BenchmarkBacktestSimulation(b *testing.B) {
	data := generateMockData("SPY", 10000)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	b.Run("WithoutCache_TypicalStrategy", func(b *testing.B) {
		mockDS := NewMockSlowDataSource()
		for _, d := range data {
			mockDS.AddData(d)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// Simulate processing 1000 bars with typical indicator lookbacks
			for barIdx := 200; barIdx < 1200; barIdx++ {
				endTime := baseTime.Add(time.Duration(barIdx) * time.Minute)
				// Typical strategy might call these per bar:
				_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 20) // Fast MA
				_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50) // Slow MA
				_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 14) // RSI
			}
		}
	})

	b.Run("WithCache_TypicalStrategy", func(b *testing.B) {
		mockDS := NewMockSlowDataSource()
		for _, d := range data {
			mockDS.AddData(d)
		}
		slidingDS := NewSlidingWindowDataSource(mockDS, 100)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			slidingDS.ClearCache()
			// Simulate processing 1000 bars with typical indicator lookbacks
			for barIdx := 200; barIdx < 1200; barIdx++ {
				// Add current bar to cache (as backtest engine would)
				slidingDS.AddToCache(data[barIdx])

				endTime := baseTime.Add(time.Duration(barIdx) * time.Minute)
				// Typical strategy might call these per bar:
				_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 20) // Fast MA
				_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50) // Slow MA
				_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 14) // RSI
			}
		}
	})
}

// TestBenchmarkResults runs and prints benchmark results in a more readable format
func TestBenchmarkResults(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping benchmark comparison in short mode")
	}

	data := generateMockData("SPY", 10000)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Setup without cache
	mockDS := NewMockSlowDataSource()
	for _, d := range data {
		mockDS.AddData(d)
	}

	// Setup with cache
	slidingDS := NewSlidingWindowDataSource(mockDS, 1000)
	for i := 9000; i < 10000; i++ {
		slidingDS.AddToCache(data[i])
	}

	iterations := 1000
	endTime := baseTime.Add(9999 * time.Minute)

	// Benchmark GetPreviousDataPoints without cache
	start := time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = mockDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50)
	}
	withoutCacheDuration := time.Since(start)

	// Benchmark GetPreviousDataPoints with cache
	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = slidingDS.GetPreviousNumberOfDataPoints(endTime, "SPY", 50)
	}
	withCacheDuration := time.Since(start)

	t.Logf("\n=== Benchmark Results (10k data points, %d iterations) ===", iterations)
	t.Logf("GetPreviousDataPoints(50):")
	t.Logf("  Without cache: %v (%.2f µs/op)", withoutCacheDuration, float64(withoutCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  With cache:    %v (%.2f µs/op)", withCacheDuration, float64(withCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  Speedup:       %.2fx", float64(withoutCacheDuration)/float64(withCacheDuration))

	// Benchmark GetMarketData
	targetTime := baseTime.Add(9500 * time.Minute)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = mockDS.GetMarketData("SPY", targetTime)
	}
	withoutCacheDuration = time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = slidingDS.GetMarketData("SPY", targetTime)
	}
	withCacheDuration = time.Since(start)

	t.Logf("\nGetMarketData:")
	t.Logf("  Without cache: %v (%.2f µs/op)", withoutCacheDuration, float64(withoutCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  With cache:    %v (%.2f µs/op)", withCacheDuration, float64(withCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  Speedup:       %.2fx", float64(withoutCacheDuration)/float64(withCacheDuration))

	// Benchmark GetRange
	rangeStart := baseTime.Add(9800 * time.Minute)
	rangeEnd := baseTime.Add(9900 * time.Minute)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = mockDS.GetRange(rangeStart, rangeEnd, optional.None[Interval]())
	}
	withoutCacheDuration = time.Since(start)

	start = time.Now()
	for i := 0; i < iterations; i++ {
		_, _ = slidingDS.GetRange(rangeStart, rangeEnd, optional.None[Interval]())
	}
	withCacheDuration = time.Since(start)

	t.Logf("\nGetRange(100 points):")
	t.Logf("  Without cache: %v (%.2f µs/op)", withoutCacheDuration, float64(withoutCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  With cache:    %v (%.2f µs/op)", withCacheDuration, float64(withCacheDuration.Microseconds())/float64(iterations))
	t.Logf("  Speedup:       %.2fx", float64(withoutCacheDuration)/float64(withCacheDuration))
}

package datasource

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// createTestLogger creates a silent logger for benchmarks
func createTestLogger() *logger.Logger {
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, _ := loggerConfig.Build()
	return &logger.Logger{Logger: zapLogger}
}

// setupBenchmarkData creates a DuckDB datasource with generated test data
func setupBenchmarkData(b *testing.B, count int) (DataSource, []types.MarketData) {
	log := createTestLogger()

	// Generate test data
	gen := mocks.NewDataGenerator(42)
	config := mocks.DefaultConfig()
	config.Symbol = "TEST"
	config.Count = count
	data := gen.Generate(config)

	// Create in-memory DuckDB datasource
	ds, err := NewDataSource(":memory:", log)
	if err != nil {
		b.Fatal(err)
	}

	// Create table and insert data manually for benchmarks
	db := ds.(*DuckDBDataSource).db
	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol VARCHAR,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		);
	`)
	if err != nil {
		b.Fatal(err)
	}

	// Insert data in batches for efficiency
	stmt, err := db.Prepare(`INSERT INTO market_data VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	if err != nil {
		b.Fatal(err)
	}

	for _, md := range data {
		_, err = stmt.Exec(md.Time, md.Symbol, md.Open, md.High, md.Low, md.Close, md.Volume)
		if err != nil {
			b.Fatal(err)
		}
	}
	stmt.Close()

	// Create index
	_, err = db.Exec(`CREATE INDEX idx_market_data_symbol_time ON market_data(symbol, time);`)
	if err != nil {
		b.Logf("Warning: could not create index: %v", err)
	}

	return ds, data
}

// BenchmarkDuckDBGetPreviousDataPoints benchmarks DuckDB queries for historical data
func BenchmarkDuckDBGetPreviousDataPoints(b *testing.B) {
	counts := []int{100, 1000, 10000}

	for _, count := range counts {
		b.Run(formatCount(count), func(b *testing.B) {
			ds, data := setupBenchmarkData(b, count)
			defer ds.Close()

			// Use a point in the middle of the data for queries
			midIdx := len(data) / 2
			midTime := data[midIdx].Time
			symbol := "TEST"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := ds.GetPreviousNumberOfDataPoints(midTime, symbol, 26)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkInMemoryIndexedGetPreviousDataPoints benchmarks in-memory indexed lookups
func BenchmarkInMemoryIndexedGetPreviousDataPoints(b *testing.B) {
	counts := []int{100, 1000, 10000}

	for _, count := range counts {
		b.Run(formatCount(count), func(b *testing.B) {
			ds, data := setupBenchmarkData(b, count)
			defer ds.Close()

			// Wrap with in-memory indexed datasource and preload
			indexedDS := NewInMemoryIndexedDataSource(ds)
			err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
			if err != nil {
				b.Fatal(err)
			}

			// Use a point in the middle of the data for queries
			midIdx := len(data) / 2
			midTime := data[midIdx].Time
			symbol := "TEST"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := indexedDS.GetPreviousNumberOfDataPoints(midTime, symbol, 26)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkInMemoryIndexedGetPreviousNBars benchmarks direct bar-indexed lookups (fastest)
func BenchmarkInMemoryIndexedGetPreviousNBars(b *testing.B) {
	counts := []int{100, 1000, 10000}

	for _, count := range counts {
		b.Run(formatCount(count), func(b *testing.B) {
			ds, _ := setupBenchmarkData(b, count)
			defer ds.Close()

			// Wrap with in-memory indexed datasource and preload
			indexedDS := NewInMemoryIndexedDataSource(ds)
			err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
			if err != nil {
				b.Fatal(err)
			}

			// Set current bar index to middle of data
			midIdx := count / 2
			indexedDS.SetCurrentBarIndex(midIdx)
			symbol := "TEST"

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				_, err := indexedDS.GetPreviousNBars(symbol, 26)
				if err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// BenchmarkReadAllComparison compares ReadAll performance
func BenchmarkReadAllComparison(b *testing.B) {
	b.Run("DuckDB_10k", func(b *testing.B) {
		ds, _ := setupBenchmarkData(b, 10000)
		defer ds.Close()

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for _, err := range ds.ReadAll(optional.None[time.Time](), optional.None[time.Time]()) {
				if err != nil {
					b.Fatal(err)
				}
				count++
			}
			if count != 10000 {
				b.Fatalf("expected 10000 records, got %d", count)
			}
		}
	})

	b.Run("InMemoryIndexed_10k", func(b *testing.B) {
		ds, _ := setupBenchmarkData(b, 10000)
		defer ds.Close()

		indexedDS := NewInMemoryIndexedDataSource(ds)
		err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
		if err != nil {
			b.Fatal(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			count := 0
			for _, err := range indexedDS.ReadAll(optional.None[time.Time](), optional.None[time.Time]()) {
				if err != nil {
					b.Fatal(err)
				}
				count++
			}
			if count != 10000 {
				b.Fatalf("expected 10000 records, got %d", count)
			}
		}
	})
}

// BenchmarkMultipleIndicatorQueries simulates indicator calculation pattern
func BenchmarkMultipleIndicatorQueries(b *testing.B) {
	// Indicator calculation typically queries multiple periods: 12, 14, 20, 26
	periods := []int{12, 14, 20, 26}

	b.Run("DuckDB_10k", func(b *testing.B) {
		ds, data := setupBenchmarkData(b, 10000)
		defer ds.Close()

		midIdx := len(data) / 2
		midTime := data[midIdx].Time
		symbol := "TEST"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, period := range periods {
				_, err := ds.GetPreviousNumberOfDataPoints(midTime, symbol, period)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("InMemoryIndexed_10k", func(b *testing.B) {
		ds, data := setupBenchmarkData(b, 10000)
		defer ds.Close()

		indexedDS := NewInMemoryIndexedDataSource(ds)
		err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
		if err != nil {
			b.Fatal(err)
		}

		midIdx := len(data) / 2
		midTime := data[midIdx].Time
		symbol := "TEST"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, period := range periods {
				_, err := indexedDS.GetPreviousNumberOfDataPoints(midTime, symbol, period)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})

	b.Run("InMemoryIndexedNBars_10k", func(b *testing.B) {
		ds, _ := setupBenchmarkData(b, 10000)
		defer ds.Close()

		indexedDS := NewInMemoryIndexedDataSource(ds)
		err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
		if err != nil {
			b.Fatal(err)
		}

		midIdx := 5000
		indexedDS.SetCurrentBarIndex(midIdx)
		symbol := "TEST"

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, period := range periods {
				_, err := indexedDS.GetPreviousNBars(symbol, period)
				if err != nil {
					b.Fatal(err)
				}
			}
		}
	})
}

// TestPerformanceImprovement verifies that in-memory indexing provides significant speedup
func TestPerformanceImprovement(t *testing.T) {
	log := createTestLogger()

	// Generate 10k test data
	gen := mocks.NewDataGenerator(42)
	config := mocks.DefaultConfig()
	config.Symbol = "TEST"
	config.Count = 10000
	data := gen.Generate(config)

	// Create in-memory DuckDB datasource
	ds, err := NewDataSource(":memory:", log)
	assert.NoError(t, err)
	defer ds.Close()

	// Create table and insert data
	db := ds.(*DuckDBDataSource).db
	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol VARCHAR,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		);
	`)
	assert.NoError(t, err)

	stmt, err := db.Prepare(`INSERT INTO market_data VALUES ($1, $2, $3, $4, $5, $6, $7)`)
	assert.NoError(t, err)

	for _, md := range data {
		_, err = stmt.Exec(md.Time, md.Symbol, md.Open, md.High, md.Low, md.Close, md.Volume)
		assert.NoError(t, err)
	}
	stmt.Close()

	// Create index
	_, _ = db.Exec(`CREATE INDEX idx_market_data_symbol_time ON market_data(symbol, time);`)

	// Create in-memory indexed datasource and preload
	indexedDS := NewInMemoryIndexedDataSource(ds)
	err = indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	midIdx := len(data) / 2
	midTime := data[midIdx].Time
	symbol := "TEST"
	iterations := 1000

	// Measure DuckDB query time
	duckDBStart := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := ds.GetPreviousNumberOfDataPoints(midTime, symbol, 26)
		assert.NoError(t, err)
	}
	duckDBDuration := time.Since(duckDBStart)

	// Measure in-memory indexed query time
	indexedStart := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := indexedDS.GetPreviousNumberOfDataPoints(midTime, symbol, 26)
		assert.NoError(t, err)
	}
	indexedDuration := time.Since(indexedStart)

	// Measure direct bar-indexed query time
	indexedDS.SetCurrentBarIndex(midIdx)
	barIndexedStart := time.Now()
	for i := 0; i < iterations; i++ {
		_, err := indexedDS.GetPreviousNBars(symbol, 26)
		assert.NoError(t, err)
	}
	barIndexedDuration := time.Since(barIndexedStart)

	speedup := float64(duckDBDuration) / float64(indexedDuration)
	barSpeedup := float64(duckDBDuration) / float64(barIndexedDuration)

	t.Logf("Performance comparison (10k data, %d iterations):", iterations)
	t.Logf("  DuckDB queries:          %v", duckDBDuration)
	t.Logf("  In-memory indexed:       %v (%.1fx faster)", indexedDuration, speedup)
	t.Logf("  Direct bar-indexed:      %v (%.1fx faster)", barIndexedDuration, barSpeedup)

	// Assert at least 10x improvement for direct bar-indexed access
	assert.Greater(t, barSpeedup, 10.0, "Expected at least 10x performance improvement with bar-indexed access")
}

func formatCount(count int) string {
	switch {
	case count >= 10000:
		return "10k"
	case count >= 1000:
		return "1k"
	default:
		return "100"
	}
}

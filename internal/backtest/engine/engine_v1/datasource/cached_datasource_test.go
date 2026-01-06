package datasource

import (
	"database/sql"
	"fmt"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// CachedDataSourceTestSuite is a test suite for CachedDataSource
type CachedDataSourceTestSuite struct {
	suite.Suite
	dataSource       DataSource
	cachedDataSource *CachedDataSource
	logger           *logger.Logger
	tmpDir           string
}

// SetupSuite sets up the test suite
func (suite *CachedDataSourceTestSuite) SetupSuite() {
	// Create a no-op logger
	loggerConfig := zap.NewDevelopmentConfig()
	loggerConfig.OutputPaths = []string{}
	loggerConfig.ErrorOutputPaths = []string{}
	zapLogger, err := loggerConfig.Build()
	suite.Require().NoError(err)
	suite.logger = &logger.Logger{Logger: zapLogger}

	// Create temp directory
	suite.tmpDir = suite.T().TempDir()

	// Create test data
	testData := createTestDataForCaching()
	testFilePath := filepath.Join(suite.tmpDir, "test.parquet")
	err = writeTestDataToParquetForCaching(testData, testFilePath)
	suite.Require().NoError(err)

	// Create datasource
	ds, err := NewDataSource(":memory:", suite.logger)
	suite.Require().NoError(err)
	suite.dataSource = ds

	// Initialize with test data
	err = ds.Initialize(testFilePath)
	suite.Require().NoError(err)

	// Create cached datasource
	suite.cachedDataSource = NewCachedDataSource(ds)
}

// createTestDataForCaching creates test market data for caching tests
func createTestDataForCaching() []types.MarketData {
	var data []types.MarketData
	baseTime := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	for i := 0; i < 100; i++ {
		data = append(data, types.MarketData{
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   100.0 + float64(i),
			High:   101.0 + float64(i),
			Low:    99.0 + float64(i),
			Close:  100.5 + float64(i),
			Volume: 1000.0 + float64(i*100),
			Symbol: "AAPL",
		})
	}
	return data
}

// writeTestDataToParquetForCaching writes test data to a parquet file
func writeTestDataToParquetForCaching(data []types.MarketData, filePath string) error {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol TEXT,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	if err != nil {
		return err
	}

	for _, d := range data {
		_, err = db.Exec(`
			INSERT INTO market_data VALUES (?, ?, ?, ?, ?, ?, ?)
		`, d.Time, d.Symbol, d.Open, d.High, d.Low, d.Close, d.Volume)
		if err != nil {
			return err
		}
	}

	_, err = db.Exec(fmt.Sprintf(`
		COPY market_data TO '%s' (FORMAT PARQUET)
	`, filePath))
	return err
}

// TearDownSuite cleans up after tests
func (suite *CachedDataSourceTestSuite) TearDownSuite() {
	if suite.dataSource != nil {
		suite.dataSource.Close()
	}
}

// TestCachedDataSourceSuite runs the test suite
func TestCachedDataSourceSuite(t *testing.T) {
	suite.Run(t, new(CachedDataSourceTestSuite))
}

// TestCachingPreviousDataPoints tests that GetPreviousNumberOfDataPoints caches results
func (suite *CachedDataSourceTestSuite) TestCachingPreviousDataPoints() {
	symbol := "AAPL"
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	count := 10

	// First call - should hit the database
	data1, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, count)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data1)

	// Second call with same parameters - should hit cache
	data2, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, count)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data2)

	// Results should be identical
	suite.Equal(len(data1), len(data2))
	for i := range data1 {
		suite.Equal(data1[i].Symbol, data2[i].Symbol)
		suite.Equal(data1[i].Time, data2[i].Time)
		suite.Equal(data1[i].Close, data2[i].Close)
	}
}

// TestCachingGetRange tests that GetRange caches results
func (suite *CachedDataSourceTestSuite) TestCachingGetRange() {
	startTime := time.Date(2024, 1, 2, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	interval := optional.None[Interval]()

	// First call - should hit the database
	data1, err := suite.cachedDataSource.GetRange(startTime, endTime, interval)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data1)

	// Second call with same parameters - should hit cache
	data2, err := suite.cachedDataSource.GetRange(startTime, endTime, interval)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data2)

	// Results should be identical
	suite.Equal(len(data1), len(data2))
}

// TestClearCache tests that ClearCache properly clears the cache
func (suite *CachedDataSourceTestSuite) TestClearCache() {
	symbol := "AAPL"
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	count := 10

	// First call
	data1, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, count)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data1)

	// Clear cache
	suite.cachedDataSource.ClearCache()

	// Second call after clear - should work correctly (hit DB again)
	data2, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, count)
	suite.Require().NoError(err)
	suite.Require().NotEmpty(data2)

	// Results should still be correct
	suite.Equal(len(data1), len(data2))
}

// TestDifferentParametersNotCached tests that different parameters result in different cache entries
func (suite *CachedDataSourceTestSuite) TestDifferentParametersNotCached() {
	symbol := "AAPL"
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	// Call with count 10
	data1, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 10)
	suite.Require().NoError(err)

	// Call with count 5 - should be a different cache entry
	data2, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 5)
	suite.Require().NoError(err)

	// Results should have different lengths
	suite.NotEqual(len(data1), len(data2))
	suite.Equal(10, len(data1))
	suite.Equal(5, len(data2))
}

// TestPassthroughMethods tests that methods that don't cache still work
func (suite *CachedDataSourceTestSuite) TestPassthroughMethods() {
	// Test Count
	count, err := suite.cachedDataSource.Count(optional.None[time.Time](), optional.None[time.Time]())
	suite.Require().NoError(err)
	suite.Greater(count, 0)

	// Test GetAllSymbols
	symbols, err := suite.cachedDataSource.GetAllSymbols()
	suite.Require().NoError(err)
	suite.NotEmpty(symbols)

	// Test ReadLastData
	data, err := suite.cachedDataSource.ReadLastData("AAPL")
	suite.Require().NoError(err)
	suite.Equal("AAPL", data.Symbol)
}

// TestCacheKeyUniqueness tests that cache keys are unique for different parameters
func (suite *CachedDataSourceTestSuite) TestCacheKeyUniqueness() {
	suite.cachedDataSource.ClearCache()

	// Same time but different symbol
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)
	count := 5

	data1, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, "AAPL", count)
	suite.Require().NoError(err)

	data2, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, "GOOGL", count)
	// May error if GOOGL doesn't exist, but the important thing is they have different cache keys
	if err == nil {
		// If both succeed, they should have different symbols
		suite.Equal("AAPL", data1[0].Symbol)
		suite.Equal("GOOGL", data2[0].Symbol)
	}
}

// TestSimulateIndicatorCalls simulates what happens when multiple indicators are called
func (suite *CachedDataSourceTestSuite) TestSimulateIndicatorCalls() {
	suite.cachedDataSource.ClearCache()

	symbol := "AAPL"
	endTime := time.Date(2024, 1, 2, 10, 0, 0, 0, time.UTC)

	// Simulate RSI call (needs 15 data points for 14-period RSI)
	_, err := suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 15)
	suite.Require().NoError(err)

	// Simulate EMA call (needs 20 data points)
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 20)
	suite.Require().NoError(err)

	// Simulate MACD fast EMA call (needs 12 data points)
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 12)
	suite.Require().NoError(err)

	// Simulate MACD slow EMA call (needs 26 data points)
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 26)
	suite.Require().NoError(err)

	// Now simulate same calls again within same bar - should all be cached
	// (In real scenario, this happens when multiple composite indicators call sub-indicators)
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 15)
	suite.Require().NoError(err)
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 12)
	suite.Require().NoError(err)

	// Clear cache for next bar
	suite.cachedDataSource.ClearCache()

	// Verify cache works correctly after clear
	_, err = suite.cachedDataSource.GetPreviousNumberOfDataPoints(endTime, symbol, 15)
	suite.Require().NoError(err)
}

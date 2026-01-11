package indicator

import (
	"fmt"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// MockDataSource is a mock implementation of DataSource for indicator testing
type MockDataSource struct {
	data map[string][]types.MarketData
}

func NewMockDataSource() *MockDataSource {
	return &MockDataSource{
		data: make(map[string][]types.MarketData),
	}
}

func (m *MockDataSource) Initialize(path string) error {
	return nil
}

func (m *MockDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		for _, symbolData := range m.data {
			for _, d := range symbolData {
				// Apply time filters
				if start.IsSome() && d.Time.Before(start.Unwrap()) {
					continue
				}
				if end.IsSome() && d.Time.After(end.Unwrap()) {
					continue
				}
				if !yield(d, nil) {
					return
				}
			}
		}
	}
}

func (m *MockDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[datasource.Interval]) ([]types.MarketData, error) {
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

func (m *MockDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	symbolData, ok := m.data[symbol]
	if !ok {
		return nil, fmt.Errorf("symbol %s not found", symbol)
	}

	var result []types.MarketData
	for i := len(symbolData) - 1; i >= 0 && len(result) < count; i-- {
		if !symbolData[i].Time.After(end) {
			result = append([]types.MarketData{symbolData[i]}, result...)
		}
	}

	if len(result) < count {
		return nil, fmt.Errorf("insufficient data points for symbol %s: requested %d, got %d", symbol, count, len(result))
	}
	return result, nil
}

func (m *MockDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
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

func (m *MockDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	symbolData, ok := m.data[symbol]
	if !ok || len(symbolData) == 0 {
		return types.MarketData{}, fmt.Errorf("no data")
	}
	return symbolData[len(symbolData)-1], nil
}

func (m *MockDataSource) ExecuteSQL(query string, params ...interface{}) ([]datasource.SQLResult, error) {
	return nil, nil
}

func (m *MockDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	count := 0
	for _, symbolData := range m.data {
		count += len(symbolData)
	}
	return count, nil
}

func (m *MockDataSource) Close() error {
	return nil
}

func (m *MockDataSource) GetAllSymbols() ([]string, error) {
	symbols := make([]string, 0, len(m.data))
	for s := range m.data {
		symbols = append(symbols, s)
	}
	return symbols, nil
}

func (m *MockDataSource) AddData(data types.MarketData) {
	if _, ok := m.data[data.Symbol]; !ok {
		m.data[data.Symbol] = make([]types.MarketData, 0)
	}
	m.data[data.Symbol] = append(m.data[data.Symbol], data)
}

// generateTestData generates n data points for testing with realistic price movements
func generateTestData(symbol string, n int) []types.MarketData {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data := make([]types.MarketData, n)

	// Start with a base price and add some variation
	basePrice := 100.0
	for i := 0; i < n; i++ {
		// Simple price variation for testing
		variation := float64(i%10) * 0.5
		price := basePrice + variation

		data[i] = types.MarketData{
			Id:     fmt.Sprintf("%s_%d", symbol, i),
			Symbol: symbol,
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   price - 0.5,
			High:   price + 1.0,
			Low:    price - 1.0,
			Close:  price,
			Volume: 1000 + float64(i),
		}
	}

	return data
}

// SlidingWindowIndicatorTestSuite tests indicators with SlidingWindowDataSource
type SlidingWindowIndicatorTestSuite struct {
	suite.Suite
	mockDS   *MockDataSource
	registry IndicatorRegistry
	cache    cache.Cache
	baseTime time.Time
}

// SetupSuite sets up the test suite
func (suite *SlidingWindowIndicatorTestSuite) SetupSuite() {
	suite.baseTime = time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Create mock datasource with 500 data points
	suite.mockDS = NewMockDataSource()
	data := generateTestData("AAPL", 500)
	for _, d := range data {
		suite.mockDS.AddData(d)
	}

	// Add another symbol for multi-symbol tests
	spyData := generateTestData("SPY", 500)
	for _, d := range spyData {
		suite.mockDS.AddData(d)
	}

	// Setup indicator registry
	suite.registry = NewIndicatorRegistry()
	suite.registry.RegisterIndicator(NewEMA())
	suite.registry.RegisterIndicator(NewRSI())
	suite.registry.RegisterIndicator(NewMACD())
	suite.registry.RegisterIndicator(NewATR())
	suite.registry.RegisterIndicator(NewBollingerBands())
	suite.registry.RegisterIndicator(NewMA())

	suite.cache = cache.NewCacheV1()
}

func TestSlidingWindowIndicatorSuite(t *testing.T) {
	suite.Run(t, new(SlidingWindowIndicatorTestSuite))
}

// populateCache populates the sliding window cache with data from a time range
func (suite *SlidingWindowIndicatorTestSuite) populateCache(ds *datasource.SlidingWindowDataSource, start, end time.Time) int {
	count := 0
	for data := range suite.mockDS.ReadAll(optional.Some(start), optional.Some(end)) {
		ds.AddToCache(data)
		count++
	}
	return count
}

// =============================================================================
// Cache Eviction Scenarios
// =============================================================================

// TestIndicatorWithCacheSmallerThanPeriod tests when indicator period > cache size
// The indicator should fall back to DB when cache can't satisfy the request
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorWithCacheSmallerThanPeriod() {
	// Create a sliding window with small cache (30 data points)
	cacheSize := 30
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	// Use MA with period 50 (larger than cache)
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)

	// Populate cache with limited data
	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// MA-50 should still work by falling back to DB
	value, err := ma.RawValue("AAPL", endTime, ctx, 50)
	suite.NoError(err, "MA-50 should work even with cache size 30 by falling back to DB")
	suite.NotZero(value, "MA value should not be zero")
}

// TestIndicatorFallbackToDB verifies DB fallback works correctly
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorFallbackToDB() {
	// Create sliding window with moderate cache
	cacheSize := 50
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)

	// Populate cache with data starting AFTER what we need
	// This forces fallback to DB for older data
	recentStart := endTime.Add(-20 * time.Minute)
	suite.populateCache(ds, recentStart, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// RSI-14 needs 15 data points (14 + 1) - should work from cache
	rsi, err := suite.registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Require().NoError(err)

	rsiValue, err := rsi.RawValue("AAPL", endTime, ctx, 14)
	suite.NoError(err, "RSI should work with recent data in cache")
	suite.NotZero(rsiValue)

	// MA-50 needs 50 data points - cache only has 20, must fall back to DB
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)

	maValue, err := ma.RawValue("AAPL", endTime, ctx, 50)
	suite.NoError(err, "MA-50 should fall back to DB when cache insufficient")
	suite.NotZero(maValue)
}

// TestMultipleIndicatorsWithDifferentPeriods tests multiple indicators simultaneously
func (suite *SlidingWindowIndicatorTestSuite) TestMultipleIndicatorsWithDifferentPeriods() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// RSI-14 (needs 15 data points)
	rsi, err := suite.registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Require().NoError(err)
	rsiValue, err := rsi.RawValue("AAPL", endTime, ctx, 14)
	suite.NoError(err)
	suite.NotZero(rsiValue)

	// MA-20 (needs 20 data points)
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	ma20Value, err := ma.RawValue("AAPL", endTime, ctx, 20)
	suite.NoError(err)
	suite.NotZero(ma20Value)

	// MA-50 (needs 50 data points)
	ma50Value, err := ma.RawValue("AAPL", endTime, ctx, 50)
	suite.NoError(err)
	suite.NotZero(ma50Value)

	// EMA-26 (needs 26 data points)
	ema, err := suite.registry.GetIndicator(types.IndicatorTypeEMA)
	suite.Require().NoError(err)
	emaValue, err := ema.RawValue("AAPL", endTime, ctx, 26)
	suite.NoError(err)
	suite.NotZero(emaValue)
}

// =============================================================================
// Cache Boundary Tests
// =============================================================================

// TestIndicatorWithExactCacheSize tests when period == cache size
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorWithExactCacheSize() {
	cacheSize := 50
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	count := suite.populateCache(ds, startTime, endTime)

	suite.GreaterOrEqual(count, cacheSize, "Should have populated enough data")

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// MA with period equal to cache size
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	value, err := ma.RawValue("AAPL", endTime, ctx, cacheSize)
	suite.NoError(err, "MA with period == cache size should work")
	suite.NotZero(value)
}

// TestIndicatorWithCacheSizeMinusOne tests when period == cache size - 1
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorWithCacheSizeMinusOne() {
	cacheSize := 50
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// MA with period == cache size - 1 (should definitely be cache hit)
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	period := cacheSize - 1
	value, err := ma.RawValue("AAPL", endTime, ctx, period)
	suite.NoError(err, "MA with period == cache size - 1 should work from cache")
	suite.NotZero(value)
}

// TestIndicatorWithCacheSizePlusOne tests when period == cache size + 1
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorWithCacheSizePlusOne() {
	cacheSize := 50
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// MA with period == cache size + 1 (should fall back to DB)
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	period := cacheSize + 1
	value, err := ma.RawValue("AAPL", endTime, ctx, period)
	suite.NoError(err, "MA with period == cache size + 1 should work via DB fallback")
	suite.NotZero(value)
}

// TestIndicatorAtCacheCapacity tests when cache is at maximum capacity
func (suite *SlidingWindowIndicatorTestSuite) TestIndicatorAtCacheCapacity() {
	cacheSize := 30
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	// Populate with more data than cache size to trigger eviction
	startTime := endTime.Add(-time.Duration(cacheSize*2) * time.Minute)

	count := 0
	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		if data.Symbol == "AAPL" {
			ds.AddToCache(data)
			count++
		}
	}

	// Cache should be at capacity (cacheSize)
	suite.Equal(cacheSize, ds.GetCache().Size("AAPL"), "Cache should be at capacity")

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Indicator within cache size should work
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	value, err := ma.RawValue("AAPL", endTime, ctx, 20)
	suite.NoError(err)
	suite.NotZero(value)
}

// =============================================================================
// Multi-Symbol Handling
// =============================================================================

// TestMultipleSymbolsIndependentCaches tests that each symbol has independent cache
func (suite *SlidingWindowIndicatorTestSuite) TestMultipleSymbolsIndependentCaches() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)

	// Populate cache with data for all symbols
	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		ds.AddToCache(data)
	}

	// Get symbols in cache
	aaplSize := ds.GetCache().Size("AAPL")
	spySize := ds.GetCache().Size("SPY")
	suite.Greater(aaplSize, 0, "AAPL should have data in cache")
	suite.Greater(spySize, 0, "SPY should have data in cache")

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Calculate indicator for AAPL
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	aaplValue, err := ma.RawValue("AAPL", endTime, ctx, 20)
	suite.NoError(err)
	suite.NotZero(aaplValue)

	// Calculate indicator for SPY
	spyValue, err := ma.RawValue("SPY", endTime, ctx, 20)
	suite.NoError(err)
	suite.NotZero(spyValue)

	// Both caches should still be intact
	suite.Equal(aaplSize, ds.GetCache().Size("AAPL"), "AAPL cache should be unchanged")
	suite.Equal(spySize, ds.GetCache().Size("SPY"), "SPY cache should be unchanged")
}

// TestSymbolNotInCache tests querying a symbol that's not in cache
func (suite *SlidingWindowIndicatorTestSuite) TestSymbolNotInCache() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-50 * time.Minute)

	// Populate cache with AAPL only (filter by symbol)
	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		if data.Symbol == "AAPL" {
			ds.AddToCache(data)
		}
	}

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Query for AAPL should work from cache
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	aaplValue, err := ma.RawValue("AAPL", endTime, ctx, 20)
	suite.NoError(err)
	suite.NotZero(aaplValue)

	// Query for SPY not in cache should fall back to DB
	spyValue, err := ma.RawValue("SPY", endTime, ctx, 20)
	suite.NoError(err, "SPY query should fall back to DB")
	suite.NotZero(spyValue)
}

// TestMixedCacheHitMiss tests scenarios with partial cache coverage
func (suite *SlidingWindowIndicatorTestSuite) TestMixedCacheHitMiss() {
	cacheSize := 30
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	// Only populate recent data
	recentStart := endTime.Add(-25 * time.Minute)
	suite.populateCache(ds, recentStart, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)

	// Small period - should hit cache
	smallValue, err := ma.RawValue("AAPL", endTime, ctx, 10)
	suite.NoError(err)
	suite.NotZero(smallValue)

	// Large period - should fall back to DB
	largeValue, err := ma.RawValue("AAPL", endTime, ctx, 50)
	suite.NoError(err)
	suite.NotZero(largeValue)
}

// =============================================================================
// Indicator-Specific Edge Cases
// =============================================================================

// TestRSIWithSlidingWindow verifies RSI calculation correctness with sliding window cache
func (suite *SlidingWindowIndicatorTestSuite) TestRSIWithSlidingWindow() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	rsi, err := suite.registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Require().NoError(err)

	// Calculate RSI with sliding window
	cachedRSI, err := rsi.RawValue("AAPL", endTime, ctx, 14)
	suite.NoError(err)

	// Calculate RSI directly from underlying datasource
	directCtx := IndicatorContext{
		DataSource:        suite.mockDS,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}
	directRSI, err := rsi.RawValue("AAPL", endTime, directCtx, 14)
	suite.NoError(err)

	// Values should match (within floating point tolerance)
	suite.InDelta(directRSI, cachedRSI, 0.0001, "RSI from cache should match direct calculation")
}

// TestEMAWithSlidingWindow verifies EMA calculation correctness
func (suite *SlidingWindowIndicatorTestSuite) TestEMAWithSlidingWindow() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	ema, err := suite.registry.GetIndicator(types.IndicatorTypeEMA)
	suite.Require().NoError(err)

	// Test various EMA periods
	periods := []int{12, 20, 26}
	for _, period := range periods {
		cachedEMA, err := ema.RawValue("AAPL", endTime, ctx, period)
		suite.NoError(err, "EMA-%d should work", period)
		suite.NotZero(cachedEMA, "EMA-%d value should not be zero", period)

		// Verify against direct calculation
		directCtx := IndicatorContext{
			DataSource:        suite.mockDS,
			IndicatorRegistry: suite.registry,
			Cache:             suite.cache,
		}
		directEMA, _ := ema.RawValue("AAPL", endTime, directCtx, period)
		suite.InDelta(directEMA, cachedEMA, 0.0001, "EMA-%d from cache should match direct", period)
	}
}

// TestMACDWithSlidingWindow verifies MACD calculation (uses multiple EMAs)
func (suite *SlidingWindowIndicatorTestSuite) TestMACDWithSlidingWindow() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	endTime := suite.baseTime.Add(400 * time.Minute)
	startTime := endTime.Add(-time.Duration(cacheSize) * time.Minute)
	suite.populateCache(ds, startTime, endTime)

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	macd, err := suite.registry.GetIndicator(types.IndicatorTypeMACD)
	suite.Require().NoError(err)

	// MACD with default parameters (12, 26, 9)
	_, err = macd.RawValue("AAPL", endTime, ctx, 12, 26, 9)
	suite.NoError(err, "MACD should work with sliding window cache")
}

// =============================================================================
// Backtest Simulation
// =============================================================================

// TestBacktestSimulationWithIndicators simulates an actual backtest loop
func (suite *SlidingWindowIndicatorTestSuite) TestBacktestSimulationWithIndicators() {
	cacheSize := 100
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	// Simulate backtest: process data chronologically
	startTime := suite.baseTime
	endTime := suite.baseTime.Add(200 * time.Minute)

	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	rsi, err := suite.registry.GetIndicator(types.IndicatorTypeRSI)
	suite.Require().NoError(err)

	processedCount := 0
	indicatorCalculations := 0

	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		if data.Symbol != "AAPL" {
			continue
		}

		// Add data to cache (simulating backtest data feed)
		ds.AddToCache(data)
		processedCount++

		// Skip first N bars to have enough data
		if processedCount < 20 {
			continue
		}

		ctx := IndicatorContext{
			DataSource:        ds,
			IndicatorRegistry: suite.registry,
			Cache:             suite.cache,
		}

		// Calculate indicators at each bar
		_, err := ma.RawValue(data.Symbol, data.Time, ctx, 14)
		if err == nil {
			indicatorCalculations++
		}

		_, err = rsi.RawValue(data.Symbol, data.Time, ctx, 14)
		if err == nil {
			indicatorCalculations++
		}
	}

	suite.Greater(processedCount, 0, "Should have processed some data")
	suite.Greater(indicatorCalculations, 0, "Should have calculated some indicators")
}

// TestCacheEvictionDuringBacktest tests cache eviction behavior during backtest
func (suite *SlidingWindowIndicatorTestSuite) TestCacheEvictionDuringBacktest() {
	// Very small cache to force eviction
	cacheSize := 20
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, cacheSize)

	startTime := suite.baseTime
	endTime := suite.baseTime.Add(100 * time.Minute)

	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)

	dataCount := 0
	successfulCalculations := 0

	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		if data.Symbol != "AAPL" {
			continue
		}

		ds.AddToCache(data)
		dataCount++

		// Cache should never exceed maxSize
		actualSize := ds.GetCache().Size(data.Symbol)
		suite.LessOrEqual(actualSize, cacheSize, "Cache should not exceed maxSize")

		if dataCount < 15 {
			continue
		}

		ctx := IndicatorContext{
			DataSource:        ds,
			IndicatorRegistry: suite.registry,
			Cache:             suite.cache,
		}

		// MA-10 should always work (within cache or via fallback)
		_, err := ma.RawValue(data.Symbol, data.Time, ctx, 10)
		if err == nil {
			successfulCalculations++
		}
	}

	suite.Greater(dataCount, cacheSize, "Should have processed more data than cache size")
	suite.Greater(successfulCalculations, 0, "Should have successful calculations despite eviction")
}

// TestZeroCacheSizeFallback tests that zero cache size falls back to DB for all queries
func (suite *SlidingWindowIndicatorTestSuite) TestZeroCacheSizeFallback() {
	// Zero cache size - all queries go to DB
	ds := datasource.NewSlidingWindowDataSource(suite.mockDS, 0)

	endTime := suite.baseTime.Add(400 * time.Minute)

	// Adding data should be no-op
	startTime := endTime.Add(-50 * time.Minute)
	for data := range suite.mockDS.ReadAll(optional.Some(startTime), optional.Some(endTime)) {
		ds.AddToCache(data)
	}

	// Cache should remain empty
	suite.Equal(0, ds.GetCache().TotalSize(), "Cache should remain empty with zero size")

	ctx := IndicatorContext{
		DataSource:        ds,
		IndicatorRegistry: suite.registry,
		Cache:             suite.cache,
	}

	// Indicators should still work via DB fallback
	ma, err := suite.registry.GetIndicator(types.IndicatorTypeMA)
	suite.Require().NoError(err)
	value, err := ma.RawValue("AAPL", endTime, ctx, 20)
	suite.NoError(err, "Indicator should work via DB fallback with zero cache size")
	suite.NotZero(value)
}

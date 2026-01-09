package datasource

import (
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type SlidingWindowCacheTestSuite struct {
	suite.Suite
}

func TestSlidingWindowCacheTestSuite(t *testing.T) {
	suite.Run(t, new(SlidingWindowCacheTestSuite))
}

func (s *SlidingWindowCacheTestSuite) createMarketData(symbol string, timeOffset int, close float64) types.MarketData {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return types.MarketData{
		Id:     symbol + "_" + baseTime.Add(time.Duration(timeOffset)*time.Minute).Format("20060102150405"),
		Symbol: symbol,
		Time:   baseTime.Add(time.Duration(timeOffset) * time.Minute),
		Open:   close - 1,
		High:   close + 1,
		Low:    close - 2,
		Close:  close,
		Volume: 1000,
	}
}

func (s *SlidingWindowCacheTestSuite) TestNewSlidingWindowCache() {
	cache := NewSlidingWindowCache(100)
	assert.NotNil(s.T(), cache)
	assert.Equal(s.T(), 100, cache.MaxSize())
	assert.Equal(s.T(), 0, cache.TotalSize())
}

func (s *SlidingWindowCacheTestSuite) TestAddAndSize() {
	cache := NewSlidingWindowCache(5)

	// Add data for a symbol
	cache.Add(s.createMarketData("SPY", 0, 100))
	assert.Equal(s.T(), 1, cache.Size("SPY"))
	assert.Equal(s.T(), 0, cache.Size("AAPL"))

	cache.Add(s.createMarketData("SPY", 1, 101))
	assert.Equal(s.T(), 2, cache.Size("SPY"))

	// Add data for another symbol
	cache.Add(s.createMarketData("AAPL", 0, 150))
	assert.Equal(s.T(), 1, cache.Size("AAPL"))
	assert.Equal(s.T(), 3, cache.TotalSize())
}

func (s *SlidingWindowCacheTestSuite) TestSlidingWindowEviction() {
	cache := NewSlidingWindowCache(3)

	// Add 5 data points, cache should only keep latest 3
	for i := 0; i < 5; i++ {
		cache.Add(s.createMarketData("SPY", i, float64(100+i)))
	}

	assert.Equal(s.T(), 3, cache.Size("SPY"))

	// Verify oldest data was evicted
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Data at minute 0 and 1 should be evicted
	_, ok := cache.GetMarketData("SPY", baseTime)
	assert.False(s.T(), ok)

	_, ok = cache.GetMarketData("SPY", baseTime.Add(time.Minute))
	assert.False(s.T(), ok)

	// Data at minutes 2, 3, 4 should exist
	data, ok := cache.GetMarketData("SPY", baseTime.Add(2*time.Minute))
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 102.0, data.Close)

	data, ok = cache.GetMarketData("SPY", baseTime.Add(4*time.Minute))
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 104.0, data.Close)
}

func (s *SlidingWindowCacheTestSuite) TestGetRange() {
	cache := NewSlidingWindowCache(10)

	// Add 5 data points
	for i := 0; i < 5; i++ {
		cache.Add(s.createMarketData("SPY", i, float64(100+i)))
	}

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Get range that's fully in cache
	data, ok := cache.GetRange(baseTime.Add(time.Minute), baseTime.Add(3*time.Minute))
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 3, len(data))
	assert.Equal(s.T(), 101.0, data[0].Close)
	assert.Equal(s.T(), 102.0, data[1].Close)
	assert.Equal(s.T(), 103.0, data[2].Close)

	// Get range that starts before cache coverage
	data, ok = cache.GetRange(baseTime.Add(-time.Minute), baseTime.Add(2*time.Minute))
	assert.False(s.T(), ok)
	assert.Nil(s.T(), data)
}

func (s *SlidingWindowCacheTestSuite) TestGetPreviousDataPoints() {
	cache := NewSlidingWindowCache(10)

	// Add 5 data points
	for i := 0; i < 5; i++ {
		cache.Add(s.createMarketData("SPY", i, float64(100+i)))
	}

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Get last 3 data points ending at minute 4
	data, ok := cache.GetPreviousDataPoints(baseTime.Add(4*time.Minute), "SPY", 3)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 3, len(data))
	assert.Equal(s.T(), 102.0, data[0].Close) // minute 2
	assert.Equal(s.T(), 103.0, data[1].Close) // minute 3
	assert.Equal(s.T(), 104.0, data[2].Close) // minute 4

	// Request more data than available
	data, ok = cache.GetPreviousDataPoints(baseTime.Add(2*time.Minute), "SPY", 10)
	assert.False(s.T(), ok)
	assert.Nil(s.T(), data)

	// Request for non-existent symbol
	data, ok = cache.GetPreviousDataPoints(baseTime.Add(4*time.Minute), "AAPL", 3)
	assert.False(s.T(), ok)
	assert.Nil(s.T(), data)
}

func (s *SlidingWindowCacheTestSuite) TestGetLastData() {
	cache := NewSlidingWindowCache(10)

	// Empty cache
	_, ok := cache.GetLastData("SPY")
	assert.False(s.T(), ok)

	// Add data
	cache.Add(s.createMarketData("SPY", 0, 100))
	cache.Add(s.createMarketData("SPY", 1, 101))
	cache.Add(s.createMarketData("SPY", 2, 102))

	data, ok := cache.GetLastData("SPY")
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 102.0, data.Close)
}

func (s *SlidingWindowCacheTestSuite) TestGetMarketData() {
	cache := NewSlidingWindowCache(10)

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Add data
	cache.Add(s.createMarketData("SPY", 0, 100))
	cache.Add(s.createMarketData("SPY", 2, 102))

	// Get existing data
	data, ok := cache.GetMarketData("SPY", baseTime)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 100.0, data.Close)

	// Get non-existent time
	_, ok = cache.GetMarketData("SPY", baseTime.Add(time.Minute))
	assert.False(s.T(), ok)

	// Get non-existent symbol
	_, ok = cache.GetMarketData("AAPL", baseTime)
	assert.False(s.T(), ok)
}

func (s *SlidingWindowCacheTestSuite) TestClear() {
	cache := NewSlidingWindowCache(10)

	cache.Add(s.createMarketData("SPY", 0, 100))
	cache.Add(s.createMarketData("AAPL", 0, 150))
	assert.Equal(s.T(), 2, cache.TotalSize())

	cache.Clear()
	assert.Equal(s.T(), 0, cache.TotalSize())
	assert.Equal(s.T(), 0, cache.Size("SPY"))
	assert.Equal(s.T(), 0, cache.Size("AAPL"))
}

func (s *SlidingWindowCacheTestSuite) TestZeroMaxSize() {
	cache := NewSlidingWindowCache(0)

	// Adding should be a no-op
	cache.Add(s.createMarketData("SPY", 0, 100))
	assert.Equal(s.T(), 0, cache.TotalSize())

	// All get methods should return false
	_, ok := cache.GetMarketData("SPY", time.Now())
	assert.False(s.T(), ok)

	_, ok = cache.GetLastData("SPY")
	assert.False(s.T(), ok)

	_, ok = cache.GetRange(time.Now(), time.Now())
	assert.False(s.T(), ok)

	_, ok = cache.GetPreviousDataPoints(time.Now(), "SPY", 1)
	assert.False(s.T(), ok)
}

func (s *SlidingWindowCacheTestSuite) TestDuplicateTimeHandling() {
	cache := NewSlidingWindowCache(10)

	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Add data at the same time
	data1 := types.MarketData{
		Symbol: "SPY",
		Time:   baseTime,
		Close:  100,
	}
	data2 := types.MarketData{
		Symbol: "SPY",
		Time:   baseTime,
		Close:  101,
	}

	cache.Add(data1)
	cache.Add(data2)

	// Should have only 1 entry, with the updated value
	assert.Equal(s.T(), 1, cache.Size("SPY"))

	retrieved, ok := cache.GetMarketData("SPY", baseTime)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 101.0, retrieved.Close)
}

func (s *SlidingWindowCacheTestSuite) TestOutOfOrderInsertion() {
	cache := NewSlidingWindowCache(10)

	// Insert out of order
	cache.Add(s.createMarketData("SPY", 2, 102))
	cache.Add(s.createMarketData("SPY", 0, 100))
	cache.Add(s.createMarketData("SPY", 1, 101))

	// Should be sorted by time
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	data, ok := cache.GetPreviousDataPoints(baseTime.Add(2*time.Minute), "SPY", 3)
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 3, len(data))
	assert.Equal(s.T(), 100.0, data[0].Close) // minute 0
	assert.Equal(s.T(), 101.0, data[1].Close) // minute 1
	assert.Equal(s.T(), 102.0, data[2].Close) // minute 2
}

func (s *SlidingWindowCacheTestSuite) TestMultipleSymbols() {
	cache := NewSlidingWindowCache(3)

	// Add data for multiple symbols
	for i := 0; i < 5; i++ {
		cache.Add(s.createMarketData("SPY", i, float64(100+i)))
		cache.Add(s.createMarketData("AAPL", i, float64(150+i)))
	}

	// Each symbol should have 3 entries (its own window)
	assert.Equal(s.T(), 3, cache.Size("SPY"))
	assert.Equal(s.T(), 3, cache.Size("AAPL"))
	assert.Equal(s.T(), 6, cache.TotalSize())

	// Verify the correct data for each symbol
	spyLast, ok := cache.GetLastData("SPY")
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 104.0, spyLast.Close)

	aaplLast, ok := cache.GetLastData("AAPL")
	assert.True(s.T(), ok)
	assert.Equal(s.T(), 154.0, aaplLast.Close)
}

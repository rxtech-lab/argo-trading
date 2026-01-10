package datasource

import (
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// SlidingWindowDataSource wraps a DataSource and uses a sliding window cache
// to store market data as it's processed. It checks the cache first for data
// requests and falls back to the underlying datasource when the cache cannot
// satisfy the request.
type SlidingWindowDataSource struct {
	underlying DataSource
	cache      *SlidingWindowCache
}

// NewSlidingWindowDataSource creates a new SlidingWindowDataSource with the specified
// cache size per symbol. If cacheSize is 0, caching is disabled and all requests
// go directly to the underlying datasource.
func NewSlidingWindowDataSource(underlying DataSource, cacheSize int) *SlidingWindowDataSource {
	return &SlidingWindowDataSource{
		underlying: underlying,
		cache:      NewSlidingWindowCache(cacheSize),
	}
}

// AddToCache adds a market data point to the sliding window cache.
// This should be called as market data is processed during backtesting.
func (s *SlidingWindowDataSource) AddToCache(data types.MarketData) {
	s.cache.Add(data)
}

// GetCache returns the underlying sliding window cache for direct access.
func (s *SlidingWindowDataSource) GetCache() *SlidingWindowCache {
	return s.cache
}

// ClearCache clears all cached data.
func (s *SlidingWindowDataSource) ClearCache() {
	s.cache.Clear()
}

// Initialize implements DataSource.
func (s *SlidingWindowDataSource) Initialize(path string) error {
	return s.underlying.Initialize(path)
}

// ReadAll implements DataSource. This bypasses the cache as it's meant
// for streaming through all data.
func (s *SlidingWindowDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return s.underlying.ReadAll(start, end)
}

// GetRange implements DataSource with cache support.
// It first checks if the sliding window cache can satisfy the request.
// If not, it falls back to the underlying datasource.
func (s *SlidingWindowDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	// If interval is specified, bypass cache (aggregation not supported in cache)
	if interval.IsSome() {
		return s.underlying.GetRange(start, end, interval)
	}

	// Try to get from cache first
	if data, ok := s.cache.GetRange(start, end); ok {
		return data, nil
	}

	// Cache miss or insufficient data, fall back to underlying datasource
	return s.underlying.GetRange(start, end, interval)
}

// GetPreviousNumberOfDataPoints implements DataSource with cache support.
// It first checks if the sliding window cache can satisfy the request.
// If not, it falls back to the underlying datasource.
func (s *SlidingWindowDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	// Try to get from cache first
	if data, ok := s.cache.GetPreviousDataPoints(end, symbol, count); ok {
		return data, nil
	}

	// Cache miss or insufficient data, fall back to underlying datasource
	return s.underlying.GetPreviousNumberOfDataPoints(end, symbol, count)
}

// GetMarketData implements DataSource with cache support.
func (s *SlidingWindowDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	// Try to get from cache first
	if data, ok := s.cache.GetMarketData(symbol, timestamp); ok {
		return data, nil
	}

	// Cache miss, fall back to underlying datasource
	return s.underlying.GetMarketData(symbol, timestamp)
}

// ReadLastData implements DataSource with cache support.
func (s *SlidingWindowDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	// Try to get from cache first
	if data, ok := s.cache.GetLastData(symbol); ok {
		return data, nil
	}

	// Cache miss, fall back to underlying datasource
	return s.underlying.ReadLastData(symbol)
}

// ExecuteSQL implements DataSource. SQL queries bypass the cache
// as they may require complex operations not supported by the cache.
func (s *SlidingWindowDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	return s.underlying.ExecuteSQL(query, params...)
}

// Count implements DataSource.
func (s *SlidingWindowDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	return s.underlying.Count(start, end)
}

// Close implements DataSource.
func (s *SlidingWindowDataSource) Close() error {
	return s.underlying.Close()
}

// GetAllSymbols implements DataSource.
func (s *SlidingWindowDataSource) GetAllSymbols() ([]string, error) {
	return s.underlying.GetAllSymbols()
}

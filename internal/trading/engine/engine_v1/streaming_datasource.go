package engine_v1

import (
	"errors"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// ErrNotSupported is returned when an operation is not supported in streaming mode.
var ErrNotSupported = errors.New("operation not supported in streaming mode")

// ErrDataNotFound is returned when the requested data is not found in the cache.
var ErrDataNotFound = errors.New("data not found in cache")

// StreamingDataSource implements datasource.DataSource for live streaming.
// It uses a sliding window cache to store recent market data for indicator calculations.
// Unlike the backtest SlidingWindowDataSource, this has no underlying datasource -
// all data comes from the real-time stream and is stored in the cache.
type StreamingDataSource struct {
	cache *datasource.SlidingWindowCache
}

// NewStreamingDataSource creates a new StreamingDataSource with the specified
// cache size per symbol.
func NewStreamingDataSource(cacheSize int) *StreamingDataSource {
	return &StreamingDataSource{
		cache: datasource.NewSlidingWindowCache(cacheSize),
	}
}

// AddToCache adds a market data point to the sliding window cache.
// This should be called as market data is received from the stream.
func (s *StreamingDataSource) AddToCache(data types.MarketData) {
	s.cache.Add(data)
}

// GetCache returns the underlying sliding window cache for direct access.
func (s *StreamingDataSource) GetCache() *datasource.SlidingWindowCache {
	return s.cache
}

// ClearCache clears all cached data.
func (s *StreamingDataSource) ClearCache() {
	s.cache.Clear()
}

// Initialize implements datasource.DataSource.
// In streaming mode, this is a no-op since data comes from the real-time stream.
func (s *StreamingDataSource) Initialize(_ string) error {
	// No initialization needed for streaming mode
	return nil
}

// ReadAll implements datasource.DataSource.
// This is not supported in streaming mode since there's no historical data to read.
func (s *StreamingDataSource) ReadAll(_ optional.Option[time.Time], _ optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		yield(types.MarketData{}, ErrNotSupported) //nolint:exhaustruct // zero value for error response
	}
}

// GetRange implements datasource.DataSource.
// Returns data from the cache if available.
func (s *StreamingDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[datasource.Interval]) ([]types.MarketData, error) {
	// Aggregation (interval) is not supported in streaming cache
	if interval.IsSome() {
		return nil, ErrNotSupported
	}

	// Try to get from cache
	if data, ok := s.cache.GetRange(start, end); ok {
		return data, nil
	}

	// Cache miss - in streaming mode, we can only return what's in the cache
	return nil, ErrDataNotFound
}

// GetPreviousNumberOfDataPoints implements datasource.DataSource.
// Returns data from the cache if available.
func (s *StreamingDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	// Try to get from cache
	if data, ok := s.cache.GetPreviousDataPoints(end, symbol, count); ok {
		return data, nil
	}

	// Cache miss - in streaming mode, we can only return what's in the cache
	return nil, ErrDataNotFound
}

// GetMarketData implements datasource.DataSource.
// Returns data from the cache if available.
func (s *StreamingDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	// Try to get from cache
	if data, ok := s.cache.GetMarketData(symbol, timestamp); ok {
		return data, nil
	}

	// Cache miss
	return types.MarketData{}, ErrDataNotFound
}

// ReadLastData implements datasource.DataSource.
// Returns the most recent data from the cache.
func (s *StreamingDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	// Try to get from cache
	if data, ok := s.cache.GetLastData(symbol); ok {
		return data, nil
	}

	// Cache miss
	return types.MarketData{}, ErrDataNotFound
}

// ExecuteSQL implements datasource.DataSource.
// SQL queries are not supported in streaming mode since there's no database.
func (s *StreamingDataSource) ExecuteSQL(_ string, _ ...interface{}) ([]datasource.SQLResult, error) {
	return nil, ErrNotSupported
}

// Count implements datasource.DataSource.
// Returns the total number of cached entries across all symbols.
func (s *StreamingDataSource) Count(_ optional.Option[time.Time], _ optional.Option[time.Time]) (int, error) {
	return s.cache.TotalSize(), nil
}

// Close implements datasource.DataSource.
// Clears the cache and releases resources.
func (s *StreamingDataSource) Close() error {
	s.cache.Clear()

	return nil
}

// GetAllSymbols implements datasource.DataSource.
// This is not supported in streaming mode since we don't have a complete dataset.
func (s *StreamingDataSource) GetAllSymbols() ([]string, error) {
	// We could potentially return symbols that have data in cache,
	// but that would be inconsistent behavior. Return error instead.
	return nil, ErrNotSupported
}

// Verify StreamingDataSource implements datasource.DataSource interface.
var _ datasource.DataSource = (*StreamingDataSource)(nil)

package datasource

import (
	"fmt"
	"sync"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// CachedDataSource wraps a DataSource and caches repeated queries within the same bar/time period.
// This significantly improves performance when multiple indicators query similar data.
type CachedDataSource struct {
	underlying           DataSource
	previousDataCache    map[string][]types.MarketData
	previousDataErrCache map[string]error
	rangeCache           map[string][]types.MarketData
	rangeErrCache        map[string]error
	mu                   sync.RWMutex
}

// NewCachedDataSource creates a new CachedDataSource wrapping the given DataSource.
func NewCachedDataSource(underlying DataSource) *CachedDataSource {
	return &CachedDataSource{
		underlying:           underlying,
		previousDataCache:    make(map[string][]types.MarketData),
		previousDataErrCache: make(map[string]error),
		rangeCache:           make(map[string][]types.MarketData),
		rangeErrCache:        make(map[string]error),
	}
}

// ClearCache clears all cached data. Call this when moving to a new bar/time period.
func (c *CachedDataSource) ClearCache() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.previousDataCache = make(map[string][]types.MarketData)
	c.previousDataErrCache = make(map[string]error)
	c.rangeCache = make(map[string][]types.MarketData)
	c.rangeErrCache = make(map[string]error)
}

// Initialize implements DataSource.
func (c *CachedDataSource) Initialize(path string) error {
	return c.underlying.Initialize(path)
}

// ReadAll implements DataSource.
func (c *CachedDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return c.underlying.ReadAll(start, end)
}

// GetRange implements DataSource with caching.
func (c *CachedDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	key := c.buildRangeKey(start, end, interval)

	// Check cache first (read lock)
	c.mu.RLock()
	if data, ok := c.rangeCache[key]; ok {
		err := c.rangeErrCache[key]
		c.mu.RUnlock()
		return data, err
	}
	c.mu.RUnlock()

	// Cache miss - fetch from underlying (write lock)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if data, ok := c.rangeCache[key]; ok {
		return data, c.rangeErrCache[key]
	}

	data, err := c.underlying.GetRange(start, end, interval)
	c.rangeCache[key] = data
	c.rangeErrCache[key] = err
	return data, err
}

// GetPreviousNumberOfDataPoints implements DataSource with caching.
func (c *CachedDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	key := c.buildPreviousDataKey(end, symbol, count)

	// Check cache first (read lock)
	c.mu.RLock()
	if data, ok := c.previousDataCache[key]; ok {
		err := c.previousDataErrCache[key]
		c.mu.RUnlock()
		return data, err
	}
	c.mu.RUnlock()

	// Cache miss - fetch from underlying (write lock)
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if data, ok := c.previousDataCache[key]; ok {
		return data, c.previousDataErrCache[key]
	}

	data, err := c.underlying.GetPreviousNumberOfDataPoints(end, symbol, count)
	c.previousDataCache[key] = data
	c.previousDataErrCache[key] = err
	return data, err
}

// GetMarketData implements DataSource.
func (c *CachedDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	return c.underlying.GetMarketData(symbol, timestamp)
}

// ReadLastData implements DataSource.
func (c *CachedDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	return c.underlying.ReadLastData(symbol)
}

// ExecuteSQL implements DataSource.
func (c *CachedDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	return c.underlying.ExecuteSQL(query, params...)
}

// Count implements DataSource.
func (c *CachedDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	return c.underlying.Count(start, end)
}

// Close implements DataSource.
func (c *CachedDataSource) Close() error {
	return c.underlying.Close()
}

// GetAllSymbols implements DataSource.
func (c *CachedDataSource) GetAllSymbols() ([]string, error) {
	return c.underlying.GetAllSymbols()
}

// buildPreviousDataKey creates a cache key for GetPreviousNumberOfDataPoints.
func (c *CachedDataSource) buildPreviousDataKey(end time.Time, symbol string, count int) string {
	return fmt.Sprintf("prev:%s:%d:%d", symbol, end.UnixNano(), count)
}

// buildRangeKey creates a cache key for GetRange.
func (c *CachedDataSource) buildRangeKey(start time.Time, end time.Time, interval optional.Option[Interval]) string {
	intervalStr := "none"
	if interval.IsSome() {
		intervalStr = string(interval.Unwrap())
	}
	return fmt.Sprintf("range:%d:%d:%s", start.UnixNano(), end.UnixNano(), intervalStr)
}

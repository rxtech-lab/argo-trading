package datasource

import (
	"sort"
	"sync"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// SlidingWindowCache stores market data using a sliding window algorithm.
// It maintains a fixed-size cache per symbol, automatically evicting the oldest
// entries when the cache reaches capacity.
type SlidingWindowCache struct {
	maxSize int
	// data stores market data per symbol, ordered by time (oldest first)
	data map[string][]types.MarketData
	mu   sync.RWMutex
}

// NewSlidingWindowCache creates a new SlidingWindowCache with the specified maximum size per symbol.
func NewSlidingWindowCache(maxSize int) *SlidingWindowCache {
	return &SlidingWindowCache{
		maxSize: maxSize,
		data:    make(map[string][]types.MarketData),
		mu:      sync.RWMutex{},
	}
}

// Add adds a market data point to the cache. If the cache for this symbol
// exceeds maxSize, the oldest entry is evicted.
// Optimized for the common case where data is added in chronological order (backtesting).
func (c *SlidingWindowCache) Add(data types.MarketData) {
	if c.maxSize <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	symbol := data.Symbol
	if _, ok := c.data[symbol]; !ok {
		c.data[symbol] = make([]types.MarketData, 0, c.maxSize)
	}

	symbolData := c.data[symbol]

	// Fast path: chronological append (common case in backtesting)
	// Check if data is after or equal to the last entry
	if len(symbolData) > 0 {
		lastTime := symbolData[len(symbolData)-1].Time
		if data.Time.After(lastTime) {
			// Append at end - O(1)
			symbolData = append(symbolData, data)
			// Evict oldest if over capacity
			if len(symbolData) > c.maxSize {
				symbolData = symbolData[1:]
			}

			c.data[symbol] = symbolData

			return
		}

		if data.Time.Equal(lastTime) {
			// Update last entry - O(1)
			symbolData[len(symbolData)-1] = data

			return
		}
	} else {
		// Empty slice, just append
		c.data[symbol] = append(symbolData, data)

		return
	}

	// Slow path: out-of-order insertion (rare in backtesting)
	// Insert data in sorted order by time using binary search
	insertIdx := sort.Search(len(symbolData), func(i int) bool {
		return symbolData[i].Time.After(data.Time) || symbolData[i].Time.Equal(data.Time)
	})

	// Check if data at this time already exists (avoid duplicates)
	if insertIdx < len(symbolData) && symbolData[insertIdx].Time.Equal(data.Time) {
		// Update existing entry
		symbolData[insertIdx] = data

		return
	}

	// Insert at the correct position
	symbolData = append(symbolData, types.MarketData{}) //nolint:exhaustruct // placeholder for slice expansion
	copy(symbolData[insertIdx+1:], symbolData[insertIdx:])
	symbolData[insertIdx] = data

	// Evict oldest entries if over capacity
	if len(symbolData) > c.maxSize {
		// Remove from the beginning (oldest data)
		symbolData = symbolData[len(symbolData)-c.maxSize:]
	}

	c.data[symbol] = symbolData
}

// GetRange returns market data within the specified time range for all symbols.
// Returns nil if the requested range cannot be fully satisfied from cache.
func (c *SlidingWindowCache) GetRange(start time.Time, end time.Time) ([]types.MarketData, bool) {
	if c.maxSize <= 0 {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	var result []types.MarketData

	for _, symbolData := range c.data {
		if len(symbolData) == 0 {
			continue
		}

		// Check if cache covers the requested range for this symbol
		oldestTime := symbolData[0].Time
		if start.Before(oldestTime) {
			// Cache doesn't have data from the start of the range
			return nil, false
		}

		// Binary search for start index
		startIdx := sort.Search(len(symbolData), func(i int) bool {
			return !symbolData[i].Time.Before(start)
		})

		// Binary search for end index
		endIdx := sort.Search(len(symbolData), func(i int) bool {
			return symbolData[i].Time.After(end)
		})

		// Collect data in range
		for i := startIdx; i < endIdx; i++ {
			result = append(result, symbolData[i])
		}
	}

	// Sort combined result by time
	sort.Slice(result, func(i, j int) bool {
		return result[i].Time.Before(result[j].Time)
	})

	return result, true
}

// GetPreviousDataPoints returns the specified number of historical data points
// for a given symbol, ending at the specified time.
// Returns data in chronological order (oldest to newest).
// Returns nil if the requested count cannot be satisfied from cache.
func (c *SlidingWindowCache) GetPreviousDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, bool) {
	if c.maxSize <= 0 || count <= 0 {
		return nil, false
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	symbolData, ok := c.data[symbol]
	if !ok || len(symbolData) == 0 {
		return nil, false
	}

	// Binary search for the end index (first element after end time)
	endIdx := sort.Search(len(symbolData), func(i int) bool {
		return symbolData[i].Time.After(end)
	})

	// Check if we have enough data points before the end time
	if endIdx < count {
		return nil, false
	}

	// Extract the requested data points
	startIdx := endIdx - count
	result := make([]types.MarketData, count)
	copy(result, symbolData[startIdx:endIdx])

	return result, true
}

// GetMarketData returns the market data for a specific symbol and time.
// Returns false if the data is not in cache.
func (c *SlidingWindowCache) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, bool) {
	if c.maxSize <= 0 {
		return types.MarketData{}, false //nolint:exhaustruct // zero value for not found
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	symbolData, ok := c.data[symbol]
	if !ok || len(symbolData) == 0 {
		return types.MarketData{}, false //nolint:exhaustruct // zero value for not found
	}

	// Binary search for the exact time
	idx := sort.Search(len(symbolData), func(i int) bool {
		return !symbolData[i].Time.Before(timestamp)
	})

	if idx < len(symbolData) && symbolData[idx].Time.Equal(timestamp) {
		return symbolData[idx], true
	}

	return types.MarketData{}, false //nolint:exhaustruct // zero value for not found
}

// GetLastData returns the most recent market data for a specific symbol.
// Returns false if no data exists for the symbol.
func (c *SlidingWindowCache) GetLastData(symbol string) (types.MarketData, bool) {
	if c.maxSize <= 0 {
		return types.MarketData{}, false //nolint:exhaustruct // zero value for not found
	}

	c.mu.RLock()
	defer c.mu.RUnlock()

	symbolData, ok := c.data[symbol]
	if !ok || len(symbolData) == 0 {
		return types.MarketData{}, false //nolint:exhaustruct // zero value for not found
	}

	return symbolData[len(symbolData)-1], true
}

// Size returns the current number of cached entries for a symbol.
func (c *SlidingWindowCache) Size(symbol string) int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if symbolData, ok := c.data[symbol]; ok {
		return len(symbolData)
	}

	return 0
}

// TotalSize returns the total number of cached entries across all symbols.
func (c *SlidingWindowCache) TotalSize() int {
	c.mu.RLock()
	defer c.mu.RUnlock()

	total := 0
	for _, symbolData := range c.data {
		total += len(symbolData)
	}

	return total
}

// Clear removes all cached data.
func (c *SlidingWindowCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.data = make(map[string][]types.MarketData)
}

// MaxSize returns the maximum cache size per symbol.
func (c *SlidingWindowCache) MaxSize() int {
	return c.maxSize
}

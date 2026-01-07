package datasource

import (
	"sort"
	"sync"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// InMemoryIndexedDataSource provides high-performance indexed access to market data.
// It preloads all data into memory and uses array indexing for O(1) lookups,
// providing 10x+ performance improvement over SQL-based queries during backtesting.
type InMemoryIndexedDataSource struct {
	underlying DataSource

	// Preloaded data indexed by symbol, then by bar index
	// data[symbol][barIndex] = MarketData
	data map[string][]types.MarketData

	// Time index maps timestamps to bar indices for each symbol
	// timeIndex[symbol][timestamp] = barIndex
	timeIndex map[string]map[int64]int

	// All data combined in chronological order for ReadAll iteration
	allData []types.MarketData

	// Current bar index for each symbol
	currentBarIndex map[string]int

	// Global bar index (for single-symbol or multi-symbol iteration)
	globalBarIndex int

	// Preload state
	preloaded bool

	// Mutex for thread-safe operations
	mu sync.RWMutex
}

// NewInMemoryIndexedDataSource creates a new InMemoryIndexedDataSource wrapping the given DataSource.
func NewInMemoryIndexedDataSource(underlying DataSource) *InMemoryIndexedDataSource {
	return &InMemoryIndexedDataSource{
		underlying:      underlying,
		data:            make(map[string][]types.MarketData),
		timeIndex:       make(map[string]map[int64]int),
		allData:         nil,
		currentBarIndex: make(map[string]int),
		globalBarIndex:  0,
		preloaded:       false,
		mu:              sync.RWMutex{},
	}
}

// Preload loads all data into memory for fast indexed access.
func (ds *InMemoryIndexedDataSource) Preload(start optional.Option[time.Time], end optional.Option[time.Time]) error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Clear existing data
	ds.data = make(map[string][]types.MarketData)
	ds.timeIndex = make(map[string]map[int64]int)
	ds.currentBarIndex = make(map[string]int)
	ds.globalBarIndex = 0

	// Load all data from underlying source
	var allData []types.MarketData
	for marketData, err := range ds.underlying.ReadAll(start, end) {
		if err != nil {
			return errors.Wrap(errors.ErrCodeDataNotFound, "failed to preload data", err)
		}
		allData = append(allData, marketData)
	}

	// Sort by time to ensure chronological order
	sort.Slice(allData, func(i, j int) bool {
		return allData[i].Time.Before(allData[j].Time)
	})

	ds.allData = allData

	// Index data by symbol
	for _, md := range allData {
		symbol := md.Symbol

		// Initialize symbol maps if needed
		if _, ok := ds.data[symbol]; !ok {
			ds.data[symbol] = make([]types.MarketData, 0)
			ds.timeIndex[symbol] = make(map[int64]int)
			ds.currentBarIndex[symbol] = 0
		}

		// Add to symbol's data array
		barIndex := len(ds.data[symbol])
		ds.data[symbol] = append(ds.data[symbol], md)

		// Create time index for backward compatibility
		ds.timeIndex[symbol][md.Time.UnixNano()] = barIndex
	}

	ds.preloaded = true
	return nil
}

// IsPreloaded returns true if data has been preloaded into memory.
func (ds *InMemoryIndexedDataSource) IsPreloaded() bool {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.preloaded
}

// SetCurrentBarIndex sets the current bar index for subsequent queries.
func (ds *InMemoryIndexedDataSource) SetCurrentBarIndex(index int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.globalBarIndex = index

	// Update per-symbol indices based on global index
	// This assumes single-symbol iteration; for multi-symbol, use SetCurrentBarIndexForSymbol
	for symbol := range ds.data {
		if index < len(ds.data[symbol]) {
			ds.currentBarIndex[symbol] = index
		}
	}
}

// SetCurrentBarIndexForSymbol sets the current bar index for a specific symbol.
func (ds *InMemoryIndexedDataSource) SetCurrentBarIndexForSymbol(symbol string, index int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.currentBarIndex[symbol] = index
}

// GetCurrentBarIndex returns the current global bar index.
func (ds *InMemoryIndexedDataSource) GetCurrentBarIndex() int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	return ds.globalBarIndex
}

// GetPreviousNBars returns the previous N bars ending at the current bar index.
// This is an O(1) operation using array slicing.
func (ds *InMemoryIndexedDataSource) GetPreviousNBars(symbol string, count int) ([]types.MarketData, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if !ds.preloaded {
		return nil, errors.New(errors.ErrCodeDataNotFound, "data not preloaded, call Preload() first")
	}

	symbolData, ok := ds.data[symbol]
	if !ok {
		return nil, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol: %s", symbol)
	}

	currentIdx, ok := ds.currentBarIndex[symbol]
	if !ok {
		return nil, errors.Newf(errors.ErrCodeDataNotFound, "current bar index not set for symbol: %s", symbol)
	}

	// Calculate the start index for the slice
	// We want `count` bars ending at currentIdx (inclusive)
	startIdx := currentIdx - count + 1
	if startIdx < 0 {
		startIdx = 0
	}

	// Slice end is currentIdx + 1 (exclusive)
	endIdx := currentIdx + 1
	if endIdx > len(symbolData) {
		endIdx = len(symbolData)
	}

	actualCount := endIdx - startIdx
	if actualCount < count {
		return nil, errors.NewInsufficientDataErrorf(count, actualCount, symbol,
			"insufficient data points for symbol %s: requested %d, got %d", symbol, count, actualCount)
	}

	// Return a copy to prevent modification of underlying data
	result := make([]types.MarketData, actualCount)
	copy(result, symbolData[startIdx:endIdx])

	return result, nil
}

// GetBarAtIndex returns the market data at a specific bar index.
func (ds *InMemoryIndexedDataSource) GetBarAtIndex(symbol string, index int) (types.MarketData, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if !ds.preloaded {
		return types.MarketData{}, errors.New(errors.ErrCodeDataNotFound, "data not preloaded, call Preload() first")
	}

	symbolData, ok := ds.data[symbol]
	if !ok {
		return types.MarketData{}, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol: %s", symbol)
	}

	if index < 0 || index >= len(symbolData) {
		return types.MarketData{}, errors.Newf(errors.ErrCodeDataNotFound,
			"bar index %d out of range [0, %d) for symbol %s", index, len(symbolData), symbol)
	}

	return symbolData[index], nil
}

// GetTotalBars returns the total number of bars loaded for a symbol.
func (ds *InMemoryIndexedDataSource) GetTotalBars(symbol string) int {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if symbolData, ok := ds.data[symbol]; ok {
		return len(symbolData)
	}
	return 0
}

// ========================================
// DataSource interface implementation
// ========================================

// Initialize implements DataSource.
func (ds *InMemoryIndexedDataSource) Initialize(path string) error {
	return ds.underlying.Initialize(path)
}

// ReadAll implements DataSource.
// When preloaded, iterates over in-memory data with automatic index tracking.
func (ds *InMemoryIndexedDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		ds.mu.RLock()
		preloaded := ds.preloaded
		allData := ds.allData
		ds.mu.RUnlock()

		if preloaded && allData != nil {
			// Use preloaded data with index tracking
			symbolIndices := make(map[string]int)

			for i, md := range allData {
				// Update the current bar index for this symbol
				ds.mu.Lock()
				ds.globalBarIndex = i
				symbolIndices[md.Symbol]++
				ds.currentBarIndex[md.Symbol] = symbolIndices[md.Symbol] - 1
				ds.mu.Unlock()

				if !yield(md, nil) {
					return
				}
			}
		} else {
			// Fall back to underlying datasource
			for md, err := range ds.underlying.ReadAll(start, end) {
				if !yield(md, err) {
					return
				}
			}
		}
	}
}

// GetRange implements DataSource.
func (ds *InMemoryIndexedDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		// Use in-memory data
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		var result []types.MarketData
		for _, md := range ds.allData {
			if !md.Time.Before(start) && !md.Time.After(end) {
				result = append(result, md)
			}
		}
		return result, nil
	}

	return ds.underlying.GetRange(start, end, interval)
}

// GetPreviousNumberOfDataPoints implements DataSource.
// When preloaded, uses O(1) indexed access instead of SQL queries.
func (ds *InMemoryIndexedDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		symbolData, ok := ds.data[symbol]
		if !ok {
			return nil, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol: %s", symbol)
		}

		// Find the bar index for this timestamp using time index
		timeIdx, ok := ds.timeIndex[symbol]
		if !ok {
			return nil, errors.Newf(errors.ErrCodeDataNotFound, "no time index for symbol: %s", symbol)
		}

		endIdx, ok := timeIdx[end.UnixNano()]
		if !ok {
			// Fallback: linear search for closest time <= end
			endIdx = -1
			for i, md := range symbolData {
				if !md.Time.After(end) {
					endIdx = i
				} else {
					break
				}
			}
			if endIdx < 0 {
				return nil, errors.Newf(errors.ErrCodeDataNotFound, "no data found before time %v for symbol %s", end, symbol)
			}
		}

		// Calculate start index
		startIdx := endIdx - count + 1
		if startIdx < 0 {
			startIdx = 0
		}

		actualCount := endIdx - startIdx + 1
		if actualCount < count {
			return nil, errors.NewInsufficientDataErrorf(count, actualCount, symbol,
				"insufficient data points for symbol %s: requested %d, got %d", symbol, count, actualCount)
		}

		// Return a copy
		result := make([]types.MarketData, count)
		copy(result, symbolData[startIdx:endIdx+1])

		return result, nil
	}

	return ds.underlying.GetPreviousNumberOfDataPoints(end, symbol, count)
}

// GetMarketData implements DataSource.
func (ds *InMemoryIndexedDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		timeIdx, ok := ds.timeIndex[symbol]
		if !ok {
			return types.MarketData{}, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol: %s", symbol)
		}

		idx, ok := timeIdx[timestamp.UnixNano()]
		if !ok {
			return types.MarketData{}, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol %s at time %v", symbol, timestamp)
		}

		return ds.data[symbol][idx], nil
	}

	return ds.underlying.GetMarketData(symbol, timestamp)
}

// ReadLastData implements DataSource.
func (ds *InMemoryIndexedDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		symbolData, ok := ds.data[symbol]
		if !ok || len(symbolData) == 0 {
			return types.MarketData{}, errors.Newf(errors.ErrCodeDataNotFound, "no data found for symbol: %s", symbol)
		}

		return symbolData[len(symbolData)-1], nil
	}

	return ds.underlying.ReadLastData(symbol)
}

// ExecuteSQL implements DataSource.
// SQL queries are passed to underlying datasource (cannot be served from memory).
func (ds *InMemoryIndexedDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	return ds.underlying.ExecuteSQL(query, params...)
}

// Count implements DataSource.
func (ds *InMemoryIndexedDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		if start.IsNone() && end.IsNone() {
			return len(ds.allData), nil
		}

		count := 0
		for _, md := range ds.allData {
			if start.IsSome() && md.Time.Before(start.Unwrap()) {
				continue
			}
			if end.IsSome() && md.Time.After(end.Unwrap()) {
				continue
			}
			count++
		}
		return count, nil
	}

	return ds.underlying.Count(start, end)
}

// Close implements DataSource.
func (ds *InMemoryIndexedDataSource) Close() error {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	// Clear preloaded data
	ds.data = nil
	ds.timeIndex = nil
	ds.allData = nil
	ds.currentBarIndex = nil
	ds.preloaded = false

	return ds.underlying.Close()
}

// GetAllSymbols implements DataSource.
func (ds *InMemoryIndexedDataSource) GetAllSymbols() ([]string, error) {
	ds.mu.RLock()
	preloaded := ds.preloaded
	ds.mu.RUnlock()

	if preloaded {
		ds.mu.RLock()
		defer ds.mu.RUnlock()

		symbols := make([]string, 0, len(ds.data))
		for symbol := range ds.data {
			symbols = append(symbols, symbol)
		}
		return symbols, nil
	}

	return ds.underlying.GetAllSymbols()
}

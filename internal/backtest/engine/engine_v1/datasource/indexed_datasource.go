package datasource

import (
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// IndexedDataSource extends DataSource with bar-index-based access methods.
// This interface enables O(1) lookups during backtesting by using array indices
// instead of time-based SQL queries, providing significant performance improvements.
type IndexedDataSource interface {
	DataSource

	// SetCurrentBarIndex sets the current bar index for subsequent queries.
	// This must be called before each bar is processed during backtesting.
	SetCurrentBarIndex(index int)

	// GetCurrentBarIndex returns the current bar index.
	GetCurrentBarIndex() int

	// GetPreviousNBars returns the previous N bars ending at the current bar index.
	// This is an O(1) operation using array slicing instead of SQL queries.
	// Returns data in chronological order (oldest to newest).
	GetPreviousNBars(symbol string, count int) ([]types.MarketData, error)

	// GetBarAtIndex returns the market data at a specific bar index.
	GetBarAtIndex(symbol string, index int) (types.MarketData, error)

	// GetTotalBars returns the total number of bars loaded for a symbol.
	GetTotalBars(symbol string) int

	// Preload loads all data into memory for fast indexed access.
	// This should be called once before backtesting begins.
	Preload(start optional.Option[time.Time], end optional.Option[time.Time]) error

	// IsPreloaded returns true if data has been preloaded into memory.
	IsPreloaded() bool
}

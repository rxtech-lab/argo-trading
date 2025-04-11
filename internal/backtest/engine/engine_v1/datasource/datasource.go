package datasource

import (
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/internal/types"
)

type Interval string

const (
	Interval1m  Interval = "1m"
	Interval5m  Interval = "5m"
	Interval15m Interval = "15m"
	Interval30m Interval = "30m"
	Interval1h  Interval = "1h"
	Interval4h  Interval = "4h"
	Interval6h  Interval = "6h"
	Interval8h  Interval = "8h"
	Interval12h Interval = "12h"
	Interval1d  Interval = "1d"
	Interval1w  Interval = "1w"
	Interval1M  Interval = "1M"
)

// SQLResult represents a row of data from a SQL query
type SQLResult struct {
	Values map[string]interface{}
}

type DataSource interface {
	// Initialize initializes the data source with the given data path in parquet format
	Initialize(path string) error
	// ReadAll reads all the data from the data source and yields it to the caller
	ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool)
	// GetRange reads a range of data from the data source and yields it to the caller
	GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error)
	// ReadLastData reads the last data from the data source for a specific symbol
	ReadLastData(symbol string) (types.MarketData, error)
	// ExecuteSQL executes a raw SQL query and returns the results as SQLResult
	ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error)
	// Count returns the number of rows in the data source
	Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error)
	// Close closes the data source and releases any resources
	Close() error
}

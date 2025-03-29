package datasource

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
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

type DataSource interface {
	// Initialize initializes the data source with the given data path in parquet format
	Initialize(path string) error
	// ReadAll reads all the data from the data source and yields it to the caller
	ReadAll(yield func(types.MarketData, error) bool)

	// ReadRange reads a range of data from the data source and yields it to the caller
	ReadRange(start time.Time, end time.Time, interval Interval) ([]types.MarketData, error)
}

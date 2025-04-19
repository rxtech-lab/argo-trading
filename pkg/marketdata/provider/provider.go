package provider

import (
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

type Provider interface {
	// ConfigWriter configures the writer for the provider
	// Writer is used to write the market data to the database.
	// It could be a file, a database, etc.
	ConfigWriter(writer writer.MarketDataWriter)
	// Download downloads the data for the given ticker and date range
	// example:
	// Download("AAPL", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC), 1, models.TimespanMinute)
	Download(ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan) (path string, err error)
}

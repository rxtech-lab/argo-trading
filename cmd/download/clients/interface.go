package clients

import "time"

type Downloader interface {
	// Download downloads the data for the given ticker and date range
	// example:
	// Download("AAPL", "data/AAPL.csv", time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC), time.Date(2020, 1, 31, 0, 0, 0, 0, time.UTC))
	Download(ticker string, toPath string, startDate time.Time, endDate time.Time) (path string, err error)
}

type TradingData struct {
	Time   time.Time `csv:"time"`
	Open   float64   `csv:"open"`
	High   float64   `csv:"high"`
	Low    float64   `csv:"low"`
	Close  float64   `csv:"close"`
	Volume float64   `csv:"volume"`
}

package types

import "time"

type MarketDataSource interface {
	// Iterator iterates over the market data row by row
	Iterator(startTime, endTime time.Time) func(yield func(MarketData) bool)
	// GetDataForTimeRange returns market data within a specific time range
	GetDataForTimeRange(startTime, endTime time.Time) []MarketData
}

type MarketData struct {
	Time   time.Time `csv:"time"`
	Open   float64   `csv:"open"`
	High   float64   `csv:"high"`
	Low    float64   `csv:"low"`
	Close  float64   `csv:"close"`
	Volume float64   `csv:"volume"`
}

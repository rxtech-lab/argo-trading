package types

import "time"

type MarketData struct {
	Id     string    `csv:"id"`
	Symbol string    `csv:"symbol"`
	Time   time.Time `csv:"time"`
	Open   float64   `csv:"open"`
	High   float64   `csv:"high"`
	Low    float64   `csv:"low"`
	Close  float64   `csv:"close"`
	Volume float64   `csv:"volume"`
}

package types

import "time"

type Trade struct {
	Order         Order
	ExecutedAt    time.Time
	ExecutedQty   float64
	ExecutedPrice float64
	Commission    float64
	PnL           float64 // Profit and Loss for this trade
}

// Position represents current holdings of an asset
type Position struct {
	Symbol        string
	Quantity      float64
	AveragePrice  float64
	OpenTimestamp time.Time
}

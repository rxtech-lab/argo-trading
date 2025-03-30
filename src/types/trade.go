package types

import "time"

type Trade struct {
	Order         Order     `csv:"order"`
	ExecutedAt    time.Time `csv:"executed_at"`
	ExecutedQty   float64   `csv:"executed_qty"`
	ExecutedPrice float64   `csv:"executed_price"`
	Commission    float64   `csv:"commission"`
	PnL           float64   `csv:"pnl"` // Profit and Loss for this trade
}

// Position represents current holdings of an asset
type Position struct {
	Symbol        string    `csv:"symbol"`
	Quantity      float64   `csv:"quantity"`
	AveragePrice  float64   `csv:"average_price"`
	OpenTimestamp time.Time `csv:"open_timestamp"`
	StrategyName  string    `csv:"strategy_name"`
}

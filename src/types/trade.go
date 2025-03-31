package types

import (
	"time"

	"github.com/shopspring/decimal"
)

type Trade struct {
	Order         Order     `csv:"order"`
	ExecutedAt    time.Time `csv:"executed_at"`
	ExecutedQty   float64   `csv:"executed_qty"`
	ExecutedPrice float64   `csv:"executed_price"`
	// Fee is the fee for this trade
	Fee float64 `csv:"fee"`
	// PnL is the profit and loss for this trade
	// For example, you have 300 shares of AAPL at $100.01 average entry price
	// And you sell 100 shares at $110.0
	// Then the PnL is (110.0-100.01)*100 = $999.
	// Note, the fee is included in the average entry price and pnl would be 0 if the order is buy.
	PnL float64 `csv:"pnl"`
}

// Position represents current holdings of an asset
type Position struct {
	Symbol           string    `csv:"symbol"`
	Quantity         float64   `csv:"quantity"`
	TotalInQuantity  float64   `csv:"total_in_quantity"`
	TotalOutQuantity float64   `csv:"total_out_quantity"`
	TotalInAmount    float64   `csv:"total_in_amount"`
	TotalOutAmount   float64   `csv:"total_out_amount"`
	TotalInFee       float64   `csv:"total_in_fee"`
	TotalOutFee      float64   `csv:"total_out_fee"`
	OpenTimestamp    time.Time `csv:"open_timestamp"`
	StrategyName     string    `csv:"strategy_name"`
}

// GetAverageEntryPrice calculates the average entry price including fees
func (p *Position) GetAverageEntryPrice() float64 {
	if p.TotalInQuantity == 0 {
		return 0
	}
	return (p.TotalInAmount + p.TotalInFee) / p.TotalInQuantity
}

// GetAverageExitPrice calculates the average exit price including fees
func (p *Position) GetAverageExitPrice() float64 {
	if p.TotalOutQuantity == 0 {
		return 0
	}
	return (p.TotalOutAmount - p.TotalOutFee) / p.TotalOutQuantity
}

func (p *Position) GetTotalPnL() float64 {
	if p.TotalInQuantity == 0 {
		return 0
	}

	if p.TotalOutQuantity == 0 {
		return 0
	}

	entryDec := decimal.NewFromFloat(p.TotalOutQuantity).Mul(decimal.NewFromFloat(p.GetAverageEntryPrice()))
	exitDec := decimal.NewFromFloat(p.TotalOutQuantity).Mul(decimal.NewFromFloat(p.GetAverageExitPrice()))
	resultDec := exitDec.Sub(entryDec)
	result, _ := resultDec.Float64()
	return result
}

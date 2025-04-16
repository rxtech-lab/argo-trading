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

// Position represents current holdings of an asset.
type Position struct {
	Symbol                     string  `csv:"symbol"`
	TotalLongPositionQuantity  float64 `csv:"long_position_quantity"`
	TotalShortPositionQuantity float64 `csv:"short_position_quantity"`

	TotalLongInPositionQuantity  float64 `csv:"total_in_long_position_quantity"`
	TotalLongOutPositionQuantity float64 `csv:"total_out_long_position_quantity"`
	TotalLongInPositionAmount    float64 `csv:"total_in_long_position_amount"`
	TotalLongOutPositionAmount   float64 `csv:"total_out_long_position_amount"`

	TotalShortInPositionQuantity  float64 `csv:"total_in_short_position_quantity"`
	TotalShortOutPositionQuantity float64 `csv:"total_out_short_position_quantity"`
	TotalShortInPositionAmount    float64 `csv:"total_in_short_position_amount"`
	TotalShortOutPositionAmount   float64 `csv:"total_out_short_position_amount"`

	TotalLongInFee   float64 `csv:"total_long_in_fee"`
	TotalLongOutFee  float64 `csv:"total_long_out_fee"`
	TotalShortInFee  float64 `csv:"total_short_in_fee"`
	TotalShortOutFee float64 `csv:"total_short_out_fee"`

	OpenTimestamp time.Time `csv:"open_timestamp"`
	StrategyName  string    `csv:"strategy_name"`
}

// GetAverageLongPositionEntryPrice calculates the average entry price including fees.
func (p *Position) GetAverageLongPositionEntryPrice() float64 {
	if p.TotalLongInPositionQuantity == 0 {
		return 0
	}

	return (p.TotalLongInPositionAmount + p.TotalLongInFee) / p.TotalLongInPositionQuantity
}

func (p *Position) GetAverageShortPositionEntryPrice() float64 {
	if p.TotalShortInPositionQuantity == 0 {
		return 0
	}

	return (p.TotalShortInPositionAmount - p.TotalShortInFee) / p.TotalShortInPositionQuantity
}

// GetAverageLongPositionExitPrice calculates the average exit price including fees.
func (p *Position) GetAverageLongPositionExitPrice() float64 {
	if p.TotalLongOutPositionQuantity == 0 {
		return 0
	}

	return (p.TotalLongOutPositionAmount - p.TotalLongOutFee) / p.TotalLongOutPositionQuantity
}

// GetAverageShortPositionExitPrice calculates the average exit price including fees.
func (p *Position) GetAverageShortPositionExitPrice() float64 {
	if p.TotalShortOutPositionQuantity == 0 {
		return 0
	}

	return (p.TotalShortOutPositionAmount + p.TotalShortOutFee) / p.TotalShortOutPositionQuantity
}

// GetTotalShortPositionPnl calculates the total pnl for short position.
func (p *Position) GetTotalShortPositionPnl() decimal.Decimal {
	if p.TotalShortInPositionQuantity == 0 {
		return decimal.Zero
	}

	if p.TotalShortOutPositionQuantity == 0 {
		return decimal.Zero
	}

	shortEntryDec := decimal.NewFromFloat(p.TotalShortOutPositionQuantity).Mul(decimal.NewFromFloat(p.GetAverageShortPositionEntryPrice()))
	shortExitDec := decimal.NewFromFloat(p.TotalShortOutPositionQuantity).Mul(decimal.NewFromFloat(p.GetAverageShortPositionExitPrice()))
	// the way we calculate short pnl is the opposite of long pnl
	// for example, if the exit price is higher than the entry price, the pnl is negative
	shortResultDec := shortEntryDec.Sub(shortExitDec)

	return shortResultDec
}

func (p *Position) GetTotalLongPositionPnl() decimal.Decimal {
	if p.TotalLongInPositionQuantity == 0 {
		return decimal.Zero
	}

	if p.TotalLongOutPositionQuantity == 0 {
		return decimal.Zero
	}

	longEntryDec := decimal.NewFromFloat(p.TotalLongOutPositionQuantity).Mul(decimal.NewFromFloat(p.GetAverageLongPositionEntryPrice()))
	longExitDec := decimal.NewFromFloat(p.TotalLongOutPositionQuantity).Mul(decimal.NewFromFloat(p.GetAverageLongPositionExitPrice()))
	longResultDec := longExitDec.Sub(longEntryDec)

	return longResultDec
}

func (p *Position) GetTotalPnL() float64 {
	longResult := p.GetTotalLongPositionPnl()
	shortResult := p.GetTotalShortPositionPnl()

	result, _ := longResult.Add(shortResult).Float64()

	return result
}

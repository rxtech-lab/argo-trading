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
	// PnL is the individual profit and loss for this trade using FIFO matching.
	// For sell orders, it matches against the earliest unmatched buy orders to calculate
	// the individual PnL for this specific trade. For buy orders, PnL is 0.
	PnL float64 `csv:"pnl"`
	// CumulativePnL is the running sum of per-trade PnL for this symbol up to and
	// including this trade. For example, if three sells produce PnL +8, +8, -2,
	// the CumulativePnL values are 8, 16, 14. Buys contribute 0 to the sum and
	// inherit the prior cumulative value unchanged.
	CumulativePnL float64 `csv:"cumulative_pnl"`
	// OpenPositionQty is the open position quantity after this trade.
	// For long positions, it is the net long quantity. For short positions, it is the net short quantity.
	OpenPositionQty float64 `csv:"open_position_qty"`
	// Balance is the cash balance after this trade.
	// Calculated as: initialCapital - Σ(buy_cost + buy_fee) + Σ(sell_proceeds - sell_fee).
	Balance float64 `csv:"balance"`
	// HoldTime is the holding time in seconds for a closing trade, calculated using
	// FIFO matching against the prior unmatched entry trades. For long positions it
	// is computed on the sell trade; for short positions, on the covering buy trade.
	// It is the quantity-weighted average duration (in seconds) between the closing
	// trade and each matched entry trade. For entry (opening) trades, HoldTime is 0.
	HoldTime int `csv:"hold_time"`
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

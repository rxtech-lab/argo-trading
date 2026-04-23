package commission_fee

// BinanceSpotRegularFeeRate is the standard Binance Spot maker/taker fee rate
// for a Regular User (VIP 0) without the BNB fee discount, charged as a
// fraction of the trade's notional value (price * quantity).
//
// Reference: https://www.binance.com/en/fee/trading
const BinanceSpotRegularFeeRate = 0.001 // 0.1%

// BinanceCommissionFee implements CommissionFee for Binance Spot trading.
// Binance charges a percentage of the notional value (price * quantity)
// rather than a per-share fee.
type BinanceCommissionFee struct {
	// FeeRate is the fee rate applied to the notional value, expressed as a
	// decimal fraction (e.g. 0.001 for 0.1%).
	FeeRate float64
}

// NewBinanceCommissionFee creates a BinanceCommissionFee using the standard
// Regular User Spot fee rate (0.1% of notional).
func NewBinanceCommissionFee() CommissionFee {
	return &BinanceCommissionFee{FeeRate: BinanceSpotRegularFeeRate}
}

// NewBinanceCommissionFeeWithRate creates a BinanceCommissionFee with a
// custom fee rate, useful for modelling VIP tiers or the BNB fee discount.
// Negative rates are clamped to zero.
func NewBinanceCommissionFeeWithRate(rate float64) CommissionFee {
	if rate < 0 {
		rate = 0
	}

	return &BinanceCommissionFee{FeeRate: rate}
}

// Calculate returns the Binance trading fee in USD for the given quantity
// and per-unit price. The fee is computed as |quantity| * |price| * FeeRate
// so that it remains non-negative regardless of the order side representation.
func (c *BinanceCommissionFee) Calculate(quantity float64, price float64) float64 {
	if quantity < 0 {
		quantity = -quantity
	}

	if price < 0 {
		price = -price
	}

	return quantity * price * c.FeeRate
}

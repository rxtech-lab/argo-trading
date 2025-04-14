package commission_fee

// ZeroCommissionFee implements CommissionFee interface with zero commission.
type ZeroCommissionFee struct{}

// NewZeroCommissionFee creates a new zero commission fee.
func NewZeroCommissionFee() CommissionFee {
	return &ZeroCommissionFee{}
}

// Calculate returns 0 for any quantity.
func (c *ZeroCommissionFee) Calculate(quantity float64) float64 {
	return 0.0
}

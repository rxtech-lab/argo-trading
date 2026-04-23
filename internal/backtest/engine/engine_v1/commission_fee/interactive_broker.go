package commission_fee

type InteractiveBrokerCommissionFee struct {
}

func NewInteractiveBrokerCommissionFee() CommissionFee {
	return &InteractiveBrokerCommissionFee{}
}

func (c *InteractiveBrokerCommissionFee) Calculate(quantity float64, price float64) float64 {
	_ = price
	fee := 0.005 * quantity
	if fee < 1.0 {
		return 1.0
	}

	return fee
}

package commission_fee

type CommissionFee interface {
	// Calculate the commission fee for a given quantity and returns the fee in USD
	Calculate(quantity float64) float64
}

type Broker string

const (
	BrokerInteractiveBroker Broker = "interactive_broker"
	BrokerZero             Broker = "zero_commission"
)

var AllBrokers = []any{
	BrokerInteractiveBroker,
	BrokerZero,
}

func GetCommissionFeeHandler(broker Broker) CommissionFee {
	switch broker {
	case BrokerInteractiveBroker:
		return NewInteractiveBrokerCommissionFee()
	case BrokerZero:
		return NewZeroCommissionFee()
	default:
		return NewZeroCommissionFee()
	}
}

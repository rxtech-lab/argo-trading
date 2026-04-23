package commission_fee

type CommissionFee interface {
	// Calculate the commission fee for a given quantity and price (per unit)
	// and returns the fee in USD. Implementations that depend only on
	// quantity (e.g. per-share brokers) may ignore the price argument, while
	// implementations that depend on notional value (e.g. crypto exchanges
	// such as Binance) use both arguments.
	Calculate(quantity float64, price float64) float64
}

type Broker string

const (
	BrokerInteractiveBroker Broker = "interactive_broker"
	BrokerZero              Broker = "zero_commission"
	BrokerBinance           Broker = "binance"
)

var AllBrokers = []any{
	BrokerInteractiveBroker,
	BrokerZero,
	BrokerBinance,
}

func GetCommissionFeeHandler(broker Broker) CommissionFee {
	switch broker {
	case BrokerInteractiveBroker:
		return NewInteractiveBrokerCommissionFee()
	case BrokerZero:
		return NewZeroCommissionFee()
	case BrokerBinance:
		return NewBinanceCommissionFee()
	default:
		return NewZeroCommissionFee()
	}
}

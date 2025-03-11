package indicator

type Indicator interface {
	Calculate(data []MarketData) (interface{}, error)
}

package indicator

import (
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/types"
)

type IndicatorContext struct {
	DataSource datasource.DataSource
}

// Indicator interface defines methods that any technical indicator must implement
type Indicator interface {
	// BuySignal returns a signal to buy
	GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error)
	// Name returns the name of the indicator
	Name() types.Indicator
}

package indicator

import (
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

type IndicatorContext struct {
	DataSource        datasource.DataSource
	IndicatorRegistry IndicatorRegistry
	Cache             cache.Cache
}

// Indicator interface defines methods that any technical indicator must implement
type Indicator interface {
	// BuySignal returns a signal to buy
	GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error)
	// Name returns the name of the indicator
	Name() types.IndicatorType
	// RawValue returns the raw value of the indicator
	RawValue(params ...any) (float64, error)
	Config(params ...any) error
}

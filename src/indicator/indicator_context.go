package indicator

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// marketDataIndicatorContext implements indicator.IndicatorContext using MarketDataSource
type marketDataIndicatorContext struct {
	source types.MarketDataSource
}

// GetDataForTimeRange returns market data within a specific time range
func (ctx *marketDataIndicatorContext) GetDataForTimeRange(startTime, endTime time.Time) []types.MarketData {
	return ctx.source.GetDataForTimeRange(startTime, endTime)
}

// CreateIndicatorContext creates an IndicatorContext from a MarketDataSource
func CreateIndicatorContext(source types.MarketDataSource) IndicatorContext {
	return &marketDataIndicatorContext{
		source: source,
	}
}

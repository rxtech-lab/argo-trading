package indicator

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// IndicatorContext provides data access methods for indicators
type IndicatorContext interface {
	// GetDataForTimeRange returns market data within a specific time range
	GetDataForTimeRange(startTime, endTime time.Time) []types.MarketData
}

// Indicator interface defines methods that any technical indicator must implement
type Indicator interface {
	// Calculate computes the indicator value using the provided context
	Calculate(ctx IndicatorContext) (interface{}, error)

	// Name returns the name of the indicator
	Name() types.Indicator

	// SetParams allows setting parameters for the indicator
	SetParams(params map[string]interface{}) error

	// GetParams returns the current parameters of the indicator
	GetParams() map[string]interface{}
}

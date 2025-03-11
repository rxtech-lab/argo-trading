package indicator

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// IndicatorContext provides data access methods for indicators
type IndicatorContext interface {
	// GetData returns all available market data
	GetData() []types.MarketData

	// GetDataForTimeRange returns market data within a specific time range
	GetDataForTimeRange(startTime, endTime time.Time) []types.MarketData
}

// Indicator interface defines methods that any technical indicator must implement
type Indicator interface {
	// Calculate computes the indicator value using the provided context
	Calculate(ctx IndicatorContext) (interface{}, error)

	// Name returns the name of the indicator
	Name() string

	// SetParams allows setting parameters for the indicator
	SetParams(params map[string]interface{}) error

	// GetParams returns the current parameters of the indicator
	GetParams() map[string]interface{}
}

// DefaultIndicatorContext is a default implementation of IndicatorContext
type DefaultIndicatorContext struct {
	data []types.MarketData
}

// NewIndicatorContext creates a new indicator context with the provided data
func NewIndicatorContext(data []types.MarketData) IndicatorContext {
	return &DefaultIndicatorContext{
		data: data,
	}
}

// GetData returns all available market data
func (c *DefaultIndicatorContext) GetData() []types.MarketData {
	return c.data
}

// GetDataForTimeRange returns market data within a specific time range
func (c *DefaultIndicatorContext) GetDataForTimeRange(startTime, endTime time.Time) []types.MarketData {
	var result []types.MarketData

	for _, d := range c.data {
		if (d.Time.Equal(startTime) || d.Time.After(startTime)) &&
			(d.Time.Equal(endTime) || d.Time.Before(endTime)) {
			result = append(result, d)
		}
	}

	return result
}

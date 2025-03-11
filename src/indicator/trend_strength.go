package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// TrendStrength indicator implementation
// This indicator combines price action with moving averages to determine trend strength
type TrendStrength struct {
	shortPeriod int
	longPeriod  int
	params      map[string]interface{}
	startTime   time.Time
	endTime     time.Time
}

// NewTrendStrength creates a new TrendStrength indicator
func NewTrendStrength(startTime, endTime time.Time, shortPeriod, longPeriod int) Indicator {
	return &TrendStrength{
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		params: map[string]interface{}{
			"shortPeriod": shortPeriod,
			"longPeriod":  longPeriod,
		},
		startTime: startTime,
		endTime:   endTime,
	}
}

// Name returns the name of the indicator
func (t *TrendStrength) Name() types.Indicator {
	return types.IndicatorTrendStrength
}

// SetParams allows setting parameters for the indicator
func (t *TrendStrength) SetParams(params map[string]interface{}) error {
	if shortPeriod, ok := params["shortPeriod"]; ok {
		if p, ok := shortPeriod.(int); ok {
			t.shortPeriod = p
			t.params["shortPeriod"] = p
		} else {
			return fmt.Errorf("shortPeriod must be an integer")
		}
	}

	if longPeriod, ok := params["longPeriod"]; ok {
		if p, ok := longPeriod.(int); ok {
			t.longPeriod = p
			t.params["longPeriod"] = p
		} else {
			return fmt.Errorf("longPeriod must be an integer")
		}
	}

	return nil
}

// GetParams returns the current parameters of the indicator
func (t *TrendStrength) GetParams() map[string]interface{} {
	return t.params
}

// Calculate computes the trend strength value
func (t *TrendStrength) Calculate(ctx IndicatorContext) (interface{}, error) {
	data := ctx.GetDataForTimeRange(t.startTime, t.endTime)

	if len(data) < t.longPeriod {
		return nil, fmt.Errorf("not enough data points for trend strength calculation, need at least %d", t.longPeriod)
	}

	// Calculate short and long moving averages
	shortMA := calculateSMA(data, t.shortPeriod)
	longMA := calculateSMA(data, t.longPeriod)

	// Calculate trend strength values
	trendStrengthValues := make([]float64, len(shortMA))

	for i := 0; i < len(shortMA); i++ {
		// Skip if we don't have corresponding long MA value
		if i >= len(longMA) {
			break
		}

		// Calculate trend strength based on:
		// 1. Difference between short and long MA (direction and magnitude)
		// 2. Rate of change of short MA (momentum)
		// 3. Price position relative to both MAs

		// 1. MA difference component (normalized)
		maDiff := (shortMA[i] - longMA[i]) / longMA[i] * 100

		// 2. Short MA momentum (rate of change)
		var momentum float64
		if i > 0 {
			momentum = (shortMA[i] - shortMA[i-1]) / shortMA[i-1] * 100
		}

		// 3. Price position relative to MAs
		priceIndex := i + t.shortPeriod - 1
		if priceIndex >= len(data) {
			priceIndex = len(data) - 1
		}

		priceToShortMA := (data[priceIndex].Close - shortMA[i]) / shortMA[i] * 100
		priceToLongMA := (data[priceIndex].Close - longMA[i]) / longMA[i] * 100

		// Combine components with weights
		trendStrength := (maDiff * 0.4) + (momentum * 0.3) + (priceToShortMA * 0.15) + (priceToLongMA * 0.15)

		// Scale to a -100 to +100 range
		trendStrength = math.Max(-100, math.Min(100, trendStrength*5))

		trendStrengthValues[i] = trendStrength
	}

	return trendStrengthValues, nil
}

// Helper function to calculate Simple Moving Average
func calculateSMA(data []types.MarketData, period int) []float64 {
	if len(data) < period {
		return []float64{}
	}

	smaValues := make([]float64, len(data)-period+1)

	// Calculate first SMA
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += data[i].Close
	}
	smaValues[0] = sum / float64(period)

	// Calculate remaining SMAs using previous sum
	for i := period; i < len(data); i++ {
		sum = sum - data[i-period].Close + data[i].Close
		smaValues[i-period+1] = sum / float64(period)
	}

	return smaValues
}

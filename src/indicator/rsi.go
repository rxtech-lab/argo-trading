package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// RSI (Relative Strength Index) indicator implementation
type RSI struct {
	period    int
	params    map[string]interface{}
	startTime time.Time
	endTime   time.Time
}

// NewRSI creates a new RSI indicator with the specified period
func NewRSI(startTime, endTime time.Time, period int) Indicator {
	return &RSI{
		period: period,
		params: map[string]interface{}{
			"period": period,
		},
		startTime: startTime,
		endTime:   endTime,
	}
}

// Name returns the name of the indicator
func (r *RSI) Name() types.Indicator {
	return types.IndicatorRSI
}

// SetParams allows setting parameters for the indicator
func (r *RSI) SetParams(params map[string]interface{}) error {
	if period, ok := params["period"]; ok {
		if p, ok := period.(int); ok {
			r.period = p
			r.params["period"] = p
			return nil
		}
		return fmt.Errorf("period must be an integer")
	}
	return fmt.Errorf("period parameter is required")
}

// GetParams returns the current parameters of the indicator
func (r *RSI) GetParams() map[string]interface{} {
	return r.params
}

// Calculate computes the RSI value using the provided context
func (r *RSI) Calculate(ctx IndicatorContext) (interface{}, error) {
	data := ctx.GetDataForTimeRange(r.startTime, r.endTime)

	if len(data) < r.period+1 {
		return nil, fmt.Errorf("not enough data points for RSI calculation, need at least %d", r.period+1)
	}

	// Calculate price changes
	changes := make([]float64, len(data)-1)
	for i := 1; i < len(data); i++ {
		changes[i-1] = data[i].Close - data[i-1].Close
	}

	// Calculate gains and losses
	gains := make([]float64, len(changes))
	losses := make([]float64, len(changes))
	for i, change := range changes {
		if change > 0 {
			gains[i] = change
		} else {
			losses[i] = math.Abs(change)
		}
	}

	// Calculate average gains and losses
	var avgGain, avgLoss float64
	for i := 0; i < r.period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}
	avgGain /= float64(r.period)
	avgLoss /= float64(r.period)

	// Calculate RSI values
	rsiValues := make([]float64, len(data)-r.period)

	// First RSI value
	rs := avgGain / math.Max(avgLoss, 0.0001) // Avoid division by zero
	rsiValues[0] = 100 - (100 / (1 + rs))

	// Calculate remaining RSI values using smoothed method
	for i := r.period; i < len(changes); i++ {
		avgGain = ((avgGain * float64(r.period-1)) + gains[i-1]) / float64(r.period)
		avgLoss = ((avgLoss * float64(r.period-1)) + losses[i-1]) / float64(r.period)

		rs = avgGain / math.Max(avgLoss, 0.0001) // Avoid division by zero
		rsiValues[i-r.period] = 100 - (100 / (1 + rs))
	}

	return rsiValues, nil
}

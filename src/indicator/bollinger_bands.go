package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/types"
)

type InsufficientDataError struct {
	Message string
}

func (e *InsufficientDataError) Error() string {
	return e.Message
}

// BollingerBands implements the Indicator interface for Bollinger Bands
type BollingerBands struct {
	period   int     // Number of periods for moving average
	stdDev   float64 // Number of standard deviations
	lookback time.Duration
}

// NewBollingerBands creates a new Bollinger Bands indicator
func NewBollingerBands(period int, stdDev float64, lookback time.Duration) *BollingerBands {
	return &BollingerBands{
		period:   period,
		stdDev:   stdDev,
		lookback: lookback,
	}
}

// Name returns the name of the indicator
func (bb *BollingerBands) Name() types.Indicator {
	return types.IndicatorBollingerBands
}

// GetSignal generates trading signals based on Bollinger Bands
func (bb *BollingerBands) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Get historical data for the lookback period
	startTime := marketData.Time.Add(-bb.lookback)
	endTime := marketData.Time

	historicalData, err := ctx.DataSource.ReadRange(startTime, endTime, datasource.Interval1m)
	if err != nil {
		return types.Signal{}, err
	}

	// Calculate Bollinger Bands
	upper, middle, lower, err := bb.calculateBands(historicalData)
	if err != nil {
		// if the error is insufficient data, return a no action signal
		if _, ok := err.(*InsufficientDataError); ok {
			return types.Signal{
				Time: marketData.Time,
				Type: types.SignalTypeNoAction,
				Name: "Bollinger Bands",
			}, nil
		}
		return types.Signal{}, err
	}

	// Generate signals based on price position relative to bands
	currentPrice := marketData.Close

	// Buy signal when price crosses below lower band
	if currentPrice < lower {
		return types.Signal{
			Time:     marketData.Time,
			Type:     types.SignalTypeBuy,
			Name:     "Bollinger Bands",
			Reason:   "Price below lower band",
			RawValue: map[string]float64{"upper": upper, "middle": middle, "lower": lower},
		}, nil
	}

	// Sell signal when price crosses above upper band
	if currentPrice > upper {
		return types.Signal{
			Time:     marketData.Time,
			Type:     types.SignalTypeSell,
			Name:     "Bollinger Bands",
			Reason:   "Price above upper band",
			RawValue: map[string]float64{"upper": upper, "middle": middle, "lower": lower},
		}, nil
	}

	// No action signal
	return types.Signal{
		Time:     marketData.Time,
		Type:     types.SignalTypeNoAction,
		Name:     "Bollinger Bands",
		Reason:   "Price within bands",
		RawValue: map[string]float64{"upper": upper, "middle": middle, "lower": lower},
	}, nil
}

// calculateBands calculates the Bollinger Bands values
func (bb *BollingerBands) calculateBands(data []types.MarketData) (upper, middle, lower float64, err error) {
	if len(data) < bb.period {
		return 0, 0, 0, &InsufficientDataError{
			Message: fmt.Sprintf("insufficient data points for period %d", bb.period),
		}
	}

	// Calculate SMA (middle band)
	var sum float64
	for i := len(data) - bb.period; i < len(data); i++ {
		sum += data[i].Close
	}
	middle = sum / float64(bb.period)

	// Calculate standard deviation
	var squaredDiffSum float64
	for i := len(data) - bb.period; i < len(data); i++ {
		diff := data[i].Close - middle
		squaredDiffSum += diff * diff
	}
	stdDev := math.Sqrt(squaredDiffSum / float64(bb.period))

	// Calculate upper and lower bands
	upper = middle + (bb.stdDev * stdDev)
	lower = middle - (bb.stdDev * stdDev)

	return upper, middle, lower, nil
}

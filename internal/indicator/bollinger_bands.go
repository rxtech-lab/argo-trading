package indicator

import (
	"math"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// BollingerBands implements the Indicator interface for Bollinger Bands.
type BollingerBands struct {
	period   int     // Number of periods for moving average
	stdDev   float64 // Number of standard deviations
	lookback time.Duration
}

// NewBollingerBands creates a new Bollinger Bands indicator with default configuration.
func NewBollingerBands() Indicator {
	return &BollingerBands{
		period:   20,             // Default period
		stdDev:   2.0,            // Default standard deviation
		lookback: time.Hour * 24, // Default lookback period
	}
}

// Name returns the name of the indicator.
func (bb *BollingerBands) Name() types.IndicatorType {
	return types.IndicatorTypeBollingerBands
}

// Config configures the Bollinger Bands indicator. Expected parameters: period (int), stdDev (float64), lookback (time.Duration).
func (bb *BollingerBands) Config(params ...any) error {
	if len(params) != 3 {
		return errors.New(errors.ErrCodeMissingParameter, "Config expects 3 parameters: period (int), stdDev (float64), lookback (time.Duration)")
	}

	period, ok := params[0].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for period parameter, expected int")
	}

	if period <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "period must be a positive integer, got %d", period)
	}

	stdDev, ok := params[1].(float64)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for stdDev parameter, expected float64")
	}

	if stdDev <= 0 {
		return errors.Newf(errors.ErrCodeInvalidStdDevPeriod, "stdDev must be a positive number, got %f", stdDev)
	}

	lookback, ok := params[2].(time.Duration)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for lookback parameter, expected time.Duration")
	}

	if lookback <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "lookback must be a positive duration, got %v", lookback)
	}

	bb.period = period
	bb.stdDev = stdDev
	bb.lookback = lookback

	return nil
}

// RawValue implements Indicator. It calculates the middle Bollinger Band (SMA)
// for the given market data point.
// It expects types.MarketData as the first parameter and IndicatorContext as the second.
func (bb *BollingerBands) RawValue(params ...any) (float64, error) {
	// thinking process:
	// 1. Validate and extract parameters. RawValue needs market data and context.
	// 2. Fetch historical data required for calculation.
	// 3. Calculate the Bollinger Bands using the existing helper function.
	// 4. Return the middle band value.
	if len(params) < 2 {
		return 0, errors.New(errors.ErrCodeMissingParameter, "RawValue requires at least 2 parameters: types.MarketData and IndicatorContext")
	}

	marketData, ok := params[0].(types.MarketData)
	if !ok {
		return 0, errors.New(errors.ErrCodeInvalidType, "first parameter must be of type types.MarketData")
	}

	ctx, ok := params[1].(IndicatorContext)
	if !ok {
		return 0, errors.New(errors.ErrCodeInvalidType, "second parameter must be of type IndicatorContext")
	}

	// Get historical data for the lookback period
	startTime := marketData.Time.Add(-bb.lookback)
	endTime := marketData.Time

	historicalData, err := ctx.DataSource.GetRange(startTime, endTime, optional.None[datasource.Interval]())
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeHistoricalDataFailed, "failed to get historical data", err)
	}

	// Calculate Bollinger Bands
	_, middle, _, err := bb.calculateBands(historicalData)
	if err != nil {
		// Return 0 and the error (e.g., InsufficientDataError)
		return 0, err
	}

	// Return the middle band value
	return middle, nil
}

// GetSignal generates trading signals based on Bollinger Bands.
func (bb *BollingerBands) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Get historical data for the lookback period
	startTime := marketData.Time.Add(-bb.lookback)
	endTime := marketData.Time

	historicalData, err := ctx.DataSource.GetRange(startTime, endTime, optional.None[datasource.Interval]())
	if err != nil {
		return types.Signal{}, err
	}

	// Calculate Bollinger Bands
	upper, middle, lower, err := bb.calculateBands(historicalData)
	if err != nil {
		// if the error is insufficient data, return a no action signal
		if errors.IsInsufficientDataError(err) {
			return types.Signal{
				Time:      marketData.Time,
				Type:      types.SignalTypeNoAction,
				Name:      "Bollinger Bands",
				Reason:    "",
				RawValue:  nil,
				Symbol:    marketData.Symbol,
				Indicator: bb.Name(),
			}, nil
		}

		return types.Signal{}, err
	}

	// Generate signals based on price position relative to bands
	currentPrice := marketData.Close

	// Buy signal when price crosses below lower band
	if currentPrice < lower {
		return types.Signal{
			Time:      marketData.Time,
			Type:      types.SignalTypeBuyLong,
			Name:      "Bollinger Bands",
			Reason:    "Price below lower band",
			RawValue:  map[string]float64{"upper": upper, "middle": middle, "lower": lower},
			Symbol:    marketData.Symbol,
			Indicator: bb.Name(),
		}, nil
	}

	// Sell signal when price crosses above upper band
	if currentPrice > upper {
		return types.Signal{
			Time:      marketData.Time,
			Type:      types.SignalTypeSellLong,
			Name:      "Bollinger Bands",
			Reason:    "Price above upper band",
			RawValue:  map[string]float64{"upper": upper, "middle": middle, "lower": lower},
			Symbol:    marketData.Symbol,
			Indicator: bb.Name(),
		}, nil
	}

	// No action signal
	return types.Signal{
		Time:      marketData.Time,
		Type:      types.SignalTypeNoAction,
		Name:      "Bollinger Bands",
		Reason:    "Price within bands",
		RawValue:  map[string]float64{"upper": upper, "middle": middle, "lower": lower},
		Symbol:    marketData.Symbol,
		Indicator: bb.Name(),
	}, nil
}

// calculateBands calculates the Bollinger Bands values.
func (bb *BollingerBands) calculateBands(data []types.MarketData) (upper, middle, lower float64, err error) {
	if len(data) < bb.period {
		return 0, 0, 0, errors.NewInsufficientDataErrorf(bb.period, len(data), "", "insufficient data points for Bollinger Bands: required %d, got %d", bb.period, len(data))
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

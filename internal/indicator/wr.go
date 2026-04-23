package indicator

import (
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// WR represents the Williams %R indicator.
//
// Williams %R is a momentum oscillator that measures the level of the close
// relative to the highest high for the lookback period. The value oscillates
// between -100 and 0, where readings above the overbought threshold (default
// -20) indicate an overbought market and readings below the oversold threshold
// (default -80) indicate an oversold market.
type WR struct {
	period              int
	overboughtThreshold float64 // typically -20
	oversoldThreshold   float64 // typically -80
}

// NewWR creates a new Williams %R indicator with default configuration.
func NewWR() Indicator {
	return &WR{
		period:              14,
		overboughtThreshold: -20,
		oversoldThreshold:   -80,
	}
}

// Name returns the name of the indicator.
func (w *WR) Name() types.IndicatorType {
	return types.IndicatorTypeWilliamsR
}

// Config configures the Williams %R indicator.
// Expected parameters: period (int), overboughtThreshold (float64, optional),
// oversoldThreshold (float64, optional).
func (w *WR) Config(params ...any) error {
	if len(params) < 1 {
		return errors.New(errors.ErrCodeMissingParameter, "Config expects at least 1 parameter: period (int)")
	}

	period, ok := params[0].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for period parameter, expected int")
	}

	if period <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "period must be a positive integer, got %d", period)
	}

	w.period = period

	if len(params) >= 2 {
		threshold, ok := params[1].(float64)
		if !ok {
			return errors.New(errors.ErrCodeInvalidType, "invalid type for overboughtThreshold parameter, expected float64")
		}

		w.overboughtThreshold = threshold
	}

	if len(params) >= 3 {
		threshold, ok := params[2].(float64)
		if !ok {
			return errors.New(errors.ErrCodeInvalidType, "invalid type for oversoldThreshold parameter, expected float64")
		}

		w.oversoldThreshold = threshold
	}

	return nil
}

// GetSignal calculates the Williams %R signal.
func (w *WR) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	wrValue, err := w.RawValue(marketData.Symbol, marketData.Time, ctx)
	if err != nil {
		return types.Signal{}, err
	}

	signalType := types.SignalTypeNoAction
	reason := "No signal"

	if wrValue < w.oversoldThreshold {
		signalType = types.SignalTypeBuyLong
		reason = fmt.Sprintf("Williams %%R oversold (value=%.2f)", wrValue)
	} else if wrValue > w.overboughtThreshold {
		signalType = types.SignalTypeSellShort
		reason = fmt.Sprintf("Williams %%R overbought (value=%.2f)", wrValue)
	}

	return types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(w.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"wr": wrValue,
		},
		Symbol:    marketData.Symbol,
		Indicator: w.Name(),
	}, nil
}

// RawValue computes the Williams %R value.
// Parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext).
func (w *WR) RawValue(params ...any) (float64, error) {
	if len(params) < 3 {
		return 0, errors.New(errors.ErrCodeMissingParameter, "RawValue requires at least 3 parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext)")
	}

	symbol, ok := params[0].(string)
	if !ok {
		return 0, errors.New(errors.ErrCodeInvalidType, "first parameter must be of type string (symbol)")
	}

	currentTime, ok := params[1].(time.Time)
	if !ok {
		return 0, errors.New(errors.ErrCodeInvalidType, "second parameter must be of type time.Time")
	}

	ctx, ok := params[2].(IndicatorContext)
	if !ok {
		return 0, errors.New(errors.ErrCodeInvalidType, "third parameter must be of type IndicatorContext")
	}

	historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, w.period)
	if err != nil {
		return 0, errors.Wrapf(errors.ErrCodeHistoricalDataFailed, err, "failed to get historical data for symbol %s", symbol)
	}

	if len(historicalData) < w.period {
		return 0, errors.NewInsufficientDataErrorf(w.period, len(historicalData), symbol, "insufficient historical data for Williams %%R calculation for symbol %s: required %d, got %d", symbol, w.period, len(historicalData))
	}

	highestHigh := historicalData[0].High
	lowestLow := historicalData[0].Low
	currentClose := historicalData[len(historicalData)-1].Close

	for _, data := range historicalData {
		if data.High > highestHigh {
			highestHigh = data.High
		}

		if data.Low < lowestLow {
			lowestLow = data.Low
		}
	}

	denominator := highestHigh - lowestLow
	if denominator == 0 {
		// Flat range - treat as neutral/zero %R value.
		return 0, nil
	}

	wr := ((highestHigh - currentClose) / denominator) * -100

	return wr, nil
}

package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// ATR represents the Average True Range indicator.
type ATR struct {
	period int
}

// NewATR creates a new ATR indicator with default configuration.
func NewATR() Indicator {
	return &ATR{
		period: 14, // Default period
	}
}

// Name returns the name of the indicator.
func (a *ATR) Name() types.IndicatorType {
	return types.IndicatorTypeATR
}

// Config configures the ATR indicator. Expected parameters: period (int).
func (a *ATR) Config(params ...any) error {
	if len(params) != 1 {
		return errors.New(errors.ErrCodeMissingParameter, "Config expects 1 parameter: period (int)")
	}

	period, ok := params[0].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for period parameter, expected int")
	}

	if period <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "period must be a positive integer, got %d", period)
	}

	a.period = period

	return nil
}

// GetSignal calculates the ATR signal.
func (a *ATR) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	atrValue, err := a.RawValue(marketData.Symbol, marketData.Time, ctx, a.period)
	if err != nil {
		return types.Signal{}, err
	}

	return types.Signal{
		Time:   marketData.Time,
		Type:   types.SignalTypeNoAction, // ATR is typically used for volatility, not direct signals
		Name:   string(a.Name()),
		Reason: fmt.Sprintf("ATR value: %.4f", atrValue),
		RawValue: map[string]float64{
			"atr": atrValue,
		},
		Symbol:    marketData.Symbol,
		Indicator: a.Name(),
	}, nil
}

// RawValue implements the Indicator interface.
func (a *ATR) RawValue(params ...any) (float64, error) {
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

	// Get market data
	var marketData types.MarketData

	var err error

	if !currentTime.IsZero() {
		endTime := currentTime
		startTime := endTime.Add(-time.Hour * 24)

		historicalData, err := ctx.DataSource.GetRange(startTime, endTime, optional.None[datasource.Interval]())
		if err != nil {
			return 0, errors.Wrap(errors.ErrCodeHistoricalDataFailed, "failed to get historical data", err)
		}

		if len(historicalData) == 0 {
			return 0, errors.New(errors.ErrCodeNoDataFound, "no historical data available for the specified time range")
		}

		marketData = historicalData[len(historicalData)-1]
	} else {
		marketData, err = ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return 0, errors.Wrap(errors.ErrCodeDataNotFound, "failed to get latest market data", err)
		}
	}

	// Calculate True Range
	tr := math.Max(
		math.Max(
			marketData.High-marketData.Low,
			math.Abs(marketData.High-marketData.Close),
		),
		math.Abs(marketData.Low-marketData.Close),
	)

	// Get EMA indicator for smoothing
	emaIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorNotFound, "failed to get EMA indicator", err)
	}

	// Calculate ATR using EMA
	atrValue, err := emaIndicator.RawValue(marketData.Symbol, marketData.Time, ctx, a.period, tr)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorCalculation, "failed to calculate ATR", err)
	}

	return atrValue, nil
}

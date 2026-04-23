package indicator

import (
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// PSY represents the Psychological Line indicator.
//
// The Psychological Line (PSY) is a sentiment oscillator that measures the
// percentage of periods within the lookback window that closed higher than the
// previous period. A reading above the upper threshold (default 75) is
// considered overbought while a reading below the lower threshold (default 25)
// is considered oversold.
type PSY struct {
	period         int
	upperThreshold float64
	lowerThreshold float64
}

// NewPSY creates a new Psychological Line indicator with default configuration.
func NewPSY() Indicator {
	return &PSY{
		period:         12,
		upperThreshold: 75,
		lowerThreshold: 25,
	}
}

// Name returns the name of the indicator.
func (p *PSY) Name() types.IndicatorType {
	return types.IndicatorTypePSY
}

// Config configures the PSY indicator.
// Expected parameters: period (int), upperThreshold (float64, optional),
// lowerThreshold (float64, optional).
func (p *PSY) Config(params ...any) error {
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

	p.period = period

	if len(params) >= 2 {
		threshold, ok := params[1].(float64)
		if !ok {
			return errors.New(errors.ErrCodeInvalidType, "invalid type for upperThreshold parameter, expected float64")
		}

		p.upperThreshold = threshold
	}

	if len(params) >= 3 {
		threshold, ok := params[2].(float64)
		if !ok {
			return errors.New(errors.ErrCodeInvalidType, "invalid type for lowerThreshold parameter, expected float64")
		}

		p.lowerThreshold = threshold
	}

	return nil
}

// GetSignal calculates the PSY signal.
func (p *PSY) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	psyValue, err := p.RawValue(marketData.Symbol, marketData.Time, ctx)
	if err != nil {
		return types.Signal{}, err
	}

	signalType := types.SignalTypeNoAction
	reason := "No signal"

	if psyValue < p.lowerThreshold {
		signalType = types.SignalTypeBuyLong
		reason = fmt.Sprintf("PSY oversold (value=%.2f)", psyValue)
	} else if psyValue > p.upperThreshold {
		signalType = types.SignalTypeSellShort
		reason = fmt.Sprintf("PSY overbought (value=%.2f)", psyValue)
	}

	return types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(p.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"psy": psyValue,
		},
		Symbol:    marketData.Symbol,
		Indicator: p.Name(),
	}, nil
}

// RawValue computes the PSY value as the percentage of up days in the lookback
// window.
// Parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext).
func (p *PSY) RawValue(params ...any) (float64, error) {
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

	// We need period+1 data points so we can compare each of the last `period`
	// closes against its previous close.
	required := p.period + 1

	historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, required)
	if err != nil {
		return 0, errors.Wrapf(errors.ErrCodeHistoricalDataFailed, err, "failed to get historical data for symbol %s", symbol)
	}

	if len(historicalData) < required {
		return 0, errors.NewInsufficientDataErrorf(required, len(historicalData), symbol, "insufficient historical data for PSY calculation for symbol %s: required %d, got %d", symbol, required, len(historicalData))
	}

	upCount := 0

	for i := 1; i < len(historicalData); i++ {
		if historicalData[i].Close > historicalData[i-1].Close {
			upCount++
		}
	}

	psy := (float64(upCount) / float64(p.period)) * 100

	return psy, nil
}

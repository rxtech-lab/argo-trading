package indicator

import (
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// MACD represents the Moving Average Convergence Divergence indicator.
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
}

// NewMACD creates a new MACD indicator with default configuration.
func NewMACD() Indicator {
	return &MACD{
		fastPeriod:   12, // Default fast period
		slowPeriod:   26, // Default slow period
		signalPeriod: 9,  // Default signal period
	}
}

// Name returns the name of the indicator.
func (m *MACD) Name() types.IndicatorType {
	return types.IndicatorTypeMACD
}

// Config configures the MACD indicator. Expected parameters: fastPeriod (int), slowPeriod (int), signalPeriod (int).
func (m *MACD) Config(params ...any) error {
	if len(params) != 3 {
		return errors.New(errors.ErrCodeMissingParameter, "Config expects 3 parameters: fastPeriod (int), slowPeriod (int), signalPeriod (int)")
	}

	fastPeriod, ok := params[0].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for fastPeriod parameter, expected int")
	}

	if fastPeriod <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "fastPeriod must be a positive integer, got %d", fastPeriod)
	}

	slowPeriod, ok := params[1].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for slowPeriod parameter, expected int")
	}

	if slowPeriod <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "slowPeriod must be a positive integer, got %d", slowPeriod)
	}

	signalPeriod, ok := params[2].(int)
	if !ok {
		return errors.New(errors.ErrCodeInvalidType, "invalid type for signalPeriod parameter, expected int")
	}

	if signalPeriod <= 0 {
		return errors.Newf(errors.ErrCodeInvalidPeriod, "signalPeriod must be a positive integer, got %d", signalPeriod)
	}

	m.fastPeriod = fastPeriod
	m.slowPeriod = slowPeriod
	m.signalPeriod = signalPeriod

	return nil
}

// GetSignal calculates the MACD signal.
func (m *MACD) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	macdValue, err := m.RawValue(marketData.Symbol, marketData.Time, ctx, m.fastPeriod, m.slowPeriod, m.signalPeriod)
	if err != nil {
		return types.Signal{}, err
	}

	signalType := types.SignalTypeNoAction
	reason := "No signal"

	if macdValue > 0 {
		signalType = types.SignalTypeBuyLong
		reason = fmt.Sprintf("MACD bullish (value=%.4f)", macdValue)
	} else if macdValue < 0 {
		signalType = types.SignalTypeSellShort
		reason = fmt.Sprintf("MACD bearish (value=%.4f)", macdValue)
	}

	return types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(m.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"macd": macdValue,
		},
		Symbol:    marketData.Symbol,
		Indicator: m.Name(),
	}, nil
}

// RawValue implements the Indicator interface.
func (m *MACD) RawValue(params ...any) (float64, error) {
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
		// Get historical data points
		historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, m.slowPeriod)
		if err != nil {
			return 0, errors.Wrap(errors.ErrCodeHistoricalDataFailed, "failed to get historical data", err)
		}

		if len(historicalData) == 0 {
			return 0, errors.Newf(errors.ErrCodeNoDataFound, "no historical data available for symbol %s", symbol)
		}

		// Check if we have enough data points for MACD calculation
		if len(historicalData) < m.slowPeriod {
			return 0, errors.NewInsufficientDataErrorf(m.slowPeriod, len(historicalData), symbol, "insufficient historical data for MACD calculation for symbol %s: required %d, got %d", symbol, m.slowPeriod, len(historicalData))
		}

		marketData = historicalData[len(historicalData)-1]
	} else {
		marketData, err = ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return 0, errors.Wrap(errors.ErrCodeDataNotFound, "failed to get latest market data", err)
		}
	}

	// Calculate MACD
	fastEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorNotFound, "failed to get EMA indicator", err)
	}

	slowEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorNotFound, "failed to get EMA indicator", err)
	}

	fastValue, err := fastEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.fastPeriod)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorCalculation, "failed to calculate fast EMA", err)
	}

	slowValue, err := slowEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.slowPeriod)
	if err != nil {
		return 0, errors.Wrap(errors.ErrCodeIndicatorCalculation, "failed to calculate slow EMA", err)
	}

	macdValue := fastValue - slowValue

	return macdValue, nil
}

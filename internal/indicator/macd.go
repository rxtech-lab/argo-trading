package indicator

import (
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
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
		return fmt.Errorf("Config expects 3 parameters: fastPeriod (int), slowPeriod (int), signalPeriod (int)")
	}

	fastPeriod, ok := params[0].(int)
	if !ok {
		return fmt.Errorf("invalid type for fastPeriod parameter, expected int")
	}

	if fastPeriod <= 0 {
		return fmt.Errorf("fastPeriod must be a positive integer, got %d", fastPeriod)
	}

	slowPeriod, ok := params[1].(int)
	if !ok {
		return fmt.Errorf("invalid type for slowPeriod parameter, expected int")
	}

	if slowPeriod <= 0 {
		return fmt.Errorf("slowPeriod must be a positive integer, got %d", slowPeriod)
	}

	signalPeriod, ok := params[2].(int)
	if !ok {
		return fmt.Errorf("invalid type for signalPeriod parameter, expected int")
	}

	if signalPeriod <= 0 {
		return fmt.Errorf("signalPeriod must be a positive integer, got %d", signalPeriod)
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
		Symbol: marketData.Symbol,
	}, nil
}

// RawValue implements the Indicator interface.
func (m *MACD) RawValue(params ...any) (float64, error) {
	if len(params) < 3 {
		return 0, fmt.Errorf("RawValue requires at least 3 parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext)")
	}

	symbol, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("first parameter must be of type string (symbol)")
	}

	currentTime, ok := params[1].(time.Time)
	if !ok {
		return 0, fmt.Errorf("second parameter must be of type time.Time")
	}

	ctx, ok := params[2].(IndicatorContext)
	if !ok {
		return 0, fmt.Errorf("third parameter must be of type IndicatorContext")
	}

	// Get market data
	var marketData types.MarketData

	var err error

	if !currentTime.IsZero() {
		// Get historical data points
		historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, m.slowPeriod)
		if err != nil {
			return 0, fmt.Errorf("failed to get historical data: %w", err)
		}

		if len(historicalData) == 0 {
			return 0, fmt.Errorf("no historical data available for symbol %s", symbol)
		}

		marketData = historicalData[len(historicalData)-1]
	} else {
		marketData, err = ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get latest market data: %w", err)
		}
	}

	// Calculate MACD
	fastEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		return 0, fmt.Errorf("failed to get EMA indicator: %w", err)
	}

	slowEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
	if err != nil {
		return 0, fmt.Errorf("failed to get EMA indicator: %w", err)
	}

	fastValue, err := fastEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.fastPeriod)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate fast EMA: %w", err)
	}

	slowValue, err := slowEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.slowPeriod)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate slow EMA: %w", err)
	}

	macdValue := fastValue - slowValue

	return macdValue, nil
}

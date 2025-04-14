package indicator

import (
	"fmt"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MA indicator implements Simple Moving Average calculation.
type MA struct {
	period int
}

// NewMA creates a new MA indicator with default configuration.
func NewMA() Indicator {
	return &MA{
		period: 20, // Default period
	}
}

// Name returns the name of the indicator.
func (m *MA) Name() types.IndicatorType {
	return types.IndicatorTypeMA
}

// Expected parameters: period (int).
func (m *MA) Config(params ...any) error {
	if len(params) != 1 {
		return fmt.Errorf("Config expects 1 parameter: period (int)")
	}

	period, ok := params[0].(int)
	if !ok {
		// Try to convert to float first
		periodFloat, ok := params[0].(float64)
		if !ok {
			return fmt.Errorf("invalid type for period parameter, expected int or float")
		}

		period = int(periodFloat)
	}

	if period <= 0 {
		return fmt.Errorf("period must be a positive integer, got %d", period)
	}

	m.period = period

	return nil
}

// GetSignal calculates the MA signal based on market data.
func (m *MA) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate MA
	maValue, err := m.RawValue(marketData.Symbol, marketData.Time, ctx, m.period)
	if err != nil {
		return types.Signal{}, fmt.Errorf("failed to calculate MA: %w", err)
	}

	// Determine signal type (basic example - in a real application you might want different logic)
	signalType := types.SignalTypeNoAction
	reason := "MA indicator calculated"

	// Create signal struct
	signal := types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(m.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"ma": maValue,
		},
		Symbol: marketData.Symbol,
	}

	return signal, nil
}

// RawValue calculates the MA value for a given symbol, time, context, and period.
// It accepts parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext), period (int).
func (m *MA) RawValue(params ...any) (float64, error) {
	// Ensure correct number and types of parameters
	if len(params) < 3 {
		return 0, fmt.Errorf("RawValue requires at least 3 parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext)")
	}

	symbol, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("invalid type for symbol parameter, expected string")
	}

	currentTime, ok := params[1].(time.Time)
	if !ok {
		return 0, fmt.Errorf("invalid type for currentTime parameter, expected time.Time")
	}

	ctx, ok := params[2].(IndicatorContext)
	if !ok {
		return 0, fmt.Errorf("invalid type for ctx parameter, expected IndicatorContext")
	}

	// Default to the configured period if not specified
	period := m.period

	// If period is explicitly provided as fourth parameter
	if len(params) >= 4 {
		switch p := params[3].(type) {
		case int:
			period = p
		case optional.Option[int]:
			if !p.IsNone() {
				periodValue, err := p.Take()
				if err != nil {
					return 0, fmt.Errorf("failed to get period value: %w", err)
				}

				period = periodValue
			}
		default:
			return 0, fmt.Errorf("invalid type for period parameter, expected int or optional.Option[int]")
		}
	}

	if period <= 0 {
		return 0, fmt.Errorf("period must be a positive integer, got %d", period)
	}

	// Get historical data
	historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, period)
	if err != nil {
		return 0, fmt.Errorf("failed to get historical data: %w", err)
	}

	if len(historicalData) == 0 {
		return 0, fmt.Errorf("no historical data available for symbol %s", symbol)
	}

	// Calculate MA
	return calculateSimpleMovingAverage(historicalData), nil
}

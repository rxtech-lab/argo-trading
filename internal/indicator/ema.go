package indicator

import (
	"fmt"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// EMA indicator implements Exponential Moving Average calculation.
type EMA struct {
	period int
}

// NewEMA creates a new EMA indicator with default configuration.
func NewEMA() Indicator {
	return &EMA{
		period: 20, // Default period
	}
}

// Name returns the name of the indicator.
func (e *EMA) Name() types.IndicatorType {
	return types.IndicatorTypeEMA
}

// Config configures the EMA indicator. Expected parameters: period (int).
func (e *EMA) Config(params ...any) error {
	if len(params) != 1 {
		return fmt.Errorf("Config expects 1 parameter: period (int)")
	}

	period, ok := params[0].(int)
	if !ok {
		return fmt.Errorf("invalid type for period parameter, expected int")
	}

	if period <= 0 {
		return fmt.Errorf("period must be a positive integer, got %d", period)
	}

	e.period = period

	return nil
}

// GetSignal calculates the EMA signal based on market data.
func (e *EMA) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate EMA
	emaValue, err := e.RawValue(marketData.Symbol, marketData.Time, ctx, e.period)
	if err != nil {
		return types.Signal{}, fmt.Errorf("failed to calculate EMA: %w", err)
	}

	// Determine signal type (basic example - in a real application you might want different logic)
	signalType := types.SignalTypeNoAction
	reason := "EMA indicator calculated"

	// Create signal struct
	signal := types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(e.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"ema": emaValue,
		},
		Symbol:    marketData.Symbol,
		Indicator: e.Name(),
	}

	return signal, nil
}

// RawValue calculates the EMA value for a given symbol, time, context, and period.
// It accepts parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext), period (int).
func (e *EMA) RawValue(params ...any) (float64, error) {
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
	period := e.period

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

	// If we don't have enough data points for a proper EMA, use simple average
	if len(historicalData) < period {
		return calculateSimpleMovingAverage(historicalData), nil
	}

	// Calculate EMA
	return calculateExponentialMovingAverage(historicalData, period), nil
}

// calculateSimpleMovingAverage calculates a simple moving average from the given historical data.
func calculateSimpleMovingAverage(data []types.MarketData) float64 {
	sum := 0.0
	for _, d := range data {
		sum += d.Close
	}

	return sum / float64(len(data))
}

// where Multiplier = 2 / (Period + 1).
func calculateExponentialMovingAverage(data []types.MarketData, period int) float64 {
	// Check if we have enough data
	if len(data) == 0 {
		return 0
	}

	// Sort data by time (oldest first)
	// Note: This assumes data is already sorted by time in ascending order

	// Start with SMA for the first EMA value
	sma := 0.0
	for i := 0; i < period && i < len(data); i++ {
		sma += data[i].Close
	}

	sma /= float64(period)

	// Calculate multiplier
	// Use alpha = 2/(span+1) to match pandas ewm implementation with adjust=False
	// In pandas, when using span parameter, the formula is slightly different from typical EMA
	alpha := 2.0 / float64(period+1)

	// Calculate EMA
	ema := sma
	for i := period; i < len(data); i++ {
		// Apply the pandas ewm formula: EMA = price * alpha + EMA_prev * (1 - alpha)
		ema = (data[i].Close * alpha) + (ema * (1 - alpha))
	}

	return ema
}

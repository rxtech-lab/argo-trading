package indicator

import (
	"fmt"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// RSI represents the Relative Strength Index indicator.
type RSI struct {
	period            int
	rsiLowerThreshold float64
	rsiUpperThreshold float64
}

// NewRSI creates a new RSI indicator with default configuration.
func NewRSI() Indicator {
	return &RSI{
		period:            14, // Default period
		rsiLowerThreshold: 30,
		rsiUpperThreshold: 70,
	}
}

// Name returns the name of the indicator.
func (r *RSI) Name() types.IndicatorType {
	return types.IndicatorTypeRSI
}

// Config configures the RSI indicator. Expected parameters: period (int).
func (r *RSI) Config(params ...any) error {
	if len(params) < 1 {
		return fmt.Errorf("Config expects at least 1 parameter: period (int)")
	}

	period, ok := params[0].(int)
	if !ok {
		return fmt.Errorf("invalid type for period parameter, expected int")
	}

	if period <= 0 {
		return fmt.Errorf("period must be a positive integer, got %d", period)
	}

	r.period = period

	// setup thresholds
	if len(params) == 2 {
		threshold, ok := params[1].(float64)
		if !ok {
			return fmt.Errorf("invalid type for threshold parameter, expected float64")
		}

		r.rsiLowerThreshold = threshold
	}

	if len(params) == 3 {
		threshold, ok := params[2].(float64)
		if !ok {
			return fmt.Errorf("invalid type for threshold parameter, expected float64")
		}

		r.rsiUpperThreshold = threshold
	}

	return nil
}

// GetSignal calculates the RSI signal.
func (r *RSI) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	rsiValue, err := r.RawValue(marketData.Symbol, marketData.Time, ctx, r.period)
	if err != nil {
		return types.Signal{}, err
	}

	signalType := types.SignalTypeNoAction
	reason := "No signal"

	if rsiValue < r.rsiLowerThreshold {
		signalType = types.SignalTypeBuyLong
		reason = fmt.Sprintf("RSI oversold (value=%.2f)", rsiValue)
	} else if rsiValue > r.rsiUpperThreshold {
		signalType = types.SignalTypeSellShort
		reason = fmt.Sprintf("RSI overbought (value=%.2f)", rsiValue)
	}

	return types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(r.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"rsi": rsiValue,
		},
		Symbol: marketData.Symbol,
	}, nil
}

// RawValue implements the Indicator interface.
func (r *RSI) RawValue(params ...any) (float64, error) {
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

	historicalData, err := ctx.DataSource.GetPreviousNumberOfDataPoints(currentTime, symbol, r.period+1)
	if err != nil {
		return 0, fmt.Errorf("failed to get historical data for symbol %s: %w", symbol, err)
	}

	if len(historicalData) < r.period+1 {
		return 0, fmt.Errorf("insufficient historical data for RSI calculation for symbol %s", symbol)
	}

	// Calculate price changes
	gains := make([]float64, 0)
	losses := make([]float64, 0)

	for i := 1; i < len(historicalData); i++ {
		change := historicalData[i].Close - historicalData[i-1].Close
		if change > 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Calculate average gains and losses
	avgGain := 0.0
	avgLoss := 0.0

	// First average
	for i := 0; i < r.period; i++ {
		avgGain += gains[i]
		avgLoss += losses[i]
	}

	avgGain /= float64(r.period)
	avgLoss /= float64(r.period)

	// Subsequent averages using Wilder's smoothing method
	for i := r.period; i < len(gains); i++ {
		avgGain = (avgGain*float64(r.period-1) + gains[i]) / float64(r.period)
		avgLoss = (avgLoss*float64(r.period-1) + losses[i]) / float64(r.period)
	}

	// Calculate RS and RSI
	if avgLoss == 0 {
		return 100, nil // Perfect uptrend
	}

	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi, nil
}

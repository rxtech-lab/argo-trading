package indicator

import (
	"fmt"
	"time"
)

// MACD (Moving Average Convergence Divergence) indicator implementation
type MACD struct {
	fastPeriod   int
	slowPeriod   int
	signalPeriod int
	params       map[string]interface{}
}

// MACDResult represents the result of MACD calculation
type MACDResult struct {
	MACD      []float64 // MACD line
	Signal    []float64 // Signal line
	Histogram []float64 // Histogram (MACD - Signal)
}

// NewMACD creates a new MACD indicator with the specified periods
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) *MACD {
	return &MACD{
		fastPeriod:   fastPeriod,
		slowPeriod:   slowPeriod,
		signalPeriod: signalPeriod,
		params: map[string]interface{}{
			"fastPeriod":   fastPeriod,
			"slowPeriod":   slowPeriod,
			"signalPeriod": signalPeriod,
		},
	}
}

// Name returns the name of the indicator
func (m *MACD) Name() string {
	return "MACD"
}

// SetParams allows setting parameters for the indicator
func (m *MACD) SetParams(params map[string]interface{}) error {
	if fast, ok := params["fastPeriod"]; ok {
		if p, ok := fast.(int); ok {
			m.fastPeriod = p
			m.params["fastPeriod"] = p
		} else {
			return fmt.Errorf("fastPeriod must be an integer")
		}
	}

	if slow, ok := params["slowPeriod"]; ok {
		if p, ok := slow.(int); ok {
			m.slowPeriod = p
			m.params["slowPeriod"] = p
		} else {
			return fmt.Errorf("slowPeriod must be an integer")
		}
	}

	if signal, ok := params["signalPeriod"]; ok {
		if p, ok := signal.(int); ok {
			m.signalPeriod = p
			m.params["signalPeriod"] = p
		} else {
			return fmt.Errorf("signalPeriod must be an integer")
		}
	}

	return nil
}

// GetParams returns the current parameters of the indicator
func (m *MACD) GetParams() map[string]interface{} {
	return m.params
}

// Calculate computes the MACD values using the provided context
func (m *MACD) Calculate(ctx IndicatorContext) (interface{}, error) {
	// Get all available data using a very wide time range
	startTime := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Now().AddDate(100, 0, 0) // 100 years in the future
	data := ctx.GetDataForTimeRange(startTime, endTime)

	if len(data) < m.slowPeriod+m.signalPeriod {
		return nil, fmt.Errorf("not enough data points for MACD calculation, need at least %d", m.slowPeriod+m.signalPeriod)
	}

	// Extract closing prices
	closePrices := make([]float64, len(data))
	for i, d := range data {
		closePrices[i] = d.Close
	}

	// Calculate EMAs
	fastEMA := calculateEMA(closePrices, m.fastPeriod)
	slowEMA := calculateEMA(closePrices, m.slowPeriod)

	// Calculate MACD line (fast EMA - slow EMA)
	macdLine := make([]float64, len(slowEMA))
	for i := 0; i < len(slowEMA); i++ {
		// Adjust index for fastEMA since it might have more values
		fastIndex := i + (len(fastEMA) - len(slowEMA))
		if fastIndex >= 0 && fastIndex < len(fastEMA) {
			macdLine[i] = fastEMA[fastIndex] - slowEMA[i]
		}
	}

	// Calculate signal line (EMA of MACD line)
	signalLine := calculateEMA(macdLine, m.signalPeriod)

	// Calculate histogram (MACD line - signal line)
	histogram := make([]float64, len(signalLine))
	for i := 0; i < len(signalLine); i++ {
		// Adjust index for macdLine
		macdIndex := i + (len(macdLine) - len(signalLine))
		if macdIndex >= 0 && macdIndex < len(macdLine) {
			histogram[i] = macdLine[macdIndex] - signalLine[i]
		}
	}

	return MACDResult{
		MACD:      macdLine,
		Signal:    signalLine,
		Histogram: histogram,
	}, nil
}

// calculateEMA calculates the Exponential Moving Average for the given period
func calculateEMA(prices []float64, period int) []float64 {
	if len(prices) < period {
		return []float64{}
	}

	// Calculate initial SMA
	var sum float64
	for i := 0; i < period; i++ {
		sum += prices[i]
	}
	sma := sum / float64(period)

	// Calculate multiplier
	multiplier := 2.0 / float64(period+1)

	// Calculate EMAs
	ema := make([]float64, len(prices)-period+1)
	ema[0] = sma

	for i := 1; i < len(ema); i++ {
		ema[i] = (prices[period+i-1]-ema[i-1])*multiplier + ema[i-1]
	}

	return ema
}

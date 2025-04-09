package indicator

import (
	"fmt"
	"log"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// EMA indicator implements Exponential Moving Average calculation
type EMA struct {
	Period int
}

// NewEMA creates a new EMA indicator
func NewEMA(period int) Indicator {
	if period <= 0 {
		period = 20 // Default period if invalid
	}
	return &EMA{
		Period: period,
	}
}

// Name returns the name of the indicator
func (e *EMA) Name() types.Indicator {
	return types.IndicatorEMA
}

// GetSignal calculates the EMA signal based on market data
func (e *EMA) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate EMA using SQL query from data source
	emaValue, err := e.CalculateEMAFromSQL(marketData.Symbol, marketData.Time, ctx)
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
		Symbol: marketData.Symbol,
	}

	return signal, nil
}

// CalculateEMAFromSQL calculates EMA using SQL query from data source
func (e *EMA) CalculateEMAFromSQL(symbol string, currentTime time.Time, ctx IndicatorContext) (float64, error) {
	// Calculate the time range for getting historical data
	// We need at least Period*3 data points to get a reliable EMA
	dataPointsNeeded := e.Period * 3

	// SQL query to get historical data
	query := `
		SELECT 
			close 
		FROM 
			market_data 
		WHERE 
			symbol = ? 
			AND time <= ? 
		ORDER BY 
			time DESC 
		LIMIT ?
	`

	results, err := ctx.DataSource.ExecuteSQL(query, symbol, currentTime, dataPointsNeeded)
	if err != nil {
		return 0, fmt.Errorf("error executing SQL query: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no data found for symbol %s", symbol)
	}

	// Extract close prices from query results
	prices := make([]float64, 0, len(results))
	for i := len(results) - 1; i >= 0; i-- { // Reverse to get chronological order
		closeVal, ok := results[i].Values["close"].(float64)
		if !ok {
			continue // Skip if not a valid float
		}
		prices = append(prices, closeVal)
	}

	if len(prices) == 0 {
		return 0, fmt.Errorf("no valid price data found for symbol %s", symbol)
	}

	// If we have fewer data points than the period, use what we have
	if len(prices) <= e.Period {
		// Calculate simple average as EMA seed
		sum := 0.0
		for _, price := range prices {
			sum += price
		}
		return sum / float64(len(prices)), nil
	}

	// Calculate EMA
	// First, calculate SMA as the seed value
	sum := 0.0
	for i := 0; i < e.Period; i++ {
		sum += prices[i]
	}
	ema := sum / float64(e.Period)

	// Then calculate EMA using multiplier
	multiplier := 2.0 / (float64(e.Period) + 1.0)

	for i := e.Period; i < len(prices); i++ {
		ema = (prices[i]-ema)*multiplier + ema
	}

	return ema, nil
}
func (e *EMA) RawValue(params ...any) (float64, error) {
	value, err := e.CalculateEMAFromSQL(params[0].(string), params[1].(time.Time), params[2].(IndicatorContext))
	if err != nil {
		// Log the error and return 0 as fallback
		log.Printf("Error calculating EMA: %v", err)
		return 0, err
	}
	return value, nil
}

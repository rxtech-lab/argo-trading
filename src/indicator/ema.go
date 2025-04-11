package indicator

import (
	"fmt"
	"log"
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/types"
)

// EMA indicator implements Exponential Moving Average calculation
type EMA struct {
	period int
}

// NewEMA creates a new EMA indicator with default configuration
func NewEMA() Indicator {
	return &EMA{
		period: 20, // Default period
	}
}

// Name returns the name of the indicator
func (e *EMA) Name() types.Indicator {
	return types.IndicatorEMA
}

// Config configures the EMA indicator with the given parameters
// Expected parameters: period (int)
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

// GetSignal calculates the EMA signal based on market data
func (e *EMA) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate EMA using SQL query from data source, passing the struct's period
	emaValue, err := e.calculateEMAFromSQL(marketData.Symbol, marketData.Time, &ctx, optional.Some(e.period))
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
// It now accepts a period parameter to allow for flexible EMA calculation.
func (e *EMA) calculateEMAFromSQL(symbol string, currentTime time.Time, ctx *IndicatorContext, period optional.Option[int]) (float64, error) {
	if period.IsNone() {
		period = optional.Some(e.period)
	}

	periodValue, err := period.Take()
	if err != nil {
		return 0, fmt.Errorf("failed to get period value: %w", err)
	}

	// First, check if we have enough data points
	countQuery := `
		SELECT COUNT(*) as count
		FROM market_data
		WHERE symbol = ? AND time <= ?
	`

	countResults, err := ctx.DataSource.ExecuteSQL(countQuery, symbol, currentTime)
	if err != nil {
		return 0, fmt.Errorf("error executing count query: %w", err)
	}

	if len(countResults) == 0 {
		return 0, fmt.Errorf("no data found for symbol %s", symbol)
	}

	count, ok := countResults[0].Values["count"].(int64)
	if !ok {
		return 0, fmt.Errorf("invalid count value returned from database")
	}

	// If count is zero, return no data found error
	if count == 0 {
		return 0, fmt.Errorf("no data found for symbol %s", symbol)
	}

	// If we have fewer data points than the period, use simple average
	if count < int64(periodValue) {
		simpleAvgQuery := `
			SELECT AVG(close) as avg_close
			FROM market_data
			WHERE symbol = ? AND time <= ?
		`

		avgResults, err := ctx.DataSource.ExecuteSQL(simpleAvgQuery, symbol, currentTime)
		if err != nil {
			return 0, fmt.Errorf("error executing simple average query: %w", err)
		}

		if len(avgResults) == 0 {
			return 0, fmt.Errorf("no data found for symbol %s", symbol)
		}

		avgVal, ok := avgResults[0].Values["avg_close"].(float64)
		if !ok {
			return 0, fmt.Errorf("invalid average value returned from database")
		}

		return avgVal, nil
	}

	// SQL query using window functions to calculate EMA directly in the database
	// The query uses a recursive CTE to implement the EMA algorithm
	query := `
		WITH RECURSIVE price_data AS (
			SELECT
				time,
				close,
				ROW_NUMBER() OVER (ORDER BY time ASC) as row_num
			FROM
				market_data
			WHERE
				symbol = ?
				AND time <= ?
			ORDER BY
				time ASC
		),
		ema_calc AS (
			-- Base case: first period points use SMA as seed
			SELECT
				time,
				close,
				row_num,
				AVG(close) OVER (
					ORDER BY time ASC
					ROWS BETWEEN ? - 1 PRECEDING AND CURRENT ROW
				) as ema_value
			FROM
				price_data
			WHERE
				row_num = ?
			
			UNION ALL
			
			-- Recursive case: calculate EMA using the formula: EMA = (Price - Previous EMA) * Multiplier + Previous EMA
			SELECT
				p.time,
				p.close,
				p.row_num,
				(p.close - e.ema_value) * (2.0 / (? + 1.0)) + e.ema_value as ema_value
			FROM
				price_data p
			JOIN
				ema_calc e ON p.row_num = e.row_num + 1
			WHERE
				p.row_num > ?
		)
		SELECT
			ema_value
		FROM
			ema_calc
		ORDER BY
			time DESC
		LIMIT 1;
	`

	results, err := ctx.DataSource.ExecuteSQL(query, symbol, currentTime, periodValue, periodValue, periodValue, periodValue)
	if err != nil {
		return 0, fmt.Errorf("error executing SQL query: %w", err)
	}

	if len(results) == 0 {
		return 0, fmt.Errorf("no data found for symbol %s", symbol)
	}

	// Extract EMA value from query result
	emaVal, ok := results[0].Values["ema_value"].(float64)
	if !ok {
		return 0, fmt.Errorf("invalid EMA value returned from database")
	}

	return emaVal, nil
}

// RawValue calculates the EMA value for a given symbol, time, context, and period.
// It accepts parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext), period (int).
func (e *EMA) RawValue(params ...any) (float64, error) {
	// Ensure correct number and types of parameters
	if len(params) != 4 { // Now expects 4 parameters
		return 0, fmt.Errorf("RawValue expects 4 parameters: symbol (string), currentTime (time.Time), ctx (IndicatorContext), period (int)")
	}
	symbol, ok := params[0].(string)
	if !ok {
		return 0, fmt.Errorf("invalid type for symbol parameter, expected string")
	}
	currentTime, ok := params[1].(time.Time)
	if !ok {
		return 0, fmt.Errorf("invalid type for currentTime parameter, expected time.Time")
	}
	ctx, ok := params[2].(IndicatorContext) // Keep assertion as IndicatorContext
	if !ok {
		return 0, fmt.Errorf("invalid type for ctx parameter, expected IndicatorContext")
	}
	period, ok := params[3].(optional.Option[int]) // Add period parameter check
	if !ok {
		return 0, fmt.Errorf("invalid type for period parameter, expected int")
	}
	if period.IsNone() {
		periodValue, err := period.Take()
		if err == nil && periodValue <= 0 {
			return 0, fmt.Errorf("period must be a positive integer, got %d", periodValue)
		}
	}

	value, err := e.calculateEMAFromSQL(symbol, currentTime, &ctx, period) // Pass period parameter
	if err != nil {
		// Log the error and return 0 as fallback
		log.Printf("Error calculating EMA: %v", err)
		return 0, err
	}
	return value, nil
}

package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// --- RangeFilter Indicator ---
// The indicator struct itself holds configuration parameters and state.
type RangeFilter struct {
	period     int
	multiplier float64
}

// NewRangeFilter creates a new Range Filter indicator with default configuration.
func NewRangeFilter() Indicator {
	return &RangeFilter{
		period:     100, // Default period
		multiplier: 3.0, // Default multiplier
	}
}

// Name returns the name of the indicator.
func (rf *RangeFilter) Name() types.IndicatorType {
	return types.IndicatorTypeRangeFilter
}

// Config configures the Range Filter indicator with the given parameters
// Expected parameters: period (int), multiplier (float64)
func (rf *RangeFilter) Config(params ...any) error {
	if len(params) != 2 {
		return fmt.Errorf("Config expects 2 parameters: period (int), multiplier (float64)")
	}

	period, ok := params[0].(int)
	if !ok {
		return fmt.Errorf("invalid type for period parameter, expected int")
	}
	if period <= 0 {
		return fmt.Errorf("period must be a positive integer, got %d", period)
	}

	multiplier, ok := params[1].(float64)
	if !ok {
		return fmt.Errorf("invalid type for multiplier parameter, expected float64")
	}
	if multiplier <= 0 {
		return fmt.Errorf("multiplier must be a positive number, got %f", multiplier)
	}

	rf.period = period
	rf.multiplier = multiplier
	return nil
}

// GetSignal calculates the Range Filter signal based on market data and stored state.
func (rf *RangeFilter) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate the Range Filter values
	filterData, err := rf.calculateFilter(marketData, ctx)
	if err != nil {
		return types.Signal{}, err
	}

	// Determine Signal Type
	signalType := types.SignalTypeNoAction
	reason := "No trend detected or uninitialized"
	if filterData.initialized { // Only generate signals after initialization
		reason = "No trend detected"
		if filterData.upward > 0 {
			signalType = types.SignalTypeBuyLong
			reason = fmt.Sprintf("Range Filter upward trend (filt=%.4f > prevFilt=%.4f)", filterData.filt, filterData.prevFilt)
		} else if filterData.downward > 0 {
			signalType = types.SignalTypeSellShort
			reason = fmt.Sprintf("Range Filter downward trend (filt=%.4f < prevFilt=%.4f)", filterData.filt, filterData.prevFilt)
		}
	}

	// Create Signal struct
	signal := types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(rf.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"filter":         filterData.filt,
			"smooth_range":   filterData.smrng,
			"upward_count":   filterData.upward,
			"downward_count": filterData.downward,
		},
		Symbol: marketData.Symbol,
	}

	return signal, nil
}

// RangeFilterData holds the calculated filter values
type RangeFilterData struct {
	filt        float64
	smrng       float64
	prevFilt    float64
	upward      float64
	downward    float64
	initialized bool
}

// calculateFilter performs the actual Range Filter calculation
// This shared function is used by both GetSignal and RawValue
func (rf *RangeFilter) calculateFilter(marketData types.MarketData, ctx IndicatorContext) (RangeFilterData, error) {
	src := marketData.Close
	result := RangeFilterData{}

	// Initialize state if needed
	if ctx.Cache.RangeFilterState.IsNone() {
		ctx.Cache.RangeFilterState = optional.Some(cache.RangeFilterState{
			PrevFilt:   math.NaN(),
			PrevSource: math.NaN(),
			Symbol:     marketData.Symbol,
		})
	}

	value, err := ctx.Cache.RangeFilterState.Take()
	if err != nil {
		return result, err
	}
	if value.Symbol != marketData.Symbol {
		ctx.Cache.RangeFilterState = optional.Some(cache.RangeFilterState{
			PrevFilt:   math.NaN(),
			PrevSource: math.NaN(),
			Symbol:     marketData.Symbol,
		})
		value, _ = ctx.Cache.RangeFilterState.Take()
	}

	var smrng, filt float64
	currentUpward := value.Upward     // Start with current state
	currentDownward := value.Downward // Start with current state

	// Initialize on the first valid data point if state indicates not initialized
	if !value.Initialized {
		if !math.IsNaN(src) {
			value.PrevFilt = src
			value.PrevSource = src
			value.Initialized = true
			filt = value.PrevFilt // Use initial value for the first run
			smrng = 0             // Initially zero
			currentUpward = 0     // Reset counts on initialization
			currentDownward = 0   // Reset counts on initialization
			result.initialized = true
		}
	} else {
		result.initialized = true
		// Calculate smoothed average range (smrng) using EMA calculations
		absChange := math.Abs(src - value.PrevSource)

		// Get EMA indicators from registry
		shortEMAIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
		if err != nil {
			return result, fmt.Errorf("failed to get EMA indicator from registry: %w", err)
		}

		// Use type assertion to get the EMA indicator
		shortEMA, ok := shortEMAIndicator.(*EMA)
		if !ok {
			return result, fmt.Errorf("indicator is not an EMA indicator")
		}

		// Clone for long EMA
		longEMAIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorTypeEMA)
		if err != nil {
			return result, fmt.Errorf("failed to get EMA indicator from registry: %w", err)
		}

		longEMA, ok := longEMAIndicator.(*EMA)
		if !ok {
			return result, fmt.Errorf("indicator is not an EMA indicator")
		}

		// Calculate both EMAs using RawValue with the period parameter
		shortEMAPeriod := rf.period
		shortEMAValue, err := shortEMA.RawValue(marketData.Symbol, marketData.Time, ctx, shortEMAPeriod)
		if err != nil {
			return result, fmt.Errorf("failed to calculate short EMA (period %d): %w", shortEMAPeriod, err)
		}

		longEMAPeriod := rf.period*2 - 1
		// Ensure long period is at least 1
		if longEMAPeriod < 1 {
			longEMAPeriod = 1
		}
		longEMAValue, err := longEMA.RawValue(marketData.Symbol, marketData.Time, ctx, longEMAPeriod)
		if err != nil {
			return result, fmt.Errorf("failed to calculate long EMA (period %d): %w", longEMAPeriod, err)
		}

		// Apply absolute change to calculate the smooth range
		smrng = absChange * (shortEMAValue*0.4 + longEMAValue*0.6) * rf.multiplier

		// Calculate filter value (filt)
		prevFilt := value.PrevFilt
		if src > prevFilt {
			if (src - smrng) < prevFilt {
				filt = prevFilt
			} else {
				filt = src - smrng
			}
		} else {
			if (src + smrng) > prevFilt {
				filt = prevFilt
			} else {
				filt = src + smrng
			}
		}

		// Calculate Upward/Downward trend detection
		if filt > prevFilt {
			currentUpward = value.Upward + 1
			currentDownward = 0
		} else if filt < prevFilt {
			currentUpward = 0
			currentDownward = value.Downward + 1
		}

		result.prevFilt = prevFilt
	}

	// --- Update state variables for saving ---
	value.PrevSource = src
	value.PrevFilt = filt
	value.Upward = currentUpward
	value.Downward = currentDownward
	ctx.Cache.RangeFilterState = optional.Some(value)

	// Set the results
	result.filt = filt
	result.smrng = smrng
	result.upward = currentUpward
	result.downward = currentDownward

	return result, nil
}

// RawValue implements the Indicator interface.
// It returns the current filter value for the given parameters.
// Parameters expected:
// - symbol (string)
// - currentTime (time.Time)
// - ctx (IndicatorContext)
// - optional period (int) - if provided, overrides the default Period
func (rf *RangeFilter) RawValue(params ...any) (float64, error) {
	// Validate and extract parameters
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

	// Check if we have a custom period parameter
	origPeriod := rf.period
	if len(params) > 3 {
		if customPeriod, ok := params[3].(int); ok && customPeriod > 0 {
			// Temporarily override the period for this calculation
			rf.period = customPeriod
			// Restore the original period when we're done
			defer func() { rf.period = origPeriod }()
		}
	}

	// Get market data
	var marketData types.MarketData
	var err error

	// If a specific time is provided, get data up to that time
	if !currentTime.IsZero() {
		// Get data from a range ending at currentTime
		endTime := currentTime
		startTime := endTime.Add(-time.Hour * 24) // Get last 24 hours of data

		historicalData, err := ctx.DataSource.GetRange(startTime, endTime, optional.None[datasource.Interval]())
		if err != nil {
			return 0, fmt.Errorf("failed to get historical data: %w", err)
		}

		if len(historicalData) == 0 {
			return 0, fmt.Errorf("no historical data available for the specified time range")
		}

		// Use the last data point
		marketData = historicalData[len(historicalData)-1]
	} else {
		// Get the latest data if no specific time provided
		marketData, err = ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get latest market data: %w", err)
		}
	}

	// Use the shared calculation function
	filterData, err := rf.calculateFilter(marketData, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate Range Filter: %w", err)
	}

	// Return the filter value
	return filterData.filt, nil
}

package indicator

import (
	"fmt"
	"math"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/cache"
	"github.com/sirily11/argo-trading-go/src/types"
)

// --- RangeFilter Indicator ---
// The indicator struct itself holds configuration parameters and state.
type RangeFilter struct {
	Period     int
	Multiplier float64
}

// NewRangeFilter creates a new Range Filter indicator.
func NewRangeFilter(period int, multiplier float64) *RangeFilter {
	if period <= 0 {
		period = 100 // Use default if invalid
	}
	if multiplier <= 0 {
		multiplier = 3.0 // Use default if invalid
	}
	return &RangeFilter{
		Period:     period,
		Multiplier: multiplier,
	}
}

// Name returns the name of the indicator.
func (rf *RangeFilter) Name() types.Indicator {
	return types.IndicatorRangeFilter
}

// GetSignal calculates the Range Filter signal based on market data and stored state.
func (rf *RangeFilter) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	src := marketData.Close

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
		return types.Signal{}, err
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
		}
	} else {
		// Calculate smoothed average range (smrng) using EMA calculations
		absChange := math.Abs(src - value.PrevSource)

		// Get EMA indicators from registry
		shortEMAIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorEMA)
		if err != nil {
			return types.Signal{}, fmt.Errorf("failed to get EMA indicator from registry: %w", err)
		}

		// Use type assertion to get the EMA indicator
		shortEMA, ok := shortEMAIndicator.(*EMA)
		if !ok {
			return types.Signal{}, fmt.Errorf("indicator is not an EMA indicator")
		}

		// Clone for long EMA
		longEMAIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorEMA)
		if err != nil {
			return types.Signal{}, fmt.Errorf("failed to get EMA indicator from registry: %w", err)
		}

		longEMA, ok := longEMAIndicator.(*EMA)
		if !ok {
			return types.Signal{}, fmt.Errorf("indicator is not an EMA indicator")
		}

		// Configure periods
		shortEMA.Period = rf.Period
		longEMA.Period = rf.Period*2 - 1

		// Create a context for each EMA calculation
		absChangeCtx := IndicatorContext{
			DataSource:        ctx.DataSource,
			IndicatorRegistry: ctx.IndicatorRegistry,
			Cache:             ctx.Cache,
		}

		// Calculate both EMAs
		shortEMAValue, err := shortEMA.CalculateEMAFromSQL(marketData.Symbol, marketData.Time, absChangeCtx)
		if err != nil {
			return types.Signal{}, fmt.Errorf("failed to calculate EMA of absolute changes: %w", err)
		}

		longEMAValue, err := longEMA.CalculateEMAFromSQL(marketData.Symbol, marketData.Time, absChangeCtx)
		if err != nil {
			return types.Signal{}, fmt.Errorf("failed to calculate EMA of smooth range: %w", err)
		}

		// Apply absolute change to calculate the smooth range
		smrng = absChange * (shortEMAValue*0.4 + longEMAValue*0.6) * rf.Multiplier

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
		} else {
			// currentUpward and currentDownward retain their current state values
		}
	}

	// Determine Signal Type
	signalType := types.SignalTypeNoAction
	reason := "No trend detected or uninitialized"
	if value.Initialized { // Only generate signals after initialization
		reason = "No trend detected"
		if currentUpward > 0 {
			signalType = types.SignalTypeBuyLong
			reason = fmt.Sprintf("Range Filter upward trend (filt=%.4f > prevFilt=%.4f)", filt, value.PrevFilt)
		} else if currentDownward > 0 {
			signalType = types.SignalTypeSellShort
			reason = fmt.Sprintf("Range Filter downward trend (filt=%.4f < prevFilt=%.4f)", filt, value.PrevFilt)
		}
	}

	// Create Signal struct
	signal := types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(rf.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"filter":         filt,
			"smooth_range":   smrng,
			"upward_count":   currentUpward,
			"downward_count": currentDownward,
		},
		Symbol: marketData.Symbol,
	}

	// --- Update state variables for saving ---
	value.PrevSource = src
	value.PrevFilt = filt
	value.Upward = currentUpward
	value.Downward = currentDownward
	ctx.Cache.RangeFilterState = optional.Some(value)

	return signal, nil
}

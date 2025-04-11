package indicator

import (
	"fmt"
	"math"
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/cache"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/types"
)

// WaddahAttar represents the Waddah Attar Explosion indicator
type WaddahAttar struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
	ATRPeriod    int
	Multiplier   float64
}

// NewWaddahAttar creates a new Waddah Attar Explosion indicator
func NewWaddahAttar(fastPeriod, slowPeriod, signalPeriod, atrPeriod int, multiplier float64) Indicator {
	if fastPeriod <= 0 {
		fastPeriod = 20 // Default fast period
	}
	if slowPeriod <= 0 {
		slowPeriod = 40 // Default slow period
	}
	if signalPeriod <= 0 {
		signalPeriod = 9 // Default signal period
	}
	if atrPeriod <= 0 {
		atrPeriod = 14 // Default ATR period
	}
	if multiplier <= 0 {
		multiplier = 150.0 // Default multiplier
	}
	return &WaddahAttar{
		FastPeriod:   fastPeriod,
		SlowPeriod:   slowPeriod,
		SignalPeriod: signalPeriod,
		ATRPeriod:    atrPeriod,
		Multiplier:   multiplier,
	}
}

// Name returns the name of the indicator
func (wa *WaddahAttar) Name() types.Indicator {
	return types.IndicatorWaddahAttar
}

// WaddahAttarData holds the calculated indicator values
type WaddahAttarData struct {
	macd        float64
	signal      float64
	hist        float64
	atr         float64
	trend       float64
	explosion   float64
	initialized bool
}

// GetSignal calculates the Waddah Attar Explosion signal
func (wa *WaddahAttar) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	// Calculate the Waddah Attar values
	waData, err := wa.calculateWaddahAttar(marketData, ctx)
	if err != nil {
		return types.Signal{}, err
	}

	// Determine Signal Type
	signalType := types.SignalTypeNoAction
	reason := "No trend detected or uninitialized"
	if waData.initialized {
		reason = "No trend detected"
		if waData.explosion > 0 && waData.trend > 0 {
			signalType = types.SignalTypeBuyLong
			reason = fmt.Sprintf("Waddah Attar bullish explosion (explosion=%.4f, trend=%.4f)", waData.explosion, waData.trend)
		} else if waData.explosion > 0 && waData.trend < 0 {
			signalType = types.SignalTypeSellShort
			reason = fmt.Sprintf("Waddah Attar bearish explosion (explosion=%.4f, trend=%.4f)", waData.explosion, waData.trend)
		}
	}

	// Create Signal struct
	signal := types.Signal{
		Time:   marketData.Time,
		Type:   signalType,
		Name:   string(wa.Name()),
		Reason: reason,
		RawValue: map[string]float64{
			"macd":      waData.macd,
			"signal":    waData.signal,
			"histogram": waData.hist,
			"atr":       waData.atr,
			"trend":     waData.trend,
			"explosion": waData.explosion,
		},
		Symbol: marketData.Symbol,
	}

	return signal, nil
}

// calculateWaddahAttar performs the actual Waddah Attar calculation
func (wa *WaddahAttar) calculateWaddahAttar(marketData types.MarketData, ctx IndicatorContext) (WaddahAttarData, error) {
	result := WaddahAttarData{}

	// Initialize state if needed
	if ctx.Cache.WaddahAttarState.IsNone() {
		ctx.Cache.WaddahAttarState = optional.Some(cache.WaddahAttarState{
			PrevMACD:   math.NaN(),
			PrevSignal: math.NaN(),
			PrevHist:   math.NaN(),
			PrevATR:    math.NaN(),
			Symbol:     marketData.Symbol,
		})
	}

	value, err := ctx.Cache.WaddahAttarState.Take()
	if err != nil {
		return result, err
	}
	if value.Symbol != marketData.Symbol {
		ctx.Cache.WaddahAttarState = optional.Some(cache.WaddahAttarState{
			PrevMACD:   math.NaN(),
			PrevSignal: math.NaN(),
			PrevHist:   math.NaN(),
			PrevATR:    math.NaN(),
			Symbol:     marketData.Symbol,
		})
		value, _ = ctx.Cache.WaddahAttarState.Take()
	}

	// Get MACD indicator
	macdIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorMACD)
	if err != nil {
		return result, fmt.Errorf("failed to get MACD indicator: %w", err)
	}

	macd, ok := macdIndicator.(*MACD)
	if !ok {
		return result, fmt.Errorf("indicator is not a MACD indicator")
	}

	// Calculate MACD values
	macdValue, err := macd.RawValue(marketData.Symbol, marketData.Time, ctx, wa.FastPeriod, wa.SlowPeriod, wa.SignalPeriod)
	if err != nil {
		return result, fmt.Errorf("failed to calculate MACD: %w", err)
	}

	// Get ATR indicator
	atrIndicator, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorATR)
	if err != nil {
		return result, fmt.Errorf("failed to get ATR indicator: %w", err)
	}

	atr, ok := atrIndicator.(*ATR)
	if !ok {
		return result, fmt.Errorf("indicator is not an ATR indicator")
	}

	// Calculate ATR
	atrValue, err := atr.RawValue(marketData.Symbol, marketData.Time, ctx, wa.ATRPeriod)
	if err != nil {
		return result, fmt.Errorf("failed to calculate ATR: %w", err)
	}

	// Calculate trend and explosion
	trend := macdValue * wa.Multiplier
	explosion := atrValue * wa.Multiplier

	// Update state
	value.PrevMACD = macdValue
	value.PrevATR = atrValue
	value.Initialized = true
	ctx.Cache.WaddahAttarState = optional.Some(value)

	// Set results
	result.macd = macdValue
	result.atr = atrValue
	result.trend = trend
	result.explosion = explosion
	result.initialized = true

	return result, nil
}

// RawValue implements the Indicator interface
func (wa *WaddahAttar) RawValue(params ...any) (float64, error) {
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

	// Get market data
	var marketData types.MarketData
	var err error

	if !currentTime.IsZero() {
		endTime := currentTime
		startTime := endTime.Add(-time.Hour * 24)
		historicalData, err := ctx.DataSource.GetRange(startTime, endTime, datasource.Interval1m)
		if err != nil {
			return 0, fmt.Errorf("failed to get historical data: %w", err)
		}
		if len(historicalData) == 0 {
			return 0, fmt.Errorf("no historical data available for the specified time range")
		}
		marketData = historicalData[len(historicalData)-1]
	} else {
		marketData, err = ctx.DataSource.ReadLastData(symbol)
		if err != nil {
			return 0, fmt.Errorf("failed to get latest market data: %w", err)
		}
	}

	// Calculate Waddah Attar values
	waData, err := wa.calculateWaddahAttar(marketData, ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate Waddah Attar: %w", err)
	}

	// Return the explosion value
	return waData.explosion, nil
}

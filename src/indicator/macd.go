package indicator

import (
	"fmt"
	"time"

	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/src/types"
)

// MACD represents the Moving Average Convergence Divergence indicator
type MACD struct {
	FastPeriod   int
	SlowPeriod   int
	SignalPeriod int
}

// NewMACD creates a new MACD indicator
func NewMACD(fastPeriod, slowPeriod, signalPeriod int) Indicator {
	if fastPeriod <= 0 {
		fastPeriod = 12 // Default fast period
	}
	if slowPeriod <= 0 {
		slowPeriod = 26 // Default slow period
	}
	if signalPeriod <= 0 {
		signalPeriod = 9 // Default signal period
	}
	return &MACD{
		FastPeriod:   fastPeriod,
		SlowPeriod:   slowPeriod,
		SignalPeriod: signalPeriod,
	}
}

// Name returns the name of the indicator
func (m *MACD) Name() types.Indicator {
	return types.IndicatorMACD
}

// GetSignal calculates the MACD signal
func (m *MACD) GetSignal(marketData types.MarketData, ctx IndicatorContext) (types.Signal, error) {
	macdValue, err := m.RawValue(marketData.Symbol, marketData.Time, ctx, m.FastPeriod, m.SlowPeriod, m.SignalPeriod)
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

// RawValue implements the Indicator interface
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

	// Calculate MACD
	fastEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorEMA)
	if err != nil {
		return 0, fmt.Errorf("failed to get EMA indicator: %w", err)
	}

	slowEMA, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorEMA)
	if err != nil {
		return 0, fmt.Errorf("failed to get EMA indicator: %w", err)
	}

	fastValue, err := fastEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.FastPeriod)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate fast EMA: %w", err)
	}

	slowValue, err := slowEMA.RawValue(marketData.Symbol, marketData.Time, ctx, m.SlowPeriod)
	if err != nil {
		return 0, fmt.Errorf("failed to calculate slow EMA: %w", err)
	}

	macdValue := fastValue - slowValue
	return macdValue, nil
}

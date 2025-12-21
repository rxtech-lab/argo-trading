package runtime

import (
	"testing"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/assert"
)

func TestStrategyIntervalToDataSourceInterval(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.Interval
		expected optional.Option[datasource.Interval]
	}{
		{
			name:     "1 minute interval",
			input:    strategy.Interval_INTERVAL_1M,
			expected: optional.Some(datasource.Interval1m),
		},
		{
			name:     "5 minute interval",
			input:    strategy.Interval_INTERVAL_5M,
			expected: optional.Some(datasource.Interval5m),
		},
		{
			name:     "15 minute interval",
			input:    strategy.Interval_INTERVAL_15M,
			expected: optional.Some(datasource.Interval15m),
		},
		{
			name:     "30 minute interval",
			input:    strategy.Interval_INTERVAL_30M,
			expected: optional.Some(datasource.Interval30m),
		},
		{
			name:     "unknown interval returns None",
			input:    strategy.Interval(999),
			expected: optional.None[datasource.Interval](),
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyIntervalToDataSourceInterval(tc.input)
			if tc.expected.IsNone() {
				assert.True(t, result.IsNone())
			} else {
				assert.True(t, result.IsSome())
				assert.Equal(t, tc.expected.Unwrap(), result.Unwrap())
			}
		})
	}
}

func TestStrategyPurchaseTypeToPurchaseType(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.PurchaseType
		expected types.PurchaseType
	}{
		{
			name:     "buy purchase type",
			input:    strategy.PurchaseType_PURCHASE_TYPE_BUY,
			expected: types.PurchaseTypeBuy,
		},
		{
			name:     "sell purchase type",
			input:    strategy.PurchaseType_PURCHASE_TYPE_SELL,
			expected: types.PurchaseTypeSell,
		},
		{
			name:     "unknown defaults to buy",
			input:    strategy.PurchaseType(999),
			expected: types.PurchaseTypeBuy,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyPurchaseTypeToPurchaseType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStrategyPositionTypeToPositionType(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.PositionType
		expected types.PositionType
	}{
		{
			name:     "long position type",
			input:    strategy.PositionType_POSITION_TYPE_LONG,
			expected: types.PositionTypeLong,
		},
		{
			name:     "short position type",
			input:    strategy.PositionType_POSITION_TYPE_SHORT,
			expected: types.PositionTypeShort,
		},
		{
			name:     "unknown defaults to long",
			input:    strategy.PositionType(999),
			expected: types.PositionTypeLong,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyPositionTypeToPositionType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStrategyOrderTypeToOrderType(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.OrderType
		expected types.OrderType
	}{
		{
			name:     "market order type",
			input:    strategy.OrderType_ORDER_TYPE_MARKET,
			expected: types.OrderTypeMarket,
		},
		{
			name:     "limit order type",
			input:    strategy.OrderType_ORDER_TYPE_LIMIT,
			expected: types.OrderTypeLimit,
		},
		{
			name:     "unknown defaults to market",
			input:    strategy.OrderType(999),
			expected: types.OrderTypeMarket,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyOrderTypeToOrderType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStrategySignalTypeToSignalType(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.SignalType
		expected types.SignalType
	}{
		{
			name:     "buy long signal",
			input:    strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
			expected: types.SignalTypeBuyLong,
		},
		{
			name:     "sell long signal",
			input:    strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
			expected: types.SignalTypeSellLong,
		},
		{
			name:     "buy short signal",
			input:    strategy.SignalType_SIGNAL_TYPE_BUY_SHORT,
			expected: types.SignalTypeBuyShort,
		},
		{
			name:     "sell short signal",
			input:    strategy.SignalType_SIGNAL_TYPE_SELL_SHORT,
			expected: types.SignalTypeSellShort,
		},
		{
			name:     "unknown defaults to no action",
			input:    strategy.SignalType(999),
			expected: types.SignalTypeNoAction,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategySignalTypeToSignalType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStrategyMarkShapeToMarkShape(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.MarkShape
		expected types.MarkShape
	}{
		{
			name:     "circle shape",
			input:    strategy.MarkShape_MARK_SHAPE_CIRCLE,
			expected: types.MarkShapeCircle,
		},
		{
			name:     "square shape",
			input:    strategy.MarkShape_MARK_SHAPE_SQUARE,
			expected: types.MarkShapeSquare,
		},
		{
			name:     "triangle shape",
			input:    strategy.MarkShape_MARK_SHAPE_TRIANGLE,
			expected: types.MarkShapeTriangle,
		},
		{
			name:     "unknown defaults to circle",
			input:    strategy.MarkShape(999),
			expected: types.MarkShapeCircle,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyMarkShapeToMarkShape(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestStrategyIndicatorTypeToIndicatorType(t *testing.T) {
	tests := []struct {
		name     string
		input    strategy.IndicatorType
		expected types.IndicatorType
	}{
		{
			name:     "RSI indicator",
			input:    strategy.IndicatorType_INDICATOR_RSI,
			expected: types.IndicatorTypeRSI,
		},
		{
			name:     "MACD indicator",
			input:    strategy.IndicatorType_INDICATOR_MACD,
			expected: types.IndicatorTypeMACD,
		},
		{
			name:     "Williams R indicator",
			input:    strategy.IndicatorType_INDICATOR_WILLIAMS_R,
			expected: types.IndicatorTypeWilliamsR,
		},
		{
			name:     "ADX indicator",
			input:    strategy.IndicatorType_INDICATOR_ADX,
			expected: types.IndicatorTypeADX,
		},
		{
			name:     "CCI indicator",
			input:    strategy.IndicatorType_INDICATOR_CCI,
			expected: types.IndicatorTypeCCI,
		},
		{
			name:     "AO indicator",
			input:    strategy.IndicatorType_INDICATOR_AO,
			expected: types.IndicatorTypeAO,
		},
		{
			name:     "Trend Strength indicator",
			input:    strategy.IndicatorType_INDICATOR_TREND_STRENGTH,
			expected: types.IndicatorTypeTrendStrength,
		},
		{
			name:     "Range Filter indicator",
			input:    strategy.IndicatorType_INDICATOR_RANGE_FILTER,
			expected: types.IndicatorTypeRangeFilter,
		},
		{
			name:     "EMA indicator",
			input:    strategy.IndicatorType_INDICATOR_EMA,
			expected: types.IndicatorTypeEMA,
		},
		{
			name:     "Waddah Attar indicator",
			input:    strategy.IndicatorType_INDICATOR_WADDAH_ATTAR,
			expected: types.IndicatorTypeWaddahAttar,
		},
		{
			name:     "ATR indicator",
			input:    strategy.IndicatorType_INDICATOR_ATR,
			expected: types.IndicatorTypeATR,
		},
		{
			name:     "MA indicator",
			input:    strategy.IndicatorType_INDICATOR_MA,
			expected: types.IndicatorTypeMA,
		},
		{
			name:     "unknown defaults to RSI",
			input:    strategy.IndicatorType(999),
			expected: types.IndicatorTypeRSI,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := StrategyIndicatorTypeToIndicatorType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestSignalTypeToStrategySignalType(t *testing.T) {
	tests := []struct {
		name     string
		input    types.SignalType
		expected strategy.SignalType
	}{
		{
			name:     "buy long signal",
			input:    types.SignalTypeBuyLong,
			expected: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
		},
		{
			name:     "sell long signal",
			input:    types.SignalTypeSellLong,
			expected: strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
		},
		{
			name:     "buy short signal",
			input:    types.SignalTypeBuyShort,
			expected: strategy.SignalType_SIGNAL_TYPE_BUY_SHORT,
		},
		{
			name:     "sell short signal",
			input:    types.SignalTypeSellShort,
			expected: strategy.SignalType_SIGNAL_TYPE_SELL_SHORT,
		},
		{
			name:     "no action signal",
			input:    types.SignalTypeNoAction,
			expected: strategy.SignalType_SIGNAL_TYPE_NO_ACTION,
		},
		{
			name:     "close position signal",
			input:    types.SignalTypeClosePosition,
			expected: strategy.SignalType_SIGNAL_TYPE_CLOSE_POSITION,
		},
		{
			name:     "wait signal",
			input:    types.SignalTypeWait,
			expected: strategy.SignalType_SIGNAL_TYPE_WAIT,
		},
		{
			name:     "abort signal",
			input:    types.SignalTypeAbort,
			expected: strategy.SignalType_SIGNAL_TYPE_ABORT,
		},
		{
			name:     "unknown defaults to no action",
			input:    types.SignalType("unknown"),
			expected: strategy.SignalType_SIGNAL_TYPE_NO_ACTION,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := SignalTypeToStrategySignalType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestIndicatorTypeToStrategyIndicatorType(t *testing.T) {
	tests := []struct {
		name     string
		input    types.IndicatorType
		expected strategy.IndicatorType
	}{
		{
			name:     "RSI indicator",
			input:    types.IndicatorTypeRSI,
			expected: strategy.IndicatorType_INDICATOR_RSI,
		},
		{
			name:     "MACD indicator",
			input:    types.IndicatorTypeMACD,
			expected: strategy.IndicatorType_INDICATOR_MACD,
		},
		{
			name:     "Williams R indicator",
			input:    types.IndicatorTypeWilliamsR,
			expected: strategy.IndicatorType_INDICATOR_WILLIAMS_R,
		},
		{
			name:     "ADX indicator",
			input:    types.IndicatorTypeADX,
			expected: strategy.IndicatorType_INDICATOR_ADX,
		},
		{
			name:     "CCI indicator",
			input:    types.IndicatorTypeCCI,
			expected: strategy.IndicatorType_INDICATOR_CCI,
		},
		{
			name:     "AO indicator",
			input:    types.IndicatorTypeAO,
			expected: strategy.IndicatorType_INDICATOR_AO,
		},
		{
			name:     "Trend Strength indicator",
			input:    types.IndicatorTypeTrendStrength,
			expected: strategy.IndicatorType_INDICATOR_TREND_STRENGTH,
		},
		{
			name:     "Range Filter indicator",
			input:    types.IndicatorTypeRangeFilter,
			expected: strategy.IndicatorType_INDICATOR_RANGE_FILTER,
		},
		{
			name:     "EMA indicator",
			input:    types.IndicatorTypeEMA,
			expected: strategy.IndicatorType_INDICATOR_EMA,
		},
		{
			name:     "Waddah Attar indicator",
			input:    types.IndicatorTypeWaddahAttar,
			expected: strategy.IndicatorType_INDICATOR_WADDAH_ATTAR,
		},
		{
			name:     "ATR indicator",
			input:    types.IndicatorTypeATR,
			expected: strategy.IndicatorType_INDICATOR_ATR,
		},
		{
			name:     "MA indicator",
			input:    types.IndicatorTypeMA,
			expected: strategy.IndicatorType_INDICATOR_MA,
		},
		{
			name:     "unknown defaults to RSI",
			input:    types.IndicatorType("unknown"),
			expected: strategy.IndicatorType_INDICATOR_RSI,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := IndicatorTypeToStrategyIndicatorType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestOrderTypeToStrategyOrderType(t *testing.T) {
	tests := []struct {
		name     string
		input    types.OrderType
		expected strategy.OrderType
	}{
		{
			name:     "market order type",
			input:    types.OrderTypeMarket,
			expected: strategy.OrderType_ORDER_TYPE_MARKET,
		},
		{
			name:     "limit order type",
			input:    types.OrderTypeLimit,
			expected: strategy.OrderType_ORDER_TYPE_LIMIT,
		},
		{
			name:     "unknown defaults to market",
			input:    types.OrderType("unknown"),
			expected: strategy.OrderType_ORDER_TYPE_MARKET,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := OrderTypeToStrategyOrderType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestMarkShapeToStrategyMarkShape(t *testing.T) {
	tests := []struct {
		name     string
		input    types.MarkShape
		expected strategy.MarkShape
	}{
		{
			name:     "circle shape",
			input:    types.MarkShapeCircle,
			expected: strategy.MarkShape_MARK_SHAPE_CIRCLE,
		},
		{
			name:     "square shape",
			input:    types.MarkShapeSquare,
			expected: strategy.MarkShape_MARK_SHAPE_SQUARE,
		},
		{
			name:     "triangle shape",
			input:    types.MarkShapeTriangle,
			expected: strategy.MarkShape_MARK_SHAPE_TRIANGLE,
		},
		{
			name:     "unknown defaults to circle",
			input:    types.MarkShape("unknown"),
			expected: strategy.MarkShape_MARK_SHAPE_CIRCLE,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := MarkShapeToStrategyMarkShape(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPurchaseTypeToStrategyPurchaseType(t *testing.T) {
	tests := []struct {
		name     string
		input    types.PurchaseType
		expected strategy.PurchaseType
	}{
		{
			name:     "buy purchase type",
			input:    types.PurchaseTypeBuy,
			expected: strategy.PurchaseType_PURCHASE_TYPE_BUY,
		},
		{
			name:     "sell purchase type",
			input:    types.PurchaseTypeSell,
			expected: strategy.PurchaseType_PURCHASE_TYPE_SELL,
		},
		{
			name:     "unknown defaults to buy",
			input:    types.PurchaseType("unknown"),
			expected: strategy.PurchaseType_PURCHASE_TYPE_BUY,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := PurchaseTypeToStrategyPurchaseType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestPositionTypeToStrategyPositionType(t *testing.T) {
	tests := []struct {
		name     string
		input    types.PositionType
		expected strategy.PositionType
	}{
		{
			name:     "long position type",
			input:    types.PositionTypeLong,
			expected: strategy.PositionType_POSITION_TYPE_LONG,
		},
		{
			name:     "short position type",
			input:    types.PositionTypeShort,
			expected: strategy.PositionType_POSITION_TYPE_SHORT,
		},
		{
			name:     "unknown defaults to long",
			input:    types.PositionType("unknown"),
			expected: strategy.PositionType_POSITION_TYPE_LONG,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := PositionTypeToStrategyPositionType(tc.input)
			assert.Equal(t, tc.expected, result)
		})
	}
}

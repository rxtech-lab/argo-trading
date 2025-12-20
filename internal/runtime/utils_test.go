package runtime

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
)

type RuntimeUtilsTestSuite struct {
	suite.Suite
}

func TestRuntimeUtilsSuite(t *testing.T) {
	suite.Run(t, new(RuntimeUtilsTestSuite))
}

func (suite *RuntimeUtilsTestSuite) TestStrategyIntervalToDataSourceInterval() {
	tests := []struct {
		name           string
		input          strategy.Interval
		expectedSome   bool
		expectedResult datasource.Interval
	}{
		{"1 minute", strategy.Interval_INTERVAL_1M, true, datasource.Interval1m},
		{"5 minutes", strategy.Interval_INTERVAL_5M, true, datasource.Interval5m},
		{"15 minutes", strategy.Interval_INTERVAL_15M, true, datasource.Interval15m},
		{"30 minutes", strategy.Interval_INTERVAL_30M, true, datasource.Interval30m},
		{"unknown interval", strategy.Interval_INTERVAL_UNSPECIFIED, false, ""},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyIntervalToDataSourceInterval(tc.input)
			if tc.expectedSome {
				suite.True(result.IsSome())
				suite.Equal(tc.expectedResult, result.Unwrap())
			} else {
				suite.True(result.IsNone())
			}
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategyPurchaseTypeToPurchaseType() {
	tests := []struct {
		name     string
		input    strategy.PurchaseType
		expected types.PurchaseType
	}{
		{"buy", strategy.PurchaseType_PURCHASE_TYPE_BUY, types.PurchaseTypeBuy},
		{"sell", strategy.PurchaseType_PURCHASE_TYPE_SELL, types.PurchaseTypeSell},
		{"unspecified defaults to buy", strategy.PurchaseType_PURCHASE_TYPE_UNSPECIFIED, types.PurchaseTypeBuy},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyPurchaseTypeToPurchaseType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategyPositionTypeToPositionType() {
	tests := []struct {
		name     string
		input    strategy.PositionType
		expected types.PositionType
	}{
		{"long", strategy.PositionType_POSITION_TYPE_LONG, types.PositionTypeLong},
		{"short", strategy.PositionType_POSITION_TYPE_SHORT, types.PositionTypeShort},
		{"unspecified defaults to long", strategy.PositionType_POSITION_TYPE_UNSPECIFIED, types.PositionTypeLong},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyPositionTypeToPositionType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategyOrderTypeToOrderType() {
	tests := []struct {
		name     string
		input    strategy.OrderType
		expected types.OrderType
	}{
		{"market", strategy.OrderType_ORDER_TYPE_MARKET, types.OrderTypeMarket},
		{"limit", strategy.OrderType_ORDER_TYPE_LIMIT, types.OrderTypeLimit},
		{"unspecified defaults to market", strategy.OrderType_ORDER_TYPE_UNSPECIFIED, types.OrderTypeMarket},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyOrderTypeToOrderType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategySignalTypeToSignalType() {
	tests := []struct {
		name     string
		input    strategy.SignalType
		expected types.SignalType
	}{
		{"buy long", strategy.SignalType_SIGNAL_TYPE_BUY_LONG, types.SignalTypeBuyLong},
		{"sell long", strategy.SignalType_SIGNAL_TYPE_SELL_LONG, types.SignalTypeSellLong},
		{"buy short", strategy.SignalType_SIGNAL_TYPE_BUY_SHORT, types.SignalTypeBuyShort},
		{"sell short", strategy.SignalType_SIGNAL_TYPE_SELL_SHORT, types.SignalTypeSellShort},
		{"unspecified defaults to no action", strategy.SignalType_SIGNAL_TYPE_UNSPECIFIED, types.SignalTypeNoAction},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategySignalTypeToSignalType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategyMarkShapeToMarkShape() {
	tests := []struct {
		name     string
		input    strategy.MarkShape
		expected types.MarkShape
	}{
		{"circle", strategy.MarkShape_MARK_SHAPE_CIRCLE, types.MarkShapeCircle},
		{"square", strategy.MarkShape_MARK_SHAPE_SQUARE, types.MarkShapeSquare},
		{"triangle", strategy.MarkShape_MARK_SHAPE_TRIANGLE, types.MarkShapeTriangle},
		{"unspecified defaults to circle", strategy.MarkShape_MARK_SHAPE_UNSPECIFIED, types.MarkShapeCircle},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyMarkShapeToMarkShape(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestStrategyIndicatorTypeToIndicatorType() {
	tests := []struct {
		name     string
		input    strategy.IndicatorType
		expected types.IndicatorType
	}{
		{"RSI", strategy.IndicatorType_INDICATOR_RSI, types.IndicatorTypeRSI},
		{"MACD", strategy.IndicatorType_INDICATOR_MACD, types.IndicatorTypeMACD},
		{"Williams R", strategy.IndicatorType_INDICATOR_WILLIAMS_R, types.IndicatorTypeWilliamsR},
		{"ADX", strategy.IndicatorType_INDICATOR_ADX, types.IndicatorTypeADX},
		{"CCI", strategy.IndicatorType_INDICATOR_CCI, types.IndicatorTypeCCI},
		{"AO", strategy.IndicatorType_INDICATOR_AO, types.IndicatorTypeAO},
		{"Trend Strength", strategy.IndicatorType_INDICATOR_TREND_STRENGTH, types.IndicatorTypeTrendStrength},
		{"Range Filter", strategy.IndicatorType_INDICATOR_RANGE_FILTER, types.IndicatorTypeRangeFilter},
		{"EMA", strategy.IndicatorType_INDICATOR_EMA, types.IndicatorTypeEMA},
		{"Waddah Attar", strategy.IndicatorType_INDICATOR_WADDAH_ATTAR, types.IndicatorTypeWaddahAttar},
		{"ATR", strategy.IndicatorType_INDICATOR_ATR, types.IndicatorTypeATR},
		{"MA", strategy.IndicatorType_INDICATOR_MA, types.IndicatorTypeMA},
		{"unspecified defaults to RSI", strategy.IndicatorType_INDICATOR_UNSPECIFIED, types.IndicatorTypeRSI},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := StrategyIndicatorTypeToIndicatorType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestSignalTypeToStrategySignalType() {
	tests := []struct {
		name     string
		input    types.SignalType
		expected strategy.SignalType
	}{
		{"buy long", types.SignalTypeBuyLong, strategy.SignalType_SIGNAL_TYPE_BUY_LONG},
		{"sell long", types.SignalTypeSellLong, strategy.SignalType_SIGNAL_TYPE_SELL_LONG},
		{"buy short", types.SignalTypeBuyShort, strategy.SignalType_SIGNAL_TYPE_BUY_SHORT},
		{"sell short", types.SignalTypeSellShort, strategy.SignalType_SIGNAL_TYPE_SELL_SHORT},
		{"no action", types.SignalTypeNoAction, strategy.SignalType_SIGNAL_TYPE_NO_ACTION},
		{"close position", types.SignalTypeClosePosition, strategy.SignalType_SIGNAL_TYPE_CLOSE_POSITION},
		{"wait", types.SignalTypeWait, strategy.SignalType_SIGNAL_TYPE_WAIT},
		{"abort", types.SignalTypeAbort, strategy.SignalType_SIGNAL_TYPE_ABORT},
		{"unknown defaults to no action", types.SignalType("unknown"), strategy.SignalType_SIGNAL_TYPE_NO_ACTION},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := SignalTypeToStrategySignalType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestIndicatorTypeToStrategyIndicatorType() {
	tests := []struct {
		name     string
		input    types.IndicatorType
		expected strategy.IndicatorType
	}{
		{"RSI", types.IndicatorTypeRSI, strategy.IndicatorType_INDICATOR_RSI},
		{"MACD", types.IndicatorTypeMACD, strategy.IndicatorType_INDICATOR_MACD},
		{"Williams R", types.IndicatorTypeWilliamsR, strategy.IndicatorType_INDICATOR_WILLIAMS_R},
		{"ADX", types.IndicatorTypeADX, strategy.IndicatorType_INDICATOR_ADX},
		{"CCI", types.IndicatorTypeCCI, strategy.IndicatorType_INDICATOR_CCI},
		{"AO", types.IndicatorTypeAO, strategy.IndicatorType_INDICATOR_AO},
		{"Trend Strength", types.IndicatorTypeTrendStrength, strategy.IndicatorType_INDICATOR_TREND_STRENGTH},
		{"Range Filter", types.IndicatorTypeRangeFilter, strategy.IndicatorType_INDICATOR_RANGE_FILTER},
		{"EMA", types.IndicatorTypeEMA, strategy.IndicatorType_INDICATOR_EMA},
		{"Waddah Attar", types.IndicatorTypeWaddahAttar, strategy.IndicatorType_INDICATOR_WADDAH_ATTAR},
		{"ATR", types.IndicatorTypeATR, strategy.IndicatorType_INDICATOR_ATR},
		{"MA", types.IndicatorTypeMA, strategy.IndicatorType_INDICATOR_MA},
		{"unknown defaults to RSI", types.IndicatorType("unknown"), strategy.IndicatorType_INDICATOR_RSI},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := IndicatorTypeToStrategyIndicatorType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestOrderTypeToStrategyOrderType() {
	tests := []struct {
		name     string
		input    types.OrderType
		expected strategy.OrderType
	}{
		{"market", types.OrderTypeMarket, strategy.OrderType_ORDER_TYPE_MARKET},
		{"limit", types.OrderTypeLimit, strategy.OrderType_ORDER_TYPE_LIMIT},
		{"unknown defaults to market", types.OrderType("unknown"), strategy.OrderType_ORDER_TYPE_MARKET},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := OrderTypeToStrategyOrderType(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *RuntimeUtilsTestSuite) TestMarkShapeToStrategyMarkShape() {
	tests := []struct {
		name     string
		input    types.MarkShape
		expected strategy.MarkShape
	}{
		{"circle", types.MarkShapeCircle, strategy.MarkShape_MARK_SHAPE_CIRCLE},
		{"square", types.MarkShapeSquare, strategy.MarkShape_MARK_SHAPE_SQUARE},
		{"triangle", types.MarkShapeTriangle, strategy.MarkShape_MARK_SHAPE_TRIANGLE},
		{"unknown defaults to circle", types.MarkShape("unknown"), strategy.MarkShape_MARK_SHAPE_CIRCLE},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := MarkShapeToStrategyMarkShape(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

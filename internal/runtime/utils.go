package runtime

import (
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

func StrategyIntervalToDataSourceInterval(interval strategy.Interval) optional.Option[datasource.Interval] {
	switch interval {
	case strategy.Interval_INTERVAL_1M:
		return optional.Some(datasource.Interval1m)
	case strategy.Interval_INTERVAL_5M:
		return optional.Some(datasource.Interval5m)
	case strategy.Interval_INTERVAL_15M:
		return optional.Some(datasource.Interval15m)
	case strategy.Interval_INTERVAL_30M:
		return optional.Some(datasource.Interval30m)
	default:
		return optional.None[datasource.Interval]()
	}
}

func StrategyPurchaseTypeToPurchaseType(purchaseType strategy.PurchaseType) types.PurchaseType {
	switch purchaseType {
	case strategy.PurchaseType_PURCHASE_TYPE_BUY:
		return types.PurchaseTypeBuy
	case strategy.PurchaseType_PURCHASE_TYPE_SELL:
		return types.PurchaseTypeSell
	default:
		return types.PurchaseTypeBuy
	}
}

func StrategyPositionTypeToPositionType(positionType strategy.PositionType) types.PositionType {
	switch positionType {
	case strategy.PositionType_POSITION_TYPE_LONG:
		return types.PositionTypeLong
	case strategy.PositionType_POSITION_TYPE_SHORT:
		return types.PositionTypeShort
	default:
		return types.PositionTypeLong
	}
}

func StrategyOrderTypeToOrderType(orderType strategy.OrderType) types.OrderType {
	switch orderType {
	case strategy.OrderType_ORDER_TYPE_MARKET:
		return types.OrderTypeMarket
	case strategy.OrderType_ORDER_TYPE_LIMIT:
		return types.OrderTypeLimit
	default:
		return types.OrderTypeMarket
	}
}

func StrategySignalTypeToSignalType(signalType strategy.SignalType) types.SignalType {
	switch signalType {
	case strategy.SignalType_SIGNAL_TYPE_BUY_LONG:
		return types.SignalTypeBuyLong
	case strategy.SignalType_SIGNAL_TYPE_SELL_LONG:
		return types.SignalTypeSellLong
	case strategy.SignalType_SIGNAL_TYPE_BUY_SHORT:
		return types.SignalTypeBuyShort
	case strategy.SignalType_SIGNAL_TYPE_SELL_SHORT:
		return types.SignalTypeSellShort
	default:
		return types.SignalTypeNoAction
	}
}

func StrategyMarkShapeToMarkShape(markShape strategy.MarkShape) types.MarkShape {
	switch markShape {
	case strategy.MarkShape_MARK_SHAPE_CIRCLE:
		return types.MarkShapeCircle
	case strategy.MarkShape_MARK_SHAPE_SQUARE:
		return types.MarkShapeSquare
	case strategy.MarkShape_MARK_SHAPE_TRIANGLE:
		return types.MarkShapeTriangle
	default:
		return types.MarkShapeCircle
	}
}

func StrategyIndicatorTypeToIndicatorType(indicatorType strategy.IndicatorType) types.IndicatorType {
	switch indicatorType {
	case strategy.IndicatorType_INDICATOR_RSI:
		return types.IndicatorTypeRSI
	case strategy.IndicatorType_INDICATOR_MACD:
		return types.IndicatorTypeMACD
	case strategy.IndicatorType_INDICATOR_WILLIAMS_R:
		return types.IndicatorTypeWilliamsR
	case strategy.IndicatorType_INDICATOR_ADX:
		return types.IndicatorTypeADX
	case strategy.IndicatorType_INDICATOR_CCI:
		return types.IndicatorTypeCCI
	case strategy.IndicatorType_INDICATOR_AO:
		return types.IndicatorTypeAO
	case strategy.IndicatorType_INDICATOR_TREND_STRENGTH:
		return types.IndicatorTypeTrendStrength
	case strategy.IndicatorType_INDICATOR_RANGE_FILTER:
		return types.IndicatorTypeRangeFilter
	case strategy.IndicatorType_INDICATOR_EMA:
		return types.IndicatorTypeEMA
	case strategy.IndicatorType_INDICATOR_WADDAH_ATTAR:
		return types.IndicatorTypeWaddahAttar
	case strategy.IndicatorType_INDICATOR_ATR:
		return types.IndicatorTypeATR
	case strategy.IndicatorType_INDICATOR_MA:
		return types.IndicatorTypeMA
	default:
		return types.IndicatorTypeRSI
	}
}

func SignalTypeToStrategySignalType(signalType types.SignalType) strategy.SignalType {
	switch signalType {
	case types.SignalTypeBuyLong:
		return strategy.SignalType_SIGNAL_TYPE_BUY_LONG
	case types.SignalTypeSellLong:
		return strategy.SignalType_SIGNAL_TYPE_SELL_LONG
	case types.SignalTypeBuyShort:
		return strategy.SignalType_SIGNAL_TYPE_BUY_SHORT
	case types.SignalTypeSellShort:
		return strategy.SignalType_SIGNAL_TYPE_SELL_SHORT
	case types.SignalTypeNoAction:
		return strategy.SignalType_SIGNAL_TYPE_NO_ACTION
	case types.SignalTypeClosePosition:
		return strategy.SignalType_SIGNAL_TYPE_CLOSE_POSITION
	case types.SignalTypeWait:
		return strategy.SignalType_SIGNAL_TYPE_WAIT
	case types.SignalTypeAbort:
		return strategy.SignalType_SIGNAL_TYPE_ABORT
	default:
		return strategy.SignalType_SIGNAL_TYPE_NO_ACTION
	}
}

func IndicatorTypeToStrategyIndicatorType(indicatorType types.IndicatorType) strategy.IndicatorType {
	switch indicatorType {
	case types.IndicatorTypeRSI:
		return strategy.IndicatorType_INDICATOR_RSI
	case types.IndicatorTypeMACD:
		return strategy.IndicatorType_INDICATOR_MACD
	case types.IndicatorTypeWilliamsR:
		return strategy.IndicatorType_INDICATOR_WILLIAMS_R
	case types.IndicatorTypeADX:
		return strategy.IndicatorType_INDICATOR_ADX
	case types.IndicatorTypeCCI:
		return strategy.IndicatorType_INDICATOR_CCI
	case types.IndicatorTypeAO:
		return strategy.IndicatorType_INDICATOR_AO
	case types.IndicatorTypeTrendStrength:
		return strategy.IndicatorType_INDICATOR_TREND_STRENGTH
	case types.IndicatorTypeRangeFilter:
		return strategy.IndicatorType_INDICATOR_RANGE_FILTER
	case types.IndicatorTypeEMA:
		return strategy.IndicatorType_INDICATOR_EMA
	case types.IndicatorTypeWaddahAttar:
		return strategy.IndicatorType_INDICATOR_WADDAH_ATTAR
	case types.IndicatorTypeATR:
		return strategy.IndicatorType_INDICATOR_ATR
	case types.IndicatorTypeMA:
		return strategy.IndicatorType_INDICATOR_MA
	default:
		return strategy.IndicatorType_INDICATOR_RSI
	}
}

func OrderTypeToStrategyOrderType(orderType types.OrderType) strategy.OrderType {
	switch orderType {
	case types.OrderTypeMarket:
		return strategy.OrderType_ORDER_TYPE_MARKET
	case types.OrderTypeLimit:
		return strategy.OrderType_ORDER_TYPE_LIMIT
	default:
		return strategy.OrderType_ORDER_TYPE_MARKET
	}
}

func MarkShapeToStrategyMarkShape(markShape types.MarkShape) strategy.MarkShape {
	switch markShape {
	case types.MarkShapeCircle:
		return strategy.MarkShape_MARK_SHAPE_CIRCLE
	case types.MarkShapeSquare:
		return strategy.MarkShape_MARK_SHAPE_SQUARE
	case types.MarkShapeTriangle:
		return strategy.MarkShape_MARK_SHAPE_TRIANGLE
	default:
		return strategy.MarkShape_MARK_SHAPE_CIRCLE
	}
}

func PurchaseTypeToStrategyPurchaseType(pt types.PurchaseType) strategy.PurchaseType {
	switch pt {
	case types.PurchaseTypeBuy:
		return strategy.PurchaseType_PURCHASE_TYPE_BUY
	case types.PurchaseTypeSell:
		return strategy.PurchaseType_PURCHASE_TYPE_SELL
	default:
		return strategy.PurchaseType_PURCHASE_TYPE_BUY
	}
}

func PositionTypeToStrategyPositionType(pt types.PositionType) strategy.PositionType {
	switch pt {
	case types.PositionTypeLong:
		return strategy.PositionType_POSITION_TYPE_LONG
	case types.PositionTypeShort:
		return strategy.PositionType_POSITION_TYPE_SHORT
	default:
		return strategy.PositionType_POSITION_TYPE_LONG
	}
}

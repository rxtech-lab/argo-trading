package runtime

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	i "github.com/rxtech-lab/argo-trading/internal/indicator"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type strategyApiForWasm struct {
	runtimeContext *RuntimeContext
}

// CancelAllOrders implements strategy.StrategyApi.
func (s strategyApiForWasm) CancelAllOrders(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := (s.runtimeContext.TradingSystem).CancelAllOrders()
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// CancelOrder implements strategy.StrategyApi.
func (s strategyApiForWasm) CancelOrder(ctx context.Context, req *strategy.CancelOrderRequest) (*emptypb.Empty, error) {
	err := (s.runtimeContext.TradingSystem).CancelOrder(req.OrderId)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// ConfigureIndicator implements strategy.StrategyApi.
func (s strategyApiForWasm) ConfigureIndicator(ctx context.Context, req *strategy.ConfigureRequest) (*emptypb.Empty, error) {
	registry := s.runtimeContext.IndicatorRegistry
	indicator, err := registry.GetIndicator(types.IndicatorType(req.IndicatorType))
	if err != nil {
		return nil, err
	}
	err = indicator.Config(req.Config)
	if err != nil {
		return nil, err
	}
	return &emptypb.Empty{}, nil
}

// Count implements strategy.StrategyApi.
func (s strategyApiForWasm) Count(ctx context.Context, req *strategy.CountRequest) (*strategy.CountResponse, error) {
	startTime := optional.Some(req.StartTime.AsTime())
	endTime := optional.Some(req.EndTime.AsTime())
	count, err := s.runtimeContext.DataSource.Count(startTime, endTime)
	if err != nil {
		return nil, err
	}
	return &strategy.CountResponse{
		Count: int32(count),
	}, nil
}

// ExecuteSQL implements strategy.StrategyApi.
func (s strategyApiForWasm) ExecuteSQL(ctx context.Context, req *strategy.ExecuteSQLRequest) (*strategy.ExecuteSQLResponse, error) {
	params := make([]interface{}, len(req.Params))
	for i, param := range req.Params {
		params[i] = param
	}
	results, err := s.runtimeContext.DataSource.ExecuteSQL(req.Query, params...)
	if err != nil {
		return nil, err
	}

	response := &strategy.ExecuteSQLResponse{
		Results: make([]*strategy.SQLResult, len(results)),
	}

	for i, result := range results {
		fields := make(map[string]string)
		for k, v := range result.Values {
			if strVal, ok := v.(string); ok {
				fields[k] = strVal
			} else {
				fields[k] = fmt.Sprintf("%v", v)
			}
		}
		response.Results[i] = &strategy.SQLResult{
			Fields: fields,
		}
	}

	return response, nil
}

// GetCache implements strategy.StrategyApi.
func (s strategyApiForWasm) GetCache(ctx context.Context, req *strategy.GetRequest) (*strategy.GetResponse, error) {
	cache := s.runtimeContext.Cache

	value, ok := cache.Get(req.Key)
	if !ok {
		return nil, fmt.Errorf("cache key not found: %s", req.Key)
	}
	// json marshal
	jsonValue, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}

	return &strategy.GetResponse{
		Value: string(jsonValue),
	}, nil
}

// GetMarkers implements strategy.StrategyApi.
func (s strategyApiForWasm) GetMarkers(ctx context.Context, _ *emptypb.Empty) (*strategy.GetMarkersResponse, error) {
	markers, err := (s.runtimeContext.Marker).GetMarkers()
	if err != nil {
		return nil, err
	}

	response := &strategy.GetMarkersResponse{
		Markers: make([]*strategy.Mark, len(markers)),
	}

	for i, marker := range markers {
		marketData := &strategy.MarketData{
			Symbol: marker.Signal.Symbol,
			High:   0, // These fields are not available in the Signal type
			Low:    0,
			Open:   0,
			Close:  0,
			Volume: 0,
			Time:   timestamppb.New(marker.Signal.Time),
		}

		var signalType strategy.SignalType
		switch marker.Signal.Type {
		case types.SignalTypeBuyLong:
			signalType = strategy.SignalType_SIGNAL_TYPE_BUY_LONG
		case types.SignalTypeSellLong:
			signalType = strategy.SignalType_SIGNAL_TYPE_SELL_LONG
		case types.SignalTypeBuyShort:
			signalType = strategy.SignalType_SIGNAL_TYPE_BUY_SHORT
		case types.SignalTypeSellShort:
			signalType = strategy.SignalType_SIGNAL_TYPE_SELL_SHORT
		case types.SignalTypeNoAction:
			signalType = strategy.SignalType_SIGNAL_TYPE_NO_ACTION
		case types.SignalTypeClosePosition:
			signalType = strategy.SignalType_SIGNAL_TYPE_CLOSE_POSITION
		case types.SignalTypeWait:
			signalType = strategy.SignalType_SIGNAL_TYPE_WAIT
		case types.SignalTypeAbort:
			signalType = strategy.SignalType_SIGNAL_TYPE_ABORT
		default:
			signalType = strategy.SignalType_SIGNAL_TYPE_NO_ACTION
		}

		response.Markers[i] = &strategy.Mark{
			MarketData: marketData,
			Signal:     signalType,
			Reason:     marker.Reason,
			Timestamp:  timestamppb.New(marker.Signal.Time),
		}
	}

	return response, nil
}

// GetOrderStatus implements strategy.StrategyApi.
func (s strategyApiForWasm) GetOrderStatus(ctx context.Context, req *strategy.GetOrderStatusRequest) (*strategy.GetOrderStatusResponse, error) {
	status, err := (s.runtimeContext.TradingSystem).GetOrderStatus(req.OrderId)
	if err != nil {
		return nil, err
	}

	var orderStatus strategy.OrderStatus
	switch status {
	case types.OrderStatusPending:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_PENDING
	case types.OrderStatusFilled:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_FILLED
	case types.OrderStatusCancelled:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_CANCELLED
	case types.OrderStatusRejected:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_REJECTED
	case types.OrderStatusFailed:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_FAILED
	default:
		orderStatus = strategy.OrderStatus_ORDER_STATUS_PENDING
	}

	return &strategy.GetOrderStatusResponse{
		Status: orderStatus,
	}, nil
}

// GetPosition implements strategy.StrategyApi.
func (s strategyApiForWasm) GetPosition(ctx context.Context, req *strategy.GetPositionRequest) (*strategy.Position, error) {
	position, err := (s.runtimeContext.TradingSystem).GetPosition(req.Symbol)
	if err != nil {
		return nil, err
	}

	return &strategy.Position{
		Symbol:           position.Symbol,
		Quantity:         position.Quantity,
		TotalInQuantity:  position.TotalInQuantity,
		TotalOutQuantity: position.TotalOutQuantity,
		TotalInAmount:    position.TotalInAmount,
		TotalOutAmount:   position.TotalOutAmount,
		TotalInFee:       position.TotalInFee,
		TotalOutFee:      position.TotalOutFee,
		OpenTimestamp:    timestamppb.New(position.OpenTimestamp),
		StrategyName:     position.StrategyName,
	}, nil
}

// GetPositions implements strategy.StrategyApi.
func (s strategyApiForWasm) GetPositions(ctx context.Context, _ *emptypb.Empty) (*strategy.GetPositionsResponse, error) {
	positions, err := (s.runtimeContext.TradingSystem).GetPositions()
	if err != nil {
		return nil, err
	}

	response := &strategy.GetPositionsResponse{
		Positions: make([]*strategy.Position, len(positions)),
	}

	for i, position := range positions {
		response.Positions[i] = &strategy.Position{
			Symbol:           position.Symbol,
			Quantity:         position.Quantity,
			TotalInQuantity:  position.TotalInQuantity,
			TotalOutQuantity: position.TotalOutQuantity,
			TotalInAmount:    position.TotalInAmount,
			TotalOutAmount:   position.TotalOutAmount,
			TotalInFee:       position.TotalInFee,
			TotalOutFee:      position.TotalOutFee,
			OpenTimestamp:    timestamppb.New(position.OpenTimestamp),
			StrategyName:     position.StrategyName,
		}
	}

	return response, nil
}

// GetRange implements strategy.StrategyApi.
func (s strategyApiForWasm) GetRange(ctx context.Context, req *strategy.GetRangeRequest) (*strategy.GetRangeResponse, error) {
	intervalValue := strategyIntervalToDataSourceInterval(req.Interval)
	data, err := s.runtimeContext.DataSource.GetRange(req.StartTime.AsTime(), req.EndTime.AsTime(), intervalValue)
	if err != nil {
		return nil, err
	}

	response := &strategy.GetRangeResponse{
		Data: make([]*strategy.MarketData, len(data)),
	}

	for i, d := range data {
		response.Data[i] = &strategy.MarketData{
			Symbol: d.Symbol,
			High:   d.High,
			Low:    d.Low,
			Open:   d.Open,
			Close:  d.Close,
			Volume: d.Volume,
			Time:   timestamppb.New(d.Time),
		}
	}

	return response, nil
}

// GetSignal implements strategy.StrategyApi.
func (s strategyApiForWasm) GetSignal(ctx context.Context, req *strategy.GetSignalRequest) (*strategy.GetSignalResponse, error) {
	registry := s.runtimeContext.IndicatorRegistry
	indicator, err := registry.GetIndicator(types.IndicatorType(req.IndicatorType))
	if err != nil {
		return nil, err
	}

	marketData := types.MarketData{
		Symbol: req.MarketData.Symbol,
		High:   req.MarketData.High,
		Low:    req.MarketData.Low,
		Open:   req.MarketData.Open,
		Close:  req.MarketData.Close,
		Volume: req.MarketData.Volume,
		Time:   req.MarketData.Time.AsTime(),
	}

	cache := (s.runtimeContext.Cache).(*cache.CacheV1)
	indicatorContext := i.IndicatorContext{
		DataSource:        s.runtimeContext.DataSource,
		IndicatorRegistry: s.runtimeContext.IndicatorRegistry,
		Cache:             cache,
	}

	signal, err := indicator.GetSignal(marketData, indicatorContext)
	if err != nil {
		return nil, err
	}

	// stringify signal.RawValue
	rawValue, err := json.Marshal(signal.RawValue)
	if err != nil {
		return nil, err
	}

	return &strategy.GetSignalResponse{
		Timestamp:     timestamppb.New(signal.Time),
		Type:          signalTypeToStrategySignalType(signal.Type),
		Name:          signal.Name,
		Reason:        signal.Reason,
		RawValue:      string(rawValue),
		Symbol:        signal.Symbol,
		IndicatorType: strategy.IndicatorType(req.IndicatorType),
	}, nil
}

// Mark implements strategy.StrategyApi.
func (s strategyApiForWasm) Mark(ctx context.Context, req *strategy.MarkRequest) (*emptypb.Empty, error) {
	panic("not implemented")
}

// PlaceMultipleOrders implements strategy.StrategyApi.
func (s strategyApiForWasm) PlaceMultipleOrders(ctx context.Context, req *strategy.PlaceMultipleOrdersRequest) (*emptypb.Empty, error) {
	orders := make([]types.ExecuteOrder, len(req.Orders))
	for i, order := range req.Orders {
		orders[i] = types.ExecuteOrder{
			ID:           order.Id,
			Symbol:       order.Symbol,
			Side:         strategyPurchaseTypeToPurchaseType(order.Side),
			OrderType:    strategyOrderTypeToOrderType(order.OrderType),
			Price:        order.Price,
			StrategyName: order.StrategyName,
			Quantity:     order.Quantity,
			Reason: types.Reason{
				Reason:  order.Reason.Reason,
				Message: order.Reason.Message,
			},
		}

		if order.TakeProfit != nil {
			orders[i].TakeProfit = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
				Symbol:    order.TakeProfit.Symbol,
				Side:      strategyPurchaseTypeToPurchaseType(order.TakeProfit.Side),
				OrderType: strategyOrderTypeToOrderType(order.TakeProfit.OrderType),
			})
		}

		if order.StopLoss != nil {
			orders[i].StopLoss = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
				Symbol:    order.StopLoss.Symbol,
				Side:      strategyPurchaseTypeToPurchaseType(order.StopLoss.Side),
				OrderType: strategyOrderTypeToOrderType(order.StopLoss.OrderType),
			})
		}
	}

	err := (s.runtimeContext.TradingSystem).PlaceMultipleOrders(orders)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// PlaceOrder implements strategy.StrategyApi.
func (s strategyApiForWasm) PlaceOrder(ctx context.Context, req *strategy.ExecuteOrder) (*emptypb.Empty, error) {
	order := types.ExecuteOrder{
		ID:           req.Id,
		Symbol:       req.Symbol,
		Side:         strategyPurchaseTypeToPurchaseType(req.Side),
		OrderType:    strategyOrderTypeToOrderType(req.OrderType),
		Price:        req.Price,
		StrategyName: req.StrategyName,
		Quantity:     req.Quantity,
		Reason: types.Reason{
			Reason:  req.Reason.Reason,
			Message: req.Reason.Message,
		},
	}

	if req.TakeProfit != nil {
		order.TakeProfit = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
			Symbol:    req.TakeProfit.Symbol,
			Side:      strategyPurchaseTypeToPurchaseType(req.TakeProfit.Side),
			OrderType: strategyOrderTypeToOrderType(req.TakeProfit.OrderType),
		})
	}

	if req.StopLoss != nil {
		order.StopLoss = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
			Symbol:    req.StopLoss.Symbol,
			Side:      strategyPurchaseTypeToPurchaseType(req.StopLoss.Side),
			OrderType: strategyOrderTypeToOrderType(req.StopLoss.OrderType),
		})
	}

	err := (s.runtimeContext.TradingSystem).PlaceOrder(order)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ReadLastData implements strategy.StrategyApi.
func (s strategyApiForWasm) ReadLastData(ctx context.Context, req *strategy.ReadLastDataRequest) (*strategy.MarketData, error) {
	data, err := s.runtimeContext.DataSource.ReadLastData(req.Symbol)
	if err != nil {
		return nil, err
	}

	return &strategy.MarketData{
		Symbol: data.Symbol,
		High:   data.High,
		Low:    data.Low,
		Open:   data.Open,
		Close:  data.Close,
		Volume: data.Volume,
		Time:   timestamppb.New(data.Time),
	}, nil
}

// SetCache implements strategy.StrategyApi.
func (s strategyApiForWasm) SetCache(ctx context.Context, req *strategy.SetRequest) (*emptypb.Empty, error) {
	cache := s.runtimeContext.Cache

	err := (cache).Set(req.Key, req.Value)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func NewWasmStrategyApi(ctx *RuntimeContext) strategy.StrategyApi {
	return strategyApiForWasm{
		runtimeContext: ctx,
	}
}

// GetStrategyApi returns the strategy api that provides host functions
// for wasm runtime strategies to interact with the trading system.
func (r *RuntimeContext) GetStrategyApiForWasm() strategy.StrategyApi {
	return NewWasmStrategyApi(r)
}

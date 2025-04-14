package wasm

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/moznion/go-optional"
	i "github.com/rxtech-lab/argo-trading/internal/indicator"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type StrategyApiForWasm struct {
	runtimeContext *runtime.RuntimeContext
}

// CancelAllOrders implements strategy.StrategyApi.
func (s StrategyApiForWasm) CancelAllOrders(ctx context.Context, _ *emptypb.Empty) (*emptypb.Empty, error) {
	err := (s.runtimeContext.TradingSystem).CancelAllOrders()
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// CancelOrder implements strategy.StrategyApi.
func (s StrategyApiForWasm) CancelOrder(ctx context.Context, req *strategy.CancelOrderRequest) (*emptypb.Empty, error) {
	err := (s.runtimeContext.TradingSystem).CancelOrder(req.OrderId)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ConfigureIndicator implements strategy.StrategyApi.
func (s StrategyApiForWasm) ConfigureIndicator(ctx context.Context, req *strategy.ConfigureRequest) (*emptypb.Empty, error) {
	registry := s.runtimeContext.IndicatorRegistry

	indicator, err := registry.GetIndicator(runtime.StrategyIndicatorTypeToIndicatorType(req.IndicatorType))
	if err != nil {
		return nil, err
	}
	// JSON unmarshal config to any[]
	var configArray []any

	err = json.Unmarshal([]byte(req.Config), &configArray)
	if err != nil {
		return nil, err
	}

	err = indicator.Config(configArray...)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// Count implements strategy.StrategyApi.
func (s StrategyApiForWasm) Count(ctx context.Context, req *strategy.CountRequest) (*strategy.CountResponse, error) {
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
func (s StrategyApiForWasm) ExecuteSQL(ctx context.Context, req *strategy.ExecuteSQLRequest) (*strategy.ExecuteSQLResponse, error) {
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
func (s StrategyApiForWasm) GetCache(ctx context.Context, req *strategy.GetRequest) (*strategy.GetResponse, error) {
	cache := s.runtimeContext.Cache

	value, ok := cache.Get(req.Key)
	if !ok {
		return &strategy.GetResponse{}, nil
	}
	// check if value is a string
	if strVal, ok := value.(string); ok {
		return &strategy.GetResponse{
			Value: strVal,
		}, nil
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
func (s StrategyApiForWasm) GetMarkers(ctx context.Context, _ *emptypb.Empty) (*strategy.GetMarkersResponse, error) {
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
func (s StrategyApiForWasm) GetOrderStatus(ctx context.Context, req *strategy.GetOrderStatusRequest) (*strategy.GetOrderStatusResponse, error) {
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
func (s StrategyApiForWasm) GetPosition(ctx context.Context, req *strategy.GetPositionRequest) (*strategy.Position, error) {
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
func (s StrategyApiForWasm) GetPositions(ctx context.Context, _ *emptypb.Empty) (*strategy.GetPositionsResponse, error) {
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
func (s StrategyApiForWasm) GetRange(ctx context.Context, req *strategy.GetRangeRequest) (*strategy.GetRangeResponse, error) {
	intervalValue := runtime.StrategyIntervalToDataSourceInterval(req.Interval)

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
func (s StrategyApiForWasm) GetSignal(ctx context.Context, req *strategy.GetSignalRequest) (*strategy.GetSignalResponse, error) {
	registry := s.runtimeContext.IndicatorRegistry

	indicator, err := registry.GetIndicator(runtime.StrategyIndicatorTypeToIndicatorType(req.IndicatorType))
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

	indicatorContext := i.IndicatorContext{
		DataSource:        s.runtimeContext.DataSource,
		IndicatorRegistry: s.runtimeContext.IndicatorRegistry,
		Cache:             s.runtimeContext.Cache,
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
		Type:          runtime.SignalTypeToStrategySignalType(signal.Type),
		Name:          signal.Name,
		Reason:        signal.Reason,
		RawValue:      string(rawValue),
		Symbol:        signal.Symbol,
		IndicatorType: req.IndicatorType,
	}, nil
}

// Mark implements strategy.StrategyApi.
func (s StrategyApiForWasm) Mark(ctx context.Context, req *strategy.MarkRequest) (*emptypb.Empty, error) {
	panic("not implemented")
}

// PlaceMultipleOrders implements strategy.StrategyApi.
func (s StrategyApiForWasm) PlaceMultipleOrders(ctx context.Context, req *strategy.PlaceMultipleOrdersRequest) (*emptypb.Empty, error) {
	orders := make([]types.ExecuteOrder, len(req.Orders))
	for i, order := range req.Orders {
		orders[i] = types.ExecuteOrder{
			ID:           order.Id,
			Symbol:       order.Symbol,
			Side:         runtime.StrategyPurchaseTypeToPurchaseType(order.Side),
			OrderType:    runtime.StrategyOrderTypeToOrderType(order.OrderType),
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
				Side:      runtime.StrategyPurchaseTypeToPurchaseType(order.TakeProfit.Side),
				OrderType: runtime.StrategyOrderTypeToOrderType(order.TakeProfit.OrderType),
			})
		}

		if order.StopLoss != nil {
			orders[i].StopLoss = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
				Symbol:    order.StopLoss.Symbol,
				Side:      runtime.StrategyPurchaseTypeToPurchaseType(order.StopLoss.Side),
				OrderType: runtime.StrategyOrderTypeToOrderType(order.StopLoss.OrderType),
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
func (s StrategyApiForWasm) PlaceOrder(ctx context.Context, req *strategy.ExecuteOrder) (*emptypb.Empty, error) {
	var reasonName string

	var reasonMessage string

	if req.Reason != nil {
		reasonName = req.Reason.Reason
		reasonMessage = req.Reason.Message
	}

	order := types.ExecuteOrder{
		ID:           req.Id,
		Symbol:       req.Symbol,
		Side:         runtime.StrategyPurchaseTypeToPurchaseType(req.Side),
		OrderType:    runtime.StrategyOrderTypeToOrderType(req.OrderType),
		Price:        req.Price,
		StrategyName: req.StrategyName,
		Quantity:     req.Quantity,
		Reason: types.Reason{
			Reason:  reasonName,
			Message: reasonMessage,
		},
	}

	if req.TakeProfit != nil {
		order.TakeProfit = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
			Symbol:    req.TakeProfit.Symbol,
			Side:      runtime.StrategyPurchaseTypeToPurchaseType(req.TakeProfit.Side),
			OrderType: runtime.StrategyOrderTypeToOrderType(req.TakeProfit.OrderType),
		})
	}

	if req.StopLoss != nil {
		order.StopLoss = optional.Some(types.ExecuteOrderTakeProfitOrStopLoss{
			Symbol:    req.StopLoss.Symbol,
			Side:      runtime.StrategyPurchaseTypeToPurchaseType(req.StopLoss.Side),
			OrderType: runtime.StrategyOrderTypeToOrderType(req.StopLoss.OrderType),
		})
	}

	err := (s.runtimeContext.TradingSystem).PlaceOrder(order)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// ReadLastData implements strategy.StrategyApi.
func (s StrategyApiForWasm) ReadLastData(ctx context.Context, req *strategy.ReadLastDataRequest) (*strategy.MarketData, error) {
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
func (s StrategyApiForWasm) SetCache(ctx context.Context, req *strategy.SetRequest) (*emptypb.Empty, error) {
	cache := s.runtimeContext.Cache

	err := (cache).Set(req.Key, req.Value)
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

func NewWasmStrategyApi(ctx *runtime.RuntimeContext) strategy.StrategyApi {
	return StrategyApiForWasm{
		runtimeContext: ctx,
	}
}

// GetStrategyApi returns the strategy api that provides host functions
// for wasm runtime strategies to interact with the trading system.
func GetStrategyApiForWasm(r *runtime.RuntimeContext) strategy.StrategyApi {
	return NewWasmStrategyApi(r)
}

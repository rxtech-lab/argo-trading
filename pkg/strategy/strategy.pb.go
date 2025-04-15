// Code generated by protoc-gen-go-plugin. DO NOT EDIT.
// versions:
// 	protoc-gen-go-plugin v0.1.0
// 	protoc               v5.29.3
// source: strategy.proto

package strategy

import (
	context "context"
	emptypb "github.com/knqyf263/go-plugin/types/known/emptypb"
	timestamppb "github.com/knqyf263/go-plugin/types/known/timestamppb"
	protoreflect "google.golang.org/protobuf/reflect/protoreflect"
	protoimpl "google.golang.org/protobuf/runtime/protoimpl"
)

const (
	// Verify that this generated code is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(20 - protoimpl.MinVersion)
	// Verify that runtime/protoimpl is sufficiently up-to-date.
	_ = protoimpl.EnforceVersion(protoimpl.MaxVersion - 20)
)

type Interval int32

const (
	Interval_INTERVAL_1M     Interval = 0
	Interval_INTERVAL_5M     Interval = 1
	Interval_INTERVAL_15M    Interval = 2
	Interval_INTERVAL_30M    Interval = 3
	Interval_INTERVAL_1H     Interval = 4
	Interval_INTERVAL_4H     Interval = 5
	Interval_INTERVAL_6H     Interval = 6
	Interval_INTERVAL_8H     Interval = 7
	Interval_INTERVAL_12H    Interval = 8
	Interval_INTERVAL_1D     Interval = 9
	Interval_INTERVAL_1W     Interval = 10
	Interval_INTERVAL_1MONTH Interval = 11
)

// Enum value maps for Interval.
var (
	Interval_name = map[int32]string{
		0:  "INTERVAL_1M",
		1:  "INTERVAL_5M",
		2:  "INTERVAL_15M",
		3:  "INTERVAL_30M",
		4:  "INTERVAL_1H",
		5:  "INTERVAL_4H",
		6:  "INTERVAL_6H",
		7:  "INTERVAL_8H",
		8:  "INTERVAL_12H",
		9:  "INTERVAL_1D",
		10: "INTERVAL_1W",
		11: "INTERVAL_1MONTH",
	}
	Interval_value = map[string]int32{
		"INTERVAL_1M":     0,
		"INTERVAL_5M":     1,
		"INTERVAL_15M":    2,
		"INTERVAL_30M":    3,
		"INTERVAL_1H":     4,
		"INTERVAL_4H":     5,
		"INTERVAL_6H":     6,
		"INTERVAL_8H":     7,
		"INTERVAL_12H":    8,
		"INTERVAL_1D":     9,
		"INTERVAL_1W":     10,
		"INTERVAL_1MONTH": 11,
	}
)

func (x Interval) Enum() *Interval {
	p := new(Interval)
	*p = x
	return p
}

type PositionType int32

const (
	PositionType_POSITION_TYPE_LONG  PositionType = 0
	PositionType_POSITION_TYPE_SHORT PositionType = 1
)

// Enum value maps for PositionType.
var (
	PositionType_name = map[int32]string{
		0: "POSITION_TYPE_LONG",
		1: "POSITION_TYPE_SHORT",
	}
	PositionType_value = map[string]int32{
		"POSITION_TYPE_LONG":  0,
		"POSITION_TYPE_SHORT": 1,
	}
)

func (x PositionType) Enum() *PositionType {
	p := new(PositionType)
	*p = x
	return p
}

type SignalType int32

const (
	SignalType_SIGNAL_TYPE_BUY_LONG       SignalType = 0
	SignalType_SIGNAL_TYPE_SELL_LONG      SignalType = 1
	SignalType_SIGNAL_TYPE_BUY_SHORT      SignalType = 2
	SignalType_SIGNAL_TYPE_SELL_SHORT     SignalType = 3
	SignalType_SIGNAL_TYPE_NO_ACTION      SignalType = 4
	SignalType_SIGNAL_TYPE_CLOSE_POSITION SignalType = 5
	SignalType_SIGNAL_TYPE_WAIT           SignalType = 6
	SignalType_SIGNAL_TYPE_ABORT          SignalType = 7
)

// Enum value maps for SignalType.
var (
	SignalType_name = map[int32]string{
		0: "SIGNAL_TYPE_BUY_LONG",
		1: "SIGNAL_TYPE_SELL_LONG",
		2: "SIGNAL_TYPE_BUY_SHORT",
		3: "SIGNAL_TYPE_SELL_SHORT",
		4: "SIGNAL_TYPE_NO_ACTION",
		5: "SIGNAL_TYPE_CLOSE_POSITION",
		6: "SIGNAL_TYPE_WAIT",
		7: "SIGNAL_TYPE_ABORT",
	}
	SignalType_value = map[string]int32{
		"SIGNAL_TYPE_BUY_LONG":       0,
		"SIGNAL_TYPE_SELL_LONG":      1,
		"SIGNAL_TYPE_BUY_SHORT":      2,
		"SIGNAL_TYPE_SELL_SHORT":     3,
		"SIGNAL_TYPE_NO_ACTION":      4,
		"SIGNAL_TYPE_CLOSE_POSITION": 5,
		"SIGNAL_TYPE_WAIT":           6,
		"SIGNAL_TYPE_ABORT":          7,
	}
)

func (x SignalType) Enum() *SignalType {
	p := new(SignalType)
	*p = x
	return p
}

type IndicatorType int32

const (
	IndicatorType_INDICATOR_RSI                   IndicatorType = 0
	IndicatorType_INDICATOR_MACD                  IndicatorType = 1
	IndicatorType_INDICATOR_BOLLINGER_BANDS       IndicatorType = 2
	IndicatorType_INDICATOR_STOCHASTIC_OSCILLATOR IndicatorType = 3
	IndicatorType_INDICATOR_WILLIAMS_R            IndicatorType = 4
	IndicatorType_INDICATOR_ADX                   IndicatorType = 5
	IndicatorType_INDICATOR_CCI                   IndicatorType = 6
	IndicatorType_INDICATOR_AO                    IndicatorType = 7
	IndicatorType_INDICATOR_TREND_STRENGTH        IndicatorType = 8
	IndicatorType_INDICATOR_RANGE_FILTER          IndicatorType = 9
	IndicatorType_INDICATOR_EMA                   IndicatorType = 10
	IndicatorType_INDICATOR_WADDAH_ATTAR          IndicatorType = 11
	IndicatorType_INDICATOR_ATR                   IndicatorType = 12
	IndicatorType_INDICATOR_MA                    IndicatorType = 13
)

// Enum value maps for IndicatorType.
var (
	IndicatorType_name = map[int32]string{
		0:  "INDICATOR_RSI",
		1:  "INDICATOR_MACD",
		2:  "INDICATOR_BOLLINGER_BANDS",
		3:  "INDICATOR_STOCHASTIC_OSCILLATOR",
		4:  "INDICATOR_WILLIAMS_R",
		5:  "INDICATOR_ADX",
		6:  "INDICATOR_CCI",
		7:  "INDICATOR_AO",
		8:  "INDICATOR_TREND_STRENGTH",
		9:  "INDICATOR_RANGE_FILTER",
		10: "INDICATOR_EMA",
		11: "INDICATOR_WADDAH_ATTAR",
		12: "INDICATOR_ATR",
		13: "INDICATOR_MA",
	}
	IndicatorType_value = map[string]int32{
		"INDICATOR_RSI":                   0,
		"INDICATOR_MACD":                  1,
		"INDICATOR_BOLLINGER_BANDS":       2,
		"INDICATOR_STOCHASTIC_OSCILLATOR": 3,
		"INDICATOR_WILLIAMS_R":            4,
		"INDICATOR_ADX":                   5,
		"INDICATOR_CCI":                   6,
		"INDICATOR_AO":                    7,
		"INDICATOR_TREND_STRENGTH":        8,
		"INDICATOR_RANGE_FILTER":          9,
		"INDICATOR_EMA":                   10,
		"INDICATOR_WADDAH_ATTAR":          11,
		"INDICATOR_ATR":                   12,
		"INDICATOR_MA":                    13,
	}
)

func (x IndicatorType) Enum() *IndicatorType {
	p := new(IndicatorType)
	*p = x
	return p
}

type OrderStatus int32

const (
	OrderStatus_ORDER_STATUS_PENDING   OrderStatus = 0
	OrderStatus_ORDER_STATUS_FILLED    OrderStatus = 1
	OrderStatus_ORDER_STATUS_CANCELLED OrderStatus = 2
	OrderStatus_ORDER_STATUS_REJECTED  OrderStatus = 3
	OrderStatus_ORDER_STATUS_FAILED    OrderStatus = 4
)

// Enum value maps for OrderStatus.
var (
	OrderStatus_name = map[int32]string{
		0: "ORDER_STATUS_PENDING",
		1: "ORDER_STATUS_FILLED",
		2: "ORDER_STATUS_CANCELLED",
		3: "ORDER_STATUS_REJECTED",
		4: "ORDER_STATUS_FAILED",
	}
	OrderStatus_value = map[string]int32{
		"ORDER_STATUS_PENDING":   0,
		"ORDER_STATUS_FILLED":    1,
		"ORDER_STATUS_CANCELLED": 2,
		"ORDER_STATUS_REJECTED":  3,
		"ORDER_STATUS_FAILED":    4,
	}
)

func (x OrderStatus) Enum() *OrderStatus {
	p := new(OrderStatus)
	*p = x
	return p
}

type PurchaseType int32

const (
	PurchaseType_PURCHASE_TYPE_BUY  PurchaseType = 0
	PurchaseType_PURCHASE_TYPE_SELL PurchaseType = 1
)

// Enum value maps for PurchaseType.
var (
	PurchaseType_name = map[int32]string{
		0: "PURCHASE_TYPE_BUY",
		1: "PURCHASE_TYPE_SELL",
	}
	PurchaseType_value = map[string]int32{
		"PURCHASE_TYPE_BUY":  0,
		"PURCHASE_TYPE_SELL": 1,
	}
)

func (x PurchaseType) Enum() *PurchaseType {
	p := new(PurchaseType)
	*p = x
	return p
}

type OrderType int32

const (
	OrderType_ORDER_TYPE_MARKET OrderType = 0
	OrderType_ORDER_TYPE_LIMIT  OrderType = 1
)

// Enum value maps for OrderType.
var (
	OrderType_name = map[int32]string{
		0: "ORDER_TYPE_MARKET",
		1: "ORDER_TYPE_LIMIT",
	}
	OrderType_value = map[string]int32{
		"ORDER_TYPE_MARKET": 0,
		"ORDER_TYPE_LIMIT":  1,
	}
)

func (x OrderType) Enum() *OrderType {
	p := new(OrderType)
	*p = x
	return p
}

type OrderReason int32

const (
	OrderReason_ORDER_REASON_STOP_LOSS   OrderReason = 0
	OrderReason_ORDER_REASON_TAKE_PROFIT OrderReason = 1
	OrderReason_ORDER_REASON_STRATEGY    OrderReason = 2
)

// Enum value maps for OrderReason.
var (
	OrderReason_name = map[int32]string{
		0: "ORDER_REASON_STOP_LOSS",
		1: "ORDER_REASON_TAKE_PROFIT",
		2: "ORDER_REASON_STRATEGY",
	}
	OrderReason_value = map[string]int32{
		"ORDER_REASON_STOP_LOSS":   0,
		"ORDER_REASON_TAKE_PROFIT": 1,
		"ORDER_REASON_STRATEGY":    2,
	}
)

func (x OrderReason) Enum() *OrderReason {
	p := new(OrderReason)
	*p = x
	return p
}

// MarketData represents the market data for a single point in time
type MarketData struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol string                 `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
	High   float64                `protobuf:"fixed64,2,opt,name=high,proto3" json:"high,omitempty"`
	Low    float64                `protobuf:"fixed64,3,opt,name=low,proto3" json:"low,omitempty"`
	Open   float64                `protobuf:"fixed64,4,opt,name=open,proto3" json:"open,omitempty"`
	Close  float64                `protobuf:"fixed64,5,opt,name=close,proto3" json:"close,omitempty"`
	Volume float64                `protobuf:"fixed64,6,opt,name=volume,proto3" json:"volume,omitempty"`
	Time   *timestamppb.Timestamp `protobuf:"bytes,7,opt,name=time,proto3" json:"time,omitempty"`
}

func (x *MarketData) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *MarketData) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *MarketData) GetHigh() float64 {
	if x != nil {
		return x.High
	}
	return 0
}

func (x *MarketData) GetLow() float64 {
	if x != nil {
		return x.Low
	}
	return 0
}

func (x *MarketData) GetOpen() float64 {
	if x != nil {
		return x.Open
	}
	return 0
}

func (x *MarketData) GetClose() float64 {
	if x != nil {
		return x.Close
	}
	return 0
}

func (x *MarketData) GetVolume() float64 {
	if x != nil {
		return x.Volume
	}
	return 0
}

func (x *MarketData) GetTime() *timestamppb.Timestamp {
	if x != nil {
		return x.Time
	}
	return nil
}

type InitializeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Config string `protobuf:"bytes,1,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *InitializeRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *InitializeRequest) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

// ProcessDataRequest contains the market data to process
type ProcessDataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data *MarketData `protobuf:"bytes,1,opt,name=data,proto3" json:"data,omitempty"`
}

func (x *ProcessDataRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ProcessDataRequest) GetData() *MarketData {
	if x != nil {
		return x.Data
	}
	return nil
}

// NameRequest is an empty request for getting the strategy name
type NameRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields
}

func (x *NameRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

// NameResponse contains the strategy name
type NameResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Name string `protobuf:"bytes,1,opt,name=name,proto3" json:"name,omitempty"`
}

func (x *NameResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *NameResponse) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

type GetRangeRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol    string                 `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
	StartTime *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime   *timestamppb.Timestamp `protobuf:"bytes,3,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
	Interval  Interval               `protobuf:"varint,4,opt,name=interval,proto3,enum=strategy.Interval" json:"interval,omitempty"`
}

func (x *GetRangeRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetRangeRequest) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *GetRangeRequest) GetStartTime() *timestamppb.Timestamp {
	if x != nil {
		return x.StartTime
	}
	return nil
}

func (x *GetRangeRequest) GetEndTime() *timestamppb.Timestamp {
	if x != nil {
		return x.EndTime
	}
	return nil
}

func (x *GetRangeRequest) GetInterval() Interval {
	if x != nil {
		return x.Interval
	}
	return Interval_INTERVAL_1M
}

type GetRangeResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Data []*MarketData `protobuf:"bytes,1,rep,name=data,proto3" json:"data,omitempty"`
}

func (x *GetRangeResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetRangeResponse) GetData() []*MarketData {
	if x != nil {
		return x.Data
	}
	return nil
}

type ReadLastDataRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol string `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
}

func (x *ReadLastDataRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ReadLastDataRequest) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

type ExecuteSQLRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Query  string   `protobuf:"bytes,1,opt,name=query,proto3" json:"query,omitempty"`
	Params []string `protobuf:"bytes,2,rep,name=params,proto3" json:"params,omitempty"`
}

func (x *ExecuteSQLRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ExecuteSQLRequest) GetQuery() string {
	if x != nil {
		return x.Query
	}
	return ""
}

func (x *ExecuteSQLRequest) GetParams() []string {
	if x != nil {
		return x.Params
	}
	return nil
}

type SQLResult struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Fields map[string]string `protobuf:"bytes,1,rep,name=fields,proto3" json:"fields,omitempty" protobuf_key:"bytes,1,opt,name=key,proto3" protobuf_val:"bytes,2,opt,name=value,proto3"`
}

func (x *SQLResult) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *SQLResult) GetFields() map[string]string {
	if x != nil {
		return x.Fields
	}
	return nil
}

type ExecuteSQLResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Results []*SQLResult `protobuf:"bytes,1,rep,name=results,proto3" json:"results,omitempty"`
}

func (x *ExecuteSQLResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ExecuteSQLResponse) GetResults() []*SQLResult {
	if x != nil {
		return x.Results
	}
	return nil
}

type CountRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	StartTime *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=start_time,json=startTime,proto3" json:"start_time,omitempty"`
	EndTime   *timestamppb.Timestamp `protobuf:"bytes,2,opt,name=end_time,json=endTime,proto3" json:"end_time,omitempty"`
}

func (x *CountRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *CountRequest) GetStartTime() *timestamppb.Timestamp {
	if x != nil {
		return x.StartTime
	}
	return nil
}

func (x *CountRequest) GetEndTime() *timestamppb.Timestamp {
	if x != nil {
		return x.EndTime
	}
	return nil
}

type CountResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Count int32 `protobuf:"varint,1,opt,name=count,proto3" json:"count,omitempty"`
}

func (x *CountResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *CountResponse) GetCount() int32 {
	if x != nil {
		return x.Count
	}
	return 0
}

type ConfigureRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IndicatorType IndicatorType `protobuf:"varint,1,opt,name=indicator_type,json=indicatorType,proto3,enum=strategy.IndicatorType" json:"indicator_type,omitempty"`
	Config        string        `protobuf:"bytes,2,opt,name=config,proto3" json:"config,omitempty"`
}

func (x *ConfigureRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ConfigureRequest) GetIndicatorType() IndicatorType {
	if x != nil {
		return x.IndicatorType
	}
	return IndicatorType_INDICATOR_RSI
}

func (x *ConfigureRequest) GetConfig() string {
	if x != nil {
		return x.Config
	}
	return ""
}

type GetSignalRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	IndicatorType IndicatorType `protobuf:"varint,1,opt,name=indicator_type,json=indicatorType,proto3,enum=strategy.IndicatorType" json:"indicator_type,omitempty"`
	MarketData    *MarketData   `protobuf:"bytes,2,opt,name=market_data,json=marketData,proto3" json:"market_data,omitempty"`
}

func (x *GetSignalRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetSignalRequest) GetIndicatorType() IndicatorType {
	if x != nil {
		return x.IndicatorType
	}
	return IndicatorType_INDICATOR_RSI
}

func (x *GetSignalRequest) GetMarketData() *MarketData {
	if x != nil {
		return x.MarketData
	}
	return nil
}

type GetSignalResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Timestamp     *timestamppb.Timestamp `protobuf:"bytes,1,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	Type          SignalType             `protobuf:"varint,2,opt,name=type,proto3,enum=strategy.SignalType" json:"type,omitempty"`
	Name          string                 `protobuf:"bytes,3,opt,name=name,proto3" json:"name,omitempty"`
	Reason        string                 `protobuf:"bytes,4,opt,name=reason,proto3" json:"reason,omitempty"`
	RawValue      string                 `protobuf:"bytes,5,opt,name=rawValue,proto3" json:"rawValue,omitempty"`
	Symbol        string                 `protobuf:"bytes,6,opt,name=symbol,proto3" json:"symbol,omitempty"`
	IndicatorType IndicatorType          `protobuf:"varint,7,opt,name=indicatorType,proto3,enum=strategy.IndicatorType" json:"indicatorType,omitempty"`
}

func (x *GetSignalResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetSignalResponse) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *GetSignalResponse) GetType() SignalType {
	if x != nil {
		return x.Type
	}
	return SignalType_SIGNAL_TYPE_BUY_LONG
}

func (x *GetSignalResponse) GetName() string {
	if x != nil {
		return x.Name
	}
	return ""
}

func (x *GetSignalResponse) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *GetSignalResponse) GetRawValue() string {
	if x != nil {
		return x.RawValue
	}
	return ""
}

func (x *GetSignalResponse) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *GetSignalResponse) GetIndicatorType() IndicatorType {
	if x != nil {
		return x.IndicatorType
	}
	return IndicatorType_INDICATOR_RSI
}

type GetRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
}

func (x *GetRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

type GetResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Value string `protobuf:"bytes,1,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *GetResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetResponse) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type SetRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Key   string `protobuf:"bytes,1,opt,name=key,proto3" json:"key,omitempty"`
	Value string `protobuf:"bytes,2,opt,name=value,proto3" json:"value,omitempty"`
}

func (x *SetRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *SetRequest) GetKey() string {
	if x != nil {
		return x.Key
	}
	return ""
}

func (x *SetRequest) GetValue() string {
	if x != nil {
		return x.Value
	}
	return ""
}

type PlaceMultipleOrdersRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Orders []*ExecuteOrder `protobuf:"bytes,1,rep,name=orders,proto3" json:"orders,omitempty"`
}

func (x *PlaceMultipleOrdersRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *PlaceMultipleOrdersRequest) GetOrders() []*ExecuteOrder {
	if x != nil {
		return x.Orders
	}
	return nil
}

type GetPositionsResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Positions []*Position `protobuf:"bytes,1,rep,name=positions,proto3" json:"positions,omitempty"`
}

func (x *GetPositionsResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetPositionsResponse) GetPositions() []*Position {
	if x != nil {
		return x.Positions
	}
	return nil
}

type GetPositionRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol string `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
}

func (x *GetPositionRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetPositionRequest) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

type CancelOrderRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OrderId string `protobuf:"bytes,1,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
}

func (x *CancelOrderRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *CancelOrderRequest) GetOrderId() string {
	if x != nil {
		return x.OrderId
	}
	return ""
}

type GetOrderStatusRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OrderId string `protobuf:"bytes,1,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
}

func (x *GetOrderStatusRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetOrderStatusRequest) GetOrderId() string {
	if x != nil {
		return x.OrderId
	}
	return ""
}

type GetOrderStatusResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Status OrderStatus `protobuf:"varint,1,opt,name=status,proto3,enum=strategy.OrderStatus" json:"status,omitempty"`
}

func (x *GetOrderStatusResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetOrderStatusResponse) GetStatus() OrderStatus {
	if x != nil {
		return x.Status
	}
	return OrderStatus_ORDER_STATUS_PENDING
}

type Reason struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Reason  string `protobuf:"bytes,1,opt,name=reason,proto3" json:"reason,omitempty"`
	Message string `protobuf:"bytes,2,opt,name=message,proto3" json:"message,omitempty"`
}

func (x *Reason) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *Reason) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *Reason) GetMessage() string {
	if x != nil {
		return x.Message
	}
	return ""
}

type ExecuteOrderTakeProfitOrStopLoss struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol    string       `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
	Side      PurchaseType `protobuf:"varint,2,opt,name=side,proto3,enum=strategy.PurchaseType" json:"side,omitempty"`
	OrderType OrderType    `protobuf:"varint,3,opt,name=order_type,json=orderType,proto3,enum=strategy.OrderType" json:"order_type,omitempty"`
}

func (x *ExecuteOrderTakeProfitOrStopLoss) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ExecuteOrderTakeProfitOrStopLoss) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *ExecuteOrderTakeProfitOrStopLoss) GetSide() PurchaseType {
	if x != nil {
		return x.Side
	}
	return PurchaseType_PURCHASE_TYPE_BUY
}

func (x *ExecuteOrderTakeProfitOrStopLoss) GetOrderType() OrderType {
	if x != nil {
		return x.OrderType
	}
	return OrderType_ORDER_TYPE_MARKET
}

type ExecuteOrder struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Id           string                            `protobuf:"bytes,1,opt,name=id,proto3" json:"id,omitempty"`
	Symbol       string                            `protobuf:"bytes,2,opt,name=symbol,proto3" json:"symbol,omitempty"`
	Side         PurchaseType                      `protobuf:"varint,3,opt,name=side,proto3,enum=strategy.PurchaseType" json:"side,omitempty"`
	OrderType    OrderType                         `protobuf:"varint,4,opt,name=order_type,json=orderType,proto3,enum=strategy.OrderType" json:"order_type,omitempty"`
	Reason       *Reason                           `protobuf:"bytes,5,opt,name=reason,proto3" json:"reason,omitempty"`
	Price        float64                           `protobuf:"fixed64,6,opt,name=price,proto3" json:"price,omitempty"`
	StrategyName string                            `protobuf:"bytes,7,opt,name=strategy_name,json=strategyName,proto3" json:"strategy_name,omitempty"`
	Quantity     float64                           `protobuf:"fixed64,8,opt,name=quantity,proto3" json:"quantity,omitempty"`
	TakeProfit   *ExecuteOrderTakeProfitOrStopLoss `protobuf:"bytes,9,opt,name=take_profit,json=takeProfit,proto3" json:"take_profit,omitempty"`
	StopLoss     *ExecuteOrderTakeProfitOrStopLoss `protobuf:"bytes,10,opt,name=stop_loss,json=stopLoss,proto3" json:"stop_loss,omitempty"`
	PositionType PositionType                      `protobuf:"varint,11,opt,name=position_type,json=positionType,proto3,enum=strategy.PositionType" json:"position_type,omitempty"`
}

func (x *ExecuteOrder) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *ExecuteOrder) GetId() string {
	if x != nil {
		return x.Id
	}
	return ""
}

func (x *ExecuteOrder) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *ExecuteOrder) GetSide() PurchaseType {
	if x != nil {
		return x.Side
	}
	return PurchaseType_PURCHASE_TYPE_BUY
}

func (x *ExecuteOrder) GetOrderType() OrderType {
	if x != nil {
		return x.OrderType
	}
	return OrderType_ORDER_TYPE_MARKET
}

func (x *ExecuteOrder) GetReason() *Reason {
	if x != nil {
		return x.Reason
	}
	return nil
}

func (x *ExecuteOrder) GetPrice() float64 {
	if x != nil {
		return x.Price
	}
	return 0
}

func (x *ExecuteOrder) GetStrategyName() string {
	if x != nil {
		return x.StrategyName
	}
	return ""
}

func (x *ExecuteOrder) GetQuantity() float64 {
	if x != nil {
		return x.Quantity
	}
	return 0
}

func (x *ExecuteOrder) GetTakeProfit() *ExecuteOrderTakeProfitOrStopLoss {
	if x != nil {
		return x.TakeProfit
	}
	return nil
}

func (x *ExecuteOrder) GetStopLoss() *ExecuteOrderTakeProfitOrStopLoss {
	if x != nil {
		return x.StopLoss
	}
	return nil
}

func (x *ExecuteOrder) GetPositionType() PositionType {
	if x != nil {
		return x.PositionType
	}
	return PositionType_POSITION_TYPE_LONG
}

type Order struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	OrderId      string                 `protobuf:"bytes,1,opt,name=order_id,json=orderId,proto3" json:"order_id,omitempty"`
	Symbol       string                 `protobuf:"bytes,2,opt,name=symbol,proto3" json:"symbol,omitempty"`
	Side         PurchaseType           `protobuf:"varint,3,opt,name=side,proto3,enum=strategy.PurchaseType" json:"side,omitempty"`
	Quantity     float64                `protobuf:"fixed64,4,opt,name=quantity,proto3" json:"quantity,omitempty"`
	Price        float64                `protobuf:"fixed64,5,opt,name=price,proto3" json:"price,omitempty"`
	Timestamp    *timestamppb.Timestamp `protobuf:"bytes,6,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
	IsCompleted  bool                   `protobuf:"varint,7,opt,name=is_completed,json=isCompleted,proto3" json:"is_completed,omitempty"`
	Reason       *Reason                `protobuf:"bytes,8,opt,name=reason,proto3" json:"reason,omitempty"`
	StrategyName string                 `protobuf:"bytes,9,opt,name=strategy_name,json=strategyName,proto3" json:"strategy_name,omitempty"`
	Fee          float64                `protobuf:"fixed64,10,opt,name=fee,proto3" json:"fee,omitempty"`
	PositionType PositionType           `protobuf:"varint,11,opt,name=position_type,json=positionType,proto3,enum=strategy.PositionType" json:"position_type,omitempty"`
}

func (x *Order) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *Order) GetOrderId() string {
	if x != nil {
		return x.OrderId
	}
	return ""
}

func (x *Order) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *Order) GetSide() PurchaseType {
	if x != nil {
		return x.Side
	}
	return PurchaseType_PURCHASE_TYPE_BUY
}

func (x *Order) GetQuantity() float64 {
	if x != nil {
		return x.Quantity
	}
	return 0
}

func (x *Order) GetPrice() float64 {
	if x != nil {
		return x.Price
	}
	return 0
}

func (x *Order) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

func (x *Order) GetIsCompleted() bool {
	if x != nil {
		return x.IsCompleted
	}
	return false
}

func (x *Order) GetReason() *Reason {
	if x != nil {
		return x.Reason
	}
	return nil
}

func (x *Order) GetStrategyName() string {
	if x != nil {
		return x.StrategyName
	}
	return ""
}

func (x *Order) GetFee() float64 {
	if x != nil {
		return x.Fee
	}
	return 0
}

func (x *Order) GetPositionType() PositionType {
	if x != nil {
		return x.PositionType
	}
	return PositionType_POSITION_TYPE_LONG
}

type Position struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Symbol           string                 `protobuf:"bytes,1,opt,name=symbol,proto3" json:"symbol,omitempty"`
	Quantity         float64                `protobuf:"fixed64,2,opt,name=quantity,proto3" json:"quantity,omitempty"`
	TotalInQuantity  float64                `protobuf:"fixed64,3,opt,name=total_in_quantity,json=totalInQuantity,proto3" json:"total_in_quantity,omitempty"`
	TotalOutQuantity float64                `protobuf:"fixed64,4,opt,name=total_out_quantity,json=totalOutQuantity,proto3" json:"total_out_quantity,omitempty"`
	TotalInAmount    float64                `protobuf:"fixed64,5,opt,name=total_in_amount,json=totalInAmount,proto3" json:"total_in_amount,omitempty"`
	TotalOutAmount   float64                `protobuf:"fixed64,6,opt,name=total_out_amount,json=totalOutAmount,proto3" json:"total_out_amount,omitempty"`
	TotalInFee       float64                `protobuf:"fixed64,7,opt,name=total_in_fee,json=totalInFee,proto3" json:"total_in_fee,omitempty"`
	TotalOutFee      float64                `protobuf:"fixed64,8,opt,name=total_out_fee,json=totalOutFee,proto3" json:"total_out_fee,omitempty"`
	OpenTimestamp    *timestamppb.Timestamp `protobuf:"bytes,9,opt,name=open_timestamp,json=openTimestamp,proto3" json:"open_timestamp,omitempty"`
	StrategyName     string                 `protobuf:"bytes,10,opt,name=strategy_name,json=strategyName,proto3" json:"strategy_name,omitempty"`
}

func (x *Position) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *Position) GetSymbol() string {
	if x != nil {
		return x.Symbol
	}
	return ""
}

func (x *Position) GetQuantity() float64 {
	if x != nil {
		return x.Quantity
	}
	return 0
}

func (x *Position) GetTotalInQuantity() float64 {
	if x != nil {
		return x.TotalInQuantity
	}
	return 0
}

func (x *Position) GetTotalOutQuantity() float64 {
	if x != nil {
		return x.TotalOutQuantity
	}
	return 0
}

func (x *Position) GetTotalInAmount() float64 {
	if x != nil {
		return x.TotalInAmount
	}
	return 0
}

func (x *Position) GetTotalOutAmount() float64 {
	if x != nil {
		return x.TotalOutAmount
	}
	return 0
}

func (x *Position) GetTotalInFee() float64 {
	if x != nil {
		return x.TotalInFee
	}
	return 0
}

func (x *Position) GetTotalOutFee() float64 {
	if x != nil {
		return x.TotalOutFee
	}
	return 0
}

func (x *Position) GetOpenTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.OpenTimestamp
	}
	return nil
}

func (x *Position) GetStrategyName() string {
	if x != nil {
		return x.StrategyName
	}
	return ""
}

type MarkRequest struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MarketData *MarketData `protobuf:"bytes,1,opt,name=market_data,json=marketData,proto3" json:"market_data,omitempty"`
	Signal     SignalType  `protobuf:"varint,2,opt,name=signal,proto3,enum=strategy.SignalType" json:"signal,omitempty"`
	Reason     string      `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
}

func (x *MarkRequest) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *MarkRequest) GetMarketData() *MarketData {
	if x != nil {
		return x.MarketData
	}
	return nil
}

func (x *MarkRequest) GetSignal() SignalType {
	if x != nil {
		return x.Signal
	}
	return SignalType_SIGNAL_TYPE_BUY_LONG
}

func (x *MarkRequest) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

type GetMarkersResponse struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	Markers []*Mark `protobuf:"bytes,1,rep,name=markers,proto3" json:"markers,omitempty"`
}

func (x *GetMarkersResponse) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *GetMarkersResponse) GetMarkers() []*Mark {
	if x != nil {
		return x.Markers
	}
	return nil
}

type Mark struct {
	state         protoimpl.MessageState
	sizeCache     protoimpl.SizeCache
	unknownFields protoimpl.UnknownFields

	MarketData *MarketData            `protobuf:"bytes,1,opt,name=market_data,json=marketData,proto3" json:"market_data,omitempty"`
	Signal     SignalType             `protobuf:"varint,2,opt,name=signal,proto3,enum=strategy.SignalType" json:"signal,omitempty"`
	Reason     string                 `protobuf:"bytes,3,opt,name=reason,proto3" json:"reason,omitempty"`
	Timestamp  *timestamppb.Timestamp `protobuf:"bytes,4,opt,name=timestamp,proto3" json:"timestamp,omitempty"`
}

func (x *Mark) ProtoReflect() protoreflect.Message {
	panic(`not implemented`)
}

func (x *Mark) GetMarketData() *MarketData {
	if x != nil {
		return x.MarketData
	}
	return nil
}

func (x *Mark) GetSignal() SignalType {
	if x != nil {
		return x.Signal
	}
	return SignalType_SIGNAL_TYPE_BUY_LONG
}

func (x *Mark) GetReason() string {
	if x != nil {
		return x.Reason
	}
	return ""
}

func (x *Mark) GetTimestamp() *timestamppb.Timestamp {
	if x != nil {
		return x.Timestamp
	}
	return nil
}

// TradingStrategy defines the interface for trading strategies
// go:plugin type=plugin version=1
type TradingStrategy interface {
	// Initialize sets up the strategy with a configuration string
	Initialize(context.Context, *InitializeRequest) (*emptypb.Empty, error)
	// ProcessData processes new market data and generates signals
	ProcessData(context.Context, *ProcessDataRequest) (*emptypb.Empty, error)
	// Name returns the name of the strategy
	Name(context.Context, *NameRequest) (*NameResponse, error)
}

// StrategyApi defines all the functions that the host provides to the plugin
// go:plugin type=host
type StrategyApi interface {
	// DataSource methods
	GetRange(context.Context, *GetRangeRequest) (*GetRangeResponse, error)
	ReadLastData(context.Context, *ReadLastDataRequest) (*MarketData, error)
	ExecuteSQL(context.Context, *ExecuteSQLRequest) (*ExecuteSQLResponse, error)
	Count(context.Context, *CountRequest) (*CountResponse, error)
	// Indicator methods
	ConfigureIndicator(context.Context, *ConfigureRequest) (*emptypb.Empty, error)
	GetSignal(context.Context, *GetSignalRequest) (*GetSignalResponse, error)
	// Cache methods
	GetCache(context.Context, *GetRequest) (*GetResponse, error)
	SetCache(context.Context, *SetRequest) (*emptypb.Empty, error)
	// TradingSystem methods
	PlaceOrder(context.Context, *ExecuteOrder) (*emptypb.Empty, error)
	PlaceMultipleOrders(context.Context, *PlaceMultipleOrdersRequest) (*emptypb.Empty, error)
	GetPositions(context.Context, *emptypb.Empty) (*GetPositionsResponse, error)
	GetPosition(context.Context, *GetPositionRequest) (*Position, error)
	CancelOrder(context.Context, *CancelOrderRequest) (*emptypb.Empty, error)
	CancelAllOrders(context.Context, *emptypb.Empty) (*emptypb.Empty, error)
	GetOrderStatus(context.Context, *GetOrderStatusRequest) (*GetOrderStatusResponse, error)
	// Marker methods
	Mark(context.Context, *MarkRequest) (*emptypb.Empty, error)
	GetMarkers(context.Context, *emptypb.Empty) (*GetMarkersResponse, error)
}

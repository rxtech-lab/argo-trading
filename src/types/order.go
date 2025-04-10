package types

import (
	"time"

	"github.com/moznion/go-optional"
)

type PurchaseType string

type OrderType string

type OrderStatus string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
)

const (
	PurchaseTypeBuy  PurchaseType = "BUY"
	PurchaseTypeSell PurchaseType = "SELL"
)

const (
	OrderTypeMarket OrderType = "MARKET"
	OrderTypeLimit  OrderType = "LIMIT"
)

const (
	OrderReasonStopLoss   string = "stop_loss"
	OrderReasonTakeProfit string = "take_profit"
	OrderReasonStrategy   string = "strategy"
)

type Reason struct {
	Reason  string `yaml:"reason" json:"reason" csv:"reason"`
	Message string `yaml:"message" json:"message" csv:"message"`
}

type ExecuteOrderTakeProfitOrStopLoss struct {
	Symbol    string       `yaml:"symbol" json:"symbol" csv:"symbol"`
	Side      PurchaseType `yaml:"side" json:"side" csv:"side"`
	OrderType OrderType    `yaml:"order_type" json:"order_type" csv:"order_type"`
}

type ExecuteOrder struct {
	Symbol    string       `yaml:"symbol" json:"symbol" csv:"symbol"`
	Side      PurchaseType `yaml:"side" json:"side" csv:"side"`
	OrderType OrderType    `yaml:"order_type" json:"order_type" csv:"order_type"`
	Reason    Reason       `yaml:"reason" json:"reason" csv:"reason"`
	Price     float64      `yaml:"price" json:"price" csv:"price"`
	Quantity  float64      `yaml:"quantity" json:"quantity" csv:"quantity"`
	// TakeProfit is the take profit order. Can be nil if not set.
	TakeProfit optional.Option[ExecuteOrderTakeProfitOrStopLoss] `yaml:"take_profit" json:"take_profit" csv:"take_profit"`
	// StopLoss is the stop loss order. Can be nil if not set.
	StopLoss optional.Option[ExecuteOrderTakeProfitOrStopLoss] `yaml:"stop_loss" json:"stop_loss" csv:"stop_loss"`
}

type Order struct {
	OrderID     string       `yaml:"order_id" json:"order_id" csv:"order_id"`
	Symbol      string       `yaml:"symbol" json:"symbol" csv:"symbol"`
	Side        PurchaseType `yaml:"side" json:"side" csv:"side"`
	Quantity    float64      `yaml:"quantity" json:"quantity" csv:"quantity"`
	Price       float64      `yaml:"price" json:"price" csv:"price"`
	Timestamp   time.Time    `yaml:"timestamp" json:"timestamp" csv:"timestamp"`
	IsCompleted bool         `yaml:"is_completed" json:"is_completed" csv:"is_completed"`
	// Reason is the reason for the order
	// It can be used to store the reason for the order
	// like "buy_signal", "sell_signal", "stop_loss", "take_profit", etc.
	Reason Reason `yaml:"reason" json:"reason" csv:"reason"`
	// StrategyName is the name of the strategy that created this order
	StrategyName string  `yaml:"strategy_name" json:"strategy_name" csv:"strategy_name"`
	Fee          float64 `yaml:"fee" json:"fee" csv:"fee"`
}

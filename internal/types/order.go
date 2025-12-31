package types

import (
	"time"

	"github.com/go-playground/validator/v10"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

type PurchaseType string

type OrderType string

type OrderStatus string

type PositionType string

const (
	OrderStatusPending   OrderStatus = "PENDING"
	OrderStatusFilled    OrderStatus = "FILLED"
	OrderStatusCancelled OrderStatus = "CANCELLED"
	OrderStatusRejected  OrderStatus = "REJECTED"
	OrderStatusFailed    OrderStatus = "FAILED"
)

const (
	PositionTypeLong  PositionType = "LONG"
	PositionTypeShort PositionType = "SHORT"
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
	OrderReasonStopLoss              string = "stop_loss"
	OrderReasonTakeProfit            string = "take_profit"
	OrderReasonStrategy              string = "strategy"
	OrderReasonInsufficientBuyPower  string = "insufficient_buying_power"
	OrderReasonInsufficientSellPower string = "insufficient_selling_power"
	OrderReasonInvalidQuantity       string = "invalid_quantity"
	OrderReasonInvalidPrice          string = "invalid_price"
)

type Reason struct {
	Reason  string `yaml:"reason" json:"reason" csv:"reason" validate:"required"`
	Message string `yaml:"message" json:"message" csv:"message" validate:"required"`
}

type ExecuteOrderTakeProfitOrStopLoss struct {
	Symbol    string       `yaml:"symbol" json:"symbol" csv:"symbol" validate:"required"`
	Side      PurchaseType `yaml:"side" json:"side" csv:"side" validate:"required,oneof=BUY SELL"`
	OrderType OrderType    `yaml:"order_type" json:"order_type" csv:"order_type" validate:"required,oneof=MARKET LIMIT"`
}

type ExecuteOrder struct {
	ID           string       `yaml:"id" json:"id" csv:"id" validate:"required,uuid"`
	Symbol       string       `yaml:"symbol" json:"symbol" csv:"symbol" validate:"required"`
	Side         PurchaseType `yaml:"side" json:"side" csv:"side" validate:"required,oneof=BUY SELL"`
	OrderType    OrderType    `yaml:"order_type" json:"order_type" csv:"order_type" validate:"required,oneof=MARKET LIMIT"`
	Reason       Reason       `yaml:"reason" json:"reason" csv:"reason" validate:"required"`
	Price        float64      `yaml:"price" json:"price" csv:"price" validate:"required,gt=0"`
	StrategyName string       `yaml:"strategy_name" json:"strategy_name" csv:"strategy_name" validate:"required"`
	Quantity     float64      `yaml:"quantity" json:"quantity" csv:"quantity" validate:"required,gt=0"`
	PositionType PositionType `yaml:"position_type" json:"position_type" csv:"position_type" validate:"required,oneof=LONG SHORT"`
	// TakeProfit is the take profit order. Can be nil if not set.
	TakeProfit optional.Option[ExecuteOrderTakeProfitOrStopLoss] `yaml:"take_profit" json:"take_profit" csv:"take_profit"`
	// StopLoss is the stop loss order. Can be nil if not set.
	StopLoss optional.Option[ExecuteOrderTakeProfitOrStopLoss] `yaml:"stop_loss" json:"stop_loss" csv:"stop_loss"`
}

type Order struct {
	OrderID   string       `yaml:"order_id" json:"order_id" csv:"order_id"`
	Symbol    string       `yaml:"symbol" json:"symbol" csv:"symbol" validate:"required"`
	Side      PurchaseType `yaml:"side" json:"side" csv:"side" validate:"required,oneof=BUY SELL"`
	Quantity  float64      `yaml:"quantity" json:"quantity" csv:"quantity" validate:"required,gt=0"`
	Price     float64      `yaml:"price" json:"price" csv:"price" validate:"required,gt=0"`
	Timestamp time.Time    `yaml:"timestamp" json:"timestamp" csv:"timestamp" validate:"required"`
	// IsCompleted is true if the order has been filled or cancelled
	IsCompleted bool `yaml:"is_completed" json:"is_completed" csv:"is_completed"`
	// Status is the status of the order (PENDING, FILLED, CANCELLED, REJECTED, Failed)
	Status OrderStatus `yaml:"status" json:"status" csv:"status"`
	// Reason is the reason for the order
	// It can be used to store the reason for the order
	// like "buy_signal", "sell_signal", "stop_loss", "take_profit", etc.
	Reason Reason `yaml:"reason" json:"reason" csv:"reason" validate:"required"`
	// StrategyName is the name of the strategy that created this order
	StrategyName string       `yaml:"strategy_name" json:"strategy_name" csv:"strategy_name" validate:"required"`
	Fee          float64      `yaml:"fee" json:"fee" csv:"fee" validate:"gte=0"`
	PositionType PositionType `yaml:"position_type" json:"position_type" csv:"position_type" validate:"required,oneof=LONG SHORT"`
}

// Validate validates the ExecuteOrder struct.
func (eo *ExecuteOrder) Validate() error {
	validate := validator.New()

	err := validate.Struct(eo)
	if err != nil {
		return errors.Wrap(errors.ErrCodeInvalidExecuteOrder, "invalid execute order", err)
	}

	// Validate take profit if present
	if eo.TakeProfit.IsSome() {
		tp := eo.TakeProfit.Unwrap()
		if err := validate.Struct(tp); err != nil {
			return errors.Wrap(errors.ErrCodeInvalidTakeProfit, "invalid take profit", err)
		}
	}

	// Validate stop loss if present
	if eo.StopLoss.IsSome() {
		sl := eo.StopLoss.Unwrap()
		if err := validate.Struct(sl); err != nil {
			return errors.Wrap(errors.ErrCodeInvalidStopLoss, "invalid stop loss", err)
		}
	}

	return nil
}

// Validate validates the Order struct.
func (o *Order) Validate() error {
	validate := validator.New()
	if err := validate.Struct(o); err != nil {
		return errors.Wrap(errors.ErrCodeInvalidOrder, "invalid order", err)
	}

	return nil
}

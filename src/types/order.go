package types

import "time"

type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

const (
	OrderReasonStopLoss   string = "stop_loss"
	OrderReasonTakeProfit string = "take_profit"
)

type Reason struct {
	Reason  string `yaml:"reason" json:"reason" csv:"reason"`
	Message string `yaml:"message" json:"message" csv:"message"`
}

type ExecuteOrder struct {
	Symbol    string    `yaml:"symbol" json:"symbol" csv:"symbol"`
	OrderType OrderType `yaml:"order_type" json:"order_type" csv:"order_type"`
	Reason    Reason    `yaml:"reason" json:"reason" csv:"reason"`
}

type Order struct {
	OrderID     string    `yaml:"order_id" json:"order_id" csv:"order_id"`
	Symbol      string    `yaml:"symbol" json:"symbol" csv:"symbol"`
	OrderType   OrderType `yaml:"order_type" json:"order_type" csv:"order_type"`
	Quantity    float64   `yaml:"quantity" json:"quantity" csv:"quantity"`
	Price       float64   `yaml:"price" json:"price" csv:"price"`
	Timestamp   time.Time `yaml:"timestamp" json:"timestamp" csv:"timestamp"`
	IsCompleted bool      `yaml:"is_completed" json:"is_completed" csv:"is_completed"`
	// Reason is the reason for the order
	// It can be used to store the reason for the order
	// like "buy_signal", "sell_signal", "stop_loss", "take_profit", etc.
	Reason Reason `yaml:"reason" json:"reason" csv:"reason"`
	// StrategyName is the name of the strategy that created this order
	StrategyName string  `yaml:"strategy_name" json:"strategy_name" csv:"strategy_name"`
	Fee          float64 `yaml:"fee" json:"fee" csv:"fee"`
}

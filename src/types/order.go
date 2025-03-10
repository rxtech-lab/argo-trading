package types

import "time"

type OrderType string

const (
	OrderTypeBuy  OrderType = "BUY"
	OrderTypeSell OrderType = "SELL"
)

type Order struct {
	Symbol      string
	OrderType   OrderType
	Quantity    float64
	Price       float64
	Timestamp   time.Time
	OrderID     string
	IsCompleted bool
}

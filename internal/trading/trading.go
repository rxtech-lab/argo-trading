package trading

import "github.com/rxtech-lab/argo-trading/internal/types"

type TradingSystem interface {
	// PlaceOrder places a single order
	PlaceOrder(order types.ExecuteOrder) error
	// PlaceMultipleOrders places multiple orders
	PlaceMultipleOrders(orders []types.ExecuteOrder) error
	// GetPositions returns the current positions
	GetPositions() ([]types.Position, error)
	// GetPosition returns the current position for a symbol
	GetPosition(symbol string) (types.Position, error)
	// CancelOrder cancels an order
	CancelOrder(orderID string) error
	// CancelAllOrders cancels all orders
	CancelAllOrders() error
	// GetOrderStatus returns the status of an order
	GetOrderStatus(orderID string) (types.OrderStatus, error)
}

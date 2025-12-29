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
	// GetAccountInfo returns the current account state including balance, equity, and P&L
	GetAccountInfo() (types.AccountInfo, error)
	// GetOpenOrders returns all pending/open orders that have not been executed yet
	GetOpenOrders() ([]types.ExecuteOrder, error)
	// GetTrades returns executed trades with optional filtering
	GetTrades(filter types.TradeFilter) ([]types.Trade, error)
	// GetMaxBuyQuantity returns the maximum quantity that can be bought at the given price.
	// It takes into account the current balance and commission fees.
	GetMaxBuyQuantity(symbol string, price float64) (float64, error)
	// GetMaxSellQuantity returns the maximum quantity that can be sold for a symbol.
	// This is the total long position quantity for the symbol.
	GetMaxSellQuantity(symbol string) (float64, error)
}

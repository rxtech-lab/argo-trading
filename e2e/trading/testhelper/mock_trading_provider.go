package testhelper

import (
	"context"
	"fmt"
	"sync"
	"time"

	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MockTradingProvider implements TradingSystemProvider for testing.
// It provides in-memory tracking of positions, balance, and trades
// with instant order execution at the current price.
type MockTradingProvider struct {
	mu sync.RWMutex

	// State
	balance    float64
	positions  map[string]*types.Position
	orders     []types.ExecuteOrder
	trades     []types.Trade
	openOrders []types.ExecuteOrder

	// Current market data (updated by engine or test)
	currentPrice map[string]float64

	// Behavior configuration
	FailAllOrders bool
	FailReason    string
}

// NewMockTradingProvider creates a new mock trading provider with the given initial balance.
func NewMockTradingProvider(initialBalance float64) *MockTradingProvider {
	return &MockTradingProvider{
		mu:            sync.RWMutex{},
		balance:       initialBalance,
		positions:     make(map[string]*types.Position),
		orders:        make([]types.ExecuteOrder, 0),
		trades:        make([]types.Trade, 0),
		openOrders:    make([]types.ExecuteOrder, 0),
		currentPrice:  make(map[string]float64),
		FailAllOrders: false,
		FailReason:    "",
	}
}

// SetCurrentPrice updates the current price for a symbol.
// This should be called by the test or engine when market data is received.
func (m *MockTradingProvider) SetCurrentPrice(symbol string, price float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.currentPrice[symbol] = price
}

// PlaceOrder executes an order instantly at the current price.
func (m *MockTradingProvider) PlaceOrder(order types.ExecuteOrder) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check for configured failure
	if m.FailAllOrders {
		return fmt.Errorf("order failed: %s", m.FailReason)
	}

	// Record order
	m.orders = append(m.orders, order)

	// Get execution price
	price := order.Price
	if price == 0 {
		if p, ok := m.currentPrice[order.Symbol]; ok {
			price = p
		}
	}

	if price == 0 {
		return fmt.Errorf("no price available for %s", order.Symbol)
	}

	// Calculate cost
	cost := price * order.Quantity

	// Execute based on side
	if order.Side == types.PurchaseTypeBuy {
		if cost > m.balance {
			return fmt.Errorf("insufficient balance: need %.2f, have %.2f", cost, m.balance)
		}

		m.balance -= cost

		// Update position
		pos := m.getOrCreatePosition(order.Symbol)
		pos.TotalLongPositionQuantity += order.Quantity
		pos.TotalLongInPositionQuantity += order.Quantity
		pos.TotalLongInPositionAmount += cost

		if pos.OpenTimestamp.IsZero() {
			pos.OpenTimestamp = time.Now()
		}
	} else {
		// Sell
		pos := m.getOrCreatePosition(order.Symbol)
		if order.Quantity > pos.TotalLongPositionQuantity {
			return fmt.Errorf("insufficient position: need %.2f, have %.2f",
				order.Quantity, pos.TotalLongPositionQuantity)
		}

		m.balance += cost
		pos.TotalLongPositionQuantity -= order.Quantity
		pos.TotalLongOutPositionQuantity += order.Quantity
		pos.TotalLongOutPositionAmount += cost
	}

	// Record trade
	trade := types.Trade{
		Order: types.Order{
			OrderID:      order.ID,
			Symbol:       order.Symbol,
			Side:         order.Side,
			Quantity:     order.Quantity,
			Price:        price,
			Timestamp:    time.Now(),
			IsCompleted:  true,
			Status:       types.OrderStatusFilled,
			Reason:       order.Reason,
			StrategyName: order.StrategyName,
			Fee:          0, // No fees in mock
			PositionType: order.PositionType,
		},
		ExecutedAt:    time.Now(),
		ExecutedQty:   order.Quantity,
		ExecutedPrice: price,
		Fee:           0,
		PnL:           0, // PnL would need to be calculated based on entry/exit
	}
	m.trades = append(m.trades, trade)

	return nil
}

// getOrCreatePosition returns existing position or creates a new one.
func (m *MockTradingProvider) getOrCreatePosition(symbol string) *types.Position {
	if pos, ok := m.positions[symbol]; ok {
		return pos
	}

	pos := &types.Position{
		Symbol:                        symbol,
		TotalLongPositionQuantity:     0,
		TotalShortPositionQuantity:    0,
		TotalLongInPositionQuantity:   0,
		TotalLongOutPositionQuantity:  0,
		TotalLongInPositionAmount:     0,
		TotalLongOutPositionAmount:    0,
		TotalShortInPositionQuantity:  0,
		TotalShortOutPositionQuantity: 0,
		TotalShortInPositionAmount:    0,
		TotalShortOutPositionAmount:   0,
		TotalLongInFee:                0,
		TotalLongOutFee:               0,
		TotalShortInFee:               0,
		TotalShortOutFee:              0,
		OpenTimestamp:                 time.Time{},
		StrategyName:                  "",
	}
	m.positions[symbol] = pos

	return pos
}

// PlaceMultipleOrders places multiple orders sequentially.
func (m *MockTradingProvider) PlaceMultipleOrders(orders []types.ExecuteOrder) error {
	for _, o := range orders {
		if err := m.PlaceOrder(o); err != nil {
			return err
		}
	}

	return nil
}

// GetPositions returns all positions.
func (m *MockTradingProvider) GetPositions() ([]types.Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]types.Position, 0, len(m.positions))
	for _, pos := range m.positions {
		result = append(result, *pos)
	}

	return result, nil
}

// GetPosition returns a specific position.
func (m *MockTradingProvider) GetPosition(symbol string) (types.Position, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if pos, ok := m.positions[symbol]; ok {
		return *pos, nil
	}

	return types.Position{Symbol: symbol}, nil //nolint:exhaustruct // empty position
}

// CancelOrder cancels an order (no-op in mock since orders execute instantly).
func (m *MockTradingProvider) CancelOrder(_ string) error {
	return nil
}

// CancelAllOrders cancels all orders (no-op in mock since orders execute instantly).
func (m *MockTradingProvider) CancelAllOrders() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.openOrders = make([]types.ExecuteOrder, 0)

	return nil
}

// GetOrderStatus returns the status of an order.
func (m *MockTradingProvider) GetOrderStatus(_ string) (types.OrderStatus, error) {
	// All orders in mock execute instantly
	return types.OrderStatusFilled, nil
}

// GetAccountInfo returns account information.
func (m *MockTradingProvider) GetAccountInfo() (types.AccountInfo, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return types.AccountInfo{
		Balance:       m.balance,
		Equity:        m.balance, // Simplified - doesn't include unrealized P&L
		BuyingPower:   m.balance,
		RealizedPnL:   0,
		UnrealizedPnL: 0,
		TotalFees:     0,
		MarginUsed:    0,
	}, nil
}

// GetOpenOrders returns all open orders.
func (m *MockTradingProvider) GetOpenOrders() ([]types.ExecuteOrder, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// In mock, orders execute instantly, so there are no open orders
	return m.openOrders, nil
}

// GetTrades returns executed trades with optional filtering.
func (m *MockTradingProvider) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// If no filter, return all trades
	if filter.Symbol == "" && filter.StartTime.IsZero() && filter.EndTime.IsZero() && filter.Limit == 0 {
		return m.trades, nil
	}

	result := make([]types.Trade, 0)

	for _, trade := range m.trades {
		// Filter by symbol
		if filter.Symbol != "" && trade.Order.Symbol != filter.Symbol {
			continue
		}

		// Filter by time range
		if !filter.StartTime.IsZero() && trade.ExecutedAt.Before(filter.StartTime) {
			continue
		}

		if !filter.EndTime.IsZero() && trade.ExecutedAt.After(filter.EndTime) {
			continue
		}

		result = append(result, trade)

		// Apply limit
		if filter.Limit > 0 && len(result) >= filter.Limit {
			break
		}
	}

	return result, nil
}

// GetMaxBuyQuantity returns the maximum quantity that can be bought at the given price.
func (m *MockTradingProvider) GetMaxBuyQuantity(_ string, price float64) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if price <= 0 {
		return 0, fmt.Errorf("price must be positive")
	}

	return m.balance / price, nil
}

// GetMaxSellQuantity returns the maximum quantity that can be sold for a symbol.
func (m *MockTradingProvider) GetMaxSellQuantity(symbol string) (float64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if pos, ok := m.positions[symbol]; ok {
		return pos.TotalLongPositionQuantity, nil
	}

	return 0, nil
}

// GetAllTrades returns all trades without filter (convenience for tests).
func (m *MockTradingProvider) GetAllTrades() []types.Trade {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make([]types.Trade, len(m.trades))
	copy(result, m.trades)

	return result
}

// GetBalance returns the current balance (convenience for tests).
func (m *MockTradingProvider) GetBalance() float64 {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.balance
}

// Reset clears all state (useful for test setup).
func (m *MockTradingProvider) Reset(initialBalance float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.balance = initialBalance
	m.positions = make(map[string]*types.Position)
	m.orders = make([]types.ExecuteOrder, 0)
	m.trades = make([]types.Trade, 0)
	m.openOrders = make([]types.ExecuteOrder, 0)
	m.currentPrice = make(map[string]float64)
	m.FailAllOrders = false
	m.FailReason = ""
}

// CheckConnection implements tradingprovider.TradingSystemProvider.
// For mock provider, this always returns nil as it's always connected.
func (m *MockTradingProvider) CheckConnection(_ context.Context) error {
	return nil
}

// SetOnStatusChange implements tradingprovider.TradingSystemProvider.
// For mock provider, this is a no-op as it's always connected.
func (m *MockTradingProvider) SetOnStatusChange(_ tradingprovider.OnStatusChange) {
	// No-op for mock provider
}

// Verify MockTradingProvider implements TradingSystemProvider interface.
var _ tradingprovider.TradingSystemProvider = (*MockTradingProvider)(nil)

package engine

import (
	"fmt"
	"slices"
	"time"

	"github.com/go-playground/validator"
	"github.com/google/uuid"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/trading"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/internal/utils"
)

// BacktestTrading is a trading system that is used to backtest a trading strategy.
type BacktestTrading struct {
	state         *BacktestState
	balance       float64
	marketData    types.MarketData
	pendingOrders []types.ExecuteOrder
	commission    commission_fee.CommissionFee
}

func (b *BacktestTrading) UpdateCurrentMarketData(marketData types.MarketData) {
	b.marketData = marketData
}

func (b *BacktestTrading) UpdateBalance(balance float64) {
	b.balance = balance
}

// CancelAllOrders implements trading.TradingSystem.
func (b *BacktestTrading) CancelAllOrders() error {
	b.pendingOrders = []types.ExecuteOrder{}

	return nil
}

// CancelOrder implements trading.TradingSystem.
func (b *BacktestTrading) CancelOrder(orderID string) error {
	for i, order := range b.pendingOrders {
		if order.ID == orderID {
			b.pendingOrders = slices.Delete(b.pendingOrders, i, i+1)

			return nil
		}
	}

	return nil
}

// GetOrderStatus implements trading.TradingSystem.
func (b *BacktestTrading) GetOrderStatus(orderID string) (types.OrderStatus, error) {
	order, err := b.state.GetOrderById(orderID)
	if err != nil {
		return types.OrderStatusFailed, err
	}

	if order.IsNone() {
		return types.OrderStatusFailed, fmt.Errorf("order not found")
	}

	value, err := order.Take()
	if err != nil {
		return types.OrderStatusFailed, err
	}

	if value.IsCompleted {
		return types.OrderStatusFilled, nil
	}

	// check if the order is in the pending orders
	for _, pendingOrder := range b.pendingOrders {
		if pendingOrder.ID == orderID {
			return types.OrderStatusPending, nil
		}
	}

	return types.OrderStatusFailed, nil
}

// GetPosition implements trading.TradingSystem.
func (b *BacktestTrading) GetPosition(symbol string) (types.Position, error) {
	position, err := b.state.GetPosition(symbol)
	if err != nil {
		return types.Position{}, err
	}

	return position, nil
}

// GetPositions implements trading.TradingSystem.
func (b *BacktestTrading) GetPositions() ([]types.Position, error) {
	positions, err := b.state.GetAllPositions()
	if err != nil {
		return []types.Position{}, err
	}

	return positions, nil
}

// PlaceMultipleOrders implements trading.TradingSystem.
func (b *BacktestTrading) PlaceMultipleOrders(orders []types.ExecuteOrder) error {
	for _, order := range orders {
		err := b.PlaceOrder(order)
		if err != nil {
			return err
		}
	}

	return nil
}

// PlaceOrder implements trading.TradingSystem.
func (b *BacktestTrading) PlaceOrder(order types.ExecuteOrder) error {
	// validate the order using go-playground/validator/v10
	order.ID = uuid.New().String()
	validate := validator.New()

	err := validate.Struct(order)
	if err != nil {
		return err
	}

	// check if the order's price is within the price range
	if order.Price < b.marketData.Low {
		return fmt.Errorf("order price is out of range")
	}

	// check if the order's price is above the range
	executePrice := order.Price
	if order.Price > b.marketData.High {
		executePrice = b.marketData.High
	}

	if executePrice <= 0 {
		return fmt.Errorf("order price is out of range: %f", executePrice)
	}

	// check if the order's quantity is less than the current buying power
	if order.Side == types.PurchaseTypeBuy {
		buyingPower := b.getBuyingPower()
		if order.Quantity > buyingPower {
			return fmt.Errorf("order quantity is greater than the current buying power")
		}
	} else {
		sellingPower := b.getSellingPower()
		if order.Quantity > sellingPower {
			return fmt.Errorf("order quantity is greater than the current selling power")
		}
	}
	// create the order
	commission := b.commission.Calculate(order.Quantity)
	executedOrder := types.Order{
		Symbol:       order.Symbol,
		Side:         order.Side,
		Quantity:     order.Quantity,
		Price:        executePrice,
		Timestamp:    time.Now(),
		IsCompleted:  false,
		Reason:       order.Reason,
		StrategyName: order.StrategyName,
		Fee:          commission,
	}
	// place the order
	_, err = b.state.Update([]types.Order{executedOrder})
	if err != nil {
		return err
	}

	return nil
}

func (b *BacktestTrading) getBuyingPower() float64 {
	return float64(utils.CalculateMaxQuantity(b.balance, (b.marketData.High+b.marketData.Low)/2, b.commission))
}

func (b *BacktestTrading) getSellingPower() float64 {
	// get current position
	position, err := b.GetPosition(b.marketData.Symbol)
	if err != nil {
		return 0
	}

	return position.Quantity
}

func NewBacktestTrading(state *BacktestState, initialBalance float64, commission commission_fee.CommissionFee) trading.TradingSystem {
	return &BacktestTrading{
		state:         state,
		balance:       initialBalance,
		marketData:    types.MarketData{},
		pendingOrders: []types.ExecuteOrder{},
		commission:    commission,
	}
}

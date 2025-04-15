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
	state            *BacktestState
	balance          float64
	marketData       types.MarketData
	pendingOrders    []types.ExecuteOrder
	commission       commission_fee.CommissionFee
	decimalPrecision int
}

func (b *BacktestTrading) UpdateCurrentMarketData(marketData types.MarketData) {
	b.marketData = marketData

	// Process pending orders with the updated market data
	b.processPendingOrders()
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
// Market orders:
//   - Always use the average price of the market data.
//   - Fail if AVG price * quantity > buying power for buy orders.
//   - If selling quantity > current selling power (MAX Holding), set it to the max.
//
// Limit orders:
//   - Fail if quantity * price > buying power.
//   - If total sold quantity > max holding, sell all max holding and modify the order quantity.
//   - For buy orders, if limit price is higher than market price, use market price.
//   - For sell orders, only sell if market price is >= limit price, and use limit price as execution price.
func (b *BacktestTrading) PlaceOrder(order types.ExecuteOrder) error {
	// validate the order using go-playground/validator/v10
	order.ID = uuid.New().String()
	validate := validator.New()

	err := validate.Struct(order)
	if err != nil {
		return err
	}

	// Round the quantity to respect configured decimal precision
	order.Quantity = utils.RoundToDecimalPrecision(order.Quantity, b.decimalPrecision)
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity is too small or zero after rounding to configured precision")
	}

	// Handle limit orders
	if order.OrderType == types.OrderTypeLimit {
		// Check if the order's price is valid (greater than zero)
		if order.Price <= 0 {
			return fmt.Errorf("limit order price must be greater than zero: %f", order.Price)
		}

		// For buy orders, check if quantity * price exceeds buying power
		if order.Side == types.PurchaseTypeBuy {
			// Check if we can afford this order
			totalCost := order.Quantity * order.Price
			if totalCost > b.balance {
				return fmt.Errorf("limit buy order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance)
			}

			// If current price is already below limit price, execute immediately with the current market price
			if b.marketData.Low <= order.Price {
				// Modify the order to use current market price if lower than limit price
				marketOrder := order
				// We'll let executeMarketOrder set the appropriate price
				return b.executeMarketOrder(marketOrder)
			}

			// Otherwise, add to pending orders
			b.pendingOrders = append(b.pendingOrders, order)

			return nil
		}

		// For limit sell orders, check if the quantity exceeds current holdings
		if order.Side == types.PurchaseTypeSell {
			sellingPower := b.getSellingPower()

			// If trying to sell more than available, adjust quantity to max available
			if order.Quantity > sellingPower {
				if sellingPower <= 0 {
					return fmt.Errorf("no shares available to sell")
				}

				order.Quantity = sellingPower
			}

			// If current price is already above limit price, execute immediately with the limit price
			if b.marketData.High >= order.Price {
				return b.executeMarketOrder(order)
			}

			// Otherwise, add to pending orders
			b.pendingOrders = append(b.pendingOrders, order)

			return nil
		}

		return nil
	}

	// For market orders, execute immediately
	if order.OrderType == types.OrderTypeMarket {
		// Calculate average market price
		avgPrice := (b.marketData.High + b.marketData.Low) / 2

		if avgPrice <= 0 {
			return fmt.Errorf("invalid market data: average price is zero or negative")
		}

		// Set the order price to the average price
		order.Price = avgPrice

		// For buy orders, check if we can afford this order
		if order.Side == types.PurchaseTypeBuy {
			totalCost := order.Quantity * avgPrice
			if totalCost > b.balance {
				return fmt.Errorf("market buy order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance)
			}
		} else {
			// For sell orders, adjust quantity if needed
			sellingPower := b.getSellingPower()
			if order.Quantity > sellingPower {
				if sellingPower <= 0 {
					return fmt.Errorf("no shares available to sell")
				}

				order.Quantity = sellingPower
			}
		}

		// Execute the market order
		return b.executeMarketOrder(order)
	}

	// Process take profit and stop loss orders if present
	if !order.TakeProfit.IsNone() {
		takeProfitOrder, _ := order.TakeProfit.Take()

		// Create a limit order for take profit
		tpOrder := types.ExecuteOrder{
			ID:           uuid.New().String(),
			Symbol:       order.Symbol,
			Side:         takeProfitOrder.Side,
			OrderType:    types.OrderTypeLimit,
			Reason:       types.Reason{Reason: types.OrderReasonTakeProfit, Message: "Take profit order"},
			Price:        order.Price, // This needs to be set by the caller based on the take profit level
			StrategyName: order.StrategyName,
			Quantity:     order.Quantity,
			PositionType: order.PositionType,
		}

		// Add to pending orders
		b.pendingOrders = append(b.pendingOrders, tpOrder)
	}

	if !order.StopLoss.IsNone() {
		stopLossOrder, _ := order.StopLoss.Take()

		// Create a limit order for stop loss
		slOrder := types.ExecuteOrder{
			ID:           uuid.New().String(),
			Symbol:       order.Symbol,
			Side:         stopLossOrder.Side,
			OrderType:    types.OrderTypeLimit,
			Reason:       types.Reason{Reason: types.OrderReasonStopLoss, Message: "Stop loss order"},
			Price:        order.Price, // This needs to be set by the caller based on the stop loss level
			StrategyName: order.StrategyName,
			Quantity:     order.Quantity,
			PositionType: order.PositionType,
		}

		// Add to pending orders
		b.pendingOrders = append(b.pendingOrders, slOrder)
	}

	return nil
}

func (b *BacktestTrading) getBuyingPower() float64 {
	maxQty := utils.CalculateMaxQuantity(b.balance, (b.marketData.High+b.marketData.Low)/2, b.commission)

	return utils.RoundToDecimalPrecision(maxQty, b.decimalPrecision)
}

func (b *BacktestTrading) getSellingPower() float64 {
	// get current position
	position, err := b.GetPosition(b.marketData.Symbol)
	if err != nil {
		return 0
	}

	return utils.RoundToDecimalPrecision(position.Quantity, b.decimalPrecision)
}

func NewBacktestTrading(state *BacktestState, initialBalance float64, commission commission_fee.CommissionFee, decimalPrecision int) trading.TradingSystem {
	return &BacktestTrading{
		state:            state,
		balance:          initialBalance,
		marketData:       types.MarketData{},
		pendingOrders:    []types.ExecuteOrder{},
		commission:       commission,
		decimalPrecision: decimalPrecision,
	}
}

// processPendingOrders processes all pending limit orders based on current market data.
func (b *BacktestTrading) processPendingOrders() {
	if len(b.pendingOrders) == 0 {
		return
	}

	var remainingOrders []types.ExecuteOrder

	var ordersToExecute []types.ExecuteOrder

	// Check each pending order to see if it can be executed with current market data
	for _, order := range b.pendingOrders {
		canExecute := false

		// For limit buy orders, we execute if market price has fallen below or equal to the limit price
		if order.Side == types.PurchaseTypeBuy && order.OrderType == types.OrderTypeLimit {
			// Buy when price falls to or below limit price
			if b.marketData.Low <= order.Price {
				canExecute = true
			}
		}

		// For limit sell orders, we execute if market price has risen above or equal to the limit price
		if order.Side == types.PurchaseTypeSell && order.OrderType == types.OrderTypeLimit {
			// Sell when price rises to or above limit price
			if b.marketData.High >= order.Price {
				canExecute = true
			}
		}

		if canExecute {
			ordersToExecute = append(ordersToExecute, order)
		} else {
			remainingOrders = append(remainingOrders, order)
		}
	}

	// Update the list of pending orders
	b.pendingOrders = remainingOrders

	// Execute the orders that can be executed
	for _, order := range ordersToExecute {
		// Execute the order with its original properties
		// Ignore errors - if one order fails, try to execute the rest
		_ = b.executeMarketOrder(order)
	}
}

// executeMarketOrder executes a market order immediately.
func (b *BacktestTrading) executeMarketOrder(order types.ExecuteOrder) error {
	// Validate the order (quantity, buying power, etc.)
	order.Quantity = utils.RoundToDecimalPrecision(order.Quantity, b.decimalPrecision)
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity is too small or zero after rounding to configured precision")
	}

	// Determine execution price based on order type and market data
	var executePrice float64

	if order.OrderType == types.OrderTypeMarket {
		// For market orders, always use the average price
		executePrice = (b.marketData.High + b.marketData.Low) / 2
	} else if order.OrderType == types.OrderTypeLimit {
		if order.Side == types.PurchaseTypeBuy {
			// For buy limit orders, use the lower of limit price and current market price
			executePrice = order.Price
			avgPrice := (b.marketData.High + b.marketData.Low) / 2

			if avgPrice < executePrice {
				executePrice = avgPrice
			}
		} else {
			// For sell limit orders, use the limit price
			executePrice = order.Price
		}
	}

	if executePrice <= 0 {
		return fmt.Errorf("execution price is invalid: %f", executePrice)
	}

	// Check buying/selling power again with final execution price
	if order.Side == types.PurchaseTypeBuy {
		totalCost := order.Quantity * executePrice
		if totalCost > b.balance {
			return fmt.Errorf("order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance)
		}
	} else {
		sellingPower := b.getSellingPower()
		if order.Quantity > sellingPower {
			if sellingPower <= 0 {
				return fmt.Errorf("no shares available to sell")
			}

			order.Quantity = sellingPower
		}
	}

	// Calculate commission fee
	commission := b.commission.Calculate(order.Quantity)

	// Create the executed order
	executedOrder := types.Order{
		OrderID:      order.ID,
		Symbol:       order.Symbol,
		Side:         order.Side,
		Quantity:     order.Quantity,
		Price:        executePrice,
		Timestamp:    time.Now(),
		IsCompleted:  true,
		Reason:       order.Reason,
		StrategyName: order.StrategyName,
		Fee:          commission,
		PositionType: order.PositionType,
	}

	// Update the order in the state
	_, err := b.state.Update([]types.Order{executedOrder})

	return err
}

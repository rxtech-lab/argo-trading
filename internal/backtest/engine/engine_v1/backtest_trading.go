package engine

import (
	"fmt"
	"slices"
	"time"

	"github.com/google/uuid"
	"github.com/moznion/go-optional"
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
	order.ID = uuid.New().String()

	// Check for invalid quantity before struct validation
	if order.Quantity <= 0 {
		failedOrder := b.createFailedOrder(order, order.Price, types.OrderReasonInvalidQuantity,
			fmt.Sprintf("order quantity must be greater than zero: %.2f", order.Quantity))

		return b.state.StoreFailedOrder(failedOrder)
	}

	// Check for invalid price before struct validation
	if order.Price <= 0 {
		failedOrder := b.createFailedOrder(order, order.Price, types.OrderReasonInvalidPrice,
			fmt.Sprintf("order price must be greater than zero: %.2f", order.Price))

		return b.state.StoreFailedOrder(failedOrder)
	}

	// validate the order using go-playground/validator/v10
	if err := order.Validate(); err != nil {
		return err
	}

	// Round the quantity to respect configured decimal precision
	order.Quantity = utils.RoundToDecimalPrecision(order.Quantity, b.decimalPrecision)
	if order.Quantity <= 0 {
		return fmt.Errorf("order quantity is too small or zero after rounding to configured precision")
	}

	// Check if the symbol matches current market data symbol
	// If not, add to pending orders and return (no errors)
	if order.Symbol != b.marketData.Symbol {
		b.pendingOrders = append(b.pendingOrders, order)

		return nil
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
				failedOrder := b.createFailedOrder(order, order.Price, types.OrderReasonInsufficientBuyPower,
					fmt.Sprintf("limit buy order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance))

				return b.state.StoreFailedOrder(failedOrder)
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

			// If trying to sell more than available, fail the order
			if order.Quantity > sellingPower {
				failedOrder := b.createFailedOrder(order, order.Price, types.OrderReasonInsufficientSellPower,
					fmt.Sprintf("order quantity (%.2f) exceeds selling power (%.2f)", order.Quantity, sellingPower))

				return b.state.StoreFailedOrder(failedOrder)
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
				failedOrder := b.createFailedOrder(order, avgPrice, types.OrderReasonInsufficientBuyPower,
					fmt.Sprintf("market buy order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance))

				return b.state.StoreFailedOrder(failedOrder)
			}
		} else {
			// For sell orders, fail if quantity exceeds selling power
			sellingPower := b.getSellingPower()
			if order.Quantity > sellingPower {
				failedOrder := b.createFailedOrder(order, avgPrice, types.OrderReasonInsufficientSellPower,
					fmt.Sprintf("order quantity (%.2f) exceeds selling power (%.2f)", order.Quantity, sellingPower))

				return b.state.StoreFailedOrder(failedOrder)
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
			TakeProfit:   optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
			StopLoss:     optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
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
			TakeProfit:   optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
			StopLoss:     optional.None[types.ExecuteOrderTakeProfitOrStopLoss](),
		}

		// Add to pending orders
		b.pendingOrders = append(b.pendingOrders, slOrder)
	}

	return nil
}

func (b *BacktestTrading) Reset(initialBalance float64) {
	b.pendingOrders = []types.ExecuteOrder{}
	b.balance = initialBalance
	b.marketData = types.MarketData{
		Id:     "",
		Symbol: "",
		Time:   time.Time{},
		Open:   0,
		High:   0,
		Low:    0,
		Close:  0,
		Volume: 0,
	}
}

// GetAccountInfo implements trading.TradingSystem.
// Returns the current account state including balance, equity, and P&L information.
func (b *BacktestTrading) GetAccountInfo() (types.AccountInfo, error) {
	positions, err := b.state.GetAllPositions()
	if err != nil {
		return types.AccountInfo{}, err
	}

	var realizedPnL, unrealizedPnL, totalFees float64

	for _, pos := range positions {
		// Add realized PnL from this position
		realizedPnL += pos.GetTotalPnL()

		// Calculate unrealized PnL for open long positions
		if pos.TotalLongPositionQuantity > 0 {
			currentPrice := b.marketData.Close
			if currentPrice == 0 {
				currentPrice = (b.marketData.High + b.marketData.Low) / 2
			}

			avgEntry := pos.GetAverageLongPositionEntryPrice()
			unrealizedPnL += (currentPrice - avgEntry) * pos.TotalLongPositionQuantity
		}

		// Calculate unrealized PnL for open short positions
		if pos.TotalShortPositionQuantity > 0 {
			currentPrice := b.marketData.Close
			if currentPrice == 0 {
				currentPrice = (b.marketData.High + b.marketData.Low) / 2
			}

			avgEntry := pos.GetAverageShortPositionEntryPrice()
			unrealizedPnL += (avgEntry - currentPrice) * pos.TotalShortPositionQuantity
		}

		// Accumulate total fees
		totalFees += pos.TotalLongInFee + pos.TotalLongOutFee + pos.TotalShortInFee + pos.TotalShortOutFee
	}

	equity := b.balance + unrealizedPnL
	buyingPower := b.getBuyingPower()

	return types.AccountInfo{
		Balance:       b.balance,
		Equity:        equity,
		BuyingPower:   buyingPower,
		RealizedPnL:   realizedPnL,
		UnrealizedPnL: unrealizedPnL,
		TotalFees:     totalFees,
		MarginUsed:    0, // Not implemented for backtesting
	}, nil
}

// GetOpenOrders implements trading.TradingSystem.
// Returns all pending/open orders that have not been executed yet.
func (b *BacktestTrading) GetOpenOrders() ([]types.ExecuteOrder, error) {
	// Return a copy to prevent external modification
	orders := make([]types.ExecuteOrder, len(b.pendingOrders))
	copy(orders, b.pendingOrders)

	return orders, nil
}

// GetTrades implements trading.TradingSystem.
// Returns executed trades with optional filtering by symbol, time range, and limit.
func (b *BacktestTrading) GetTrades(filter types.TradeFilter) ([]types.Trade, error) {
	allTrades, err := b.state.GetAllTrades()
	if err != nil {
		return nil, err
	}

	filteredTrades := make([]types.Trade, 0, len(allTrades))

	for _, trade := range allTrades {
		// Apply symbol filter
		if filter.Symbol != "" && trade.Order.Symbol != filter.Symbol {
			continue
		}

		// Apply start time filter
		if !filter.StartTime.IsZero() && trade.ExecutedAt.Before(filter.StartTime) {
			continue
		}

		// Apply end time filter
		if !filter.EndTime.IsZero() && trade.ExecutedAt.After(filter.EndTime) {
			continue
		}

		filteredTrades = append(filteredTrades, trade)
	}

	// Apply limit
	if filter.Limit > 0 && len(filteredTrades) > filter.Limit {
		filteredTrades = filteredTrades[:filter.Limit]
	}

	return filteredTrades, nil
}

func NewBacktestTrading(state *BacktestState, initialBalance float64, commission commission_fee.CommissionFee, decimalPrecision int) trading.TradingSystem {
	return &BacktestTrading{
		state:   state,
		balance: initialBalance,
		marketData: types.MarketData{
			Id:     "",
			Symbol: "",
			Time:   time.Time{},
			Open:   0,
			High:   0,
			Low:    0,
			Close:  0,
			Volume: 0,
		},
		pendingOrders:    []types.ExecuteOrder{},
		commission:       commission,
		decimalPrecision: decimalPrecision,
	}
}

// GetMaxBuyQuantity implements trading.TradingSystem.
// Returns the maximum quantity that can be bought for a symbol at the given price.
func (b *BacktestTrading) GetMaxBuyQuantity(symbol string, price float64) (float64, error) {
	if price <= 0 {
		return 0, fmt.Errorf("price must be greater than zero")
	}

	if b.balance <= 0 {
		return 0, nil
	}

	maxQty := utils.CalculateMaxQuantity(b.balance, price, b.commission)

	return utils.RoundToDecimalPrecision(maxQty, b.decimalPrecision), nil
}

// GetMaxSellQuantity implements trading.TradingSystem.
// Returns the maximum quantity that can be sold for a symbol (total long position quantity).
func (b *BacktestTrading) GetMaxSellQuantity(symbol string) (float64, error) {
	position, err := b.state.GetPosition(symbol)
	if err != nil {
		return 0, nil
	}

	return utils.RoundToDecimalPrecision(position.TotalLongPositionQuantity, b.decimalPrecision), nil
}

// getBuyingPower returns the maximum quantity that can be bought for the current market data.
func (b *BacktestTrading) getBuyingPower() float64 {
	maxQty := utils.CalculateMaxQuantity(b.balance, (b.marketData.High+b.marketData.Low)/2, b.commission)

	return utils.RoundToDecimalPrecision(maxQty, b.decimalPrecision)
}

// getSellingPower returns the maximum quantity that can be sold for the current market data.
func (b *BacktestTrading) getSellingPower() float64 {
	// get current position
	position, err := b.GetPosition(b.marketData.Symbol)
	if err != nil {
		return 0
	}

	return utils.RoundToDecimalPrecision(position.TotalLongPositionQuantity, b.decimalPrecision)
}

// createFailedOrder creates a failed order with the given parameters.
// This helper consolidates the repeated failed order creation logic.
func (b *BacktestTrading) createFailedOrder(order types.ExecuteOrder, executePrice float64, reason string, message string) types.Order {
	return types.Order{
		OrderID:      order.ID,
		Symbol:       order.Symbol,
		Side:         order.Side,
		Quantity:     order.Quantity,
		Price:        executePrice,
		Timestamp:    b.marketData.Time,
		IsCompleted:  true,
		Status:       types.OrderStatusFailed,
		StrategyName: order.StrategyName,
		PositionType: order.PositionType,
		Fee:          0,
		Reason: types.Reason{
			Reason:  reason,
			Message: message,
		},
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

		// check if symbol matches current market data
		if order.Symbol != b.marketData.Symbol {
			// Keep orders with different symbols in pending orders
			remainingOrders = append(remainingOrders, order)

			continue
		}

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

		// For market orders, always execute them when their symbol matches current market data
		if order.OrderType == types.OrderTypeMarket {
			canExecute = true
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

	// check if symbol matches current market data
	if order.Symbol != b.marketData.Symbol {
		return nil
	}

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
			failedOrder := b.createFailedOrder(order, executePrice, types.OrderReasonInsufficientBuyPower,
				fmt.Sprintf("order cost (%.2f) exceeds available balance (%.2f)", totalCost, b.balance))

			return b.state.StoreFailedOrder(failedOrder)
		}
	} else {
		sellingPower := b.getSellingPower()
		if order.Quantity > sellingPower {
			failedOrder := b.createFailedOrder(order, executePrice, types.OrderReasonInsufficientSellPower,
				fmt.Sprintf("order quantity (%.2f) exceeds selling power (%.2f)", order.Quantity, sellingPower))

			return b.state.StoreFailedOrder(failedOrder)
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
		Timestamp:    b.marketData.Time,
		IsCompleted:  true,
		Status:       types.OrderStatusFilled,
		Reason:       order.Reason,
		StrategyName: order.StrategyName,
		Fee:          commission,
		PositionType: order.PositionType,
	}

	// Update the order in the state
	_, err := b.state.Update([]types.Order{executedOrder})

	return err
}

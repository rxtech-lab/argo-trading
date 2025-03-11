package engine

import (
	"errors"
	"fmt"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/sirily11/argo-trading-go/src/engine/writer"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1Config struct {
	InitialCapital float64 `yaml:"initial_capital"`
	BrokerageFee   float64 `yaml:"brokerage_fee"`
	ResultsFolder  string  `yaml:"results_folder"`
}

type BacktestEngineV1 struct {
	marketDataSource MarketDataSource
	strategies       []strategyInfo
	initialCapital   float64
	currentCapital   float64
	positions        map[string]types.Position
	pendingOrders    []types.Order
	resultsWriter    writer.ResultWriter
	resultsFolder    string
	historicalData   []types.MarketData
	equityCurve      []float64
	buyAndHoldValue  float64
	buyAndHoldPrice  float64
	buyAndHoldShares float64
}

type strategyInfo struct {
	strategy strategy.TradingStrategy
	config   string
	trades   []types.Trade
	stats    types.TradeStats
}

func NewBacktestEngineV1() *BacktestEngineV1 {
	return &BacktestEngineV1{
		strategies:     make([]strategyInfo, 0),
		positions:      make(map[string]types.Position),
		pendingOrders:  make([]types.Order, 0),
		historicalData: make([]types.MarketData, 0),
		equityCurve:    make([]float64, 0),
	}
}

func (e *BacktestEngineV1) Initialize(config string) error {
	// parse config
	var cfg BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(config), &cfg)
	if err != nil {
		return err
	}

	e.initialCapital = cfg.InitialCapital
	e.currentCapital = cfg.InitialCapital
	e.resultsFolder = cfg.ResultsFolder

	// Create and initialize the results writer
	if e.resultsFolder != "" {
		fileWriter, err := writer.NewCSVWriter(e.resultsFolder)
		if err != nil {
			return fmt.Errorf("failed to create CSV writer: %w", err)
		}
		e.resultsWriter = fileWriter
	}

	return nil
}

// SetInitialCapital sets the initial capital for the backtest
func (e *BacktestEngineV1) SetInitialCapital(amount float64) error {
	if amount <= 0 {
		return errors.New("initial capital must be positive")
	}
	e.initialCapital = amount
	e.currentCapital = amount
	return nil
}

// AddStrategy adds a strategy to be tested
func (e *BacktestEngineV1) AddStrategy(strategy strategy.TradingStrategy, config string) error {
	if strategy == nil {
		return errors.New("strategy cannot be nil")
	}

	// Create a new strategy info
	info := strategyInfo{
		strategy: strategy,
		config:   config,
		trades:   make([]types.Trade, 0),
		stats:    types.TradeStats{},
	}

	// Add to strategies list
	e.strategies = append(e.strategies, info)
	return nil
}

// AddMarketDataSource adds a market data source to the backtest engine
func (e *BacktestEngineV1) AddMarketDataSource(source MarketDataSource) error {
	if source == nil {
		return errors.New("market data source cannot be nil")
	}
	e.marketDataSource = source
	return nil
}

// Run executes the backtest
func (e *BacktestEngineV1) Run() error {
	// Validate that we have everything we need
	if e.marketDataSource == nil {
		return errors.New("market data source is required")
	}

	if len(e.strategies) == 0 {
		return errors.New("at least one strategy is required")
	}

	if e.initialCapital <= 0 {
		return errors.New("initial capital must be set")
	}

	// Reset state for a new run
	e.currentCapital = e.initialCapital
	e.positions = make(map[string]types.Position)
	e.pendingOrders = make([]types.Order, 0)
	e.historicalData = make([]types.MarketData, 0)
	e.equityCurve = make([]float64, 0)
	e.buyAndHoldValue = 0
	e.buyAndHoldPrice = 0
	e.buyAndHoldShares = 0

	// Initialize strategies
	for i := range e.strategies {
		ctx := e.createStrategyContext()
		err := e.strategies[i].strategy.Initialize(e.strategies[i].config, ctx)
		if err != nil {
			return fmt.Errorf("failed to initialize strategy %s: %w", e.strategies[i].strategy.Name(), err)
		}
	}

	// Process market data
	isFirstData := true

	for data := range e.marketDataSource.Iterator() {
		// Add to historical data (we still need this for strategy context)
		e.historicalData = append(e.historicalData, data)

		// Initialize buy and hold on first data point
		if isFirstData {
			e.initializeBuyAndHold(data)
			isFirstData = false
		}

		// Update buy and hold value
		e.updateBuyAndHoldValue(data)

		// Process pending orders with the new market data
		e.processPendingOrders(data)

		// Create strategy context
		ctx := e.createStrategyContext()

		// Process data with each strategy
		for i := range e.strategies {
			orders, err := e.strategies[i].strategy.ProcessData(ctx, data)
			if err != nil {
				fmt.Printf("Error processing data with strategy %s: %v\n", e.strategies[i].strategy.Name(), err)
				continue
			}

			// Add orders to pending orders
			for _, order := range orders {
				// Generate order ID if not provided
				if order.OrderID == "" {
					order.OrderID = uuid.New().String()
				}

				// Set timestamp if not provided
				if order.Timestamp.IsZero() {
					order.Timestamp = data.Time
				}

				// Set strategy name
				order.StrategyName = e.strategies[i].strategy.Name()

				// Write order to disk immediately
				if e.resultsWriter != nil {
					if err := e.resultsWriter.WriteOrder(order); err != nil {
						fmt.Printf("Warning: failed to write order: %v\n", err)
					}
				}

				e.pendingOrders = append(e.pendingOrders, order)
			}
		}

		// Calculate current portfolio value
		e.calculatePortfolioValue(data)
	}

	// Calculate final statistics
	e.calculateStatistics()

	// Close the writer
	if e.resultsWriter != nil {
		if err := e.resultsWriter.Close(); err != nil {
			return fmt.Errorf("failed to close results writer: %w", err)
		}
	}

	return nil
}

// GetTradeStatsByStrategy returns statistics for a specific strategy
func (e *BacktestEngineV1) GetTradeStatsByStrategy(strategyName string) types.TradeStats {
	for _, s := range e.strategies {
		if s.strategy.Name() == strategyName {
			return s.stats
		}
	}
	return types.TradeStats{} // Return empty stats if strategy not found
}

// Helper methods

// createStrategyContext creates a context for strategies to use
func (e *BacktestEngineV1) createStrategyContext() strategyContext {
	return strategyContext{
		engine: e,
	}
}

// processPendingOrders processes all pending orders
func (e *BacktestEngineV1) processPendingOrders(data types.MarketData) {
	remainingOrders := make([]types.Order, 0)

	for _, order := range e.pendingOrders {
		// Skip orders for future timestamps
		if order.Timestamp.After(data.Time) {
			remainingOrders = append(remainingOrders, order)
			continue
		}

		// Execute the order
		executed := e.executeOrder(order, data)
		if !executed {
			// If not executed, keep in pending orders
			remainingOrders = append(remainingOrders, order)

			// Update the order status in the CSV file
			if e.resultsWriter != nil {
				// Mark as not completed yet
				orderCopy := order
				orderCopy.IsCompleted = false
				if err := e.resultsWriter.WriteOrder(orderCopy); err != nil {
					fmt.Printf("Warning: failed to update pending order: %v\n", err)
				}
			}
		}
	}

	e.pendingOrders = remainingOrders
}

// executeOrder attempts to execute an order
func (e *BacktestEngineV1) executeOrder(order types.Order, data types.MarketData) bool {
	// Determine execution price (using close price for simplicity)
	executionPrice := data.Close

	// Calculate commission
	commission := executionPrice * order.Quantity * 0.001 // 0.1% commission

	// Calculate cost or proceeds
	var cost float64
	var pnl float64

	if order.OrderType == types.OrderTypeBuy {
		cost = executionPrice*order.Quantity + commission
		if cost > e.currentCapital {
			// Not enough capital
			return false
		}

		// Update capital
		e.currentCapital -= cost

		// Update position
		position, exists := e.positions[order.Symbol]
		if exists {
			// Update existing position
			totalShares := position.Quantity + order.Quantity
			totalCost := (position.AveragePrice * position.Quantity) + (executionPrice * order.Quantity)
			position.AveragePrice = totalCost / totalShares
			position.Quantity = totalShares
		} else {
			// Create new position
			position = types.Position{
				Symbol:        order.Symbol,
				Quantity:      order.Quantity,
				AveragePrice:  executionPrice,
				OpenTimestamp: data.Time,
			}
		}
		e.positions[order.Symbol] = position

		// Write position to disk
		if e.resultsWriter != nil {
			if err := e.resultsWriter.WritePosition(position); err != nil {
				fmt.Printf("Warning: failed to write position: %v\n", err)
			}
		}
	} else if order.OrderType == types.OrderTypeSell {
		// Check if we have the position
		position, exists := e.positions[order.Symbol]
		if !exists || position.Quantity < order.Quantity {
			// Not enough shares
			return false
		}

		// Calculate proceeds
		proceeds := executionPrice*order.Quantity - commission
		e.currentCapital += proceeds

		// Calculate P&L
		pnl = (executionPrice-position.AveragePrice)*order.Quantity - commission

		// Update position
		position.Quantity -= order.Quantity
		if position.Quantity <= 0 {
			// Position closed
			delete(e.positions, order.Symbol)
		} else {
			// Position reduced
			e.positions[order.Symbol] = position

			// Write updated position to disk
			if e.resultsWriter != nil {
				if err := e.resultsWriter.WritePosition(position); err != nil {
					fmt.Printf("Warning: failed to write position: %v\n", err)
				}
			}
		}
	}

	// Create a trade record
	trade := types.Trade{
		Order:         order,
		ExecutedAt:    data.Time,
		ExecutedQty:   order.Quantity,
		ExecutedPrice: executionPrice,
		Commission:    commission,
		PnL:           pnl,
	}

	// Mark the order as completed
	orderCopy := order
	orderCopy.IsCompleted = true

	// Write completed order to disk
	if e.resultsWriter != nil {
		if err := e.resultsWriter.WriteOrder(orderCopy); err != nil {
			fmt.Printf("Warning: failed to write completed order: %v\n", err)
		}
	}

	// Add to strategy's trades
	for i, s := range e.strategies {
		if s.strategy.Name() == order.StrategyName {
			e.strategies[i].trades = append(e.strategies[i].trades, trade)
			break
		}
	}

	// Write trade to disk
	if e.resultsWriter != nil {
		if err := e.resultsWriter.WriteTrade(trade); err != nil {
			fmt.Printf("Warning: failed to write trade: %v\n", err)
		}
	}

	return true
}

// calculatePortfolioValue calculates the current portfolio value
func (e *BacktestEngineV1) calculatePortfolioValue(data types.MarketData) float64 {
	value := e.currentCapital
	for _, position := range e.positions {
		value += position.Quantity * data.Close
	}

	// Add to equity curve
	e.equityCurve = append(e.equityCurve, value)

	// Write equity curve point to disk
	if e.resultsWriter != nil {
		if err := e.resultsWriter.WriteEquityCurve([]float64{value}, []time.Time{data.Time}); err != nil {
			fmt.Printf("Warning: failed to write equity curve point: %v\n", err)
		}
	}

	return value
}

// initializeBuyAndHold initializes the buy and hold strategy
func (e *BacktestEngineV1) initializeBuyAndHold(data types.MarketData) {
	e.buyAndHoldPrice = data.Close
	e.buyAndHoldShares = e.initialCapital / data.Close
	e.buyAndHoldValue = e.initialCapital
}

// updateBuyAndHoldValue updates the buy and hold value
func (e *BacktestEngineV1) updateBuyAndHoldValue(data types.MarketData) {
	e.buyAndHoldValue = e.buyAndHoldShares * data.Close
}

// calculateStatistics calculates performance statistics
func (e *BacktestEngineV1) calculateStatistics() {
	// Calculate statistics for each strategy
	for i := range e.strategies {
		trades := e.strategies[i].trades
		stats := types.TradeStats{
			TotalTrades: len(trades),
		}

		for _, trade := range trades {
			if trade.PnL > 0 {
				stats.WinningTrades++
			} else if trade.PnL < 0 {
				stats.LosingTrades++
			}
			stats.TotalPnL += trade.PnL
		}

		if stats.TotalTrades > 0 {
			stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
			stats.AverageProfitLoss = stats.TotalPnL / float64(stats.TotalTrades)
		}

		e.strategies[i].stats = stats

		// Write strategy stats to disk
		if e.resultsWriter != nil {
			if err := e.resultsWriter.WriteStrategyStats(e.strategies[i].strategy.Name(), stats); err != nil {
				fmt.Printf("Warning: failed to write strategy stats: %v\n", err)
			}
		}
	}

	// Write combined stats to disk
	if e.resultsWriter != nil {
		// Calculate combined stats
		combinedStats := types.TradeStats{
			TotalTrades:       0,
			WinningTrades:     0,
			LosingTrades:      0,
			TotalPnL:          0,
			AverageProfitLoss: 0,
		}

		for _, s := range e.strategies {
			combinedStats.TotalTrades += s.stats.TotalTrades
			combinedStats.WinningTrades += s.stats.WinningTrades
			combinedStats.LosingTrades += s.stats.LosingTrades
			combinedStats.TotalPnL += s.stats.TotalPnL
		}

		if combinedStats.TotalTrades > 0 {
			combinedStats.WinRate = float64(combinedStats.WinningTrades) / float64(combinedStats.TotalTrades)
			combinedStats.AverageProfitLoss = combinedStats.TotalPnL / float64(combinedStats.TotalTrades)
		}

		// Calculate Sharpe ratio and max drawdown from equity curve
		if len(e.equityCurve) > 0 {
			combinedStats.SharpeRatio = e.calculateSharpeRatio()
			combinedStats.MaxDrawdown = e.calculateMaxDrawdown()
		}

		if err := e.resultsWriter.WriteStats(combinedStats); err != nil {
			fmt.Printf("Warning: failed to write combined stats: %v\n", err)
		}
	}

	// Print comparison with buy and hold
	if e.buyAndHoldValue > 0 {
		finalPortfolioValue := e.currentCapital
		for _, position := range e.positions {
			finalPortfolioValue += position.Quantity * e.historicalData[len(e.historicalData)-1].Close
		}

		fmt.Printf("Final portfolio value: $%.2f\n", finalPortfolioValue)
		fmt.Printf("Buy and hold value: $%.2f\n", e.buyAndHoldValue)
		fmt.Printf("Outperformance: %.2f%%\n", (finalPortfolioValue/e.buyAndHoldValue-1)*100)
	}
}

// calculateSharpeRatio calculates the Sharpe ratio
func (e *BacktestEngineV1) calculateSharpeRatio() float64 {
	if len(e.equityCurve) < 2 {
		return 0
	}

	// Calculate daily returns
	returns := make([]float64, len(e.equityCurve)-1)
	for i := 1; i < len(e.equityCurve); i++ {
		returns[i-1] = (e.equityCurve[i] - e.equityCurve[i-1]) / e.equityCurve[i-1]
	}

	// Calculate mean return
	meanReturn := 0.0
	for _, r := range returns {
		meanReturn += r
	}
	meanReturn /= float64(len(returns))

	// Calculate standard deviation
	variance := 0.0
	for _, r := range returns {
		variance += math.Pow(r-meanReturn, 2)
	}
	variance /= float64(len(returns))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return 0
	}

	// Assume risk-free rate of 0 for simplicity
	sharpeRatio := meanReturn / stdDev * math.Sqrt(252) // Annualized

	return sharpeRatio
}

// calculateMaxDrawdown calculates the maximum drawdown
func (e *BacktestEngineV1) calculateMaxDrawdown() float64 {
	if len(e.equityCurve) < 2 {
		return 0
	}

	maxDrawdown := 0.0
	peak := e.equityCurve[0]

	for _, value := range e.equityCurve {
		if value > peak {
			peak = value
		}

		drawdown := (peak - value) / peak
		if drawdown > maxDrawdown {
			maxDrawdown = drawdown
		}
	}

	return maxDrawdown
}

// strategyContext implements the strategy.StrategyContext interface
type strategyContext struct {
	engine *BacktestEngineV1
}

func (c strategyContext) GetHistoricalData() []types.MarketData {
	return c.engine.historicalData
}

func (c strategyContext) GetCurrentPositions() []types.Position {
	positions := make([]types.Position, 0, len(c.engine.positions))
	for _, pos := range c.engine.positions {
		positions = append(positions, pos)
	}
	return positions
}

func (c strategyContext) GetPendingOrders() []types.Order {
	return c.engine.pendingOrders
}

func (c strategyContext) GetExecutedTrades() []types.Trade {
	var allTrades []types.Trade
	for _, s := range c.engine.strategies {
		allTrades = append(allTrades, s.trades...)
	}
	return allTrades
}

func (c strategyContext) GetAccountBalance() float64 {
	return c.engine.currentCapital
}

// Close finalizes the backtest and cleans up resources
func (e *BacktestEngineV1) Close() error {
	if e.resultsWriter != nil {
		return e.resultsWriter.Close()
	}
	return nil
}

package engine

import (
	"errors"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/fatih/color"
	"github.com/google/uuid"
	"github.com/schollz/progressbar/v3"
	"github.com/sirily11/argo-trading-go/src/engine/writer"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1Config struct {
	InitialCapital    float64   `yaml:"initial_capital"`
	CommissionFormula string    `yaml:"commission_formula"`
	ResultsFolder     string    `yaml:"results_folder"`
	StartTime         time.Time `yaml:"start_time"`
	EndTime           time.Time `yaml:"end_time"`
	TargetSymbol      string    `yaml:"target_symbol"`
}

type BacktestEngineV1Fees struct {
	CommissionFormula string `yaml:"commission_formula"`
}

type BacktestEngineV1 struct {
	startTime        time.Time
	endTime          time.Time
	marketDataSource types.MarketDataSource
	strategies       []strategyInfo
	initialCapital   float64
	currentCapital   float64
	positions        map[string]types.Position
	pendingOrders    []types.Order
	resultsWriter    writer.ResultWriter
	resultsFolder    string
	equityCurve      []float64
	buyAndHoldValue  float64
	buyAndHoldPrice  float64
	buyAndHoldShares float64
	fees             BacktestEngineV1Fees
	targetSymbol     string
}

type strategyInfo struct {
	strategy StrategyFactory
	config   string
	trades   []types.Trade
	stats    types.TradeStats
}

func NewBacktestEngineV1() BacktestEngine {
	return &BacktestEngineV1{
		strategies:    make([]strategyInfo, 0),
		positions:     make(map[string]types.Position),
		pendingOrders: make([]types.Order, 0),
		equityCurve:   make([]float64, 0),
	}
}

func (e *BacktestEngineV1) Initialize(config string) error {
	var cfg BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(config), &cfg)
	if err != nil {
		return fmt.Errorf("failed to parse config: %w", err)
	}

	if cfg.InitialCapital <= 0 {
		return errors.New("initial capital must be positive")
	}

	e.initialCapital = cfg.InitialCapital
	e.currentCapital = cfg.InitialCapital
	e.resultsFolder = cfg.ResultsFolder
	e.startTime = cfg.StartTime
	e.endTime = cfg.EndTime
	e.targetSymbol = cfg.TargetSymbol
	e.fees = BacktestEngineV1Fees{
		CommissionFormula: cfg.CommissionFormula,
	}

	// Initialize positions map
	e.positions = make(map[string]types.Position)

	// Initialize indicators
	e.initializeIndicators()

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

// initializeIndicators registers default indicators with the indicator registry
func (e *BacktestEngineV1) initializeIndicators() {
	// No initialization needed since we're creating indicators on demand
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
func (e *BacktestEngineV1) AddStrategy(strategy StrategyFactory, config string) error {
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
func (e *BacktestEngineV1) AddMarketDataSource(source types.MarketDataSource) error {
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
		return errors.New("initial capital must be greater than zero")
	}

	log.Printf("Initializing backtest with start time: %s and end time: %s", e.startTime, e.endTime)
	// Initialize the backtest
	e.currentCapital = e.initialCapital
	e.positions = make(map[string]types.Position)
	e.pendingOrders = make([]types.Order, 0)
	e.equityCurve = make([]float64, 0)
	e.buyAndHoldValue = 0
	e.buyAndHoldPrice = 0
	e.buyAndHoldShares = 0

	// Process market data
	isFirstData := true

	// Create a progress bar with undefined length
	bar := progressbar.NewOptions(-1,
		progressbar.OptionSetDescription("Processing market data..."),
		progressbar.OptionSetItsString("data points"),
		progressbar.OptionShowIts(),
		progressbar.OptionShowCount(),
		progressbar.OptionSetTheme(progressbar.Theme{
			Saucer:        "=",
			SaucerHead:    ">",
			SaucerPadding: " ",
			BarStart:      "[",
			BarEnd:        "]",
		}),
	)
	defer bar.Close()

	dataCount := 0
	for data := range e.marketDataSource.Iterator(e.startTime, e.endTime) {
		// Increment the progress bar
		bar.Add(1)
		dataCount++

		// Initialize buy and hold on first data point
		if isFirstData {
			e.initializeBuyAndHold(data)
			isFirstData = false
		}

		// Update buy and hold value
		e.updateBuyAndHoldValue(data)

		// Create strategy context
		ctx := NewStrategyContext(e, e.startTime, data.Time)

		// Process data with each strategy
		for i := range e.strategies {
			strategy := e.strategies[i].strategy()
			strategy.Initialize(e.strategies[i].config)

			orders, err := strategy.ProcessData(ctx, data, e.targetSymbol)
			if err != nil {
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
				order.StrategyName = strategy.Name()

				// Add to pending orders
				e.pendingOrders = append(e.pendingOrders, order)
			}
			// Process pending orders with the new market data
			e.processPendingOrders(strategy, data)
		}

		// Calculate portfolio value
		portfolioValue := e.calculatePortfolioValue(data)

		// Write portfolio value to disk
		if e.resultsWriter != nil {
			if err := e.resultsWriter.WriteEquityCurve([]float64{portfolioValue}, []time.Time{data.Time}); err != nil {
				fmt.Printf("Warning: failed to write portfolio value: %v\n", err)
			}
		}
	}
	// Calculate statistics
	bar.Finish()
	fmt.Printf("\n")
	e.calculateStatistics()

	return nil
}

// GetTradeStatsByStrategy returns statistics for a specific strategy
func (e *BacktestEngineV1) GetTradeStatsByStrategy(strategyName string) types.TradeStats {
	for _, s := range e.strategies {
		if s.strategy().Name() == strategyName {
			return s.stats
		}
	}
	return types.TradeStats{} // Return empty stats if strategy not found
}

// processPendingOrders processes all pending orders
func (e *BacktestEngineV1) processPendingOrders(strategy strategy.TradingStrategy, data types.MarketData) {
	remainingOrders := make([]types.Order, 0)

	for _, order := range e.pendingOrders {
		// Skip orders for future timestamps
		if order.Timestamp.After(data.Time) {
			remainingOrders = append(remainingOrders, order)
			continue
		}

		// Execute the order
		executed := e.executeOrder(strategy, order, data)
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
func (e *BacktestEngineV1) executeOrder(strategy strategy.TradingStrategy, order types.Order, data types.MarketData) bool {
	// Determine execution price (using close price for simplicity)
	executionPrice := data.Close

	// Calculate commission using the formula
	commission, err := e.calculateCommission(order, executionPrice)
	if err != nil {
		fmt.Printf("Warning: failed to calculate commission: %v\n", err)
		return false
	}

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
			stats.RealizedPnL += trade.PnL
			stats.TotalFees += trade.Commission
		}

		if stats.TotalTrades > 0 {
			stats.WinRate = float64(stats.WinningTrades) / float64(stats.TotalTrades)
			stats.AverageProfitLoss = stats.RealizedPnL / float64(stats.TotalTrades)
		}

		// Calculate final portfolio value
		finalPortfolioValue := e.currentCapital
		marketData := e.marketDataSource.GetDataForTimeRange(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().AddDate(100, 0, 0))
		if len(marketData) > 0 {
			lastPrice := marketData[len(marketData)-1].Close
			for _, position := range e.positions {
				finalPortfolioValue += position.Quantity * lastPrice
			}
		}

		// Calculate statistics for each strategy
		e.strategies[i].stats = stats

		// Write strategy stats to disk
		if e.resultsWriter != nil {
			fmt.Printf("%s", color.HiYellowString("Writing strategy stats for %s\n", e.strategies[i].strategy.Name()))
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
			RealizedPnL:       0,
			UnrealizedPnL:     0,
			AverageProfitLoss: 0,
			TotalFees:         0,
		}

		for _, s := range e.strategies {
			combinedStats.TotalTrades += s.stats.TotalTrades
			combinedStats.WinningTrades += s.stats.WinningTrades
			combinedStats.LosingTrades += s.stats.LosingTrades
			combinedStats.RealizedPnL += s.stats.RealizedPnL
			combinedStats.TotalFees += s.stats.TotalFees
		}

		// Get the final portfolio value including unrealized PnL
		finalPortfolioValue := e.currentCapital
		marketData := e.marketDataSource.GetDataForTimeRange(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().AddDate(100, 0, 0))
		if len(marketData) > 0 {
			lastPrice := marketData[len(marketData)-1].Close
			for _, position := range e.positions {
				finalPortfolioValue += position.Quantity * lastPrice
			}
		}

		// Calculate unrealized PnL from open positions
		combinedStats.UnrealizedPnL = finalPortfolioValue - e.initialCapital - combinedStats.RealizedPnL

		// Calculate total PnL (realized + unrealized)
		combinedStats.TotalPnL = combinedStats.RealizedPnL + combinedStats.UnrealizedPnL

		if combinedStats.TotalTrades > 0 {
			combinedStats.WinRate = float64(combinedStats.WinningTrades) / float64(combinedStats.TotalTrades)
			combinedStats.AverageProfitLoss = combinedStats.RealizedPnL / float64(combinedStats.TotalTrades)
		}

		// Calculate Sharpe ratio and max drawdown from equity curve
		if len(e.equityCurve) > 0 {
			combinedStats.SharpeRatio = e.calculateSharpeRatio()
			combinedStats.MaxDrawdown = e.calculateMaxDrawdown()
		}

		if err := e.resultsWriter.WriteStats(combinedStats); err != nil {
			fmt.Printf("Warning: failed to write combined stats: %v\n", err)
		}

		// Print summary of realized vs unrealized PnL
		fmt.Printf("%s", color.HiYellowString("Realized PnL: $%.2f\n", combinedStats.RealizedPnL))
		fmt.Printf("%s", color.HiYellowString("Unrealized PnL: $%.2f\n", combinedStats.UnrealizedPnL))
		fmt.Printf("%s", color.HiYellowString("Total PnL: $%.2f\n", combinedStats.TotalPnL))
		fmt.Printf("%s", color.HiYellowString("Win Rate: %.2f%%\n", combinedStats.WinRate*100))
		fmt.Printf("%s", color.HiYellowString("Average Profit/Loss: $%.2f\n", combinedStats.AverageProfitLoss))
		fmt.Printf("%s", color.HiYellowString("Sharpe Ratio: %.2f\n", combinedStats.SharpeRatio))
		fmt.Printf("%s", color.HiYellowString("Max Drawdown: %.2f%%\n", combinedStats.MaxDrawdown*100))
		fmt.Printf("%s", color.HiYellowString("Total Fees: $%.2f\n", combinedStats.TotalFees))
		fmt.Printf("%s", color.HiYellowString("Total Trades: %d\n", combinedStats.TotalTrades))
		fmt.Printf("%s", color.HiYellowString("Winning Trades: %d\n", combinedStats.WinningTrades))
		fmt.Printf("%s", color.HiYellowString("Losing Trades: %d\n", combinedStats.LosingTrades))
	}

	// Print comparison with buy and hold
	if e.buyAndHoldValue > 0 {
		finalPortfolioValue := e.currentCapital
		marketData := e.marketDataSource.GetDataForTimeRange(time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC), time.Now().AddDate(100, 0, 0))
		if len(marketData) > 0 {
			lastPrice := marketData[len(marketData)-1].Close
			for _, position := range e.positions {
				finalPortfolioValue += position.Quantity * lastPrice
			}
		}

		fmt.Printf("%s", color.HiYellowString("Final portfolio value: $%.2f\n", finalPortfolioValue))
		fmt.Printf("%s", color.HiYellowString("Buy and hold value: $%.2f\n", e.buyAndHoldValue))
		fmt.Printf("%s", color.HiYellowString("Outperformance: %.2f%%\n", (finalPortfolioValue/e.buyAndHoldValue-1)*100))
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

// calculateCommission calculates the commission for an order using the formula from the config
func (e *BacktestEngineV1) calculateCommission(order types.Order, executionPrice float64) (float64, error) {
	// If no formula is provided, default to zero commission
	if e.fees.CommissionFormula == "" {
		return 0, nil
	}

	formula := e.fees.CommissionFormula

	return calculateCommissionWithExpression(formula, order, executionPrice)
}

// calculateCommissionWithExpression calculates commission using a human-readable expression

// TestCalculateCommission is a helper method for testing commission calculations
func (e *BacktestEngineV1) TestCalculateCommission(order types.Order, executionPrice float64) (float64, error) {
	return e.calculateCommission(order, executionPrice)
}

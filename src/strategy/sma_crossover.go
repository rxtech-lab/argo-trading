package strategy

import (
	"fmt"

	"github.com/sirily11/argo-trading-go/src/types"
)

// SimpleMovingAverageCrossover is a simple strategy that buys when the short MA crosses above the long MA
// and sells when the short MA crosses below the long MA
type SimpleMovingAverageCrossover struct {
	shortPeriod int
	longPeriod  int
	symbol      string
}

// NewSimpleMovingAverageCrossover creates a new SMA crossover strategy with the given parameters
func NewSimpleMovingAverageCrossover(shortPeriod, longPeriod int, symbol string) *SimpleMovingAverageCrossover {
	return &SimpleMovingAverageCrossover{
		shortPeriod: shortPeriod,
		longPeriod:  longPeriod,
		symbol:      symbol,
	}
}

// Name returns the name of the strategy
func (s *SimpleMovingAverageCrossover) Name() string {
	return fmt.Sprintf("SMA_Cross_%d_%d", s.shortPeriod, s.longPeriod)
}

// Initialize sets up the strategy with a configuration string and initial context
func (s *SimpleMovingAverageCrossover) Initialize(config string, initialContext StrategyContext) error {
	// In a real implementation, parse the config string
	// For now, if no values were set in the constructor, use defaults
	if s.shortPeriod == 0 {
		s.shortPeriod = 5 // Default values
	}
	if s.longPeriod == 0 {
		s.longPeriod = 20
	}
	if s.symbol == "" {
		s.symbol = "AAPL"
	}
	return nil
}

// ProcessData processes new market data and generates signals
func (s *SimpleMovingAverageCrossover) ProcessData(ctx StrategyContext, data types.MarketData) ([]types.Order, error) {
	historicalData := ctx.GetHistoricalData()

	// Need enough data for the long moving average
	if len(historicalData) < s.longPeriod {
		return nil, nil
	}

	// Calculate short and long moving averages
	shortMA := calculateSMA(historicalData, s.shortPeriod)
	longMA := calculateSMA(historicalData, s.longPeriod)

	// Get previous values if available
	var prevShortMA, prevLongMA float64
	if len(historicalData) > s.longPeriod {
		prevData := historicalData[:len(historicalData)-1]
		prevShortMA = calculateSMA(prevData, s.shortPeriod)
		prevLongMA = calculateSMA(prevData, s.longPeriod)
	} else {
		// Not enough data for previous values
		return nil, nil
	}

	// Check for crossovers
	var orders []types.Order

	// Get current positions
	positions := ctx.GetCurrentPositions()
	hasPosition := false
	for _, pos := range positions {
		if pos.Symbol == s.symbol && pos.Quantity > 0 {
			hasPosition = true
			break
		}
	}

	// Buy signal: short MA crosses above long MA
	if shortMA > longMA && prevShortMA <= prevLongMA && !hasPosition {
		// Calculate position size (invest 95% of available capital)
		capital := ctx.GetAccountBalance()
		if capital <= 0 {
			return nil, nil
		}

		quantity := (capital * 0.95) / data.Close

		// Create buy order
		order := types.Order{
			Symbol:       s.symbol,
			OrderType:    types.OrderTypeBuy,
			Quantity:     quantity,
			Price:        data.Close,
			StrategyName: s.Name(),
		}

		orders = append(orders, order)
	}

	// Sell signal: short MA crosses below long MA
	if shortMA < longMA && prevShortMA >= prevLongMA && hasPosition {
		// Find position to sell
		var positionToSell types.Position
		for _, pos := range positions {
			if pos.Symbol == s.symbol {
				positionToSell = pos
				break
			}
		}

		// Create sell order
		order := types.Order{
			Symbol:       s.symbol,
			OrderType:    types.OrderTypeSell,
			Quantity:     positionToSell.Quantity,
			Price:        data.Close,
			StrategyName: s.Name(),
		}

		orders = append(orders, order)
	}

	return orders, nil
}

// calculateSMA calculates the simple moving average for the given period
func calculateSMA(data []types.MarketData, period int) float64 {
	if len(data) < period {
		return 0
	}

	sum := 0.0
	for i := len(data) - period; i < len(data); i++ {
		sum += data[i].Close
	}

	return sum / float64(period)
}

package strategy

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirily11/argo-trading-go/src/indicator"
	"github.com/sirily11/argo-trading-go/src/types"
)

// ExampleIndicatorStrategy demonstrates how to use indicators in a strategy
type ExampleIndicatorStrategy struct {
	name           string
	symbol         string
	rsiPeriod      int
	rsiOverbought  float64
	rsiOversold    float64
	initialized    bool
	lastSignalTime time.Time
	cooldownPeriod time.Duration
}

// NewExampleIndicatorStrategy creates a new example strategy
func NewExampleIndicatorStrategy(symbol string) *ExampleIndicatorStrategy {
	return &ExampleIndicatorStrategy{
		name:           "ExampleIndicatorStrategy",
		symbol:         symbol,
		rsiPeriod:      14,
		rsiOverbought:  70.0,
		rsiOversold:    30.0,
		initialized:    false,
		cooldownPeriod: 24 * time.Hour, // Avoid frequent trading
	}
}

// Name returns the name of the strategy
func (s *ExampleIndicatorStrategy) Name() string {
	return s.name
}

// Initialize sets up the strategy with configuration
func (s *ExampleIndicatorStrategy) Initialize(config string, initialContext StrategyContext) error {
	s.initialized = true
	return nil
}

// ProcessData processes new market data and generates signals
func (s *ExampleIndicatorStrategy) ProcessData(ctx StrategyContext, data types.MarketData) ([]types.Order, error) {
	if !s.initialized {
		return nil, fmt.Errorf("strategy not initialized")
	}

	// Check if we're in cooldown period
	if !s.lastSignalTime.IsZero() && time.Since(s.lastSignalTime) < s.cooldownPeriod {
		return nil, nil
	}

	// Get current positions
	positions := ctx.GetCurrentPositions()
	hasPosition := false
	var currentPosition types.Position

	for _, pos := range positions {
		if pos.Symbol == s.symbol {
			hasPosition = true
			currentPosition = pos
			break
		}
	}

	// Get RSI indicator values
	startTime := data.Time.Add(-1 * time.Hour * 24 * 30)
	endTime := data.Time
	rsiResult, err := ctx.GetIndicator(types.IndicatorRSI, startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get RSI indicator: %w", err)
	}

	rsiValues, ok := rsiResult.([]float64)
	if !ok || len(rsiValues) == 0 {
		return nil, fmt.Errorf("invalid RSI result format")
	}

	// Get latest RSI value
	currentRSI := rsiValues[len(rsiValues)-1]

	// Get MACD indicator values
	macdResult, err := ctx.GetIndicator("MACD", startTime, endTime)
	if err != nil {
		return nil, fmt.Errorf("failed to get MACD indicator: %w", err)
	}

	macdData, ok := macdResult.(indicator.MACDResult)
	if !ok || len(macdData.MACD) == 0 || len(macdData.Signal) == 0 || len(macdData.Histogram) == 0 {
		return nil, fmt.Errorf("invalid MACD result format")
	}

	// Get latest MACD values
	macdValue := macdData.MACD[len(macdData.MACD)-1]
	signalValue := macdData.Signal[len(macdData.Signal)-1]

	// Generate orders based on indicator signals
	var orders []types.Order

	// Buy signal: RSI < oversold threshold and MACD crosses above signal line
	if currentRSI < s.rsiOversold && macdValue > signalValue && !hasPosition {
		// Calculate position size (10% of available capital)
		capital := ctx.GetAccountBalance()
		positionSize := capital * 0.1
		quantity := positionSize / data.Close

		// Create buy order
		order := types.Order{
			Symbol:       s.symbol,
			OrderType:    types.OrderTypeBuy,
			Quantity:     quantity,
			Price:        data.Close,
			Timestamp:    data.Time,
			OrderID:      uuid.New().String(),
			IsCompleted:  false,
			StrategyName: s.name,
		}

		orders = append(orders, order)
		s.lastSignalTime = data.Time
	}

	// Sell signal: RSI > overbought threshold or MACD crosses below signal line
	if (currentRSI > s.rsiOverbought || macdValue < signalValue) && hasPosition {
		// Create sell order for entire position
		order := types.Order{
			Symbol:       s.symbol,
			OrderType:    types.OrderTypeSell,
			Quantity:     currentPosition.Quantity,
			Price:        data.Close,
			Timestamp:    data.Time,
			OrderID:      uuid.New().String(),
			IsCompleted:  false,
			StrategyName: s.name,
		}

		orders = append(orders, order)
		s.lastSignalTime = data.Time
	}

	return orders, nil
}

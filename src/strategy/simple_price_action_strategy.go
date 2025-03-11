package strategy

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirily11/argo-trading-go/src/types"
)

// SimplePriceActionStrategy uses basic price action for trading decisions
type SimplePriceActionStrategy struct {
	name                 string
	shortLookbackPeriod  int
	longLookbackPeriod   int
	stopLossPercentage   float64
	takeProfitPercentage float64
	initialized          bool
	lastSignalTime       time.Time
	cooldownPeriod       time.Duration
	positionSizePercent  float64
	volumeThreshold      float64
	trendLookbackPeriod  int
}

// NewSimplePriceActionStrategy creates a new simple price action strategy
func NewSimplePriceActionStrategy() TradingStrategy {
	return &SimplePriceActionStrategy{
		name:                 "SimplePriceActionStrategy",
		shortLookbackPeriod:  8,
		longLookbackPeriod:   21,
		stopLossPercentage:   0.8, // Tighter stop loss
		takeProfitPercentage: 1.6, // Lower take profit for more frequent wins (2:1 reward-to-risk)
		initialized:          false,
		cooldownPeriod:       45 * time.Minute, // Balanced cooldown period
		positionSizePercent:  0.15,             // Reduced position size for risk management
		volumeThreshold:      1.2,              // Volume must be 1.2x average to confirm signals
		trendLookbackPeriod:  50,               // Longer period for trend determination
	}
}

// Name returns the name of the strategy
func (s *SimplePriceActionStrategy) Name() string {
	return s.name
}

// Initialize sets up the strategy with configuration
func (s *SimplePriceActionStrategy) Initialize(config string) error {
	s.initialized = true
	return nil
}

// ProcessData processes new market data and generates signals
func (s *SimplePriceActionStrategy) ProcessData(ctx StrategyContext, data types.MarketData, targetSymbol string) ([]types.Order, error) {

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
		if pos.Symbol == targetSymbol {
			hasPosition = true
			currentPosition = pos
			break
		}
	}

	// Get historical data for price action analysis
	historicalData := ctx.GetHistoricalData()
	if len(historicalData) < s.trendLookbackPeriod {
		return nil, fmt.Errorf("not enough historical data for analysis")
	}

	// Calculate short-term simple moving average (8-day)
	var shortSumPrice float64
	for i := len(historicalData) - s.shortLookbackPeriod; i < len(historicalData); i++ {
		shortSumPrice += historicalData[i].Close
	}
	shortSMA := shortSumPrice / float64(s.shortLookbackPeriod)

	// Calculate medium-term simple moving average (21-day)
	var longSumPrice float64
	for i := len(historicalData) - s.longLookbackPeriod; i < len(historicalData); i++ {
		longSumPrice += historicalData[i].Close
	}
	longSMA := longSumPrice / float64(s.longLookbackPeriod)

	// Calculate long-term trend SMA (50-day)
	var trendSumPrice float64
	for i := len(historicalData) - s.trendLookbackPeriod; i < len(historicalData); i++ {
		trendSumPrice += historicalData[i].Close
	}
	trendSMA := trendSumPrice / float64(s.trendLookbackPeriod)

	// Determine overall trend
	uptrend := data.Close > trendSMA
	downtrend := data.Close < trendSMA

	// Calculate average volume
	var sumVolume float64
	for i := len(historicalData) - s.longLookbackPeriod; i < len(historicalData); i++ {
		sumVolume += historicalData[i].Volume
	}
	avgVolume := sumVolume / float64(s.longLookbackPeriod)

	// Calculate price momentum (rate of change over short period)
	priceChange := (data.Close - historicalData[len(historicalData)-s.shortLookbackPeriod].Close) /
		historicalData[len(historicalData)-s.shortLookbackPeriod].Close * 100

	// Check if volume is above threshold
	volumeConfirmation := data.Volume > (avgVolume * s.volumeThreshold)

	// Calculate price volatility (Average True Range simplified)
	var sumATR float64
	for i := len(historicalData) - s.longLookbackPeriod + 1; i < len(historicalData); i++ {
		trueRange := max3(
			historicalData[i].High-historicalData[i].Low,
			absValue(historicalData[i].High-historicalData[i-1].Close),
			absValue(historicalData[i].Low-historicalData[i-1].Close),
		)
		sumATR += trueRange
	}
	atr := sumATR / float64(s.longLookbackPeriod-1)

	// Adjust stop loss and take profit based on volatility
	volatilityFactor := atr / data.Close * 100
	dynamicStopLoss := s.stopLossPercentage * (1 + volatilityFactor/2)
	dynamicTakeProfit := s.takeProfitPercentage * (1 + volatilityFactor/2)

	// Check if we have pending orders
	pendingOrders := ctx.GetPendingOrders()
	hasPendingOrders := false
	for _, order := range pendingOrders {
		if order.Symbol == targetSymbol {
			hasPendingOrders = true
			break
		}
	}

	// Generate orders based on price action signals
	var orders []types.Order

	// Check for stop loss or take profit if we have a position
	if hasPosition {
		unrealizedPnL := (data.Close - currentPosition.AveragePrice) / currentPosition.AveragePrice * 100

		// Stop loss triggered
		if unrealizedPnL <= -dynamicStopLoss {
			order := types.Order{
				Symbol:       targetSymbol,
				OrderType:    types.OrderTypeSell,
				Quantity:     currentPosition.Quantity,
				Price:        data.Close,
				Timestamp:    data.Time,
				OrderID:      uuid.New().String(),
				IsCompleted:  false,
				StrategyName: s.name,
				Reason: types.Reason{
					Reason: types.OrderReasonStopLoss,
					Message: fmt.Sprintf("stop loss triggered at %.2f%% loss (dynamic: %.2f%%)",
						unrealizedPnL, dynamicStopLoss),
				},
			}
			orders = append(orders, order)
			s.lastSignalTime = data.Time
			return orders, nil
		}

		// Take profit triggered
		if unrealizedPnL >= dynamicTakeProfit {
			order := types.Order{
				Symbol:       targetSymbol,
				OrderType:    types.OrderTypeSell,
				Quantity:     currentPosition.Quantity,
				Price:        data.Close,
				Timestamp:    data.Time,
				OrderID:      uuid.New().String(),
				IsCompleted:  false,
				StrategyName: s.name,
				Reason: types.Reason{
					Reason: types.OrderReasonTakeProfit,
					Message: fmt.Sprintf("take profit triggered at %.2f%% gain (dynamic: %.2f%%)",
						unrealizedPnL, dynamicTakeProfit),
				},
			}
			orders = append(orders, order)
			s.lastSignalTime = data.Time
			return orders, nil
		}
	}

	// Buy signal: Short SMA crosses above Long SMA (Golden Cross) with positive momentum
	// and price is near the short SMA (within 0.3%) in an uptrend
	isGoldenCross := shortSMA > longSMA &&
		absValue((data.Close-shortSMA)/shortSMA*100) < 0.3 &&
		priceChange > 0.3

	if isGoldenCross && volumeConfirmation && uptrend && !hasPosition && !hasPendingOrders {
		// Calculate position size based on account balance
		capital := ctx.GetAccountBalance()
		positionSize := capital * s.positionSizePercent
		quantity := positionSize / data.Close

		// Ensure minimum buy quantity is at least 0.01
		if quantity < 0.01 {
			quantity = 0.01
		}

		// Create buy order
		order := types.Order{
			Symbol:       targetSymbol,
			OrderType:    types.OrderTypeBuy,
			Quantity:     quantity,
			Price:        data.Close,
			Timestamp:    data.Time,
			OrderID:      uuid.New().String(),
			IsCompleted:  false,
			StrategyName: s.name,
			Reason: types.Reason{
				Reason: types.OrderReasonBuySignal,
				Message: fmt.Sprintf("Golden Cross: Short SMA (%.2f) > Long SMA (%.2f), Momentum: %.2f%%, Uptrend",
					shortSMA, longSMA, priceChange),
			},
		}

		orders = append(orders, order)
		s.lastSignalTime = data.Time
	}

	// Sell signal: Short SMA crosses below Long SMA (Death Cross) with negative momentum
	// and price is near the short SMA (within 0.3%) in a downtrend
	isDeathCross := shortSMA < longSMA &&
		absValue((data.Close-shortSMA)/shortSMA*100) < 0.3 &&
		priceChange < -0.3

	if isDeathCross && volumeConfirmation && downtrend && hasPosition {
		// Create sell order for entire position
		order := types.Order{
			Symbol:       targetSymbol,
			OrderType:    types.OrderTypeSell,
			Quantity:     currentPosition.Quantity,
			Price:        data.Close,
			Timestamp:    data.Time,
			OrderID:      uuid.New().String(),
			IsCompleted:  false,
			StrategyName: s.name,
			Reason: types.Reason{
				Reason: types.OrderReasonSellSignal,
				Message: fmt.Sprintf("Death Cross: Short SMA (%.2f) < Long SMA (%.2f), Momentum: %.2f%%, Downtrend",
					shortSMA, longSMA, priceChange),
			},
		}

		orders = append(orders, order)
		s.lastSignalTime = data.Time
	}

	return orders, nil
}

// Helper function for absolute value
func absValue(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

// Helper function to get maximum of three values
func max3(a, b, c float64) float64 {
	max := a
	if b > max {
		max = b
	}
	if c > max {
		max = c
	}
	return max
}

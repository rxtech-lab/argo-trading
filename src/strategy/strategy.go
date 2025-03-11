package strategy

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

type StrategyContext interface {
	// Data access methods
	GetHistoricalData() []types.MarketData
	GetCurrentPositions() []types.Position
	GetPendingOrders() []types.Order
	GetExecutedTrades() []types.Trade
	GetAccountBalance() float64
	GetIndicator(name types.Indicator, startTime, endTime time.Time) (any, error)
}

// TradingStrategy interface defines methods that any trading strategy must implement
// Strategies should be stateless - position and order management is handled by the trading system
type TradingStrategy interface {
	// Initialize sets up the strategy with a configuration string and initial context
	// The trading system is responsible for decoding the config string
	Initialize(config string) error

	// ProcessData processes new market data and generates signals
	// It receives a context object with all necessary information to make decisions
	ProcessData(ctx StrategyContext, data types.MarketData) ([]types.Order, error)

	// Name returns the name of the strategy
	Name() string
}

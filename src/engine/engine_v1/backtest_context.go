package engine

import (
	"fmt"
	"time"

	"github.com/sirily11/argo-trading-go/src/indicator"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"github.com/sirily11/argo-trading-go/src/types"
)

// strategyContext implements the strategy.StrategyContext interface
type strategyContext struct {
	engine    *BacktestEngineV1
	startTime time.Time
	endTime   time.Time
}

func NewStrategyContext(engine *BacktestEngineV1, startTime, endTime time.Time) strategy.StrategyContext {
	return &strategyContext{
		engine:    engine,
		startTime: startTime,
		endTime:   endTime,
	}
}

func (c strategyContext) GetHistoricalData() []types.MarketData {
	return c.engine.marketDataSource.GetDataForTimeRange(c.startTime, c.endTime)
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

// GetIndicator retrieves an indicator by name and calculates its value using historical data
func (c strategyContext) GetIndicator(name types.Indicator, startTime, endTime time.Time) (any, error) {
	var ind indicator.Indicator

	// if endTime is before the end of the backtest, use the end of the backtest
	targetEndtime := endTime
	if c.endTime.Before(endTime) {
		targetEndtime = c.endTime
	}

	switch name {
	case types.IndicatorRSI:
		ind = indicator.NewRSI(startTime, targetEndtime, 14) // Default period of 14
	case types.IndicatorMACD:
		ind = indicator.NewMACD(startTime, targetEndtime, 12, 26, 9) // Default periods of 12, 26, 9
	default:
		return nil, fmt.Errorf("indicator %s not found", name)
	}

	// Create indicator context
	ctx := indicator.CreateIndicatorContext(c.engine.marketDataSource)

	// Calculate indicator value
	result, err := ind.Calculate(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate indicator %s: %w", name, err)
	}
	return result, nil
}

// Close finalizes the backtest and cleans up resources
func (e *BacktestEngineV1) Close() error {
	if e.resultsWriter != nil {
		return e.resultsWriter.Close()
	}
	return nil
}

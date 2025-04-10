package strategy

import (
	"github.com/sirily11/argo-trading-go/src/types"
)

// ExampleIndicatorStrategy demonstrates how to use indicators in a strategy
type ExampleIndicatorStrategy struct {
}

// NewExampleIndicatorStrategy creates a new example strategy
func NewExampleIndicatorStrategy() TradingStrategy {
	return &ExampleIndicatorStrategy{}
}

// Name returns the name of the strategy
func (s *ExampleIndicatorStrategy) Name() string {
	return "ExampleIndicatorStrategy"
}

// Initialize sets up the strategy with configuration
func (s *ExampleIndicatorStrategy) Initialize(config string) error {
	return nil
}

// ProcessData processes new market data and generates signals
func (s *ExampleIndicatorStrategy) ProcessData(ctx StrategyContext, data types.MarketData, param string) ([]types.ExecuteOrder, error) {

	return []types.ExecuteOrder{}, nil
}

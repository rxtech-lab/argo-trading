package strategy

import (
	"github.com/sirily11/argo-trading-go/src/indicator"
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
	currentPosition, err := ctx.GetPosition(data.Symbol)
	if err != nil {
		return nil, err
	}
	hasPosition := currentPosition.Quantity > 0

	bollingerBands, err := ctx.IndicatorRegistry.GetIndicator(types.IndicatorBollingerBands)
	if err != nil {
		return nil, err
	}

	signal, err := bollingerBands.GetSignal(data, indicator.IndicatorContext{
		DataSource: ctx.DataSource,
	})

	if err != nil {
		return nil, err
	}

	if signal.Type == types.SignalTypeBuy {
		if hasPosition {
			return []types.ExecuteOrder{}, nil
		}

		return []types.ExecuteOrder{
			{
				Symbol:    data.Symbol,
				OrderType: types.OrderTypeBuy,
				Reason: types.Reason{
					Reason:  signal.Name,
					Message: "Buy signal",
				},
			},
		}, nil
	}

	if signal.Type == types.SignalTypeSell {
		if !hasPosition {
			return []types.ExecuteOrder{}, nil
		}

		return []types.ExecuteOrder{
			{
				Symbol:    data.Symbol,
				OrderType: types.OrderTypeSell,
				Reason: types.Reason{
					Reason:  signal.Name,
					Message: signal.Reason,
				},
			},
		}, nil
	}

	return []types.ExecuteOrder{}, nil
}

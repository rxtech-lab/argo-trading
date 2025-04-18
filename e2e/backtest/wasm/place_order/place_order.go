//go:build wasip1

package main

import (
	"context"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// PlaceOrderStrategy implements a strategy based on two moving averages
// Buy when fast MA crosses above slow MA, sell when fast MA crosses below slow MA
type PlaceOrderStrategy struct {
	// Strategy is stateless, state is stored in cache
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewPlaceOrderStrategy())
}

func NewPlaceOrderStrategy() strategy.TradingStrategy {
	return &PlaceOrderStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *PlaceOrderStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *PlaceOrderStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PlaceOrderStrategy"}, nil
}

// ProcessData implements strategy.TradingStrategy.
// Place only one order at a time
func (s *PlaceOrderStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	key := "place_order_key"
	// Get API for interacting with the host system
	api := strategy.NewStrategyApi()
	// get cache
	cache, err := api.GetCache(ctx, &strategy.GetRequest{Key: key})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	if cache.Value == "1" {
		return &emptypb.Empty{}, nil
	}

	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "PlaceOrderStrategy",
			Message: "PlaceOrderStrategy",
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	_, err = api.Mark(ctx, &strategy.MarkRequest{
		Signal:     strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
		MarketData: data,
		Reason:     "PlaceOrderStrategy",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to mark: %w", err)
	}

	// set cache
	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   key,
		Value: "1",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set cache: %w", err)
	}

	return &emptypb.Empty{}, nil

}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *PlaceOrderStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: ""}, nil
}

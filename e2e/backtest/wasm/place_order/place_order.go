//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// PlaceOrderStrategy implements a strategy based on two moving averages
// Buy when fast MA crosses above slow MA, sell when fast MA crosses below slow MA
type PlaceOrderStrategy struct {
	// Strategy is stateless, state is stored in cache
	config Config
}

// Config represents the configuration for the PlaceOrderStrategy
type Config struct {
	Symbol string `yaml:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
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
	var config Config
	// unmarshal json to config
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	s.config = config
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *PlaceOrderStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PlaceOrderStrategy"}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *PlaceOrderStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "A strategy that places an order when the fast MA crosses above the slow MA and sells when the fast MA crosses below the slow MA"}, nil
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
		Mark: &strategy.Mark{
			SignalType: strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
			Color:      "red",
			Shape:      strategy.MarkShape_MARK_SHAPE_CIRCLE,
			Title:      "PlaceOrderStrategy",
			Message:    "PlaceOrderStrategy",
			Category:   "PlaceOrderStrategy",
		},
		MarketData: data,
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
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

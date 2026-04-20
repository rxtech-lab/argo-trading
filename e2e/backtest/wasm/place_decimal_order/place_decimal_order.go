//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// PlaceDecimalOrderStrategy places repeated market buy orders with a fractional
// quantity (0.01) so we can exercise the engine's decimal-precision handling end-to-end.
type PlaceDecimalOrderStrategy struct {
	config Config
}

// Config represents the configuration for the PlaceDecimalOrderStrategy.
type Config struct {
	Symbol string `yaml:"symbol" json:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewPlaceDecimalOrderStrategy())
}

func NewPlaceDecimalOrderStrategy() strategy.TradingStrategy {
	return &PlaceDecimalOrderStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	var config Config
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	s.config = config
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PlaceDecimalOrderStrategy"}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "Places fractional-quantity (0.01) market buy orders to exercise decimal precision handling"}, nil
}

const maxOrders = 3

// ProcessData implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()
	key := "execution_count"

	cache, err := api.GetCache(ctx, &strategy.GetRequest{Key: key})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	count := 0
	if cache.Value != "" {
		count, _ = strconv.Atoi(cache.Value)
	}

	if count >= maxOrders {
		return &emptypb.Empty{}, nil
	}

	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     0.01,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceDecimalOrderStrategy",
		PositionType: strategy.PositionType_POSITION_TYPE_LONG,
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "fractional quantity test",
		},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to place order: %w", err)
	}

	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   key,
		Value: strconv.Itoa(count + 1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set cache: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

// GetIdentifier implements strategy.TradingStrategy.
func (s *PlaceDecimalOrderStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.e2e.place-decimal-order",
	}, nil
}

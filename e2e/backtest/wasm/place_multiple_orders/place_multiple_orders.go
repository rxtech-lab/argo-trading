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

// PlaceMultipleOrdersStrategy places one BUY order on every Nth processed bar
// (controlled by the orderEveryN config), giving the engine a non-trivial,
// deterministic amount of work per data file. The strategy is designed to
// exercise the parallel run path: every (data file, worker) combination
// produces a known number of orders.
type PlaceMultipleOrdersStrategy struct {
	config Config
}

// Config controls how often the strategy places orders.
type Config struct {
	// OrderEveryN places one BUY order every N processed bars (>=1).
	OrderEveryN int `yaml:"orderEveryN" jsonschema:"title=Order Every N,description=Place an order every N bars,default=10"`
	// Symbol restricts placement to bars matching this symbol when set.
	Symbol string `yaml:"symbol" jsonschema:"title=Symbol,description=Symbol to filter on (optional)"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewPlaceMultipleOrdersStrategy())
}

func NewPlaceMultipleOrdersStrategy() strategy.TradingStrategy {
	return &PlaceMultipleOrdersStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	cfg := Config{OrderEveryN: 10}
	if req.Config != "" {
		if err := json.Unmarshal([]byte(req.Config), &cfg); err != nil {
			return nil, fmt.Errorf("failed to parse configuration: %w", err)
		}
	}
	if cfg.OrderEveryN < 1 {
		cfg.OrderEveryN = 1
	}
	s.config = cfg
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PlaceMultipleOrdersStrategy"}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{
		Description: "Places one BUY order every N processed bars to exercise parallel runs",
	}, nil
}

// ProcessData implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	if s.config.Symbol != "" && data.Symbol != s.config.Symbol {
		return &emptypb.Empty{}, nil
	}

	api := strategy.NewStrategyApi()

	// Use the cache as a counter so the strategy stays stateless between
	// ProcessData invocations (per the framework's contract).
	const counterKey = "place_multiple_orders_counter"
	cur, err := api.GetCache(ctx, &strategy.GetRequest{Key: counterKey})
	if err != nil {
		return nil, fmt.Errorf("failed to read counter: %w", err)
	}

	processed := 0
	if cur.Value != "" {
		if v, err := strconv.Atoi(cur.Value); err == nil {
			processed = v
		}
	}
	processed++

	if processed%s.config.OrderEveryN == 0 {
		_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
			Symbol:       data.Symbol,
			Quantity:     1,
			Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
			Price:        data.Close,
			StrategyName: "PlaceMultipleOrdersStrategy",
			Reason: &strategy.Reason{
				Reason:  "PlaceMultipleOrdersStrategy",
				Message: fmt.Sprintf("bar #%d", processed),
			},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to place order: %w", err)
		}
	}

	if _, err := api.SetCache(ctx, &strategy.SetRequest{
		Key:   counterKey,
		Value: strconv.Itoa(processed),
	}); err != nil {
		return nil, fmt.Errorf("failed to update counter: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

// GetIdentifier implements strategy.TradingStrategy.
func (s *PlaceMultipleOrdersStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.e2e.place-multiple-orders",
	}, nil
}

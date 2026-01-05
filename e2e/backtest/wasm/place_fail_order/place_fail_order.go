//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strconv"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// PlaceFailOrderStrategy implements a strategy that tests various order failure scenarios
type PlaceFailOrderStrategy struct {
	config Config
}

// Config represents the configuration for the PlaceFailOrderStrategy
type Config struct {
	Symbol   string `yaml:"symbol" json:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
	TestCase string `yaml:"testCase" json:"testCase" jsonschema:"title=TestCase,description=Which test case to run"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewPlaceFailOrderStrategy())
}

func NewPlaceFailOrderStrategy() strategy.TradingStrategy {
	return &PlaceFailOrderStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	var config Config
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	s.config = config
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PlaceFailOrderStrategy"}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "A strategy that tests various order failure scenarios"}, nil
}

// ProcessData implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()
	key := "execution_count"

	// Get execution count from cache
	cache, err := api.GetCache(ctx, &strategy.GetRequest{Key: key})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	count := 0
	if cache.Value != "" {
		count, _ = strconv.Atoi(cache.Value)
	}

	// Execute test case based on configuration
	switch s.config.TestCase {
	case "exceed_buying_power":
		err = s.testExceedBuyingPower(ctx, api, data, count)
	case "exceed_selling_power":
		err = s.testExceedSellingPower(ctx, api, data, count)
	case "invalid_then_success":
		err = s.testInvalidThenSuccess(ctx, api, data, count)
	case "success_order":
		err = s.testSuccessOrder(ctx, api, data, count)
	case "mixed_orders":
		err = s.testMixedOrders(ctx, api, data, count)
	case "multiple_failed":
		err = s.testMultipleFailed(ctx, api, data, count)
	case "max_buy_order":
		err = s.testMaxBuyOrder(ctx, api, data, count)
	default:
		// Default: just place one order (for backward compatibility)
		err = s.testSuccessOrder(ctx, api, data, count)
	}

	if err != nil {
		return nil, err
	}

	// Increment execution count
	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   key,
		Value: strconv.Itoa(count + 1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set cache: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// testExceedBuyingPower places a buy order that exceeds available balance
func (s *PlaceFailOrderStrategy) testExceedBuyingPower(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	// Place order for 1000 shares at current price (will exceed 10000 balance)
	_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1000, // 1000 * ~150 = ~150000, exceeds 10000 balance
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Testing exceed buying power",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	return nil
}

// testExceedSellingPower places a sell order when we have no holdings
func (s *PlaceFailOrderStrategy) testExceedSellingPower(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	// Place sell order when we have no position (should fail with insufficient_selling_power)
	_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     10,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_SELL,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Testing exceed selling power",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	return nil
}

// testInvalidThenSuccess places an invalid order (qty=0) then a valid order
func (s *PlaceFailOrderStrategy) testInvalidThenSuccess(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 1 {
		return nil // Only execute twice
	}

	if count == 0 {
		// First: place invalid order with quantity 0
		_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
			Symbol:       data.Symbol,
			Quantity:     0, // Invalid quantity
			Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
			Price:        data.High,
			StrategyName: "PlaceFailOrderStrategy",
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Testing invalid quantity",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to place invalid order: %w", err)
		}
	} else if count == 1 {
		// Second: place valid order
		_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
			Symbol:       data.Symbol,
			Quantity:     1,
			Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
			Price:        data.High,
			StrategyName: "PlaceFailOrderStrategy",
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Testing valid order after invalid",
			},
		})
		if err != nil {
			return fmt.Errorf("failed to place valid order: %w", err)
		}
	}

	return nil
}

// testSuccessOrder places a single successful order
func (s *PlaceFailOrderStrategy) testSuccessOrder(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Testing successful order",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order: %w", err)
	}

	return nil
}

// testMixedOrders places multiple orders - some succeed, some fail
func (s *PlaceFailOrderStrategy) testMixedOrders(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	// Order 1: Buy 1 share (should succeed)
	_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Mixed test - buy success",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 1: %w", err)
	}

	// Order 2: Buy 1000 shares (should fail - exceed buying power)
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1000,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Mixed test - buy fail",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 2: %w", err)
	}

	// Order 3: Sell 1 share (should succeed - we bought 1 share)
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_SELL,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Mixed test - sell success",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 3: %w", err)
	}

	// Order 4: Sell 100 shares (should fail - we have 0 shares now)
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     100,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_SELL,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Mixed test - sell fail",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 4: %w", err)
	}

	return nil
}

// testMultipleFailed places multiple orders that all fail
func (s *PlaceFailOrderStrategy) testMultipleFailed(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	// Order 1: Buy too many shares (should fail - exceed buying power)
	_, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     1000,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Multiple failed - exceed buying power",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 1: %w", err)
	}

	// Order 2: Invalid quantity (should fail - invalid_quantity)
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     0,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Multiple failed - invalid quantity",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 2: %w", err)
	}

	// Order 3: Sell without holdings (should fail - exceed selling power)
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     10,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_SELL,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Multiple failed - exceed selling power",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place order 3: %w", err)
	}

	return nil
}

// testMaxBuyOrder uses GetAccountInfo to place the maximum affordable order
func (s *PlaceFailOrderStrategy) testMaxBuyOrder(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, count int) error {
	if count > 0 {
		return nil // Only execute once
	}

	// Get account info to determine buying power
	accountInfo, err := api.GetAccountInfo(ctx, &emptypb.Empty{})
	if err != nil {
		return fmt.Errorf("failed to get account info: %w", err)
	}

	// In backtest, BuyingPower is already the max quantity we can afford (not dollar amount)
	maxQty := math.Floor(accountInfo.BuyingPower)

	if maxQty < 1 {
		// Not enough buying power - skip placing order (this is acceptable for this test)
		return nil
	}

	// Place order for max quantity
	_, err = api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Quantity:     maxQty,
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
		Price:        data.High,
		StrategyName: "PlaceFailOrderStrategy",
		Reason: &strategy.Reason{
			Reason:  "strategy",
			Message: "Testing max buy order",
		},
	})
	if err != nil {
		return fmt.Errorf("failed to place max buy order: %w", err)
	}

	return nil
}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

// GetIdentifier implements strategy.TradingStrategy.
func (s *PlaceFailOrderStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.e2e.place-fail-order",
	}, nil
}

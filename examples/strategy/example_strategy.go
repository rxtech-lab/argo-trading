//go:build wasip1

package main

import (
	"context"
	"encoding/json"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// ConsecutiveCandlesStrategy implements a strategy that buys on 2 consecutive up candles
// and sells on 2 consecutive down candles
type ConsecutiveCandlesStrategy struct {
	// Strategy is stateless, state is stored in cache
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewConsecutiveCandlesStrategy())
}

func NewConsecutiveCandlesStrategy() strategy.TradingStrategy {
	return &ConsecutiveCandlesStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	// Nothing to initialize
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "ConsecutiveCandlesStrategy"}, nil
}

// ProcessData implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	// Get API for interacting with the host system
	api := strategy.NewStrategyApi()
	// Get previous data from cache
	cacheKey := "prev_data_" + data.Symbol
	prevDataResp, err := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})

	// If there's no previous data or an error, just store current data and return
	if err != nil || prevDataResp.Value == "" {
		// Store current data in cache
		dataJson, err := json.Marshal(data)
		if err != nil {
			return nil, err
		}

		_, err = api.SetCache(ctx, &strategy.SetRequest{
			Key:   cacheKey,
			Value: string(dataJson),
		})
		if err != nil {
			return nil, err
		}

		return &emptypb.Empty{}, nil
	}

	// Parse previous data from cache
	var prevData strategy.MarketData
	if err := json.Unmarshal([]byte(prevDataResp.Value), &prevData); err != nil {
		return nil, err
	}

	// Check for consecutive up candles
	if data.Close > data.Open && prevData.Close > prevData.Open {
		// Buy signal
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0, // Fixed quantity
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Two consecutive up candles",
			},
			StrategyName: "ConsecutiveCandlesStrategy",
		}

		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, err
		}
	}

	// Check for consecutive down candles
	if data.Close < data.Open && prevData.Close < prevData.Open {
		// Sell signal
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0, // Fixed quantity
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Two consecutive down candles",
			},
			StrategyName: "ConsecutiveCandlesStrategy",
		}

		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, err
		}
	}

	// Update cache with current data
	dataJson, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   cacheKey,
		Value: string(dataJson),
	})
	if err != nil {
		return nil, err
	}

	return &emptypb.Empty{}, nil
}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: ""}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{
		Description: "A strategy that buys on 2 consecutive up candles and sells on 2 consecutive down candles",
	}, nil
}

// GetIdentifier implements strategy.TradingStrategy.
func (s *ConsecutiveCandlesStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.examples.consecutive-candles",
	}, nil
}
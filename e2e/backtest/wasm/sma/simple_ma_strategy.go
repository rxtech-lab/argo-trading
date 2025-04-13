//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"gopkg.in/yaml.v3"
)

// SimpleMAStrategy implements a strategy based on two moving averages
// Buy when fast MA crosses above slow MA, sell when fast MA crosses below slow MA
type SimpleMAStrategy struct {
	// Strategy is stateless, state is stored in cache
}

// Config represents the configuration for the SimpleMAStrategy
type Config struct {
	FastPeriod int    `yaml:"fastPeriod"`
	SlowPeriod int    `yaml:"slowPeriod"`
	Symbol     string `yaml:"symbol"`
}

type maRawValue struct {
	MA float64 `json:"ma"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewSimpleMAStrategy())
}

func NewSimpleMAStrategy() strategy.TradingStrategy {
	return &SimpleMAStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *SimpleMAStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	// Parse configuration
	var config Config
	if err := yaml.Unmarshal([]byte(req.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	// Validate configuration
	if config.FastPeriod < 0 || config.SlowPeriod < 0 {
		return nil, fmt.Errorf("invalid periods: fast=%d, slow=%d", config.FastPeriod, config.SlowPeriod)
	} else {
		// set default values
		if config.FastPeriod == 0 {
			config.FastPeriod = 5
		}
		if config.SlowPeriod == 0 {
			config.SlowPeriod = 20
		}
	}

	if config.FastPeriod >= config.SlowPeriod {
		return nil, fmt.Errorf("fast period must be less than slow period")
	}

	// Configure MA indicators
	api := strategy.NewStrategyApi()

	// Configure Fast MA
	fastMAConfig := fmt.Sprintf(`[%d]`, config.FastPeriod)
	_, err := api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_MA,
		Config:        fastMAConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure fast MA: %w", err)
	}

	// Configure Slow MA
	slowMAConfig := fmt.Sprintf(`[%d]`, config.SlowPeriod)
	_, err = api.ConfigureIndicator(context.Background(), &strategy.ConfigureRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_MA,
		Config:        slowMAConfig,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to configure slow MA: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *SimpleMAStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "SimpleMAStrategy"}, nil
}

// ProcessData implements strategy.TradingStrategy.
func (s *SimpleMAStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data

	// Get API for interacting with the host system
	api := strategy.NewStrategyApi()

	// Get the cache key for previous signal state
	cacheKey := "ma_signal_state_" + data.Symbol
	prevStateResp, err := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	// Get MA signals - first the fast MA, then the slow MA
	fastMASignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_MA,
		MarketData:    data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get fast MA signal: %w", err)
	}

	slowMASignal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_MA,
		MarketData:    data,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get slow MA signal: %w", err)
	}

	// Parse MA values
	var fastMA maRawValue
	err = json.Unmarshal([]byte(fastMASignal.RawValue), &fastMA)
	if err != nil {
		return nil, fmt.Errorf("failed to parse fast MA value: %w", err)
	}

	var slowMA maRawValue
	err = json.Unmarshal([]byte(slowMASignal.RawValue), &slowMA)
	if err != nil {
		return nil, fmt.Errorf("failed to parse slow MA value: %w", err)
	}

	// Current state - is fast MA above slow MA?
	currentIsFastAboveSlow := fastMA.MA > slowMA.MA

	// If we have no previous state, just store the current state and return
	if prevStateResp.Value == "" {
		_, err = api.SetCache(ctx, &strategy.SetRequest{
			Key:   cacheKey,
			Value: fmt.Sprintf("%v", currentIsFastAboveSlow),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set cache: %w", err)
		}
		return &emptypb.Empty{}, nil
	}

	// Parse previous state
	prevIsFastAboveSlow := prevStateResp.Value == "true"

	// Check for crossovers
	if !prevIsFastAboveSlow && currentIsFastAboveSlow {
		// BUY SIGNAL: Fast MA crossed above Slow MA
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0,
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Fast MA crossed above Slow MA",
			},
			StrategyName: "SimpleMAStrategy",
		}

		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("failed to place buy order: %w", err)
		}

		// Mark this signal point
		_, err = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Signal:     strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
			Reason:     "Fast MA crossed above Slow MA",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to mark buy signal: %w", err)
		}
	} else if prevIsFastAboveSlow && !currentIsFastAboveSlow {
		// SELL SIGNAL: Fast MA crossed below Slow MA
		order := &strategy.ExecuteOrder{
			Symbol:    data.Symbol,
			Side:      strategy.PurchaseType_PURCHASE_TYPE_SELL,
			OrderType: strategy.OrderType_ORDER_TYPE_LIMIT,
			Quantity:  1.0,
			Price:     data.Close,
			Reason: &strategy.Reason{
				Reason:  "strategy",
				Message: "Fast MA crossed below Slow MA",
			},
			StrategyName: "SimpleMAStrategy",
		}

		_, err := api.PlaceOrder(ctx, order)
		if err != nil {
			return nil, fmt.Errorf("failed to place sell order: %w", err)
		}

		// Mark this signal point
		_, err = api.Mark(ctx, &strategy.MarkRequest{
			MarketData: data,
			Signal:     strategy.SignalType_SIGNAL_TYPE_SELL_LONG,
			Reason:     "Fast MA crossed below Slow MA",
		})
		if err != nil {
			return nil, fmt.Errorf("failed to mark sell signal: %w", err)
		}
	}

	// Store current state for next comparison
	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   cacheKey,
		Value: fmt.Sprintf("%v", currentIsFastAboveSlow),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to update cache: %w", err)
	}

	return &emptypb.Empty{}, nil
}

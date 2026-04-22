//go:build wasip1

// Package main implements an RSI(14) long-only single-share strategy used to
// validate that argo's backtest engine produces the same trades, orders and
// PnL as the equivalent strategy run with `backtesting.py`.
//
// Strategy semantics (must stay in sync with
// `internal/indicator/test_data/src/comparison/runner.py`):
//
//   - When RSI(14) < 30 and we are flat, place a BUY for 1 share at data.Close.
//   - When RSI(14) > 70 and we are long 1 share, place a SELL for 1 share.
//   - Skip bars where the RSI cannot be computed yet (insufficient history).
//
// `commission` is configured to 0 in the backtest engine config so that the
// per-bar fill price equals data.Close, which is exactly how `backtesting.py`
// fills market orders with `trade_on_close=True`.
package main

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// Strategy parameters mirror the Python comparison runner. They are not
// configurable from yaml because the comparison test is driven entirely by
// the committed reference outputs.
const (
	rsiPeriod = 14
	rsiLower  = 30.0
	rsiUpper  = 70.0
	cacheKey  = "rsi_comparison_in_position"
)

// rsiRawValue mirrors the JSON shape returned by argo's RSI indicator
// (see internal/indicator/rsi.go GetSignal RawValue).
type rsiRawValue struct {
	RSI float64 `json:"rsi"`
}

// RSIComparisonStrategy is the WASM-side RSI(14) long-only strategy.
type RSIComparisonStrategy struct{}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(&RSIComparisonStrategy{})
}

// Initialize is a no-op: argo's RSI defaults (period=14, lower=30, upper=70)
// already match our comparison configuration, so no ConfigureIndicator call
// is required. Avoiding the call also sidesteps RSI.Config rejecting JSON
// numbers (which decode as float64) for its int-typed period parameter.
func (s *RSIComparisonStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *RSIComparisonStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "RSIComparisonStrategy"}, nil
}

func (s *RSIComparisonStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{
		Description: "RSI(14) long-only single-share strategy mirrored against backtesting.py for engine parity testing.",
	}, nil
}

func (s *RSIComparisonStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{Identifier: "com.argo-trading.e2e.rsi-comparison"}, nil
}

func (s *RSIComparisonStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: `{"type":"object","properties":{}}`}, nil
}

// ProcessData implements the RSI(14) long-only strategy.
func (s *RSIComparisonStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	api := strategy.NewStrategyApi()
	data := req.Data

	signal, err := api.GetSignal(ctx, &strategy.GetSignalRequest{
		IndicatorType: strategy.IndicatorType_INDICATOR_RSI,
		MarketData:    data,
	})
	if err != nil {
		// Insufficient historical data during the warm-up window is expected;
		// the host turns it into an error here, so just skip the bar.
		return &emptypb.Empty{}, nil
	}

	var raw rsiRawValue
	if err := json.Unmarshal([]byte(signal.RawValue), &raw); err != nil {
		return nil, fmt.Errorf("failed to parse RSI raw value %q: %w", signal.RawValue, err)
	}

	inPosition, err := s.loadInPosition(ctx, api)
	if err != nil {
		return nil, err
	}

	switch {
	case raw.RSI < rsiLower && !inPosition:
		if err := s.placeOrder(ctx, api, data, strategy.PurchaseType_PURCHASE_TYPE_BUY, raw.RSI); err != nil {
			return nil, err
		}
		if err := s.storeInPosition(ctx, api, true); err != nil {
			return nil, err
		}
	case raw.RSI > rsiUpper && inPosition:
		if err := s.placeOrder(ctx, api, data, strategy.PurchaseType_PURCHASE_TYPE_SELL, raw.RSI); err != nil {
			return nil, err
		}
		if err := s.storeInPosition(ctx, api, false); err != nil {
			return nil, err
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *RSIComparisonStrategy) loadInPosition(ctx context.Context, api strategy.StrategyApi) (bool, error) {
	resp, err := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})
	if err != nil {
		return false, fmt.Errorf("failed to read cache: %w", err)
	}
	if resp == nil {
		return false, nil
	}

	return resp.Value == "1", nil
}

func (s *RSIComparisonStrategy) storeInPosition(ctx context.Context, api strategy.StrategyApi, inPosition bool) error {
	value := "0"
	if inPosition {
		value = "1"
	}
	if _, err := api.SetCache(ctx, &strategy.SetRequest{Key: cacheKey, Value: value}); err != nil {
		return fmt.Errorf("failed to update cache: %w", err)
	}

	return nil
}

func (s *RSIComparisonStrategy) placeOrder(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, side strategy.PurchaseType, rsi float64) error {
	if _, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
		Symbol:       data.Symbol,
		Side:         side,
		OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
		Quantity:     1.0,
		Price:        data.Close,
		StrategyName: "RSIComparisonStrategy",
		Reason: &strategy.Reason{
			Reason:  "rsi-threshold",
			Message: fmt.Sprintf("RSI=%.4f", rsi),
		},
	}); err != nil {
		return fmt.Errorf("failed to place %s order: %w", side, err)
	}

	return nil
}

//go:build wasip1

package main

import (
	"context"
	"fmt"
	"strconv"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// PortfolioComparisonStrategy places a deterministic sequence of 10 orders
// driven by a bar counter stored in cache. It is used by the e2e test that
// verifies the FIFO and average-cost portfolio calculation strategies agree on
// FinalBalance and TotalFees.
//
// Schedule (0-indexed bar):
//
//	buys:   10, 20, 30, 40, 50, 60   (6 buys, qty 1.0 each)
//	sells:  70, 80, 90, 95           (4 sells, qty 1.0 each)
//
// Net: 2 long positions remain open at bar 99, so stats contain non-zero
// realized (from 4 closed pairs) and non-zero unrealized (from 2 open).
type PortfolioComparisonStrategy struct{}

const cacheKey = "portfolio_comparison_bar"

func main() {}

func init() {
	strategy.RegisterTradingStrategy(&PortfolioComparisonStrategy{})
}

func (s *PortfolioComparisonStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *PortfolioComparisonStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "PortfolioComparisonStrategy"}, nil
}

func (s *PortfolioComparisonStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "Deterministic 10-order strategy for FIFO vs average-cost parity testing"}, nil
}

func (s *PortfolioComparisonStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{Identifier: "com.argo-trading.e2e.portfolio-comparison"}, nil
}

func (s *PortfolioComparisonStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: `{"type":"object","properties":{}}`}, nil
}

func (s *PortfolioComparisonStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	api := strategy.NewStrategyApi()
	data := req.Data

	cached, err := api.GetCache(ctx, &strategy.GetRequest{Key: cacheKey})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	bar := 0
	if cached != nil && cached.Value != "" {
		if v, convErr := strconv.Atoi(cached.Value); convErr == nil {
			bar = v
		}
	}

	side, place := scheduleFor(bar)
	if place {
		if _, err := api.PlaceOrder(ctx, &strategy.ExecuteOrder{
			Symbol:       data.Symbol,
			Quantity:     1.0,
			Side:         side,
			OrderType:    strategy.OrderType_ORDER_TYPE_MARKET,
			Price:        data.Close,
			StrategyName: "PortfolioComparisonStrategy",
			Reason: &strategy.Reason{
				Reason:  "PortfolioComparisonStrategy",
				Message: "bar " + strconv.Itoa(bar),
			},
		}); err != nil {
			return nil, fmt.Errorf("failed to place order at bar %d: %w", bar, err)
		}
	}

	if _, err := api.SetCache(ctx, &strategy.SetRequest{
		Key:   cacheKey,
		Value: strconv.Itoa(bar + 1),
	}); err != nil {
		return nil, fmt.Errorf("failed to set cache: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// scheduleFor returns the order side for a given bar index and whether an
// order should be placed. Split out for clarity; keep in sync with the header
// comment on PortfolioComparisonStrategy.
func scheduleFor(bar int) (strategy.PurchaseType, bool) {
	switch bar {
	case 10, 20, 30, 40, 50, 60:
		return strategy.PurchaseType_PURCHASE_TYPE_BUY, true
	case 70, 80, 90, 95:
		return strategy.PurchaseType_PURCHASE_TYPE_SELL, true
	}
	return strategy.PurchaseType_PURCHASE_TYPE_BUY, false
}

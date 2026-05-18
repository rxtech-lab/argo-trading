//go:build wasip1

// LogEveryTickStrategy is a minimal WASM strategy used by live-trading e2e
// tests. Each ProcessData call emits exactly one INFO log carrying the bar's
// symbol and close price. The tests stream N bars through a mock provider
// and assert that logs.parquet contains exactly N distinct rows — verifying
// the full host → gRPC → LogStorage → tick-loop-cursor → parquet pipeline.
package main

import (
	"context"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type LogEveryTickStrategy struct{}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(&LogEveryTickStrategy{})
}

func (s *LogEveryTickStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *LogEveryTickStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "LogEveryTickStrategy"}, nil
}

func (s *LogEveryTickStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "Logs once per ProcessData; used by live-trading e2e tests."}, nil
}

func (s *LogEveryTickStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()
	_, _ = api.Log(ctx, &strategy.LogRequest{
		Level:   strategy.LogLevel_LOG_LEVEL_INFO,
		Message: fmt.Sprintf("tick %s close=%.4f", data.Symbol, data.Close),
	})
	return &emptypb.Empty{}, nil
}

func (s *LogEveryTickStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: "{}"}, nil
}

func (s *LogEveryTickStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{Identifier: "com.argo-trading.e2e.log-every-tick"}, nil
}

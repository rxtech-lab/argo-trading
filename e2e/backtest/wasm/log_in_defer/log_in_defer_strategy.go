//go:build wasip1

// LogInDeferStrategy mirrors the real-world pattern where a strategy emits a
// "no action taken" log via a deferred function at ProcessData exit. The
// MultiConfirmStrategy uses this pattern for HOLD lines. This strategy
// proves the host correctly persists logs emitted from inside a defer.
//
// Each ProcessData call:
//   - registers a defer that calls api.Log with LOG_LEVEL_DEBUG
//   - returns normally
//
// The e2e test streams N bars and asserts logs.parquet has N rows at
// level=debug. If the defer-based log path is broken anywhere in the
// WASM → gRPC → host → LogStorage → cursor → parquet pipeline, this
// test fails.
package main

import (
	"context"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type LogInDeferStrategy struct{}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(&LogInDeferStrategy{})
}

func (s *LogInDeferStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	return &emptypb.Empty{}, nil
}

func (s *LogInDeferStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "LogInDeferStrategy"}, nil
}

func (s *LogInDeferStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "Logs a DEBUG line via defer; used by live-trading e2e tests."}, nil
}

func (s *LogInDeferStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()

	defer func() {
		_, _ = api.Log(ctx, &strategy.LogRequest{
			Level:   strategy.LogLevel_LOG_LEVEL_DEBUG,
			Message: fmt.Sprintf("defer HOLD symbol=%s close=%.4f", data.Symbol, data.Close),
		})
	}()

	return &emptypb.Empty{}, nil
}

func (s *LogInDeferStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: "{}"}, nil
}

func (s *LogInDeferStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{Identifier: "com.argo-trading.e2e.log-in-defer"}, nil
}

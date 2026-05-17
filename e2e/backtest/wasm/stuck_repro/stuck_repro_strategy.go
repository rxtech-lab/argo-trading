//go:build wasip1

// StuckReproStrategy is a minimal reproducer for the "stuck at first tick"
// bug observed in the user's MultiConfirmStrategy.
//
// The bug: a one-shot init log gated on `tickCount == 0` fires on every
// warmup bar (not just once) because `tickCount++` happens AFTER an early
// return that triggers while indicators (RSI) are still warming up.
//
// Pattern (BUG):
//
//	if tickCount == 0 { api.Log("started") }   // fires every warmup tick
//	rsi := getSignal(...)
//	if rsi == 0 { return }                     // early return before tickCount++
//	tickCount++                                // only reached once RSI is ready
//
// Result: many duplicate "started" log lines (one per RSI-warmup bar) so the
// strategy appears frozen on the first tick from a log-watcher's POV.
//
// FIX: use a dedicated `started` flag that flips on the first call, instead
// of relying on `tickCount` which gets skipped past by early-returns.
package main

import (
	"context"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type StuckReproStrategy struct{}

var (
	tickCount int
	started   bool // FIX: one-shot guard independent of early-returns
)

func main() {}

func init() { strategy.RegisterTradingStrategy(&StuckReproStrategy{}) }

func (s *StuckReproStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	return &strategy.GetConfigSchemaResponse{Schema: "{}"}, nil
}

func (s *StuckReproStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "Minimal repro for the stuck-at-first-tick log dup bug."}, nil
}

func (s *StuckReproStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{Identifier: "com.argo-trading.e2e.stuck-repro"}, nil
}

func (s *StuckReproStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "StuckReproStrategy"}, nil
}

func (s *StuckReproStrategy) Initialize(_ context.Context, _ *strategy.InitializeRequest) (*emptypb.Empty, error) {
	tickCount = 0
	started = false
	return &emptypb.Empty{}, nil
}

func (s *StuckReproStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	data := req.Data
	api := strategy.NewStrategyApi()

	// One-shot init log — guarded by a dedicated flag instead of `tickCount==0`
	// so it fires exactly once, not once per warmup bar.
	if !started {
		started = true
		_, _ = api.Log(ctx, &strategy.LogRequest{
			Level:   strategy.LogLevel_LOG_LEVEL_INFO,
			Message: "StuckReproStrategy started",
			Fields:  map[string]string{"symbol": data.Symbol},
		})
	}

	// Fetch RSI to mimic the warmup-return path from the original strategy.
	rsi := getSignal(ctx, api, data, strategy.IndicatorType_INDICATOR_RSI)
	if rsi == 0 {
		// Warmup: RSI not ready. Returning here used to leave the BUGGY
		// `tickCount == 0` guard active, causing repeated "started" logs.
		return &emptypb.Empty{}, nil
	}

	tickCount++

	// Emit a per-tick HOLD log so the test can verify ticks actually progressed.
	_, _ = api.Log(ctx, &strategy.LogRequest{
		Level:   strategy.LogLevel_LOG_LEVEL_DEBUG,
		Message: fmt.Sprintf("HOLD tick=%d close=%.2f", tickCount, data.Close),
	})

	return &emptypb.Empty{}, nil
}

func getSignal(ctx context.Context, api strategy.StrategyApi, data *strategy.MarketData, t strategy.IndicatorType) float64 {
	resp, err := api.GetSignal(ctx, &strategy.GetSignalRequest{IndicatorType: t, MarketData: data})
	if err != nil || resp == nil || resp.RawValue == "" {
		return 0
	}
	// Any non-zero RawValue means "ready" — the exact value doesn't matter
	// for this repro; we only care about ready vs warming-up.
	return 1
}

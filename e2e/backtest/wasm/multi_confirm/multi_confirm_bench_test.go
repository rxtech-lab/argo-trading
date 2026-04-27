//go:build !wasip1

package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
)

const (
	benchNumDataPoints = 100_000
	benchWasmPath      = "./multi_confirm_plugin.wasm"
)

type benchStrategyConfig struct {
	FixedNotional float64 `json:"fixedNotional"`
}

// runBenchmarkOnce executes a single 100k-bar backtest and returns wall-clock
// time + bars/sec. Mock data and config are written to t.TempDir() — the WASM
// plugin must be pre-built (run `make build` in e2e/backtest/wasm).
func runBenchmarkOnce(tb testing.TB) (elapsed time.Duration, barsPerSec float64) {
	tb.Helper()

	if _, err := os.Stat(benchWasmPath); err != nil {
		tb.Skipf("WASM plugin not built at %s — run `make build` in e2e/backtest/wasm", benchWasmPath)
	}

	tmp := tb.TempDir()
	dataPath := filepath.Join(tmp, testhelper.GenerateMockFilename("multi_confirm_100k"))

	genStart := time.Now()
	err := testhelper.GenerateAndWriteToParquet(testhelper.MockDataConfig{
		Symbol:             "BTCUSDT",
		StartTime:          time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Interval:           time.Minute,
		NumDataPoints:      benchNumDataPoints,
		Pattern:            testhelper.PatternVolatile,
		InitialPrice:       50_000,
		VolatilityPercent:  1.5,
		MaxDrawdownPercent: 25,
		Seed:               42,
	}, dataPath)
	if err != nil {
		tb.Fatalf("generate mock data: %v", err)
	}
	tb.Logf("generated %d bars in %s", benchNumDataPoints, time.Since(genStart))

	cfgBytes, err := json.Marshal(benchStrategyConfig{FixedNotional: 15000})
	if err != nil {
		tb.Fatal(err)
	}
	cfgPath := filepath.Join(tmp, "config.json")
	if err := os.WriteFile(cfgPath, cfgBytes, 0o644); err != nil {
		tb.Fatal(err)
	}

	backtest, err := v1.NewBacktestEngineV1()
	if err != nil {
		tb.Fatal(err)
	}
	if err := backtest.Initialize(`
initial_capital: 1000000
market_data_cache_size: 10000
`); err != nil {
		tb.Fatal(err)
	}

	log, err := logger.NewLogger()
	if err != nil {
		tb.Fatal(err)
	}
	ds, err := datasource.NewDataSource(":memory:", log)
	if err != nil {
		tb.Fatal(err)
	}
	if err := backtest.SetDataSource(ds); err != nil {
		tb.Fatal(err)
	}
	if err := backtest.SetDataPath(dataPath); err != nil {
		tb.Fatal(err)
	}

	rt, err := wasm.NewStrategyWasmRuntime(benchWasmPath)
	if err != nil {
		tb.Fatal(err)
	}
	if err := backtest.LoadStrategy(rt); err != nil {
		tb.Fatal(err)
	}
	if err := backtest.SetResultsFolder(filepath.Join(tmp, "results")); err != nil {
		tb.Fatal(err)
	}
	if err := backtest.SetConfigPath(cfgPath); err != nil {
		tb.Fatal(err)
	}

	runStart := time.Now()
	if err := backtest.Run(context.Background(), engine.LifecycleCallbacks{}); err != nil {
		tb.Fatalf("run: %v", err)
	}
	elapsed = time.Since(runStart)
	barsPerSec = float64(benchNumDataPoints) / elapsed.Seconds()
	return elapsed, barsPerSec
}

// TestBenchmarkMultiConfirm100k runs once and prints throughput. Suitable for
// quick perf checks: `go test -run TestBenchmarkMultiConfirm100k -v -timeout 30m`.
func TestBenchmarkMultiConfirm100k(t *testing.T) {
	elapsed, bps := runBenchmarkOnce(t)
	t.Logf("multi_confirm 100k bars: %s (%.0f bars/sec, %.2f µs/bar)",
		elapsed, bps, float64(elapsed.Microseconds())/float64(benchNumDataPoints))
}

// BenchmarkMultiConfirm100k integrates with `go test -bench`. Run with
// `-benchtime=1x` to do a single 100k-bar pass:
//
//	go test -bench BenchmarkMultiConfirm100k -benchtime=1x -timeout 30m
func BenchmarkMultiConfirm100k(b *testing.B) {
	for i := 0; i < b.N; i++ {
		elapsed, bps := runBenchmarkOnce(b)
		b.ReportMetric(bps, "bars/sec")
		b.ReportMetric(float64(elapsed.Microseconds())/float64(benchNumDataPoints), "µs/bar")
	}
}

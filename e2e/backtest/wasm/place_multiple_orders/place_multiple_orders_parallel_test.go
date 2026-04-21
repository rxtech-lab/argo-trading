package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// numDataFiles controls how many independent data files the parallel test
// uses. The value is intentionally chosen to be large enough to make the
// per-iteration WASM startup cost dwarfed by the cost of processing the bars
// while still small enough that the test runs quickly. Each data file is
// processed by exactly one worker, so with 8 files and 4 workers we get 2
// files per worker on average.
const numDataFiles = 8

// orderEveryN must match the value baked into the strategy config below; it
// is used by the test to compute the number of orders the strategy is expected
// to place per data file.
const orderEveryN = 50

// PlaceMultipleOrdersParallelSuite verifies that running the backtest engine
// with MaxConcurrency > 1 across multiple data files:
//  1. Produces the same number of total orders as the sequential run
//     (no regression in correctness).
//  2. Completes faster than the sequential run for the same workload
//     (performance gain).
//  3. Emits OnProcessData callbacks whose final cumulative count matches
//     the total number of bars processed across all data files.
type PlaceMultipleOrdersParallelSuite struct {
	testhelper.E2ETestSuite
}

func TestPlaceMultipleOrdersParallelSuite(t *testing.T) {
	suite.Run(t, new(PlaceMultipleOrdersParallelSuite))
}

// SetupTest is intentionally a no-op: each sub-test creates its own engine so
// we can run sequential and parallel back-to-back with isolated state.
func (s *PlaceMultipleOrdersParallelSuite) SetupTest() {}

// runResult bundles the observable outputs of a single full backtest run.
type runResult struct {
	duration       time.Duration
	totalStats     int
	totalOrders    int
	totalBarsProc  int // total OnProcessData invocations
	maxBarsSoFar   int // largest `current` value observed
	maxTotal       int // largest `total` value observed
	progressMonoOK bool
}

// runBacktest performs one full backtest run with the given concurrency. It
// recreates the engine from scratch each call so the two runs (sequential vs
// parallel) are completely independent.
func (s *PlaceMultipleOrdersParallelSuite) runBacktest(concurrency int, dataPattern string, configPath string, resultsRoot string) runResult {
	t := s.T()

	yamlCfg := fmt.Sprintf("initial_capital: 1000000\nmax_concurrency: %d\n", concurrency)

	backtest, err := v1.NewBacktestEngineV1()
	require.NoError(t, err)
	require.NoError(t, backtest.Initialize(yamlCfg))

	log, err := logger.NewLogger()
	require.NoError(t, err)
	ds, err := datasource.NewDataSource(":memory:", log)
	require.NoError(t, err)
	require.NoError(t, backtest.SetDataSource(ds))

	require.NoError(t, backtest.LoadStrategyFromFile("./place_multiple_orders_plugin.wasm"))
	require.NoError(t, backtest.SetDataPath(dataPattern))
	require.NoError(t, backtest.SetResultsFolder(resultsRoot))
	require.NoError(t, backtest.SetConfigPath(configPath))

	// Track progress callbacks. With multiple workers these are invoked from
	// different goroutines, so we serialize via a mutex. The wrapped
	// callbacks installed by the parallel runner already serialize calls,
	// but using our own mutex keeps the test correct under both code paths.
	var (
		mu          sync.Mutex
		lastCurrent int
		lastTotal   int
		monoOK      = true
		callCount   int
	)

	onProcess := engine.OnProcessDataCallback(func(current int, total int) error {
		mu.Lock()
		defer mu.Unlock()
		callCount++
		if current < lastCurrent {
			monoOK = false
		}
		if current > lastCurrent {
			lastCurrent = current
		}
		if total > lastTotal {
			lastTotal = total
		}
		return nil
	})

	start := time.Now()
	err = backtest.Run(context.Background(), engine.LifecycleCallbacks{
		OnProcessData: &onProcess,
	})
	require.NoError(t, err)
	duration := time.Since(start)

	stats, err := testhelper.ReadStats(&s.E2ETestSuite, resultsRoot)
	require.NoError(t, err)

	totalOrders := 0
	for _, st := range stats {
		totalOrders += st.TradeResult.NumberOfTrades
	}

	return runResult{
		duration:       duration,
		totalStats:     len(stats),
		totalOrders:    totalOrders,
		totalBarsProc:  callCount,
		maxBarsSoFar:   lastCurrent,
		maxTotal:       lastTotal,
		progressMonoOK: monoOK,
	}
}

// writeStrategyConfig writes the strategy JSON config used by both runs.
func (s *PlaceMultipleOrdersParallelSuite) writeStrategyConfig(dir string) string {
	t := s.T()
	cfgPath := filepath.Join(dir, "config.json")
	cfg := map[string]any{
		"orderEveryN": orderEveryN,
	}
	bytes, err := json.Marshal(cfg)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(dir, 0755))
	require.NoError(t, os.WriteFile(cfgPath, bytes, 0644))
	return cfgPath
}

// prepareDataFiles produces `numDataFiles` parquet files derived from the
// shared test_data fixture, each rewritten with a unique symbol so the engine
// treats them as independent datasets.
func (s *PlaceMultipleOrdersParallelSuite) prepareDataFiles(dir string) string {
	t := s.T()
	require.NoError(t, os.MkdirAll(dir, 0755))
	for i := 1; i <= numDataFiles; i++ {
		out := filepath.Join(dir, fmt.Sprintf("data_%d.parquet", i))
		require.NoError(t, testhelper.UpdateParquetSymbol(
			"../../../../internal/indicator/test_data/test_data.parquet",
			out,
			fmt.Sprintf("PARSYM%d", i),
		))
	}
	return filepath.Join(dir, "*.parquet")
}

// TestParallelMatchesSequentialAndIsFaster validates the three guarantees the
// parallel backtest runner is meant to provide. We deliberately do not assert
// "exactly N times faster" because real wallclock speedup depends on host CPU
// availability; the assertion is the looser "parallel is faster than
// sequential" which is what end users actually care about.
func (s *PlaceMultipleOrdersParallelSuite) TestParallelMatchesSequentialAndIsFaster() {
	t := s.T()

	tmp := t.TempDir()
	dataPattern := s.prepareDataFiles(filepath.Join(tmp, "data"))
	configPath := s.writeStrategyConfig(filepath.Join(tmp, "cfg"))

	sequentialResults := filepath.Join(tmp, "results-seq")
	parallelResults := filepath.Join(tmp, "results-par")

	// Sequential baseline.
	seq := s.runBacktest(1, dataPattern, configPath, sequentialResults)

	// Parallel run with 4 workers.
	par := s.runBacktest(4, dataPattern, configPath, parallelResults)

	// (1) Multiple runners should not regress: stats and order counts must
	// match between sequential and parallel runs.
	require.Equal(t, numDataFiles, seq.totalStats, "sequential should produce one stats file per data file")
	require.Equal(t, seq.totalStats, par.totalStats, "parallel run should produce the same number of stats as sequential")
	require.Equal(t, seq.totalOrders, par.totalOrders, "parallel run should place the same total number of orders as sequential")
	require.Greater(t, seq.totalOrders, 0, "strategy should place at least one order overall")

	// (2) Performance gain: parallel run should be measurably faster than
	// sequential. We allow a generous threshold (must be at least 10% faster)
	// to avoid flakiness on heavily contended CI runners while still
	// detecting regressions in the parallel scheduler.
	t.Logf("sequential=%s parallel=%s", seq.duration, par.duration)
	require.Less(t, par.duration, seq.duration*9/10,
		"parallel run (%s) should be at least 10%% faster than sequential (%s)",
		par.duration, seq.duration)

	// (3) Progress callback correctness:
	//
	// In sequential mode the engine reports progress per-iteration, so the
	// `current` value resets to 1 between data files. The relevant invariants
	// are: the user receives one OnProcessData call per processed bar, and
	// the highest `total` value observed equals the largest single iteration's
	// bar count.
	//
	// In parallel mode the runner aggregates progress across workers, so
	// `current` is monotonically non-decreasing and the final cumulative
	// value equals the total number of bars processed across all data files.
	require.Equal(t, seq.totalBarsProc, par.totalBarsProc,
		"parallel and sequential should process the same number of bars in total")
	require.True(t, par.progressMonoOK,
		"parallel progress callback's `current` value must be monotonically non-decreasing")
	require.Equal(t, par.totalBarsProc, par.maxBarsSoFar,
		"parallel progress callback's max `current` should equal the total number of OnProcessData invocations")
	require.GreaterOrEqual(t, par.maxTotal, par.maxBarsSoFar,
		"parallel progress callback's `total` should never be smaller than the cumulative `current`")
}

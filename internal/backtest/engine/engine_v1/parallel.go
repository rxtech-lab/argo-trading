package engine

import (
	"context"
	"sync"

	"github.com/google/uuid"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/cache"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"go.uber.org/zap"
)

// parallelJob describes a single (config, dataPath) iteration that a worker
// goroutine should execute as part of a parallel run.
type parallelJob struct {
	configIdx        int
	configName       string
	configContent    string
	dataIdx          int
	dataPath         string
	resultFolderPath string
}

// runStrategyParallel executes all (config, dataPath) iterations for a single
// strategy across `concurrency` worker goroutines. Each worker owns its own
// isolated iterResources (state, trading system, marker, log storage, cache,
// datasource, strategy runtime instance) so they cannot race on shared state.
//
// All callbacks invoked from worker goroutines (OnRunStart, OnRunEnd,
// OnProcessData) are guarded by a mutex so that callback implementations do
// not need to be thread-safe themselves. OnProcessData receives a cumulative
// `current` value summed across all running workers, which is what the engine
// has historically reported to consumers (a single monotonically-increasing
// progress counter).
func (b *BacktestEngineV1) runStrategyParallel(
	ctx context.Context,
	strategyIdx int,
	originalStrategy runtime.StrategyRuntime,
	strategyPath string,
	configs []configItem,
	callbacks engine.LifecycleCallbacks,
	concurrency int,
) error {
	// Build the job list (one job per config x dataPath combination).
	jobs := make([]parallelJob, 0, len(configs)*len(b.dataPaths))
	for configIdx, cfg := range configs {
		for dataIdx, dataPath := range b.dataPaths {
			jobs = append(jobs, parallelJob{
				configIdx:        configIdx,
				configName:       cfg.name,
				configContent:    cfg.content,
				dataIdx:          dataIdx,
				dataPath:         dataPath,
				resultFolderPath: getResultFolder(cfg.name, dataPath, b, originalStrategy),
			})
		}
	}

	if len(jobs) == 0 {
		return nil
	}

	// Cap concurrency at the number of jobs - more workers would just sit idle.
	if concurrency > len(jobs) {
		concurrency = len(jobs)
	}

	// Wrap user-supplied callbacks with mutexes so worker goroutines can call
	// them safely and so OnProcessData reports a global cumulative progress.
	wrappedCallbacks, _ := wrapCallbacksForParallel(callbacks)

	// Worker pool with a buffered job channel.
	jobCh := make(chan parallelJob, len(jobs))
	for _, j := range jobs {
		jobCh <- j
	}
	close(jobCh)

	// Cancellable context so that the first worker error stops the rest.
	workerCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var (
		wg       sync.WaitGroup
		errMu    sync.Mutex
		firstErr error
	)

	for w := 0; w < concurrency; w++ {
		workerIdx := w

		// Each worker gets its own resources up-front and reuses them for every
		// job it pulls from the queue.
		res, err := b.newWorkerResources(strategyIdx)
		if err != nil {
			cancel()
			wg.Wait()

			return errors.Wrap(errors.ErrCodeBacktestInitFailed, "failed to build worker resources", err)
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			defer func() {
				if res.datasource != nil {
					_ = res.datasource.Close()
				}
				if mk, ok := res.marker.(*BacktestMarker); ok {
					_ = mk.Close()
				}
			}()

			for job := range jobCh {
				select {
				case <-workerCtx.Done():
					return
				default:
				}

				params := runIterationParams{
					ctx:              workerCtx,
					res:              res,
					strategyPath:     strategyPath,
					runID:            uuid.New().String(),
					configIdx:        job.configIdx,
					configName:       job.configName,
					configContent:    job.configContent,
					dataIdx:          job.dataIdx,
					dataPath:         job.dataPath,
					callbacks:        wrappedCallbacks,
					resultFolderPath: job.resultFolderPath,
				}

				if err := b.runSingleIteration(params); err != nil {
					b.log.Error("Parallel worker iteration failed",
						zap.Int("worker", workerIdx),
						zap.String("data", job.dataPath),
						zap.Error(err),
					)

					errMu.Lock()
					if firstErr == nil {
						firstErr = err
					}
					errMu.Unlock()

					cancel()

					return
				}
			}
		}()
	}

	wg.Wait()

	if firstErr != nil {
		return firstErr
	}

	if err := ctx.Err(); err != nil {
		return err
	}

	return nil
}

// newWorkerResources constructs an isolated set of resources for a single
// parallel worker. It is the parallel-mode equivalent of the engine-level
// fields (b.state, b.tradingSystem, ...). Strategy runtimes are cloned from
// the stored strategy file path or the stored strategy bytes; if neither is
// available (i.e. the strategy was loaded via LoadStrategy with an in-memory
// instance) parallel execution cannot proceed because the underlying WASM
// plugin is stateful and cannot be safely shared across goroutines.
func (b *BacktestEngineV1) newWorkerResources(strategyIdx int) (*iterResources, error) {
	state, err := NewBacktestState(b.log)
	if err != nil {
		return nil, err
	}

	if err := state.Initialize(); err != nil {
		return nil, err
	}

	state.SetInitialBalance(b.config.InitialCapital)

	var commissionFee commission_fee.CommissionFee

	switch b.config.Broker {
	case commission_fee.BrokerInteractiveBroker:
		commissionFee = commission_fee.NewInteractiveBrokerCommissionFee()
	case commission_fee.BrokerZero:
		commissionFee = commission_fee.NewZeroCommissionFee()
	default:
		commissionFee = commission_fee.NewInteractiveBrokerCommissionFee()
	}

	trading := NewBacktestTrading(state, b.config.InitialCapital, commissionFee, b.config.DecimalPrecision)

	mk, err := NewBacktestMarker(b.log)
	if err != nil {
		return nil, err
	}

	logStorage, err := NewBacktestLog(b.log)
	if err != nil {
		return nil, err
	}

	ds, err := datasource.NewDataSource(":memory:", b.log)
	if err != nil {
		return nil, err
	}

	strategyRuntime, err := b.cloneStrategyRuntime(strategyIdx)
	if err != nil {
		return nil, err
	}

	return &iterResources{
		state:             state,
		tradingSystem:     trading,
		marker:            mk,
		logStorage:        logStorage,
		cache:             cache.NewCacheV1(),
		datasource:        ds,
		strategy:          strategyRuntime,
		indicatorRegistry: b.indicatorRegistry,
	}, nil
}

// cloneStrategyRuntime creates a fresh, independent strategy runtime instance
// for a parallel worker, using whichever source (file path or in-memory bytes)
// was supplied when the strategy was originally loaded.
func (b *BacktestEngineV1) cloneStrategyRuntime(idx int) (runtime.StrategyRuntime, error) {
	if idx < len(b.strategyPaths) && b.strategyPaths[idx] != "" {
		return wasm.NewStrategyWasmRuntime(b.strategyPaths[idx])
	}

	if idx < len(b.strategyBytes) && len(b.strategyBytes[idx]) > 0 {
		return wasm.NewStrategyWasmRuntimeFromBytes(b.strategyBytes[idx])
	}

	return nil, errors.New(errors.ErrCodeStrategyRuntimeError,
		"cannot clone strategy for parallel execution: strategy must be loaded via LoadStrategyFromFile or LoadStrategyFromBytes when MaxConcurrency > 1")
}

// progressTracker aggregates per-worker progress into a single cumulative
// counter that mirrors the sequential semantics of OnProcessData (a value
// that monotonically grows from 1 to total across the lifetime of the run).
type progressTracker struct {
	mu         sync.Mutex
	cumulative int
	grandTotal int
	totalKnown bool
}

func newProgressTracker() *progressTracker {
	return &progressTracker{}
}

// addRunTotal is invoked from OnRunStart to grow the global "total" so that
// OnProcessData callers see a `total` that represents the sum of bars across
// all iterations the parallel run will process.
func (p *progressTracker) addRunTotal(n int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.grandTotal += n
	p.totalKnown = true
}

// tick increments the cumulative counter by one (one call per processed bar)
// and returns the new (cumulative, total) pair to forward to the user
// callback. If OnRunStart hasn't fired yet for any worker (and therefore the
// grand total is unknown) the caller-supplied fallbackTotal is used so that
// the user never sees a total smaller than the values being reported.
func (p *progressTracker) tick(fallbackTotal int) (int, int) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.cumulative++

	total := p.grandTotal
	if !p.totalKnown || total < p.cumulative {
		if fallbackTotal > total {
			total = fallbackTotal
		}
	}

	return p.cumulative, total
}

// Cumulative returns the latest cumulative progress count.
func (p *progressTracker) Cumulative() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.cumulative
}

// GrandTotal returns the latest known total progress target.
func (p *progressTracker) GrandTotal() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.grandTotal
}

// wrapCallbacksForParallel rewrites callbacks for safe concurrent invocation.
// All user callbacks are serialized through a single mutex so callback
// implementations don't have to be thread-safe themselves. OnProcessData is
// re-mapped to forward a global cumulative counter / total instead of the
// per-iteration counter that worker-local code emits.
func wrapCallbacksForParallel(user engine.LifecycleCallbacks) (engine.LifecycleCallbacks, *progressTracker) {
	progress := newProgressTracker()

	var callbackMu sync.Mutex

	wrapped := engine.LifecycleCallbacks{
		OnBacktestStart: user.OnBacktestStart,
		OnBacktestEnd:   user.OnBacktestEnd,
		OnStrategyStart: user.OnStrategyStart,
		OnStrategyEnd:   user.OnStrategyEnd,
	}

	// Always wrap OnRunStart so we can accumulate the grand total, even when
	// the user did not provide their own OnRunStart hook.
	runStart := func(runID string, configIdx int, configName string, dataFileIdx int, dataFilePath string, totalDataPoints int) error {
		progress.addRunTotal(totalDataPoints)

		if user.OnRunStart != nil {
			callbackMu.Lock()
			defer callbackMu.Unlock()
			return (*user.OnRunStart)(runID, configIdx, configName, dataFileIdx, dataFilePath, totalDataPoints)
		}
		return nil
	}
	runStartCb := engine.OnRunStartCallback(runStart)
	wrapped.OnRunStart = &runStartCb

	if user.OnRunEnd != nil {
		runEnd := func(configIdx int, configName string, dataFileIdx int, dataFilePath string, resultFolderPath string) {
			callbackMu.Lock()
			defer callbackMu.Unlock()
			(*user.OnRunEnd)(configIdx, configName, dataFileIdx, dataFilePath, resultFolderPath)
		}
		runEndCb := engine.OnRunEndCallback(runEnd)
		wrapped.OnRunEnd = &runEndCb
	}

	if user.OnProcessData != nil {
		procData := func(_ int, total int) error {
			cum, gt := progress.tick(total)

			callbackMu.Lock()
			defer callbackMu.Unlock()
			return (*user.OnProcessData)(cum, gt)
		}
		procCb := engine.OnProcessDataCallback(procData)
		wrapped.OnProcessData = &procCb
	} else {
		// Even without a user OnProcessData callback, install a no-op tick
		// so progress.Cumulative() / GrandTotal() still report meaningful
		// values to internal tests that introspect the progress tracker.
		procData := func(_ int, total int) error {
			progress.tick(total)
			return nil
		}
		procCb := engine.OnProcessDataCallback(procData)
		wrapped.OnProcessData = &procCb
	}

	return wrapped, progress
}

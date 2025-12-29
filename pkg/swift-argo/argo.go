package swiftargo

import (
	"context"
	"sync"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	engine_v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

type ArgoHelper interface {
	OnBacktestStart(totalStrategies int, totalConfigs int, totalDataFiles int) error
	OnBacktestEnd(err error)
	OnStrategyStart(strategyIndex int, strategyName string, totalStrategies int) error
	OnStrategyEnd(strategyIndex int, strategyName string)
	OnRunStart(runID string, configIndex int, configName string, dataFileIndex int, dataFilePath string, totalDataPoints int) error
	OnRunEnd(configIndex int, configName string, dataFileIndex int, dataFilePath string, resultFolderPath string)
	OnProcessData(current int, total int) error
}

type Argo struct {
	helper ArgoHelper
	engine engine.Engine

	// Cancellation support
	mu         sync.Mutex
	cancelFunc context.CancelFunc
}

// GetBacktestEngineConfigSchema returns the backtest engine config schema.
func GetBacktestEngineConfigSchema() string {
	schema, err := strategy.ToJSONSchema(engine_v1.BacktestEngineV1Config{
		InitialCapital:   0,
		Broker:           "",
		StartTime:        optional.None[time.Time](),
		EndTime:          optional.None[time.Time](),
		DecimalPrecision: 0,
	})
	if err != nil {
		return ""
	}

	return schema
}

// GetBacktestEngineVersion returns the backtest engine version.
func GetBacktestEngineVersion() string {
	return version.GetVersion()
}

func NewArgo(helper ArgoHelper) (*Argo, error) {
	backtestEngine, err := engine_v1.NewBacktestEngineV1()
	if err != nil {
		return nil, err
	}

	return &Argo{
		helper:     helper,
		engine:     backtestEngine,
		mu:         sync.Mutex{},
		cancelFunc: nil,
	}, nil
}

func (a *Argo) SetConfigContent(configs StringCollection) error {
	// Convert StringCollection to []string (gomobile doesn't support slice returns)
	items := make([]string, configs.Size())
	for i := 0; i < configs.Size(); i++ {
		items[i] = configs.Get(i)
	}

	return a.engine.SetConfigContent(items)
}

func (a *Argo) SetDataPath(path string) error {
	return a.engine.SetDataPath(path)
}

// Run executes the backtest engine. This method is blocking.
// Can be cancelled by calling Cancel() from another goroutine.
func (a *Argo) Run(backtestConfig string, strategyPath string, resultsFolderPath string) error {
	// Create cancellable context
	ctx, cancel := context.WithCancel(context.Background())

	// Store cancel function with mutex protection
	a.mu.Lock()
	a.cancelFunc = cancel
	a.mu.Unlock()

	// Ensure we clean up the cancel function when done
	defer func() {
		a.mu.Lock()
		a.cancelFunc = nil
		a.mu.Unlock()
	}()

	// Create callbacks from helper interface
	onBacktestStart := engine.OnBacktestStartCallback(a.helper.OnBacktestStart)
	onBacktestEnd := engine.OnBacktestEndCallback(a.helper.OnBacktestEnd)
	onStrategyStart := engine.OnStrategyStartCallback(a.helper.OnStrategyStart)
	onStrategyEnd := engine.OnStrategyEndCallback(a.helper.OnStrategyEnd)
	onRunStart := engine.OnRunStartCallback(a.helper.OnRunStart)
	onRunEnd := engine.OnRunEndCallback(a.helper.OnRunEnd)
	onProcessData := engine.OnProcessDataCallback(a.helper.OnProcessData)

	// Initialize the engine with the given configuration file.
	if err := a.engine.Initialize(backtestConfig); err != nil {
		return err
	}

	callbacks := engine.LifecycleCallbacks{
		OnBacktestStart: &onBacktestStart,
		OnBacktestEnd:   &onBacktestEnd,
		OnStrategyStart: &onStrategyStart,
		OnStrategyEnd:   &onStrategyEnd,
		OnRunStart:      &onRunStart,
		OnRunEnd:        &onRunEnd,
		OnProcessData:   &onProcessData,
	}

	err := a.engine.LoadStrategyFromFile(strategyPath)
	if err != nil {
		return err
	}

	err = a.engine.SetResultsFolder(resultsFolderPath)
	if err != nil {
		return err
	}

	return a.engine.Run(ctx, callbacks)
}

// Cancel cancels any in-progress run.
// This method is safe to call from any goroutine (e.g., Swift's main thread).
// Returns true if a run was cancelled, false if no run was in progress.
func (a *Argo) Cancel() bool {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.cancelFunc != nil {
		a.cancelFunc()
		a.cancelFunc = nil

		return true
	}

	return false
}

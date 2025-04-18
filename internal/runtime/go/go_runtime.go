package go_runtime

import (
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// GoRuntime is a runtime for a strategy that is written in Go struct.
// So you don't need to write a wasm file to run it.
// It is used for testing and development purposes.
type GoRuntime struct {
	strategy runtime.StrategyRuntime
}

// Initialize implements StrategyRuntime.
func (g *GoRuntime) Initialize(config string) error {
	panic("unimplemented")
}

// Name implements StrategyRuntime.
func (g *GoRuntime) Name() string {
	panic("unimplemented")
}

// ProcessData implements StrategyRuntime.
func (g *GoRuntime) ProcessData(data types.MarketData) error {
	panic("unimplemented")
}

func (g *GoRuntime) InitializeApi(api strategy.StrategyApi) error {
	panic("unimplemented")
}

// GetConfigSchema implements StrategyRuntime.
func (g *GoRuntime) GetConfigSchema() (string, error) {
	panic("unimplemented")
}

func NewGoRuntime(strategy runtime.StrategyRuntime) runtime.StrategyRuntime {
	return &GoRuntime{
		strategy: strategy,
	}
}

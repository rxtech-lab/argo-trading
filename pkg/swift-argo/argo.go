package swiftargo

import (
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
)

type Argo struct {
	engine engine.Engine
}

func NewArgo() *Argo {
	return &Argo{
		engine: v1.NewBacktestEngineV1(),
	}
}

func (a *Argo) SetConfigPath(path string) {
	a.engine.SetConfigPath(path)
}

func (a *Argo) SetDataPath(path string) {
	a.engine.SetDataPath(path)
}

func (a *Argo) Run() {
	a.engine.Run(optional.Option[engine.OnProcessDataCallback]{})
}

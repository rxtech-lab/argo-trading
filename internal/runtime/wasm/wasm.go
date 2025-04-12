package wasm

import (
	"context"
	"fmt"
	"os"

	timestamppb "github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// StrategyWasmRuntime is a runtime for a strategy that is written in WebAssembly.
// It is used for production purposes.
type StrategyWasmRuntime struct {
	strategy strategy.TradingStrategy
}

// NewStrategyWasmRuntime creates a new StrategyWasmRuntime with `wasmFilePath` as the strategy file.
func NewStrategyWasmRuntime(wasmFilePath string, strategyApi strategy.StrategyApi) (runtime.StrategyRuntime, error) {
	// check if file exists
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		return nil, fmt.Errorf("file does not exist: %s", wasmFilePath)
	}

	ctx := context.Background()
	p, err := strategy.NewTradingStrategyPlugin(ctx)
	if err != nil {
		return nil, err
	}
	plugin, err := p.Load(ctx, wasmFilePath, strategyApi)
	if err != nil {
		return nil, err
	}

	return &StrategyWasmRuntime{
		strategy: plugin,
	}, nil
}

func (s *StrategyWasmRuntime) Initialize(config string) error {
	_, err := s.strategy.Initialize(context.Background(), &strategy.InitializeRequest{
		Config: config,
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *StrategyWasmRuntime) ProcessData(data types.MarketData) error {
	_, err := s.strategy.ProcessData(context.Background(), &strategy.ProcessDataRequest{
		Data: &strategy.MarketData{
			Symbol: data.Symbol,
			Volume: data.Volume,
			High:   data.High,
			Low:    data.Low,
			Open:   data.Open,
			Close:  data.Close,
			Time:   timestamppb.New(data.Time),
		},
	})
	if err != nil {
		return err
	}
	return nil
}

func (s *StrategyWasmRuntime) Name() string {
	name, err := s.strategy.Name(context.Background(), &strategy.NameRequest{})
	if err != nil {
		return ""
	}
	return name.Name
}

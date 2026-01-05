package wasm

import (
	"context"
	"os"

	timestamppb "github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// StrategyWasmRuntime is a runtime for a strategy that is written in WebAssembly.
// It is used for production purposes.
type StrategyWasmRuntime struct {
	strategy     strategy.TradingStrategy
	wasmFilePath string
	wasmBytes    []byte
}

// NewStrategyWasmRuntime creates a new StrategyWasmRuntime with `wasmFilePath` as the strategy file.
func NewStrategyWasmRuntime(wasmFilePath string) (runtime.StrategyRuntime, error) {
	// check if file exists
	if _, err := os.Stat(wasmFilePath); os.IsNotExist(err) {
		return nil, errors.Newf(errors.ErrCodeDataNotFound, "file does not exist: %s", wasmFilePath)
	}

	return &StrategyWasmRuntime{
		strategy:     nil,
		wasmFilePath: wasmFilePath,
		wasmBytes:    nil,
	}, nil
}

func NewStrategyWasmRuntimeFromBytes(wasmBytes []byte) (runtime.StrategyRuntime, error) {
	return &StrategyWasmRuntime{
		strategy:     nil,
		wasmFilePath: "",
		wasmBytes:    wasmBytes,
	}, nil
}

// GetDescription implements runtime.StrategyRuntime.
func (s *StrategyWasmRuntime) GetDescription() (string, error) {
	if s.strategy == nil {
		return "", errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized, call InitializeApi first")
	}

	description, err := s.strategy.GetDescription(context.Background(), &strategy.GetDescriptionRequest{})
	if err != nil {
		return "", err
	}

	return description.Description, nil
}

func (s *StrategyWasmRuntime) Initialize(config string) error {
	if s.strategy == nil {
		return errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized, call InitializeApi first")
	}

	_, err := s.strategy.Initialize(context.Background(), &strategy.InitializeRequest{
		Config: config,
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *StrategyWasmRuntime) InitializeApi(api strategy.StrategyApi) error {
	ctx := context.Background()

	plugin, err := s.loadPlugin(ctx, api)
	if err != nil {
		return err
	}

	s.strategy = plugin

	return nil
}

func (s *StrategyWasmRuntime) ProcessData(data types.MarketData) error {
	if s.strategy == nil {
		return errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized, call InitializeApi first")
	}

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

func (s *StrategyWasmRuntime) GetConfigSchema() (string, error) {
	plugin, err := s.loadPlugin(context.Background(), nil)
	if err != nil {
		return "", err
	}

	if plugin == nil {
		return "", errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized")
	}

	schema, err := plugin.GetConfigSchema(context.Background(), &strategy.GetConfigSchemaRequest{})
	if err != nil {
		return "", err
	}

	return schema.Schema, nil
}

func (s *StrategyWasmRuntime) Name() string {
	if s.strategy == nil {
		return ""
	}

	name, err := s.strategy.Name(context.Background(), &strategy.NameRequest{})
	if err != nil {
		return ""
	}

	return name.Name
}

// engineVersionGetter is implemented by strategies that export engine version info.
type engineVersionGetter interface {
	GetEngineVersion(ctx context.Context) string
}

// GetRuntimeEngineVersion returns the engine version the strategy was compiled against.
// Returns empty string if the strategy doesn't export version info (older strategies).
func (s *StrategyWasmRuntime) GetRuntimeEngineVersion() (string, error) {
	if s.strategy == nil {
		return "", errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized, call InitializeApi first")
	}

	// Try to get version from strategy (compiled-in version)
	if getter, ok := s.strategy.(engineVersionGetter); ok {
		return getter.GetEngineVersion(context.Background()), nil
	}

	// Older strategies without version export
	return "", nil
}

// GetIdentifier returns the unique identifier for the strategy (e.g., "com.example.strategy").
// This method is required - strategies that don't implement it will fail to load.
func (s *StrategyWasmRuntime) GetIdentifier() (string, error) {
	if s.strategy == nil {
		return "", errors.New(errors.ErrCodeStrategyNotLoaded, "strategy is not initialized, call InitializeApi first")
	}

	identifier, err := s.strategy.GetIdentifier(context.Background(), &strategy.GetIdentifierRequest{})
	if err != nil {
		return "", errors.Wrap(errors.ErrCodeStrategyRuntimeError, "failed to get strategy identifier", err)
	}

	// Validate required field
	if identifier.Identifier == "" {
		return "", errors.New(errors.ErrCodeInvalidConfiguration, "strategy identifier is required but was empty")
	}

	return identifier.Identifier, nil
}

func (s *StrategyWasmRuntime) loadPlugin(ctx context.Context, api strategy.StrategyApi) (strategy.TradingStrategy, error) {
	p, err := strategy.NewTradingStrategyPlugin(ctx)
	if err != nil {
		return nil, err
	}

	var plugin strategy.TradingStrategy
	// check if both wasmFilePath and wasmBytes are set
	// return error if both are set
	if len(s.wasmFilePath) > 0 && len(s.wasmBytes) > 0 {
		return nil, errors.New(errors.ErrCodeInvalidConfiguration, "both wasmFilePath and wasmBytes are set")
	}

	// Check if at least one of wasmFilePath or wasmBytes is set
	if len(s.wasmFilePath) == 0 && len(s.wasmBytes) == 0 {
		return nil, errors.New(errors.ErrCodeInvalidConfiguration, "either wasmFilePath or wasmBytes must be set")
	}

	if len(s.wasmFilePath) > 0 {
		plugin, err = p.Load(ctx, s.wasmFilePath, api)
		if err != nil {
			return nil, err
		}
	}

	if len(s.wasmBytes) > 0 {
		plugin, err = p.LoadFromBytes(ctx, s.wasmBytes, api)
		if err != nil {
			return nil, err
		}
	}

	return plugin, nil
}

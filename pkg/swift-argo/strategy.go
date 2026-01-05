package swiftargo

import (
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
)

// StrategyMetadata contains the metadata of a trading strategy.
type StrategyMetadata struct {
	Identifier     string
	Name           string
	Schema         string
	Description    string
	RuntimeVersion string // The engine version the strategy was compiled against
}

type StrategyApi struct {
}

func NewStrategyApi() *StrategyApi {
	return &StrategyApi{}
}

// GetStrategyMetadata loads a WASM strategy from the given path and returns its metadata.
func (s *StrategyApi) GetStrategyMetadata(path string) (*StrategyMetadata, error) {
	runtime, err := wasm.NewStrategyWasmRuntime(path)
	if err != nil {
		return nil, err
	}

	if err != nil {
		return nil, err
	}

	runtime.InitializeApi(nil)

	schema, err := runtime.GetConfigSchema()
	if err != nil {
		return nil, err
	}

	description, err := runtime.GetDescription()
	if err != nil {
		return nil, err
	}

	runtimeVersion, err := runtime.GetRuntimeEngineVersion()
	if err != nil {
		return nil, err
	}

	identifier, err := runtime.GetIdentifier()
	if err != nil {
		return nil, err
	}

	return &StrategyMetadata{
		Name:           runtime.Name(),
		Schema:         schema,
		Description:    description,
		RuntimeVersion: runtimeVersion,
		Identifier:     identifier,
	}, nil
}

package swiftargo

import (
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
)

// StrategyMetadata contains the metadata of a trading strategy.
type StrategyMetadata struct {
	Name        string
	Schema      string
	Description string
}

// GetStrategyMetadata loads a WASM strategy from the given path and returns its metadata.
func GetStrategyMetadata(path string) (StrategyMetadata, error) {
	runtime, err := wasm.NewStrategyWasmRuntime(path)
	if err != nil {
		return StrategyMetadata{}, err
	}

	schema, err := runtime.GetConfigSchema()
	if err != nil {
		return StrategyMetadata{}, err
	}

	description, err := runtime.GetDescription()
	if err != nil {
		return StrategyMetadata{}, err
	}

	return StrategyMetadata{
		Name:        runtime.Name(),
		Schema:      schema,
		Description: description,
	}, nil
}

package indicator

import (
	"fmt"
	"sync"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// IndicatorRegistry manages all available indicators.
type IndicatorRegistry interface {
	RegisterIndicator(indicator Indicator) error
	GetIndicator(name types.IndicatorType) (Indicator, error)
	ListIndicators() []types.IndicatorType
	RemoveIndicator(name types.IndicatorType) error
}

// IndicatorRegistryV1 manages all available indicators.
type IndicatorRegistryV1 struct {
	indicators map[types.IndicatorType]Indicator
	mu         sync.RWMutex
}

// NewIndicatorRegistry creates a new indicator registry.
func NewIndicatorRegistry() IndicatorRegistry {
	return &IndicatorRegistryV1{
		indicators: make(map[types.IndicatorType]Indicator),
		mu:         sync.RWMutex{},
	}
}

// RegisterIndicator adds an indicator to the registry.
func (r *IndicatorRegistryV1) RegisterIndicator(indicator Indicator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := indicator.Name()
	if _, exists := r.indicators[name]; exists {
		return fmt.Errorf("RegisterIndicator: indicator with name %s already registered", name)
	}

	r.indicators[name] = indicator

	return nil
}

// GetIndicator retrieves an indicator by name.
func (r *IndicatorRegistryV1) GetIndicator(name types.IndicatorType) (Indicator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	indicator, exists := r.indicators[name]
	if !exists {
		return nil, fmt.Errorf("GetIndicator: indicator with name %s not found", name)
	}

	return indicator, nil
}

// ListIndicators returns a list of all registered indicator names.
func (r *IndicatorRegistryV1) ListIndicators() []types.IndicatorType {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]types.IndicatorType, 0, len(r.indicators))
	for name := range r.indicators {
		names = append(names, name)
	}

	return names
}

// RemoveIndicator removes an indicator from the registry.
func (r *IndicatorRegistryV1) RemoveIndicator(name types.IndicatorType) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.indicators[name]; !exists {
		return fmt.Errorf("RemoveIndicator: indicator with name %s not found", name)
	}

	delete(r.indicators, name)

	return nil
}

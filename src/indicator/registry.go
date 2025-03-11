package indicator

import (
	"fmt"
	"sync"
)

// IndicatorRegistry manages all available indicators
type IndicatorRegistry struct {
	indicators map[string]Indicator
	mu         sync.RWMutex
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry() *IndicatorRegistry {
	return &IndicatorRegistry{
		indicators: make(map[string]Indicator),
	}
}

// RegisterIndicator adds an indicator to the registry
func (r *IndicatorRegistry) RegisterIndicator(indicator Indicator) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := indicator.Name()
	if _, exists := r.indicators[name]; exists {
		return fmt.Errorf("indicator with name %s already registered", name)
	}

	r.indicators[name] = indicator
	return nil
}

// GetIndicator retrieves an indicator by name
func (r *IndicatorRegistry) GetIndicator(name string) (Indicator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	indicator, exists := r.indicators[name]
	if !exists {
		return nil, fmt.Errorf("indicator with name %s not found", name)
	}

	return indicator, nil
}

// ListIndicators returns a list of all registered indicator names
func (r *IndicatorRegistry) ListIndicators() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.indicators))
	for name := range r.indicators {
		names = append(names, name)
	}

	return names
}

// RemoveIndicator removes an indicator from the registry
func (r *IndicatorRegistry) RemoveIndicator(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.indicators[name]; !exists {
		return fmt.Errorf("indicator with name %s not found", name)
	}

	delete(r.indicators, name)
	return nil
}

// DefaultRegistry is the global indicator registry
var DefaultRegistry = NewIndicatorRegistry()

// RegisterDefaultIndicators registers all built-in indicators with the default registry
func RegisterDefaultIndicators() {
	// Register RSI with default period of 14
	DefaultRegistry.RegisterIndicator(NewRSI(14))

	// Register MACD with default periods of 12, 26, 9
	DefaultRegistry.RegisterIndicator(NewMACD(12, 26, 9))
}

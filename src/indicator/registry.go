package indicator

import (
	"fmt"
	"sync"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
)

// IndicatorRegistry manages all available indicators
type IndicatorRegistry struct {
	indicators map[types.Indicator]Indicator
	mu         sync.RWMutex
}

// NewIndicatorRegistry creates a new indicator registry
func NewIndicatorRegistry() *IndicatorRegistry {
	return &IndicatorRegistry{
		indicators: make(map[types.Indicator]Indicator),
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
func (r *IndicatorRegistry) GetIndicator(name types.Indicator) (Indicator, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	indicator, exists := r.indicators[name]
	if !exists {
		return nil, fmt.Errorf("indicator with name %s not found", name)
	}

	return indicator, nil
}

// ListIndicators returns a list of all registered indicator names
func (r *IndicatorRegistry) ListIndicators() []types.Indicator {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]types.Indicator, 0, len(r.indicators))
	for name := range r.indicators {
		names = append(names, name)
	}

	return names
}

// RemoveIndicator removes an indicator from the registry
func (r *IndicatorRegistry) RemoveIndicator(name types.Indicator) error {
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

func RegisterIndicators(startTime, endTime time.Time) {
	DefaultRegistry.RegisterIndicator(NewRSI(startTime, endTime, 14))
	DefaultRegistry.RegisterIndicator(NewMACD(startTime, endTime, 12, 26, 9))
	DefaultRegistry.RegisterIndicator(NewTrendStrength(startTime, endTime, 5, 20))
}

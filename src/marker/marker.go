package marker

import "github.com/sirily11/argo-trading-go/src/types"

// Marker is a marker that can be used to mark a point in time with a signal and a reason
type Marker interface {
	// Mark a point in time with a signal and a reason
	Mark(marketData types.MarketData, signal types.Signal, reason string) error
	// GetMarkers returns all the markers
	GetMarkers() ([]types.Mark, error)
}

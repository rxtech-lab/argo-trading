package marker

import (
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// Marker is a marker that can be used to mark a point in time with a signal and a reason.
type Marker interface {
	// Mark a point in time with a signal and a reason
	Mark(marketData types.MarketData, mark types.Mark) error
	// GetMarkers returns all the markers
	GetMarks() ([]types.Mark, error)
}

package writer

import (
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MarketDataWriter defines the interface for writing market data to a destination.
type MarketDataWriter interface {
	// Initialize sets up the writer, potentially creating tables or files.
	Initialize() error
	// Write persists a single market data point.
	Write(data types.MarketData) error
	// Finalize completes the writing process (e.g., commits transactions, exports files).
	Finalize() (outputPath string, err error)
	// Close releases any resources held by the writer.
	Close() error
	// GetOutputPath returns the configured output file path.
	GetOutputPath() string
}

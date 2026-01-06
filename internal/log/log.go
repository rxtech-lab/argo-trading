package log

import (
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// LogEntry represents a single log entry with market data context.
type LogEntry struct {
	// Timestamp is the market data time when this log was created.
	Timestamp time.Time
	// Symbol is the trading symbol associated with this log.
	Symbol string
	// Level is the severity level of the log.
	Level types.LogLevel
	// Message is the log message content.
	Message string
	// Fields contains optional structured key-value data.
	Fields map[string]string
}

// Log is the interface for storing strategy logs.
type Log interface {
	// Log stores a log entry.
	Log(entry LogEntry) error
	// GetLogs retrieves all stored log entries.
	GetLogs() ([]LogEntry, error)
}

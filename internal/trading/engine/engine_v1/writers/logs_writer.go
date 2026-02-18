package writers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/log"
)

// LogsWriter writes logs to a parquet file with real-time persistence.
type LogsWriter struct {
	db         *sql.DB
	outputPath string
	mu         sync.Mutex
}

// NewLogsWriter creates a new LogsWriter.
// outputPath is the full path to the parquet file.
func NewLogsWriter(outputPath string) *LogsWriter {
	return &LogsWriter{
		db:         nil,
		outputPath: outputPath,
		mu:         sync.Mutex{},
	}
}

// Initialize sets up the logs writer with DuckDB.
//
//nolint:dupl // Writers have similar initialization but different table schemas
func (w *LogsWriter) Initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create the data directory if it doesn't exist
	dir := filepath.Dir(w.outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open DuckDB connection (in-memory)
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	w.db = db

	// Create sequence for log IDs
	_, err = w.db.Exec(`CREATE SEQUENCE IF NOT EXISTS log_id_seq`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create the logs table (schema matches backtest backtest_log.go)
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS logs (
			id INTEGER PRIMARY KEY,
			timestamp TIMESTAMP,
			symbol TEXT,
			level TEXT,
			message TEXT,
			fields TEXT
		)
	`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create logs table: %w", err)
	}

	// Load existing data from parquet file if it exists
	if _, err := os.Stat(w.outputPath); err == nil {
		_, err = w.db.Exec(fmt.Sprintf(`
			INSERT INTO logs
			SELECT * FROM read_parquet('%s')
		`, w.outputPath))
		if err != nil {
			// If loading fails, start fresh
			_ = err
		}
	}

	return nil
}

// Write persists a log entry and exports to parquet.
func (w *LogsWriter) Write(entry log.LogEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	// Get the next ID from the sequence
	var nextID int

	err := w.db.QueryRow("SELECT nextval('log_id_seq')").Scan(&nextID)
	if err != nil {
		return fmt.Errorf("failed to get next ID from sequence: %w", err)
	}

	// Serialize fields to JSON
	var fieldsJSON string

	if entry.Fields != nil && len(entry.Fields) > 0 {
		fieldsBytes, err := json.Marshal(entry.Fields)
		if err == nil {
			fieldsJSON = string(fieldsBytes)
		}
	}

	_, err = w.db.Exec(`
		INSERT INTO logs (id, timestamp, symbol, level, message, fields)
		VALUES (?, ?, ?, ?, ?, ?)
	`, nextID, entry.Timestamp, entry.Symbol, string(entry.Level), entry.Message, fieldsJSON)
	if err != nil {
		return fmt.Errorf("failed to insert log: %w", err)
	}

	// Export to parquet after each write
	if err := w.exportToParquet(); err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Flush forces an export to parquet.
func (w *LogsWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	return w.exportToParquet()
}

// GetOutputPath returns the parquet file path.
func (w *LogsWriter) GetOutputPath() string {
	return w.outputPath
}

// Close releases database resources.
func (w *LogsWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db != nil {
		if err := w.db.Close(); err != nil {
			return fmt.Errorf("failed to close database: %w", err)
		}

		w.db = nil
	}

	return nil
}

// exportToParquet exports the current data to the parquet file.
//
//nolint:funcorder // helper method used by Write and Flush
func (w *LogsWriter) exportToParquet() error {
	_, err := w.db.Exec(fmt.Sprintf(`
		COPY (SELECT * FROM logs ORDER BY timestamp ASC)
		TO '%s' (FORMAT PARQUET)
	`, w.outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// GetLogCount returns the number of logs stored.
func (w *LogsWriter) GetLogCount() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var count int

	err := w.db.QueryRow("SELECT COUNT(*) FROM logs").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count logs: %w", err)
	}

	return count, nil
}

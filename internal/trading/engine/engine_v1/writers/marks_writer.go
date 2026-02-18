package writers

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// MarksWriter writes marks to a parquet file with real-time persistence.
type MarksWriter struct {
	db         *sql.DB
	outputPath string
	mu         sync.Mutex
}

// NewMarksWriter creates a new MarksWriter.
// outputPath is the full path to the parquet file.
func NewMarksWriter(outputPath string) *MarksWriter {
	return &MarksWriter{
		db:         nil,
		outputPath: outputPath,
		mu:         sync.Mutex{},
	}
}

// Initialize sets up the marks writer with DuckDB.
//
//nolint:dupl // Writers have similar initialization but different table schemas
func (w *MarksWriter) Initialize() error {
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

	// Create sequence for mark IDs
	_, err = w.db.Exec(`CREATE SEQUENCE IF NOT EXISTS mark_id_seq`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create the marks table (schema matches backtest backtest_marker.go)
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS marks (
			id INTEGER PRIMARY KEY,
			market_data_id TEXT,
			signal_type TEXT,
			signal_name TEXT,
			signal_time TIMESTAMP,
			signal_symbol TEXT,
			color TEXT,
			shape TEXT,
			level TEXT,
			title TEXT,
			message TEXT,
			category TEXT
		)
	`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create marks table: %w", err)
	}

	// Load existing data from parquet file if it exists
	if _, err := os.Stat(w.outputPath); err == nil {
		_, err = w.db.Exec(fmt.Sprintf(`
			INSERT INTO marks
			SELECT * FROM read_parquet('%s')
		`, w.outputPath))
		if err != nil {
			// If loading fails, start fresh
			_ = err
		}
	}

	return nil
}

// Write persists a mark and exports to parquet.
func (w *MarksWriter) Write(mark types.Mark) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	// Get the next ID from the sequence
	var nextID int

	err := w.db.QueryRow("SELECT nextval('mark_id_seq')").Scan(&nextID)
	if err != nil {
		return fmt.Errorf("failed to get next ID from sequence: %w", err)
	}

	// Extract signal fields if present
	var signalType, signalName sql.NullString

	var signalTime sql.NullTime

	var signalSymbol sql.NullString

	if mark.Signal.IsSome() {
		signal := mark.Signal.Unwrap()
		signalType = sql.NullString{String: string(signal.Type), Valid: true}
		signalName = sql.NullString{String: signal.Name, Valid: true}
		signalTime = sql.NullTime{Time: signal.Time, Valid: true}
		signalSymbol = sql.NullString{String: signal.Symbol, Valid: true}
	}

	_, err = w.db.Exec(`
		INSERT INTO marks (id, market_data_id, signal_type, signal_name, signal_time, signal_symbol,
			color, shape, level, title, message, category)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, nextID, mark.MarketDataId, signalType, signalName, signalTime, signalSymbol,
		string(mark.Color), string(mark.Shape), string(mark.Level),
		mark.Title, mark.Message, mark.Category)
	if err != nil {
		return fmt.Errorf("failed to insert mark: %w", err)
	}

	// Export to parquet after each write
	if err := w.exportToParquet(); err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Flush forces an export to parquet.
func (w *MarksWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	return w.exportToParquet()
}

// GetOutputPath returns the parquet file path.
func (w *MarksWriter) GetOutputPath() string {
	return w.outputPath
}

// Close releases database resources.
func (w *MarksWriter) Close() error {
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
func (w *MarksWriter) exportToParquet() error {
	_, err := w.db.Exec(fmt.Sprintf(`
		COPY (SELECT * FROM marks)
		TO '%s' (FORMAT PARQUET)
	`, w.outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// GetMarkCount returns the number of marks stored.
func (w *MarksWriter) GetMarkCount() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var count int

	err := w.db.QueryRow("SELECT COUNT(*) FROM marks").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count marks: %w", err)
	}

	return count, nil
}

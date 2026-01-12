package writer

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// StreamingDuckDBWriter implements MarketDataWriter for streaming data with append support.
// It writes finalized candles to a parquet file that persists across restarts.
// The file is named: stream_data_{provider}_{interval}.parquet.
type StreamingDuckDBWriter struct {
	db         *sql.DB
	outputPath string // Full path: {dataDir}/stream_data_{provider}_{interval}.parquet
	mu         sync.Mutex
}

// NewStreamingDuckDBWriter creates a new StreamingDuckDBWriter.
// dataDir: directory where parquet files will be stored
// providerName: name of the data provider (e.g., "binance")
// interval: candle interval (e.g., "1m", "5m", "1h")
func NewStreamingDuckDBWriter(dataDir, providerName, interval string) *StreamingDuckDBWriter {
	filename := fmt.Sprintf("stream_data_%s_%s.parquet", providerName, interval)
	outputPath := filepath.Join(dataDir, filename)

	return &StreamingDuckDBWriter{
		db:         nil,
		outputPath: outputPath,
		mu:         sync.Mutex{},
	}
}

// Initialize sets up the streaming writer.
// It opens a DuckDB connection and loads existing data from the parquet file if it exists.
func (w *StreamingDuckDBWriter) Initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Create the data directory if it doesn't exist
	dir := filepath.Dir(w.outputPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create data directory: %w", err)
	}

	// Open DuckDB connection (in-memory, we'll read/write parquet directly)
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	w.db = db

	// Create the market_data table with primary key for upsert support
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS market_data (
			id TEXT,
			time TIMESTAMP,
			symbol TEXT,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE,
			PRIMARY KEY (symbol, time)
		)
	`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create table: %w", err)
	}

	// Load existing data from parquet file if it exists
	if _, err := os.Stat(w.outputPath); err == nil {
		_, err = w.db.Exec(fmt.Sprintf(`
			INSERT INTO market_data
			SELECT * FROM read_parquet('%s')
			ON CONFLICT (symbol, time) DO NOTHING
		`, w.outputPath))
		if err != nil {
			// If loading fails, log but continue (file might be corrupted or empty)
			// We'll overwrite it with new data
			_ = err // Ignore error, start fresh
		}
	}

	return nil
}

// Write persists a single market data point and exports to parquet.
// This should only be called for finalized candles (IsFinal=true).
func (w *StreamingDuckDBWriter) Write(data types.MarketData) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	id := uuid.New().String()

	// Use INSERT OR REPLACE for upsert behavior (handles duplicates)
	_, err := w.db.Exec(`
		INSERT INTO market_data (id, time, symbol, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (symbol, time) DO UPDATE SET
			id = excluded.id,
			open = excluded.open,
			high = excluded.high,
			low = excluded.low,
			close = excluded.close,
			volume = excluded.volume
	`, id, data.Time, data.Symbol, data.Open, data.High, data.Low, data.Close, data.Volume)
	if err != nil {
		return fmt.Errorf("failed to insert data: %w", err)
	}

	// Export to parquet after each write
	if err := w.exportToParquet(); err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Flush forces an export to parquet.
func (w *StreamingDuckDBWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	return w.exportToParquet()
}

// Finalize exports the data and returns the output path.
// This is called when streaming stops.
func (w *StreamingDuckDBWriter) Finalize() (string, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return "", fmt.Errorf("writer not initialized")
	}

	if err := w.exportToParquet(); err != nil {
		return "", err
	}

	return w.outputPath, nil
}

// GetOutputPath returns the parquet file path.
func (w *StreamingDuckDBWriter) GetOutputPath() string {
	return w.outputPath
}

// Close releases database resources.
func (w *StreamingDuckDBWriter) Close() error {
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
func (w *StreamingDuckDBWriter) exportToParquet() error {
	_, err := w.db.Exec(fmt.Sprintf(`
		COPY (SELECT * FROM market_data ORDER BY time ASC)
		TO '%s' (FORMAT PARQUET)
	`, w.outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Verify StreamingDuckDBWriter implements MarketDataWriter interface.
var _ MarketDataWriter = (*StreamingDuckDBWriter)(nil)

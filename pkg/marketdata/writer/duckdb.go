package writer

import (
	"database/sql"
	"fmt"
	"log"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// DuckDBWriter implements the Writer interface for DuckDB.
type DuckDBWriter struct {
	db         *sql.DB
	tx         *sql.Tx
	stmt       *sql.Stmt
	outputPath string // Directory to write the output Parquet file
}

// NewDuckDBWriter creates a new DuckDBWriter.
// outputPath specifies the directory where the final Parquet file will be saved.
func NewDuckDBWriter(outputPath string) MarketDataWriter {
	return &DuckDBWriter{
		outputPath: outputPath,
	}
}

// Initialize sets up the DuckDB writer.
// It creates a temporary database file, establishes a connection,
// creates the necessary table, begins a transaction, and prepares the insert statement.
func (w *DuckDBWriter) Initialize() (err error) {
	// Open DuckDB database connection
	w.db, err = sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	// Create table if it doesn't exist
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS market_data (
			id TEXT,
			time TIMESTAMP,
			symbol TEXT,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	if err != nil {
		w.db.Close() // Ensure DB is closed on error during init

		return fmt.Errorf("failed to create table: %w", err)
	}

	// Begin transaction
	w.tx, err = w.db.Begin()
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Prepare insert statement
	w.stmt, err = w.tx.Prepare(`
		INSERT INTO market_data (id, time, symbol, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		w.tx.Rollback()
		w.db.Close()

		return fmt.Errorf("failed to prepare statement: %w", err)
	}

	return nil
}

// Write persists a single market data point using the prepared statement within the transaction.
func (w *DuckDBWriter) Write(data types.MarketData) error {
	if w.stmt == nil {
		return fmt.Errorf("writer not initialized or statement is nil")
	}

	id := uuid.New().String()

	_, err := w.stmt.Exec(
		id,
		data.Time,
		data.Symbol, // Assuming the ticker context is needed here
		data.Open,
		data.High,
		data.Low,
		data.Close,
		data.Volume,
	)
	if err != nil {
		// Don't rollback here, let Finalize handle it or allow further writes
		return fmt.Errorf("failed to insert data: %w", err)
	}

	return nil
}

// Finalize commits the transaction and exports the data to a Parquet file.
func (w *DuckDBWriter) Finalize() (outputPath string, err error) {
	if w.tx == nil {
		return "", fmt.Errorf("writer not initialized or transaction is nil")
	}

	// Commit transaction
	if err = w.tx.Commit(); err != nil {
		// Attempt rollback on commit failure, though it might also fail
		w.tx.Rollback()

		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	w.tx = nil // Transaction is finished

	// Export the data to Parquet format
	_, err = w.db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, w.outputPath))
	if err != nil {
		return "", fmt.Errorf("failed to export to Parquet: %w", err)
	}

	log.Printf("Successfully exported data to %s", w.outputPath)

	return w.outputPath, nil
}

// Close cleans up resources used by the writer, including closing the statement,
// closing the database connection, and removing the temporary database file.
func (w *DuckDBWriter) Close() error {
	var closeErrors []error

	if w.stmt != nil {
		if err := w.stmt.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to close statement: %w", err))
		}

		w.stmt = nil
	}

	// If transaction is still active (e.g., Finalize wasn't called or failed), rollback
	if w.tx != nil {
		if err := w.tx.Rollback(); err != nil {
			// Log rollback error but continue cleanup
			log.Printf("Warning: failed to rollback transaction during close: %v", err)
		}

		w.tx = nil
	}

	if w.db != nil {
		if err := w.db.Close(); err != nil {
			closeErrors = append(closeErrors, fmt.Errorf("failed to close db connection: %w", err))
		}

		w.db = nil
	}

	if len(closeErrors) > 0 {
		// Combine errors into a single error message
		errMsg := "errors occurred during close:"
		for _, e := range closeErrors {
			errMsg += fmt.Sprintf("\n- %v", e)
		}

		return fmt.Errorf("%s", errMsg)
	}

	return nil
}

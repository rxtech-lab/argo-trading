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

// OrdersWriter writes orders to a parquet file with real-time persistence.
type OrdersWriter struct {
	db         *sql.DB
	outputPath string
	mu         sync.Mutex
}

// NewOrdersWriter creates a new OrdersWriter.
// outputPath is the full path to the parquet file.
func NewOrdersWriter(outputPath string) *OrdersWriter {
	return &OrdersWriter{
		db:         nil,
		outputPath: outputPath,
		mu:         sync.Mutex{},
	}
}

// Initialize sets up the orders writer with DuckDB.
//
//nolint:dupl // Writers have similar initialization but different table schemas
func (w *OrdersWriter) Initialize() error {
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

	// Create the orders table with primary key for upsert support
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS orders (
			order_id TEXT PRIMARY KEY,
			symbol TEXT,
			side TEXT,
			order_type TEXT,
			quantity DOUBLE,
			price DOUBLE,
			timestamp TIMESTAMP,
			is_completed BOOLEAN,
			status TEXT,
			reason TEXT,
			reason_message TEXT,
			strategy_name TEXT,
			fee DOUBLE,
			position_type TEXT
		)
	`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create orders table: %w", err)
	}

	// Load existing data from parquet file if it exists
	if _, err := os.Stat(w.outputPath); err == nil {
		_, err = w.db.Exec(fmt.Sprintf(`
			INSERT INTO orders
			SELECT * FROM read_parquet('%s')
			ON CONFLICT (order_id) DO NOTHING
		`, w.outputPath))
		if err != nil {
			// If loading fails, start fresh
			_ = err
		}
	}

	return nil
}

// Write persists an order and exports to parquet.
func (w *OrdersWriter) Write(order types.Order) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	// Use INSERT OR REPLACE for upsert behavior (handles status updates)
	_, err := w.db.Exec(`
		INSERT INTO orders (order_id, symbol, side, order_type, quantity, price, timestamp,
			is_completed, status, reason, reason_message, strategy_name, fee, position_type)
		VALUES (?, ?, ?, 'MARKET', ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT (order_id) DO UPDATE SET
			is_completed = excluded.is_completed,
			status = excluded.status,
			fee = excluded.fee
	`, order.OrderID, order.Symbol, string(order.Side), order.Quantity, order.Price,
		order.Timestamp, order.IsCompleted, string(order.Status),
		order.Reason.Reason, order.Reason.Message, order.StrategyName,
		order.Fee, string(order.PositionType))
	if err != nil {
		return fmt.Errorf("failed to insert order: %w", err)
	}

	// Export to parquet after each write
	if err := w.exportToParquet(); err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Flush forces an export to parquet.
func (w *OrdersWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	return w.exportToParquet()
}

// GetOutputPath returns the parquet file path.
func (w *OrdersWriter) GetOutputPath() string {
	return w.outputPath
}

// Close releases database resources.
func (w *OrdersWriter) Close() error {
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
func (w *OrdersWriter) exportToParquet() error {
	_, err := w.db.Exec(fmt.Sprintf(`
		COPY (SELECT * FROM orders ORDER BY timestamp ASC)
		TO '%s' (FORMAT PARQUET)
	`, w.outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// GetOrderCount returns the number of orders stored.
func (w *OrdersWriter) GetOrderCount() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var count int

	err := w.db.QueryRow("SELECT COUNT(*) FROM orders").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count orders: %w", err)
	}

	return count, nil
}

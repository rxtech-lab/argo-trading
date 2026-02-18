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

// TradesWriter writes trades to a parquet file with real-time persistence.
type TradesWriter struct {
	db         *sql.DB
	outputPath string
	mu         sync.Mutex
}

// NewTradesWriter creates a new TradesWriter.
// outputPath is the full path to the parquet file.
func NewTradesWriter(outputPath string) *TradesWriter {
	return &TradesWriter{
		db:         nil,
		outputPath: outputPath,
		mu:         sync.Mutex{},
	}
}

// Initialize sets up the trades writer with DuckDB.
//
//nolint:dupl // Writers have similar initialization but different table schemas
func (w *TradesWriter) Initialize() error {
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

	// Create the trades table (schema matches backtest state.go)
	_, err = w.db.Exec(`
		CREATE TABLE IF NOT EXISTS trades (
			order_id TEXT,
			symbol TEXT,
			order_type TEXT,
			quantity DOUBLE,
			price DOUBLE,
			timestamp TIMESTAMP,
			is_completed BOOLEAN,
			reason TEXT,
			message TEXT,
			strategy_name TEXT,
			executed_at TIMESTAMP,
			executed_qty DOUBLE,
			executed_price DOUBLE,
			commission DOUBLE,
			pnl DOUBLE,
			position_type TEXT
		)
	`)
	if err != nil {
		w.db.Close()

		return fmt.Errorf("failed to create trades table: %w", err)
	}

	// Load existing data from parquet file if it exists
	if _, err := os.Stat(w.outputPath); err == nil {
		_, err = w.db.Exec(fmt.Sprintf(`
			INSERT INTO trades
			SELECT * FROM read_parquet('%s')
		`, w.outputPath))
		if err != nil {
			// If loading fails, start fresh
			_ = err
		}
	}

	return nil
}

// Write persists a trade and exports to parquet.
func (w *TradesWriter) Write(trade types.Trade) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	_, err := w.db.Exec(`
		INSERT INTO trades (order_id, symbol, order_type, quantity, price, timestamp,
			is_completed, reason, message, strategy_name,
			executed_at, executed_qty, executed_price, commission, pnl, position_type)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, trade.Order.OrderID, trade.Order.Symbol, string(trade.Order.Side),
		trade.Order.Quantity, trade.Order.Price, trade.Order.Timestamp,
		trade.Order.IsCompleted, trade.Order.Reason.Reason, trade.Order.Reason.Message,
		trade.Order.StrategyName,
		trade.ExecutedAt, trade.ExecutedQty, trade.ExecutedPrice,
		trade.Fee, trade.PnL, string(trade.Order.PositionType))
	if err != nil {
		return fmt.Errorf("failed to insert trade: %w", err)
	}

	// Export to parquet after each write
	if err := w.exportToParquet(); err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// Flush forces an export to parquet.
func (w *TradesWriter) Flush() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return fmt.Errorf("writer not initialized")
	}

	return w.exportToParquet()
}

// GetOutputPath returns the parquet file path.
func (w *TradesWriter) GetOutputPath() string {
	return w.outputPath
}

// Close releases database resources.
func (w *TradesWriter) Close() error {
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
func (w *TradesWriter) exportToParquet() error {
	_, err := w.db.Exec(fmt.Sprintf(`
		COPY (SELECT * FROM trades ORDER BY executed_at ASC)
		TO '%s' (FORMAT PARQUET)
	`, w.outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// GetTradeCount returns the number of trades stored.
func (w *TradesWriter) GetTradeCount() (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var count int

	err := w.db.QueryRow("SELECT COUNT(*) FROM trades").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count trades: %w", err)
	}

	return count, nil
}

// GetTotalPnL returns the sum of all trade PnL.
func (w *TradesWriter) GetTotalPnL() (float64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var totalPnL sql.NullFloat64

	err := w.db.QueryRow("SELECT SUM(pnl) FROM trades").Scan(&totalPnL)
	if err != nil {
		return 0, fmt.Errorf("failed to sum PnL: %w", err)
	}

	if !totalPnL.Valid {
		return 0, nil
	}

	return totalPnL.Float64, nil
}

// GetTotalFees returns the sum of all trade fees.
func (w *TradesWriter) GetTotalFees() (float64, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.db == nil {
		return 0, fmt.Errorf("writer not initialized")
	}

	var totalFees sql.NullFloat64

	err := w.db.QueryRow("SELECT SUM(commission) FROM trades").Scan(&totalFees)
	if err != nil {
		return 0, fmt.Errorf("failed to sum fees: %w", err)
	}

	if !totalFees.Valid {
		return 0, nil
	}

	return totalFees.Float64, nil
}

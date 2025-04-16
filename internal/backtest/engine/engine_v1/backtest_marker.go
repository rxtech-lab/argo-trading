package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// BacktestMarker implements the Marker interface for backtesting purposes.
// It records market data, signals, and reasons in a DuckDB database.
type BacktestMarker struct {
	db     *sql.DB
	logger *logger.Logger
	sq     squirrel.StatementBuilderType
}

// NewBacktestMarker creates a new instance of BacktestMarker.
func NewBacktestMarker(logger *logger.Logger) (*BacktestMarker, error) {
	// Create an in-memory DuckDB database
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		logger.Error("Failed to open database", zap.Error(err))

		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test connection to ensure database is properly initialized
	if err := db.Ping(); err != nil {
		logger.Error("Failed to connect to database", zap.Error(err))
		db.Close()

		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	marker := &BacktestMarker{
		logger: logger,
		db:     db,
		sq:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}

	// Initialize the database tables
	if err := marker.initialize(); err != nil {
		db.Close()

		return nil, err
	}

	return marker, nil
}

// Mark implements the Marker interface. It records a point in time with market data, signal, and reason.
func (m *BacktestMarker) Mark(marketData types.MarketData, signal types.Signal, reason string) error {
	// Check for nil fields
	if m == nil || m.db == nil {
		return fmt.Errorf("backtest marker or database is nil")
	}

	// Get the next ID from the sequence
	var nextID int

	err := m.db.QueryRow("SELECT nextval('mark_id_seq')").Scan(&nextID)
	if err != nil {
		return fmt.Errorf("failed to get next ID from sequence: %w", err)
	}

	// Insert the mark using Squirrel
	insertQuery := m.sq.
		Insert("marks").
		Columns(
			"id", "symbol", "time", "open", "high", "low", "close", "volume", "signal_type", "signal_name", "reason",
		).
		Values(
			nextID, marketData.Symbol, marketData.Time, marketData.Open, marketData.High,
			marketData.Low, marketData.Close, marketData.Volume, string(signal.Type), signal.Name, reason,
		).
		RunWith(m.db)

	_, err = insertQuery.Exec()
	if err != nil {
		return fmt.Errorf("failed to insert mark: %w", err)
	}

	return nil
}

// GetMarks implements the Marker interface. It returns all recorded marks.
func (m *BacktestMarker) GetMarks() ([]types.Mark, error) {
	// Check for nil fields
	if m == nil || m.db == nil {
		return nil, fmt.Errorf("backtest marker or database is nil")
	}

	// Query all marks using Squirrel
	selectQuery := m.sq.
		Select(
			"id", "symbol", "time", "open", "high", "low", "close",
			"volume", "signal_type", "signal_name", "reason",
		).
		From("marks").
		OrderBy("time ASC").
		RunWith(m.db)

	rows, err := selectQuery.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query marks: %w", err)
	}
	defer rows.Close()

	var marks []types.Mark

	for rows.Next() {
		var id int

		var marketData types.MarketData

		var signal types.Signal

		var signalTypeStr string

		var mark types.Mark

		err := rows.Scan(
			&id,
			&marketData.Symbol,
			&marketData.Time,
			&marketData.Open,
			&marketData.High,
			&marketData.Low,
			&marketData.Close,
			&marketData.Volume,
			&signalTypeStr,
			&signal.Name,
			&mark.Reason,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark: %w", err)
		}

		signal.Type = types.SignalType(signalTypeStr)
		signal.Time = marketData.Time
		signal.Symbol = marketData.Symbol

		mark.Signal = signal
		marks = append(marks, mark)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating marks: %w", err)
	}

	return marks, nil
}

// Write saves the marks to a Parquet file in the specified directory.
func (m *BacktestMarker) Write(path string) error {
	// Check for nil fields
	if m == nil || m.db == nil || m.logger == nil {
		return fmt.Errorf("backtest marker, database, or logger is nil")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Export marks to Parquet
	marksPath := filepath.Join(path, "marks.parquet")

	_, err := m.db.Exec(fmt.Sprintf(`COPY marks TO '%s' (FORMAT PARQUET)`, marksPath))
	if err != nil {
		return fmt.Errorf("failed to export marks to Parquet: %w", err)
	}

	m.logger.Info("Successfully exported marks to Parquet file",
		zap.String("marks", marksPath),
	)

	return nil
}

// Cleanup resets the database state.
func (m *BacktestMarker) Cleanup() error {
	// Check for nil db
	if m == nil || m.db == nil {
		return fmt.Errorf("backtest marker or database is nil")
	}

	// Use raw SQL for dropping table and sequence
	_, err := m.db.Exec(`
		DROP TABLE IF EXISTS marks;
		DROP SEQUENCE IF EXISTS mark_id_seq;
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup marks table: %w", err)
	}

	// Reinitialize
	return m.initialize()
}

// Close closes the database connection.
func (m *BacktestMarker) Close() error {
	if m == nil || m.db == nil {
		return nil
	}

	return m.db.Close()
}

// initialize creates the necessary tables for storing marks.
func (m *BacktestMarker) initialize() error {
	// Check for nil db
	if m == nil || m.db == nil {
		return fmt.Errorf("backtest marker or database is nil")
	}

	// Create sequence for mark IDs
	_, err := m.db.Exec(`CREATE SEQUENCE IF NOT EXISTS mark_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create marks table
	_, err = m.db.Exec(`
		CREATE TABLE IF NOT EXISTS marks (
			id INTEGER PRIMARY KEY,
			symbol TEXT,
			time TIMESTAMP,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE,
			signal_type TEXT,
			signal_name TEXT,
			reason TEXT
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create marks table: %w", err)
	}

	return nil
}

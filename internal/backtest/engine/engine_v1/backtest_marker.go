package engine

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// BacktestMarker implements the Marker interface for backtesting purposes.
// It records marks with signal information in a DuckDB database.
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

// Mark implements the Marker interface. It records a mark with the given parameters.
func (m *BacktestMarker) Mark(marketData types.MarketData, mark types.Mark) error {
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

	// Get signal values if present
	var signalType, signalName, signalSymbol sql.NullString

	var signalTime sql.NullTime

	if mark.Signal.IsSome() {
		signal, err := mark.Signal.Take()
		if err == nil {
			signalType = sql.NullString{String: string(signal.Type), Valid: true}
			signalName = sql.NullString{String: signal.Name, Valid: true}
			signalSymbol = sql.NullString{String: signal.Symbol, Valid: true}
			signalTime = sql.NullTime{Time: signal.Time, Valid: true}
		}
	}

	// Insert the mark using Squirrel
	insertQuery := m.sq.
		Insert("marks").
		Columns(
			"id", "market_data_id", "signal_type", "signal_name", "signal_time", "signal_symbol",
			"color", "shape", "level", "title", "message", "category",
		).
		Values(
			nextID, mark.MarketDataId, signalType, signalName, signalTime, signalSymbol,
			mark.Color, string(mark.Shape), string(mark.Level), mark.Title, mark.Message, mark.Category,
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
			"id", "market_data_id", "signal_type", "signal_name", "signal_time", "signal_symbol",
			"color", "shape", "level", "title", "message", "category",
		).
		From("marks").
		OrderBy("id ASC").
		RunWith(m.db)

	rows, err := selectQuery.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query marks: %w", err)
	}
	defer rows.Close()

	var marks []types.Mark

	for rows.Next() {
		var id int

		var marketDataId string

		var signalTypeStr sql.NullString

		var signalName sql.NullString

		var signalTime sql.NullTime

		var signalSymbol sql.NullString

		var color types.MarkColor

		var shapeStr string

		var levelStr string

		var title string

		var message string

		var category string

		err := rows.Scan(
			&id,
			&marketDataId,
			&signalTypeStr,
			&signalName,
			&signalTime,
			&signalSymbol,
			&color,
			&shapeStr,
			&levelStr,
			&title,
			&message,
			&category,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan mark: %w", err)
		}

		// Create mark
		mark := types.Mark{
			MarketDataId: marketDataId,
			Color:        color,
			Shape:        types.MarkShape(shapeStr),
			Level:        types.MarkLevel(levelStr),
			Title:        title,
			Message:      message,
			Category:     category,
			Signal:       optional.None[types.Signal](),
		}

		// Add signal if it exists
		if signalTime.Valid && signalSymbol.Valid && signalTypeStr.Valid && signalName.Valid {
			signal := types.Signal{
				Type:      types.SignalType(signalTypeStr.String),
				Name:      signalName.String,
				Time:      signalTime.Time,
				Symbol:    signalSymbol.String,
				Reason:    "",
				RawValue:  nil,
				Indicator: "",
			}
			mark.Signal = optional.Some(signal)
		} else {
			mark.Signal = optional.None[types.Signal]()
		}

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

	// Create marks table with new schema
	_, err = m.db.Exec(`
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
		return fmt.Errorf("failed to create marks table: %w", err)
	}

	return nil
}

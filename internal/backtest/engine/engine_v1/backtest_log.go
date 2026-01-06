package engine

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Masterminds/squirrel"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/log"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"go.uber.org/zap"
)

// BacktestLog implements the Log interface for backtesting purposes.
// It records strategy log entries in a DuckDB database.
type BacktestLog struct {
	db     *sql.DB
	logger *logger.Logger
	sq     squirrel.StatementBuilderType
}

// NewBacktestLog creates a new instance of BacktestLog.
func NewBacktestLog(logger *logger.Logger) (*BacktestLog, error) {
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

	logStorage := &BacktestLog{
		logger: logger,
		db:     db,
		sq:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Question),
	}

	// Initialize the database tables
	if err := logStorage.initialize(); err != nil {
		db.Close()

		return nil, err
	}

	return logStorage, nil
}

// Log implements the Log interface. It records a log entry.
func (l *BacktestLog) Log(entry log.LogEntry) error {
	// Check for nil fields
	if l == nil || l.db == nil {
		return fmt.Errorf("backtest log or database is nil")
	}

	// Get the next ID from the sequence
	var nextID int

	err := l.db.QueryRow("SELECT nextval('log_id_seq')").Scan(&nextID)
	if err != nil {
		return fmt.Errorf("failed to get next ID from sequence: %w", err)
	}

	// Serialize fields to JSON
	var fieldsJSON string

	if entry.Fields != nil && len(entry.Fields) > 0 {
		fieldsBytes, err := json.Marshal(entry.Fields)
		if err != nil {
			return fmt.Errorf("failed to marshal fields to JSON: %w", err)
		}

		fieldsJSON = string(fieldsBytes)
	}

	// Insert the log entry using Squirrel
	insertQuery := l.sq.
		Insert("logs").
		Columns("id", "timestamp", "symbol", "level", "message", "fields").
		Values(nextID, entry.Timestamp, entry.Symbol, string(entry.Level), entry.Message, fieldsJSON).
		RunWith(l.db)

	_, err = insertQuery.Exec()
	if err != nil {
		return fmt.Errorf("failed to insert log entry: %w", err)
	}

	return nil
}

// GetLogs implements the Log interface. It returns all recorded log entries.
func (l *BacktestLog) GetLogs() ([]log.LogEntry, error) {
	// Check for nil fields
	if l == nil || l.db == nil {
		return nil, fmt.Errorf("backtest log or database is nil")
	}

	// Query all logs using Squirrel
	selectQuery := l.sq.
		Select("id", "timestamp", "symbol", "level", "message", "fields").
		From("logs").
		OrderBy("id ASC").
		RunWith(l.db)

	rows, err := selectQuery.Query()
	if err != nil {
		return nil, fmt.Errorf("failed to query logs: %w", err)
	}
	defer rows.Close()

	var logs []log.LogEntry

	for rows.Next() {
		var id int

		var entry log.LogEntry

		var levelStr string

		var fieldsJSON sql.NullString

		err := rows.Scan(
			&id,
			&entry.Timestamp,
			&entry.Symbol,
			&levelStr,
			&entry.Message,
			&fieldsJSON,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan log entry: %w", err)
		}

		entry.Level = types.LogLevel(levelStr)

		// Deserialize fields from JSON
		if fieldsJSON.Valid && fieldsJSON.String != "" {
			var fields map[string]string
			if err := json.Unmarshal([]byte(fieldsJSON.String), &fields); err != nil {
				return nil, fmt.Errorf("failed to unmarshal fields from JSON: %w", err)
			}

			entry.Fields = fields
		}

		logs = append(logs, entry)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating logs: %w", err)
	}

	return logs, nil
}

// Write saves the logs to a Parquet file in the specified directory.
func (l *BacktestLog) Write(path string) error {
	// Check for nil fields
	if l == nil || l.db == nil || l.logger == nil {
		return fmt.Errorf("backtest log, database, or logger is nil")
	}

	// Create directory if it doesn't exist
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Export logs to Parquet
	logsPath := filepath.Join(path, "logs.parquet")

	_, err := l.db.Exec(fmt.Sprintf(`COPY logs TO '%s' (FORMAT PARQUET)`, logsPath))
	if err != nil {
		return fmt.Errorf("failed to export logs to Parquet: %w", err)
	}

	l.logger.Info("Successfully exported logs to Parquet file",
		zap.String("logs", logsPath),
	)

	return nil
}

// Cleanup resets the database state.
func (l *BacktestLog) Cleanup() error {
	// Check for nil db
	if l == nil || l.db == nil {
		return fmt.Errorf("backtest log or database is nil")
	}

	// Use raw SQL for dropping table and sequence
	_, err := l.db.Exec(`
		DROP TABLE IF EXISTS logs;
		DROP SEQUENCE IF EXISTS log_id_seq;
	`)
	if err != nil {
		return fmt.Errorf("failed to cleanup logs table: %w", err)
	}

	// Reinitialize
	return l.initialize()
}

// Close closes the database connection.
func (l *BacktestLog) Close() error {
	if l == nil || l.db == nil {
		return nil
	}

	return l.db.Close()
}

// initialize creates the necessary tables for storing logs.
func (l *BacktestLog) initialize() error {
	// Check for nil db
	if l == nil || l.db == nil {
		return fmt.Errorf("backtest log or database is nil")
	}

	// Create sequence for log IDs
	_, err := l.db.Exec(`CREATE SEQUENCE IF NOT EXISTS log_id_seq`)
	if err != nil {
		return fmt.Errorf("failed to create sequence: %w", err)
	}

	// Create logs table
	_, err = l.db.Exec(`
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
		return fmt.Errorf("failed to create logs table: %w", err)
	}

	return nil
}

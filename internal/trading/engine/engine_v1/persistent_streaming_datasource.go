package engine_v1

import (
	"database/sql"
	"fmt"
	"os"
	"strings"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
)

// PersistentStreamingDataSource implements datasource.DataSource for live streaming with file persistence.
// It reads finalized market data directly from parquet files via DuckDB.
// Unlike StreamingDataSource which uses in-memory cache, this queries the parquet file directly
// each time, ensuring data consistency and supporting SQL queries for indicator calculations.
type PersistentStreamingDataSource struct {
	db          *sql.DB
	parquetPath string
	interval    string
}

// NewPersistentStreamingDataSource creates a new PersistentStreamingDataSource.
// parquetPath: path to the parquet file containing market data
// interval: candle interval (extracted from filename, e.g., "1m", "5m")
func NewPersistentStreamingDataSource(parquetPath, interval string) *PersistentStreamingDataSource {
	return &PersistentStreamingDataSource{
		db:          nil,
		parquetPath: parquetPath,
		interval:    interval,
	}
}

// Initialize opens a DuckDB connection for querying the parquet file.
// The path parameter is ignored since the parquet path is set in the constructor.
func (p *PersistentStreamingDataSource) Initialize(_ string) error {
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}

	p.db = db

	return nil
}

// ReadAll implements datasource.DataSource.
// Reads all data from the parquet file in chronological order.
func (p *PersistentStreamingDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		if !p.hasData() {
			return // No data yet
		}

		query := fmt.Sprintf(`
			SELECT time, symbol, open, high, low, close, volume
			FROM read_parquet('%s')
		`, p.parquetPath)

		var conditions []string
		var params []interface{}
		paramCount := 0

		if start.IsSome() {
			paramCount++
			conditions = append(conditions, fmt.Sprintf("time >= $%d", paramCount))
			params = append(params, start.Unwrap())
		}

		if end.IsSome() {
			paramCount++
			conditions = append(conditions, fmt.Sprintf("time <= $%d", paramCount))
			params = append(params, end.Unwrap())
		}

		if len(conditions) > 0 {
			query += " WHERE " + strings.Join(conditions, " AND ")
		}

		query += " ORDER BY time ASC"

		var rows *sql.Rows
		var err error
		if len(params) > 0 {
			rows, err = p.db.Query(query, params...)
		} else {
			rows, err = p.db.Query(query)
		}

		if err != nil {
			yield(types.MarketData{}, fmt.Errorf("failed to query data: %w", err)) //nolint:exhaustruct

			return
		}
		defer rows.Close()

		for rows.Next() {
			var md types.MarketData
			err := rows.Scan(&md.Time, &md.Symbol, &md.Open, &md.High, &md.Low, &md.Close, &md.Volume)
			if err != nil {
				yield(types.MarketData{}, fmt.Errorf("failed to scan row: %w", err)) //nolint:exhaustruct

				return
			}

			if !yield(md, nil) {
				return
			}
		}
	}
}

// GetRange implements datasource.DataSource.
// Returns data from the parquet file within the specified time range.
func (p *PersistentStreamingDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[datasource.Interval]) ([]types.MarketData, error) {
	if !p.hasData() {
		return []types.MarketData{}, nil
	}

	// Interval aggregation not supported for now - return raw data
	if interval.IsSome() {
		// Could implement aggregation using time_bucket in the future
	}

	query := fmt.Sprintf(`
		SELECT time, symbol, open, high, low, close, volume
		FROM read_parquet('%s')
		WHERE time >= $1 AND time <= $2
		ORDER BY time ASC
	`, p.parquetPath)

	rows, err := p.db.Query(query, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	result := make([]types.MarketData, 0)
	for rows.Next() {
		var md types.MarketData
		err := rows.Scan(&md.Time, &md.Symbol, &md.Open, &md.High, &md.Low, &md.Close, &md.Volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result = append(result, md)
	}

	return result, nil
}

// GetPreviousNumberOfDataPoints implements datasource.DataSource.
// Returns the specified number of historical data points for a given symbol,
// ending at the specified time, in chronological order (oldest to newest).
// This is the primary method used by indicators for calculations.
func (p *PersistentStreamingDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	if !p.hasData() {
		return nil, errors.NewInsufficientDataErrorf(count, 0, symbol,
			"no data available for symbol %s: parquet file does not exist", symbol)
	}

	query := fmt.Sprintf(`
		SELECT time, symbol, open, high, low, close, volume
		FROM read_parquet('%s')
		WHERE symbol = $1 AND time <= $2
		ORDER BY time DESC
		LIMIT $3
	`, p.parquetPath)

	rows, err := p.db.Query(query, symbol, end, count)
	if err != nil {
		return nil, fmt.Errorf("failed to query data: %w", err)
	}
	defer rows.Close()

	result := make([]types.MarketData, 0, count)
	for rows.Next() {
		var md types.MarketData
		err := rows.Scan(&md.Time, &md.Symbol, &md.Open, &md.High, &md.Low, &md.Close, &md.Volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		result = append(result, md)
	}

	// Reverse to get chronological order (oldest to newest)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	// Check if we got fewer data points than requested
	if len(result) < count {
		return result, errors.NewInsufficientDataErrorf(count, len(result), symbol,
			"insufficient data points for symbol %s: requested %d, got %d", symbol, count, len(result))
	}

	return result, nil
}

// GetMarketData implements datasource.DataSource.
// Returns the market data for a specific symbol and timestamp.
func (p *PersistentStreamingDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	if !p.hasData() {
		return types.MarketData{}, fmt.Errorf("no data available for symbol %s at time %v", symbol, timestamp)
	}

	query := fmt.Sprintf(`
		SELECT time, symbol, open, high, low, close, volume
		FROM read_parquet('%s')
		WHERE symbol = $1 AND time = $2
		LIMIT 1
	`, p.parquetPath)

	var md types.MarketData
	err := p.db.QueryRow(query, symbol, timestamp).Scan(
		&md.Time, &md.Symbol, &md.Open, &md.High, &md.Low, &md.Close, &md.Volume)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.MarketData{}, fmt.Errorf("no data found for symbol %s at time %v", symbol, timestamp)
		}

		return types.MarketData{}, fmt.Errorf("failed to query data: %w", err)
	}

	return md, nil
}

// ReadLastData implements datasource.DataSource.
// Returns the most recent data for the specified symbol.
func (p *PersistentStreamingDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	if !p.hasData() {
		return types.MarketData{}, fmt.Errorf("no data available for symbol %s", symbol)
	}

	query := fmt.Sprintf(`
		SELECT time, symbol, open, high, low, close, volume
		FROM read_parquet('%s')
		WHERE symbol = $1
		ORDER BY time DESC
		LIMIT 1
	`, p.parquetPath)

	var md types.MarketData
	err := p.db.QueryRow(query, symbol).Scan(
		&md.Time, &md.Symbol, &md.Open, &md.High, &md.Low, &md.Close, &md.Volume)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.MarketData{}, fmt.Errorf("no data found for symbol %s", symbol)
		}

		return types.MarketData{}, fmt.Errorf("failed to query data: %w", err)
	}

	return md, nil
}

// ExecuteSQL implements datasource.DataSource.
// Executes a raw SQL query against the parquet file and returns the results.
func (p *PersistentStreamingDataSource) ExecuteSQL(query string, params ...interface{}) ([]datasource.SQLResult, error) {
	if !p.hasData() {
		return []datasource.SQLResult{}, nil
	}

	rows, err := p.db.Query(query, params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	result := make([]datasource.SQLResult, 0)
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}

		result = append(result, datasource.SQLResult{Values: rowMap})
	}

	return result, nil
}

// Count implements datasource.DataSource.
// Returns the total number of rows in the parquet file.
func (p *PersistentStreamingDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	if !p.hasData() {
		return 0, nil
	}

	query := fmt.Sprintf("SELECT COUNT(*) FROM read_parquet('%s')", p.parquetPath)
	var conditions []string
	var params []interface{}
	paramCount := 0

	if start.IsSome() {
		paramCount++
		conditions = append(conditions, fmt.Sprintf("time >= $%d", paramCount))
		params = append(params, start.Unwrap())
	}

	if end.IsSome() {
		paramCount++
		conditions = append(conditions, fmt.Sprintf("time <= $%d", paramCount))
		params = append(params, end.Unwrap())
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	var count int
	var err error
	if len(params) > 0 {
		err = p.db.QueryRow(query, params...).Scan(&count)
	} else {
		err = p.db.QueryRow(query).Scan(&count)
	}

	if err != nil {
		return 0, fmt.Errorf("failed to count rows: %w", err)
	}

	return count, nil
}

// Close implements datasource.DataSource.
// Closes the DuckDB connection.
func (p *PersistentStreamingDataSource) Close() error {
	if p.db != nil {
		return p.db.Close()
	}

	return nil
}

// GetAllSymbols implements datasource.DataSource.
// Returns all distinct symbols in the parquet file.
func (p *PersistentStreamingDataSource) GetAllSymbols() ([]string, error) {
	if !p.hasData() {
		return []string{}, nil
	}

	query := fmt.Sprintf("SELECT DISTINCT symbol FROM read_parquet('%s') ORDER BY symbol", p.parquetPath)
	rows, err := p.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to get symbols: %w", err)
	}
	defer rows.Close()

	var symbols []string
	for rows.Next() {
		var symbol string
		if err := rows.Scan(&symbol); err != nil {
			return nil, fmt.Errorf("failed to scan symbol: %w", err)
		}

		symbols = append(symbols, symbol)
	}

	return symbols, nil
}

// GetInterval returns the candle interval configured for this datasource.
func (p *PersistentStreamingDataSource) GetInterval() string {
	return p.interval
}

// GetParquetPath returns the parquet file path.
func (p *PersistentStreamingDataSource) GetParquetPath() string {
	return p.parquetPath
}

// hasData checks if the parquet file exists and has data.
func (p *PersistentStreamingDataSource) hasData() bool {
	if _, err := os.Stat(p.parquetPath); os.IsNotExist(err) {
		return false
	}

	return true
}

// Verify PersistentStreamingDataSource implements datasource.DataSource interface.
var _ datasource.DataSource = (*PersistentStreamingDataSource)(nil)

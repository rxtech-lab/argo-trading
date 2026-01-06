package datasource

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/Masterminds/squirrel"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"go.uber.org/zap"
)

type DuckDBDataSource struct {
	db     *sql.DB
	logger *logger.Logger
	sq     squirrel.StatementBuilderType
}

// NewDataSource creates a new DuckDB data source instance with the specified database path.
// The path parameter specifies the DuckDB database file location.
// This is distinct from Initialize() which loads market data into the database.
// Returns a DataSource interface and any error encountered during creation.
func NewDataSource(path string, logger *logger.Logger) (DataSource, error) {
	db, err := sql.Open("duckdb", path)
	if err != nil {
		return nil, err
	}

	// Set DuckDB-specific optimizations
	_, err = db.Exec(`
		SET memory_limit='8GB';
		SET threads=4;
		SET temp_directory='./temp';
	`)
	if err != nil {
		return nil, fmt.Errorf("failed to set DuckDB optimizations: %w", err)
	}

	return &DuckDBDataSource{
		db:     db,
		logger: logger,
		sq:     squirrel.StatementBuilder.PlaceholderFormat(squirrel.Dollar),
	}, nil
}

// Initialize implements DataSource.
func (d *DuckDBDataSource) Initialize(path string) error {
	d.logger.Debug("Initializing DuckDB data source", zap.String("path", path))

	// First drop the view if it exists
	_, err := d.db.Exec(`DROP VIEW IF EXISTS market_data;`)
	if err != nil {
		return fmt.Errorf("failed to drop existing view: %w", err)
	}

	// Create a view from the parquet file - using raw SQL as Squirrel doesn't support CREATE VIEW
	// Use SELECT * to include all columns from the parquet file (including indicator columns for testing)
	query := fmt.Sprintf(`
		CREATE VIEW market_data AS
		SELECT * FROM read_parquet('%s');
	`, path)

	_, err = d.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

// Count implements DataSource.
func (d *DuckDBDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	// Use raw SQL query for Count as it's simpler for this case
	var count int

	query := "SELECT COUNT(*) FROM market_data"

	var params []interface{}

	paramCount := 0

	if start.IsSome() {
		paramCount++
		if paramCount == 1 {
			query += " WHERE"
		} else {
			query += " AND"
		}

		query += fmt.Sprintf(" time >= $%d", paramCount)

		params = append(params, start.Unwrap())
	}

	if end.IsSome() {
		paramCount++
		if paramCount == 1 {
			query += " WHERE"
		} else {
			query += " AND"
		}

		query += fmt.Sprintf(" time <= $%d", paramCount)

		params = append(params, end.Unwrap())
	}

	var rows *sql.Row
	if len(params) > 0 {
		rows = d.db.QueryRow(query, params...)
	} else {
		rows = d.db.QueryRow(query)
	}

	err := rows.Scan(&count)
	if err != nil {
		return 0, err
	}

	return count, nil
}

// ReadAll implements DataSource with batch processing.
func (d *DuckDBDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	const batchSize = 1000 // Adjust this value based on your memory constraints

	return func(yield func(types.MarketData, error) bool) {
		d.logger.Debug("Reading all data from DuckDB with batch processing")

		// Build the base query using raw SQL for better compatibility
		query := `
			SELECT time, symbol, open, high, low, close, volume 
			FROM market_data
		`

		// Add time range conditions if provided
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

		// Use a prepared statement for better performance
		stmt, err := d.db.Prepare(query)
		if err != nil {
			yield(types.MarketData{Id: "", Symbol: "", Time: time.Time{}, Open: 0, High: 0, Low: 0, Close: 0, Volume: 0}, err)

			return
		}
		defer stmt.Close()

		var rows *sql.Rows
		if len(params) > 0 {
			rows, err = stmt.Query(params...)
		} else {
			rows, err = stmt.Query()
		}

		if err != nil {
			yield(types.MarketData{Id: "", Symbol: "", Time: time.Time{}, Open: 0, High: 0, Low: 0, Close: 0, Volume: 0}, err)

			return
		}

		defer rows.Close()

		// Process rows in batches
		batch := make([]types.MarketData, 0, batchSize)

		for rows.Next() {
			var (
				timestamp                      time.Time
				open, high, low, close, volume float64
				symbol                         string
			)

			err := rows.Scan(&timestamp, &symbol, &open, &high, &low, &close, &volume)
			if err != nil {
				yield(types.MarketData{Id: "", Symbol: "", Time: time.Time{}, Open: 0, High: 0, Low: 0, Close: 0, Volume: 0}, err)

				return
			}

			marketData := types.MarketData{
				Id:     "",
				Symbol: symbol,
				Time:   timestamp,
				Open:   open,
				High:   high,
				Low:    low,
				Close:  close,
				Volume: volume,
			}

			batch = append(batch, marketData)

			// Process batch when it reaches the batch size
			if len(batch) >= batchSize {
				for _, data := range batch {
					if !yield(data, nil) {
						return
					}
				}

				batch = batch[:0] // Reset slice while keeping capacity
			}
		}

		// Process remaining rows
		for _, data := range batch {
			if !yield(data, nil) {
				return
			}
		}
	}
}

// GetRange implements DataSource with optimized query.
func (d *DuckDBDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	// Process interval parameter
	var intervalMinutes optional.Option[int] = optional.None[int]()

	if interval.IsSome() {
		minutes, err := getIntervalMinutes(interval.Unwrap())
		if err != nil {
			return nil, err
		}

		intervalMinutes = optional.Some(minutes)
	}

	// Build the query
	query, args, err := d.buildGetRangeQuery(start, end, intervalMinutes)
	if err != nil {
		return nil, err
	}

	// Use prepared statement for better performance
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice with reasonable capacity
	result := make([]types.MarketData, 0, 1000)

	for rows.Next() {
		var (
			timestamp                      time.Time
			open, high, low, close, volume float64
			symbol                         string
		)

		err := rows.Scan(&timestamp, &symbol, &open, &high, &low, &close, &volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		marketData := types.MarketData{
			Id:     "",
			Symbol: symbol,
			Time:   timestamp,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}

		result = append(result, marketData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ReadRecordsFromStart reads number of records from the start time of the database.
func (d *DuckDBDataSource) ReadRecordsFromStart(start time.Time, number int, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	intervalMinutes, err := getIntervalMinutes(interval)
	if err != nil {
		return nil, err
	}

	// Optimized query using materialized CTE and window functions
	// Using raw SQL since Squirrel doesn't directly support window functions and complex CTEs
	query := fmt.Sprintf(`
		WITH time_buckets AS MATERIALIZED (
			SELECT 
				time_bucket(INTERVAL '%d minutes', time) as bucket_time,
				symbol,
				FIRST_VALUE(open) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time) as open,
				MAX(high) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as high,
				MIN(low) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as low,
				LAST_VALUE(close) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as close,
				SUM(volume) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as volume
			FROM market_data 
			WHERE time >= $1
		)
		SELECT DISTINCT
			bucket_time as time,
			symbol,
			open,
			high,
			low,
			close,
			volume
		FROM time_buckets
		ORDER BY bucket_time ASC
		LIMIT $2
	`, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes)

	// Use prepared statement for better performance
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(start, number)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice with reasonable capacity
	result := make([]types.MarketData, 0, number)

	for rows.Next() {
		var (
			timestamp                      time.Time
			open, high, low, close, volume float64
			symbol                         string
		)

		err := rows.Scan(&timestamp, &symbol, &open, &high, &low, &close, &volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		marketData := types.MarketData{
			Id:     "",
			Symbol: symbol,
			Time:   timestamp,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}

		result = append(result, marketData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ReadRecordsFromEnd reads number of records from the end time of the database.
func (d *DuckDBDataSource) ReadRecordsFromEnd(end time.Time, number int, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	intervalMinutes, err := getIntervalMinutes(interval)
	if err != nil {
		return nil, err
	}

	// Optimized query using materialized CTE and window functions
	// Using raw SQL since Squirrel doesn't directly support window functions and complex CTEs
	query := fmt.Sprintf(`
		WITH time_buckets AS MATERIALIZED (
			SELECT 
				time_bucket(INTERVAL '%d minutes', time) as bucket_time,
				symbol,
				FIRST_VALUE(open) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time) as open,
				MAX(high) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as high,
				MIN(low) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as low,
				LAST_VALUE(close) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as close,
				SUM(volume) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as volume
			FROM market_data 
			WHERE time <= $1
		)
		SELECT DISTINCT
			bucket_time as time,
			symbol,
			open,
			high,
			low,
			close,
			volume
		FROM time_buckets
		ORDER BY bucket_time DESC
		LIMIT $2
	`, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes)

	// Use prepared statement for better performance
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(end, number)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice with reasonable capacity
	result := make([]types.MarketData, 0, number)

	for rows.Next() {
		var (
			timestamp                      time.Time
			open, high, low, close, volume float64
			symbol                         string
		)

		err := rows.Scan(&timestamp, &symbol, &open, &high, &low, &close, &volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		marketData := types.MarketData{
			Id:     "",
			Symbol: symbol,
			Time:   timestamp,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}

		result = append(result, marketData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Reverse the slice since we got it in DESC order
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	return result, nil
}

// ExecuteSQL implements DataSource.
func (d *DuckDBDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	d.logger.Debug("Executing SQL query", zap.String("query", query))

	// Use prepared statement for better performance
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(params...)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Get column names
	columns, err := rows.Columns()
	if err != nil {
		return nil, fmt.Errorf("failed to get columns: %w", err)
	}

	// Pre-allocate slice with reasonable capacity
	result := make([]SQLResult, 0, 1000)

	for rows.Next() {
		// Create a slice of interface{} to hold the values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))

		for i := range values {
			valuePtrs[i] = &values[i]
		}

		// Scan the row into the values slice
		err := rows.Scan(valuePtrs...)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		// Create a map to store the column-value pairs
		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}

		result = append(result, SQLResult{Values: rowMap})
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return result, nil
}

// ReadLastData implements DataSource.
// Returns the most recent market data for the specified symbol.
func (d *DuckDBDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	d.logger.Debug("Reading last data for symbol", zap.String("symbol", symbol))

	// Using raw SQL for simplicity and reliability
	query := `
		SELECT time, symbol, open, high, low, close, volume 
		FROM market_data 
		WHERE symbol = $1
		ORDER BY time DESC
		LIMIT 1
	`

	stmt, err := d.db.Prepare(query)
	if err != nil {
		return types.MarketData{}, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	var (
		timestamp                      time.Time
		open, high, low, close, volume float64
		symbolResult                   string
	)

	err = stmt.QueryRow(symbol).Scan(&timestamp, &symbolResult, &open, &high, &low, &close, &volume)
	if err != nil {
		if err == sql.ErrNoRows {
			return types.MarketData{}, fmt.Errorf("no data found for symbol: %s", symbol)
		}

		return types.MarketData{}, fmt.Errorf("failed to scan row: %w", err)
	}

	return types.MarketData{
		Id:     "",
		Symbol: symbolResult,
		Time:   timestamp,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}, nil
}

// Close implements DataSource.
func (d *DuckDBDataSource) Close() error {
	if d.db != nil {
		return d.db.Close()
	}

	return nil
}

func (d *DuckDBDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	// Build query using squirrel
	query, args, err := d.sq.
		Select("time", "symbol", "open", "high", "low", "close", "volume").
		From("market_data").
		Where(squirrel.And{
			squirrel.Eq{"symbol": symbol},
			squirrel.Eq{"time": timestamp},
		}).
		ToSql()

	if err != nil {
		return types.MarketData{}, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	var (
		timeResult                     time.Time
		symbolResult                   string
		open, high, low, close, volume float64
	)

	err = d.db.QueryRow(query, args...).Scan(
		&timeResult, &symbolResult, &open, &high, &low, &close, &volume)

	if err != nil {
		if err == sql.ErrNoRows {
			return types.MarketData{}, fmt.Errorf("no market data found for symbol %s at time %v", symbol, timestamp)
		}

		return types.MarketData{}, fmt.Errorf("failed to get market data: %w", err)
	}

	return types.MarketData{
		Id:     "",
		Symbol: symbolResult,
		Time:   timeResult,
		Open:   open,
		High:   high,
		Low:    low,
		Close:  close,
		Volume: volume,
	}, nil
}

// GetPreviousNumberOfDataPoints implements DataSource.
func (d *DuckDBDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	d.logger.Debug("Getting previous data points",
		zap.Time("end", end),
		zap.String("symbol", symbol),
		zap.Int("count", count))

	// Build query using squirrel
	query, args, err := d.sq.
		Select("time", "symbol", "open", "high", "low", "close", "volume").
		From("market_data").
		Where(squirrel.And{
			squirrel.Eq{"symbol": symbol},
			squirrel.LtOrEq{"time": end},
		}).
		OrderBy("time DESC").
		Limit(uint64(count)).
		ToSql()

	if err != nil {
		return nil, fmt.Errorf("failed to build query: %w", err)
	}

	// Execute the query
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query market data: %w", err)
	}
	defer rows.Close()

	// Pre-allocate slice with reasonable capacity
	result := make([]types.MarketData, 0, count)

	for rows.Next() {
		var (
			timestamp                      time.Time
			symbolResult                   string
			open, high, low, close, volume float64
		)

		err := rows.Scan(&timestamp, &symbolResult, &open, &high, &low, &close, &volume)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		marketData := types.MarketData{
			Id:     "",
			Symbol: symbolResult,
			Time:   timestamp,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  close,
			Volume: volume,
		}

		result = append(result, marketData)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	// Reverse the slice to get chronological order (oldest to newest)
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	// Check if we got fewer data points than requested
	if len(result) < count {
		return result, errors.NewInsufficientDataErrorf(count, len(result), symbol, "insufficient data points for symbol %s: requested %d, got %d", symbol, count, len(result))
	}

	return result, nil
}

// GetAllSymbols returns all distinct symbols from the market data.
func (d *DuckDBDataSource) GetAllSymbols() ([]string, error) {
	rows, err := d.db.Query("SELECT DISTINCT symbol FROM market_data ORDER BY symbol")
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

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating symbols: %w", err)
	}

	return symbols, nil
}

// buildGetRangeQuery constructs the SQL query for GetRange method.
func (d *DuckDBDataSource) buildGetRangeQuery(start time.Time, end time.Time, intervalMinutes optional.Option[int]) (string, []interface{}, error) {
	// If no interval is specified, use a simple query with squirrel
	if !intervalMinutes.IsSome() {
		query, args, err := d.sq.
			Select("time", "symbol", "open", "high", "low", "close", "volume").
			From("market_data").
			Where(squirrel.And{
				squirrel.GtOrEq{"time": start},
				squirrel.LtOrEq{"time": end},
			}).
			OrderBy("time ASC").
			ToSql()
		if err != nil {
			return "", nil, fmt.Errorf("failed to build query: %w", err)
		}

		return query, args, nil
	}

	// For interval case, use raw SQL with window functions
	minutes := intervalMinutes.Unwrap()
	query := fmt.Sprintf(`
		WITH time_buckets AS MATERIALIZED (
			SELECT 
				time_bucket(INTERVAL '%d minutes', time) as bucket_time,
				symbol,
				FIRST_VALUE(open) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time) as open,
				MAX(high) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as high,
				MIN(low) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as low,
				LAST_VALUE(close) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol ORDER BY time ROWS BETWEEN UNBOUNDED PRECEDING AND UNBOUNDED FOLLOWING) as close,
				SUM(volume) OVER (PARTITION BY time_bucket(INTERVAL '%d minutes', time), symbol) as volume
			FROM market_data 
			WHERE time >= $1 AND time <= $2
		)
		SELECT DISTINCT
			bucket_time as time,
			symbol,
			open,
			high,
			low,
			close,
			volume
		FROM time_buckets
		ORDER BY bucket_time ASC
	`, minutes, minutes, minutes, minutes, minutes, minutes)

	return query, []interface{}{start, end}, nil
}

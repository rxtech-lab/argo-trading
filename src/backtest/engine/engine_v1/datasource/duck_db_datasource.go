package datasource

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"go.uber.org/zap"
)

type DuckDBDataSource struct {
	db     *sql.DB
	logger *logger.Logger
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

	return &DuckDBDataSource{db: db, logger: logger}, nil
}

// Initialize implements DataSource.
func (d *DuckDBDataSource) Initialize(path string) error {
	d.logger.Debug("Initializing DuckDB data source", zap.String("path", path))

	// Create a view from the parquet file
	query := fmt.Sprintf(`
		CREATE VIEW market_data AS 
		SELECT * FROM read_parquet('%s');
	`, path)

	_, err := d.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

// Count implements DataSource.
func (d *DuckDBDataSource) Count() (int, error) {
	var count int
	rows := d.db.QueryRow("SELECT COUNT(*) FROM market_data")
	err := rows.Scan(&count)
	if err != nil {
		return 0, err
	}
	return count, nil
}

// ReadAll implements DataSource with batch processing.
func (d *DuckDBDataSource) ReadAll() func(yield func(types.MarketData, error) bool) {
	const batchSize = 1000 // Adjust this value based on your memory constraints

	return func(yield func(types.MarketData, error) bool) {
		d.logger.Debug("Reading all data from DuckDB with batch processing")

		// Use a prepared statement for better performance
		stmt, err := d.db.Prepare(`
			SELECT time, symbol, open, high, low, close, volume 
			FROM market_data 
			ORDER BY time ASC
		`)
		if err != nil {
			yield(types.MarketData{}, err)
			return
		}
		defer stmt.Close()

		rows, err := stmt.Query()
		if err != nil {
			yield(types.MarketData{}, err)
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
				yield(types.MarketData{}, err)
				return
			}

			marketData := types.MarketData{
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

// ReadRange implements DataSource with optimized query.
func (d *DuckDBDataSource) ReadRange(start time.Time, end time.Time, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	var intervalMinutes int
	switch interval {
	case Interval1m:
		intervalMinutes = 1
	case Interval5m:
		intervalMinutes = 5
	case Interval15m:
		intervalMinutes = 15
	case Interval30m:
		intervalMinutes = 30
	case Interval1h:
		intervalMinutes = 60
	case Interval4h:
		intervalMinutes = 240
	case Interval6h:
		intervalMinutes = 360
	case Interval8h:
		intervalMinutes = 480
	case Interval12h:
		intervalMinutes = 720
	case Interval1d:
		intervalMinutes = 1440
	case Interval1w:
		intervalMinutes = 10080
	default:
		return nil, fmt.Errorf("unsupported interval: %s", interval)
	}

	// Optimized query using materialized CTE and window functions
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
			WHERE time >= ? AND time <= ?
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
	`, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes)

	// Use prepared statement for better performance
	stmt, err := d.db.Prepare(query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare query: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.Query(start, end)
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

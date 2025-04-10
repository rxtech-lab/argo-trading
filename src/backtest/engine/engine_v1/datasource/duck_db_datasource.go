package datasource

import (
	"fmt"
	"time"

	"github.com/alifiroozi80/duckdb"
	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

// MarketDataModel represents the market data in the database
type MarketDataModel struct {
	Time   time.Time `gorm:"column:time;primaryKey"`
	Symbol string    `gorm:"column:symbol;primaryKey"`
	Open   float64   `gorm:"column:open"`
	High   float64   `gorm:"column:high"`
	Low    float64   `gorm:"column:low"`
	Close  float64   `gorm:"column:close"`
	Volume float64   `gorm:"column:volume"`
}

// TableName sets the table name for MarketDataModel
func (MarketDataModel) TableName() string {
	return "market_data"
}

// ToMarketData converts a MarketDataModel to types.MarketData
func (m MarketDataModel) ToMarketData() types.MarketData {
	return types.MarketData{
		Symbol: m.Symbol,
		Time:   m.Time,
		Open:   m.Open,
		High:   m.High,
		Low:    m.Low,
		Close:  m.Close,
		Volume: m.Volume,
	}
}

type DuckDBDataSource struct {
	db     *gorm.DB
	logger *logger.Logger
}

// NewDataSource creates a new DuckDB data source instance with the specified database path.
// The path parameter specifies the DuckDB database file location.
// This is distinct from Initialize() which loads market data into the database.
// Returns a DataSource interface and any error encountered during creation.
func NewDataSource(path string, logger *logger.Logger) (DataSource, error) {
	// Configure GORM with DuckDB
	db, err := gorm.Open(duckdb.Open(path), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	// Set DuckDB-specific optimizations
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}

	_, err = sqlDB.Exec(`
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

	// Create a view from the parquet file using raw SQL since it's a DuckDB-specific feature
	sqlDB, err := d.db.DB()
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		CREATE VIEW market_data AS 
		SELECT * FROM read_parquet('%s');
	`, path)

	_, err = sqlDB.Exec(query)
	if err != nil {
		return err
	}
	return nil
}

// Count implements DataSource.
func (d *DuckDBDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	var count int64
	query := d.db.Model(&MarketDataModel{})

	if start.IsSome() {
		query = query.Where("time >= ?", start.Unwrap())
	}

	if end.IsSome() {
		query = query.Where("time <= ?", end.Unwrap())
	}

	err := query.Count(&count).Error
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// ReadAll implements DataSource with batch processing.
func (d *DuckDBDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	const batchSize = 1000 // Adjust this value based on your memory constraints

	return func(yield func(types.MarketData, error) bool) {
		d.logger.Debug("Reading all data from DuckDB with batch processing")

		// Build the base query
		query := d.db.Model(&MarketDataModel{}).Order("time ASC")

		// Add time range conditions if provided
		if start.IsSome() {
			query = query.Where("time >= ?", start.Unwrap())
		}

		if end.IsSome() {
			query = query.Where("time <= ?", end.Unwrap())
		}

		// Initialize variables for paging
		offset := 0
		hasMore := true

		for hasMore {
			var marketDataModels []MarketDataModel
			result := query.Limit(batchSize).Offset(offset).Find(&marketDataModels)
			if result.Error != nil {
				yield(types.MarketData{}, result.Error)
				return
			}

			// If fewer records than batch size were returned, this is the last batch
			if len(marketDataModels) < batchSize {
				hasMore = false
			}

			// Process this batch
			for _, model := range marketDataModels {
				if !yield(model.ToMarketData(), nil) {
					return
				}
			}

			// Prepare for next batch
			offset += batchSize
		}
	}
}

// GetRange implements DataSource with optimized query.
func (d *DuckDBDataSource) GetRange(start time.Time, end time.Time, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	intervalMinutes, err := getIntervalMinutes(interval)
	if err != nil {
		return nil, err
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return nil, err
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

	// Execute raw SQL since DuckDB has specific functions that aren't part of standard SQL
	rows, err := sqlDB.Query(query, start, end)
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

// ReadRecordsFromStart reads number of records from the start time of the database
func (d *DuckDBDataSource) ReadRecordsFromStart(start time.Time, number int, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	intervalMinutes, err := getIntervalMinutes(interval)
	if err != nil {
		return nil, err
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return nil, err
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
			WHERE time >= ?
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
		LIMIT ?
	`, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes)

	// Execute raw SQL
	rows, err := sqlDB.Query(query, start, number)
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

// ReadRecordsFromEnd reads number of records from the end time of the database
func (d *DuckDBDataSource) ReadRecordsFromEnd(end time.Time, number int, interval Interval) ([]types.MarketData, error) {
	// Convert interval to minutes for aggregation
	intervalMinutes, err := getIntervalMinutes(interval)
	if err != nil {
		return nil, err
	}

	sqlDB, err := d.db.DB()
	if err != nil {
		return nil, err
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
			WHERE time <= ?
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
		LIMIT ?
	`, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes, intervalMinutes)

	// Execute raw SQL
	rows, err := sqlDB.Query(query, end, number)
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

	sqlDB, err := d.db.DB()
	if err != nil {
		return nil, err
	}

	// Execute raw SQL query
	rows, err := sqlDB.Query(query, params...)
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

	var marketData MarketDataModel
	result := d.db.Model(&MarketDataModel{}).
		Where("symbol = ?", symbol).
		Order("time DESC").
		Limit(1).
		First(&marketData)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return types.MarketData{}, fmt.Errorf("no data found for symbol: %s", symbol)
		}
		return types.MarketData{}, fmt.Errorf("failed to query market data: %w", result.Error)
	}

	return marketData.ToMarketData(), nil
}

// Close implements DataSource.
func (d *DuckDBDataSource) Close() error {
	if d.db != nil {
		sqlDB, err := d.db.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

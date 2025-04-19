package clients

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/schollz/progressbar/v3"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

func init() {
	// Register DuckDB driver
	// This is not needed because the import with _ already registers the driver
	// but we're leaving this comment for clarity
}

type PolygonClient struct {
	client *polygon.Client
}

func NewPolygonClient() *PolygonClient {
	apiKey := os.Getenv("POLYGON_API_KEY")
	if apiKey == "" {
		log.Fatal("POLYGON_API_KEY is not set")
	}
	client := polygon.New(apiKey)
	return &PolygonClient{
		client: client,
	}
}

func (c *PolygonClient) getParquetFileName(ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan) string {
	return fmt.Sprintf("%s_%s_%s_%d_%s.parquet", ticker, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), multiplier, timespan)
}

func (c *PolygonClient) Download(ticker string, toPath string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan) (path string, err error) {
	// check if toPath folder exists
	dir := filepath.Dir(toPath)
	id := uuid.New().String()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// create the folder
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return "", err
		}

		log.Printf("Created folder %s\n", dir)
		return "", nil
	}

	// We'll use DuckDB to both collect data and write to Parquet
	tempDBName := filepath.Join(dir, "temp.duckdb")
	outputFileName := c.getParquetFileName(ticker, startDate, endDate, multiplier, timespan)
	outputFilePath := filepath.Join(dir, outputFileName)
	log.Printf("Downloading data to %s\n", outputFilePath)

	// Open DuckDB database connection
	db, err := sql.Open("duckdb", tempDBName)
	if err != nil {
		return "", fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer func() {
		db.Close()
		// Clean up temporary database file
		os.Remove(tempDBName)
	}()

	// Create table if it doesn't exist
	_, err = db.Exec(`
		CREATE TABLE IF NOT EXISTS market_data (
			id TEXT PRIMARY KEY,
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
		return "", fmt.Errorf("failed to create table: %w", err)
	}

	// Calculate the number of days to download
	totalDays := int(endDate.Sub(startDate).Hours() / 24)

	// Create a new progress bar
	bar := progressbar.New(totalDays)

	// Begin transaction
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Prepare insert statement
	stmt, err := tx.Prepare(`
		INSERT INTO market_data (id, time, symbol, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		tx.Rollback()
		return "", fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	// Download the data from date to date
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		params := models.ListAggsParams{
			Ticker:     ticker,
			From:       models.Millis(date),
			To:         models.Millis(date.Add(24 * time.Hour).Add(-1 * time.Second)),
			Multiplier: multiplier,
			Timespan:   timespan,
		}

		iter := c.client.ListAggs(context.Background(), &params)

		if iter.Err() != nil {
			tx.Rollback()
			return "", iter.Err()
		}

		for iter.Next() {
			agg := iter.Item()
			// Insert directly into DuckDB
			_, err := stmt.Exec(
				id,
				time.Time(agg.Timestamp),
				ticker,
				agg.Open,
				agg.High,
				agg.Low,
				agg.Close,
				agg.Volume,
			)
			if err != nil {
				tx.Rollback()
				return "", fmt.Errorf("failed to insert data: %w", err)
			}
		}

		// Increment the progress bar
		bar.Add(1)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Export the data to Parquet format
	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, outputFilePath))
	if err != nil {
		return "", fmt.Errorf("failed to export to Parquet: %w", err)
	}

	return outputFilePath, nil
}

// QueryData opens a connection to a DuckDB database and executes the provided SQL query
// Can read from Parquet files using DuckDB's Parquet support
func (c *PolygonClient) QueryData(dbPath string, query string) ([]types.MarketData, error) {
	// Open DuckDB database connection
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// If the path ends with .parquet, we need to create a view for it
	if filepath.Ext(dbPath) == ".parquet" {
		_, err = db.Exec(fmt.Sprintf(`CREATE VIEW market_data AS SELECT * FROM read_parquet('%s')`, dbPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create view from Parquet: %w", err)
		}
	}

	// Execute the query
	rows, err := db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to execute query: %w", err)
	}
	defer rows.Close()

	// Parse the results
	var results []types.MarketData
	for rows.Next() {
		var data types.MarketData
		if err := rows.Scan(&data.Time, &data.Open, &data.High, &data.Low, &data.Close, &data.Volume); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, data)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return results, nil
}

// GetDatabaseStats returns statistics about the database including row count, date range, etc.
func (c *PolygonClient) GetDatabaseStats(dbPath string) (map[string]interface{}, error) {
	// Open DuckDB database connection
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return nil, fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// If the path ends with .parquet, we need to create a view for it
	if filepath.Ext(dbPath) == ".parquet" {
		_, err = db.Exec(fmt.Sprintf(`CREATE VIEW market_data AS SELECT * FROM read_parquet('%s')`, dbPath))
		if err != nil {
			return nil, fmt.Errorf("failed to create view from Parquet: %w", err)
		}
	}

	// Create stats map
	stats := make(map[string]interface{})

	// Get row count
	var count int64
	err = db.QueryRow("SELECT COUNT(*) FROM market_data").Scan(&count)
	if err != nil {
		return nil, fmt.Errorf("failed to get row count: %w", err)
	}
	stats["total_rows"] = count

	// Get date range
	var minDate, maxDate time.Time
	err = db.QueryRow("SELECT MIN(time), MAX(time) FROM market_data").Scan(&minDate, &maxDate)
	if err != nil {
		return nil, fmt.Errorf("failed to get date range: %w", err)
	}
	stats["start_date"] = minDate
	stats["end_date"] = maxDate
	stats["days_covered"] = int(maxDate.Sub(minDate).Hours() / 24)

	return stats, nil
}

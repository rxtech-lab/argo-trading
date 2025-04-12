package main

import (
	"fmt"
	"log"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/cmd/download/clients"
)

func main() {
	client := clients.NewPolygonClient()

	// Set up ticker and date range
	ticker := "AAPL"
	startDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2025, 3, 20, 0, 0, 0, 0, time.UTC)

	// Download data to DuckDB
	dbPath, err := client.Download(ticker, "data/market_data.duckdb", startDate, endDate, 1, models.Minute)
	if err != nil {
		log.Fatalf("Failed to download data: %v", err)
	}

	fmt.Printf("Downloaded data to %s\n", dbPath)

	// Get database statistics
	stats, err := client.GetDatabaseStats(dbPath)
	if err != nil {
		log.Fatalf("Failed to get database stats: %v", err)
	}

	fmt.Println("Database statistics:")
	fmt.Printf("- Total rows: %d\n", stats["total_rows"])
	fmt.Printf("- Date range: %s to %s\n", stats["start_date"], stats["end_date"])
	fmt.Printf("- Days covered: %d\n", stats["days_covered"])
}

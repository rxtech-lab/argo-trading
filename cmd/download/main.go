package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/cmd/download/clients"
)

func main() {
	// Set default end date to today
	currentDate := time.Now().Format("2006-01-02")

	// Define command-line flags
	tickerFlag := flag.String("ticker", "", "Stock ticker symbol (required)")
	startDateFlag := flag.String("start", "", "Start date in YYYY-MM-DD format (required)")
	endDateFlag := flag.String("end", currentDate, "End date in YYYY-MM-DD format (default: today)")
	dbPathFlag := flag.String("db", "data/market_data.duckdb", "Path to DuckDB database file")

	// Parse command-line flags
	flag.Parse()

	// Validate required parameters
	if *tickerFlag == "" {
		fmt.Println("Error: -ticker flag is required")
		flag.Usage()
		os.Exit(1)
	}

	if *startDateFlag == "" {
		fmt.Println("Error: -start flag is required")
		flag.Usage()
		os.Exit(1)
	}

	// Parse dates
	startDate, err := time.Parse("2006-01-02", *startDateFlag)
	if err != nil {
		log.Fatalf("Invalid start date format: %v", err)
	}

	endDate, err := time.Parse("2006-01-02", *endDateFlag)
	if err != nil {
		log.Fatalf("Invalid end date format: %v", err)
	}

	// Initialize client
	client := clients.NewPolygonClient()

	// Download data to DuckDB
	dbPath, err := client.Download(*tickerFlag, *dbPathFlag, startDate, endDate, 1, models.Minute)
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

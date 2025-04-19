package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata"
	"github.com/urfave/cli/v3"
)

// downloadAction is the core logic executed by the CLI command.
// It parses arguments, sets up the market data client, and starts the download process.
func downloadAction(ctx context.Context, cmd *cli.Command) error {
	// Retrieve flag values from the context
	ticker := cmd.String("ticker")
	startDate := cmd.Timestamp("start")
	endDate := cmd.Timestamp("end")
	providerFlag := cmd.String("provider")
	writerFlag := cmd.String("writer")
	dataPath := cmd.String("data")

	// Create client configuration
	clientConfig := marketdata.ClientConfig{
		ProviderType:  marketdata.ProviderType(providerFlag),
		WriterType:    marketdata.WriterType(writerFlag),
		DataPath:      dataPath,
		PolygonApiKey: os.Getenv("POLYGON_API_KEY"),
	}

	// Create market data client
	client, err := marketdata.NewClient(clientConfig)
	if err != nil {
		return fmt.Errorf("failed to create market data client: %w", err)
	}

	// Create download parameters
	downloadParams := marketdata.DownloadParams{
		Ticker:     ticker,
		StartDate:  startDate,
		EndDate:    endDate,
		Multiplier: 1, // Assuming 1 minute bars, could be made configurable via flags
		Timespan:   models.Minute,
	}

	// Execute download
	log.Printf("Starting download for %s from %s to %s using %s provider and %s writer...",
		ticker, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"), providerFlag, writerFlag)

	err = client.Download(downloadParams)
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	log.Println("Download completed successfully.")
	return nil
}

func main() {
	// Define the CLI application
	cmd := &cli.Command{
		Name:  "download",
		Usage: "Download historical market data",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:     "ticker",
				Aliases:  []string{"t"},
				Usage:    "Stock ticker symbol",
				Required: true,
			},
			&cli.TimestampFlag{
				Name:    "start",
				Aliases: []string{"s"},
				Usage:   "Start date in `YYYY-MM-DD` format (or other RFC3339 compatible)",
				Config: cli.TimestampConfig{
					Layouts: []string{"2006-01-02"},
				},
				Required: true,
			},
			&cli.TimestampFlag{
				Name:     "end",
				Aliases:  []string{"e"},
				Usage:    "End date in `YYYY-MM-DD` format (or other RFC3339 compatible). Defaults to today.",
				Value:    time.Now(), // Default to today
				Required: false,      // Has a default value
				Config: cli.TimestampConfig{
					Layouts: []string{"2006-01-02"},
				},
			},
			&cli.StringFlag{
				Name:     "provider",
				Aliases:  []string{"p"},
				Usage:    fmt.Sprintf("Data provider to use (e.g., %s, %s)", marketdata.ProviderPolygon, marketdata.ProviderBinance),
				Value:    string(marketdata.ProviderPolygon), // Default provider
				Required: false,
			},
			&cli.StringFlag{
				Name:     "writer",
				Aliases:  []string{"w"},
				Usage:    fmt.Sprintf("Data writer format (e.g., %s)", marketdata.WriterDuckDB),
				Value:    string(marketdata.WriterDuckDB), // Default writer
				Required: false,
			},
			&cli.StringFlag{
				Name:     "data",
				Aliases:  []string{"d"},
				Usage:    "Path to the data output directory",
				Value:    "data", // Default data directory
				Required: false,
			},
		},
		Action: downloadAction, // Assign the action function
	}

	// Run the CLI application
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

package clients

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/gocarina/gocsv"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/schollz/progressbar/v3"

	"github.com/sirily11/argo-trading-go/src/types"
)

type PolygonClient struct {
	client *polygon.Client
}

func NewPolygonClient() *PolygonClient {
	client := polygon.New(os.Getenv("POLYGON_API_KEY"))
	return &PolygonClient{
		client: client,
	}
}

func (c *PolygonClient) getFileName(ticker string, startDate time.Time, endDate time.Time) string {
	return fmt.Sprintf("%s_%s_%s.csv", ticker, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))
}

func (c *PolygonClient) Download(ticker string, toPath string, startDate time.Time, endDate time.Time) (path string, err error) {
	// check if toPath folder exists
	dir := filepath.Dir(toPath)
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		// create the folder
		err = os.MkdirAll(dir, 0755)
		if err != nil {
			return "", err
		}

		log.Printf("Created folder %s\n", dir)
		return "", nil
	}

	outputFileName := c.getFileName(ticker, startDate, endDate)
	outputFilePath := filepath.Join(dir, outputFileName)
	log.Printf("Downloading data to %s\n", outputFilePath)
	// Calculate the number of days to download
	// Calculate the number of days to download
	totalDays := int(endDate.Sub(startDate).Hours() / 24)

	// Create a new progress bar
	bar := progressbar.New(totalDays)

	// Download the data from date to date
	for date := startDate; !date.After(endDate); date = date.AddDate(0, 0, 1) {
		params := models.ListAggsParams{
			Ticker:     ticker,
			From:       models.Millis(date),
			To:         models.Millis(date.Add(24 * time.Hour).Add(-1 * time.Second)),
			Multiplier: 1,
			Timespan:   models.Minute,
		}

		iter := c.client.ListAggs(context.Background(), &params)

		if iter.Err() != nil {
			return "", iter.Err()
		}

		rows := []types.MarketData{}
		for iter.Next() {
			agg := iter.Item()
			tradingData := types.MarketData{
				Time:   time.Time(agg.Timestamp),
				Open:   agg.Open,
				High:   agg.High,
				Low:    agg.Low,
				Close:  agg.Close,
				Volume: agg.Volume,
			}
			rows = append(rows, tradingData)
		}

		// Check if file exists and get its size
		fileInfo, err := os.Stat(outputFilePath)
		fileExists := err == nil

		// Open file in append mode if it exists, or create it if it doesn't
		var csvFile *os.File
		if fileExists {
			csvFile, err = os.OpenFile(outputFilePath, os.O_APPEND|os.O_WRONLY, 0644)
		} else {
			csvFile, err = os.Create(outputFilePath)
		}

		if err != nil {
			return "", err
		}
		defer csvFile.Close()

		// Use gocsv package to marshal the data
		if !fileExists || fileInfo.Size() == 0 {
			// If file is new or empty, write with headers
			if err := gocsv.MarshalFile(&rows, csvFile); err != nil {
				return "", err
			}
		} else {
			// If file exists and has content, append without headers
			if err := gocsv.MarshalWithoutHeaders(&rows, csvFile); err != nil {
				return "", err
			}
		}

		// Increment the progress bar
		bar.Add(1)
	}

	return "", nil // Adjust the return statement as needed
}

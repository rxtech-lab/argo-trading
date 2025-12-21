package provider

import (
	"context"
	"fmt"
	"log"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/schollz/progressbar/v3"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

type PolygonClient struct {
	client *polygon.Client
	writer writer.MarketDataWriter
}

func NewPolygonClient(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	client := polygon.New(apiKey)

	return &PolygonClient{
		client: client,
		writer: nil,
	}, nil
}

func (c *PolygonClient) ConfigWriter(w writer.MarketDataWriter) {
	c.writer = w
}

func (c *PolygonClient) Download(ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error) {
	if c.writer == nil {
		return "", fmt.Errorf("no writer configured for PolygonClient. Call ConfigWriter first")
	}

	err = c.writer.Initialize()
	if err != nil {
		return "", fmt.Errorf("failed to initialize writer: %w", err)
	}

	defer func() {
		if cerr := c.writer.Close(); cerr != nil {
			if err == nil {
				err = fmt.Errorf("error closing writer: %w", cerr)
			} else {
				log.Printf("Error closing writer after another error: %v", cerr)
			}
		}
	}()

	totalIterations := int(endDate.Sub(startDate).Hours()/24) + 1

	bar := progressbar.NewOptions(totalIterations, progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", ticker)), progressbar.OptionShowCount())

	//nolint:exhaustruct // third-party struct with many optional fields
	params := models.ListAggsParams{
		Ticker:     ticker,
		Multiplier: multiplier,
		Timespan:   timespan,
		From:       models.Millis(startDate),
		To:         models.Millis(endDate),
	}.WithLimit(50000)

	iter := c.client.ListAggs(context.Background(), params)

	processedCount := 0

	for iter.Next() {
		go onProgress(float64(processedCount), float64(totalIterations), fmt.Sprintf("Downloading %s", ticker))

		agg := iter.Item()
		marketData := types.MarketData{
			Id:     "",
			Symbol: ticker,
			Time:   time.Time(agg.Timestamp),
			Open:   agg.Open,
			High:   agg.High,
			Low:    agg.Low,
			Close:  agg.Close,
			Volume: agg.Volume,
		}

		err = c.writer.Write(marketData)
		if err != nil {
			return "", fmt.Errorf("failed to write data: %w", err)
		}

		processedCount++
		if processedCount%1000 == 0 {
			currentTime := time.Time(agg.Timestamp)
			daysElapsed := int(currentTime.Sub(startDate).Hours() / 24)
			bar.Set(daysElapsed)
		}
	}

	if iter.Err() != nil {
		return "", fmt.Errorf("error iterating polygon aggregates: %w", iter.Err())
	}

	bar.Finish()
	log.Printf("Finished downloading %d data points for %s.", processedCount, ticker)

	outputPath, err := c.writer.Finalize()
	if err != nil {
		return "", fmt.Errorf("failed to finalize writer: %w", err)
	}

	return outputPath, nil
}

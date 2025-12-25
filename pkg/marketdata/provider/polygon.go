package provider

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/iter"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/schollz/progressbar/v3"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

// PolygonAggsIterator defines the interface for iterating over aggregates.
type PolygonAggsIterator interface {
	Next() bool
	Item() models.Agg
	Err() error
}

// PolygonAPIClient defines the interface for the Polygon API client.
type PolygonAPIClient interface {
	ListAggs(ctx context.Context, params *models.ListAggsParams, options ...models.RequestOption) PolygonAggsIterator
}

// polygonClientWrapper wraps the real polygon.Client to implement PolygonAPIClient.
type polygonClientWrapper struct {
	client *polygon.Client
}

func (w *polygonClientWrapper) ListAggs(ctx context.Context, params *models.ListAggsParams, options ...models.RequestOption) PolygonAggsIterator {
	return w.client.ListAggs(ctx, params, options...)
}

type PolygonClient struct {
	apiClient PolygonAPIClient
	writer    writer.MarketDataWriter
}

func NewPolygonClient(apiKey string) (Provider, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	client := polygon.New(apiKey)

	return &PolygonClient{
		apiClient: &polygonClientWrapper{client: client},
		writer:    nil,
	}, nil
}

// NewPolygonClientWithAPI creates a PolygonClient with a custom API client (for testing).
func NewPolygonClientWithAPI(apiClient PolygonAPIClient) *PolygonClient {
	return &PolygonClient{
		apiClient: apiClient,
		writer:    nil,
	}
}

// Ensure iter.Iter[models.Agg] implements PolygonAggsIterator.
var _ PolygonAggsIterator = (*iter.Iter[models.Agg])(nil)

func (c *PolygonClient) ConfigWriter(w writer.MarketDataWriter) {
	c.writer = w
}

func (c *PolygonClient) Download(ctx context.Context, ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error) {
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

	aggsIter := c.apiClient.ListAggs(ctx, params)

	processedCount := 0

	for aggsIter.Next() {
		// Check for cancellation
		select {
		case <-ctx.Done():
			if processedCount == 0 {
				c.cleanupFileIfExists()
			}

			return "", ctx.Err()
		default:
		}

		onProgress(float64(processedCount), float64(totalIterations), fmt.Sprintf("Downloading %s", ticker))

		agg := aggsIter.Item()
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
			// Cleanup file if no data was written
			if processedCount == 0 {
				c.cleanupFileIfExists()
			}

			return "", fmt.Errorf("failed to write data: %w", err)
		}

		processedCount++
		if processedCount%1000 == 0 {
			currentTime := time.Time(agg.Timestamp)
			daysElapsed := int(currentTime.Sub(startDate).Hours() / 24)
			bar.Set(daysElapsed)
		}
	}

	if aggsIter.Err() != nil {
		// Cleanup file if no data was written
		if processedCount == 0 {
			c.cleanupFileIfExists()
		}

		return "", fmt.Errorf("error iterating polygon aggregates: %w", aggsIter.Err())
	}

	bar.Finish()
	log.Printf("Finished downloading %d data points for %s.", processedCount, ticker)

	outputPath, err := c.writer.Finalize()
	if err != nil {
		return "", fmt.Errorf("failed to finalize writer: %w", err)
	}

	return outputPath, nil
}

// cleanupFileIfExists removes the output file if it exists.
// This is used to clean up when download fails and no data was written.
func (c *PolygonClient) cleanupFileIfExists() {
	if c.writer == nil {
		return
	}

	outputPath := c.writer.GetOutputPath()
	if outputPath == "" {
		return
	}

	if _, err := os.Stat(outputPath); err == nil {
		if removeErr := os.Remove(outputPath); removeErr != nil {
			log.Printf("Warning: failed to remove file %s: %v", outputPath, removeErr)
		} else {
			log.Printf("Removed file %s due to download failure with no data", outputPath)
		}
	}
}

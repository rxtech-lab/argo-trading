package provider

import (
	"context"
	"fmt"
	"strconv"
	"time"

	binance "github.com/adshao/go-binance/v2"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

type BinanceClient struct {
	client *binance.Client
	writer writer.MarketDataWriter
}

func NewBinanceClient() (Provider, error) {
	client := binance.NewClient("", "")

	return &BinanceClient{
		client: client,
		writer: nil,
	}, nil
}

func (c *BinanceClient) ConfigWriter(w writer.MarketDataWriter) {
	c.writer = w
}

// Download downloads the historical klines data for the given ticker and date range from Binance.
// It converts the binance kline format to our internal MarketData format and writes it using the configured writer.
func (c *BinanceClient) Download(ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error) {
	interval, err := convertTimespanToBinanceInterval(timespan, multiplier)
	if err != nil {
		return "", fmt.Errorf("failed to convert timespan to Binance interval: %w", err)
	}

	if c.writer == nil {
		return "", fmt.Errorf("writer is not configured")
	}

	err = c.writer.Initialize()
	if err != nil {
		return "", fmt.Errorf("failed to initialize writer: %w", err)
	}

	// Binance API uses milliseconds for timestamps
	startTimeMillis := startDate.UnixMilli()
	endTimeMillis := endDate.UnixMilli()

	// Use pagination to handle Binance API limits (max 500 data points per request)
	// Keep track of the last data point time to use as start time for next request
	currentStartTime := startTimeMillis

	for {
		klines, err := c.client.NewKlinesService().
			Symbol(ticker).
			Interval(interval).
			StartTime(currentStartTime).
			EndTime(endTimeMillis).
			Do(context.Background())
		if err != nil {
			// Attempt to finalize/close even if fetch fails
			_, finalizeErr := c.writer.Finalize()
			if finalizeErr != nil {
				return "", fmt.Errorf("failed to fetch klines from Binance: %w; also failed to finalize writer: %v", err, finalizeErr)
			}

			return "", fmt.Errorf("failed to fetch klines from Binance: %w", err)
		}

		go onProgress(float64(currentStartTime), float64(endTimeMillis), fmt.Sprintf("Downloading %s klines from Binance", ticker))

		// Break conditions: no data or less than 500 records (last page)
		if len(klines) == 0 || len(klines) < 500 {
			// Process the remaining klines if any
			if err := processKlines(c.writer, ticker, klines); err != nil {
				// Attempt to finalize/close even if processing fails
				_, finalizeErr := c.writer.Finalize()
				if finalizeErr != nil {
					return "", fmt.Errorf("failed to process klines: %w; also failed to finalize writer: %v", err, finalizeErr)
				}

				return "", fmt.Errorf("failed to process klines: %w", err)
			}

			break
		}

		// Process current page of klines
		if err := processKlines(c.writer, ticker, klines); err != nil {
			// Attempt to finalize/close even if processing fails
			_, finalizeErr := c.writer.Finalize()
			if finalizeErr != nil {
				return "", fmt.Errorf("failed to process klines: %w; also failed to finalize writer: %v", err, finalizeErr)
			}

			return "", fmt.Errorf("failed to process klines: %w", err)
		}

		// Update start time for next request
		// Use the close time of the last kline + 1ms to avoid duplicates
		lastKline := klines[len(klines)-1]
		currentStartTime = lastKline.CloseTime + 1

		// Break if we've reached or exceeded the end time
		if currentStartTime >= endTimeMillis {
			break
		}
	}

	// Finalize the writing process (e.g., save file, commit transaction)
	outputPath, err := c.writer.Finalize()
	if err != nil {
		return "", fmt.Errorf("failed to finalize writer: %w", err)
	}

	return outputPath, nil
}

// processKlines converts Binance kline data to our internal MarketData format and writes it.
func processKlines(writer writer.MarketDataWriter, ticker string, klines []*binance.Kline) error {
	for _, k := range klines {
		open, _ := strconv.ParseFloat(k.Open, 64)
		high, _ := strconv.ParseFloat(k.High, 64)
		low, _ := strconv.ParseFloat(k.Low, 64)
		closePrice, _ := strconv.ParseFloat(k.Close, 64)
		volume, _ := strconv.ParseFloat(k.Volume, 64)

		marketData := types.MarketData{
			Id:     "",
			Symbol: ticker,
			Time:   time.UnixMilli(k.OpenTime), // Using OpenTime as the timestamp for the bar
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
			// VWAP and N (trade count) might not be directly available in standard klines
		}

		if err := writer.Write(marketData); err != nil {
			return fmt.Errorf("failed to write market data: %w", err)
		}
	}

	return nil
}

// convertTimespanToBinanceInterval converts the polygon timespan and multiplier to a Binance interval string.
// Binance intervals: 1m, 3m, 5m, 15m, 30m, 1h, 2h, 4h, 6h, 8h, 12h, 1d, 3d, 1w, 1M
// Ref: https://binance-docs.github.io/apidocs/spot/en/#kline-candlestick-data
func convertTimespanToBinanceInterval(timespan models.Timespan, multiplier int) (string, error) {
	switch timespan {
	case models.Minute:
		return fmt.Sprintf("%dm", multiplier), nil
	case models.Hour:
		return fmt.Sprintf("%dh", multiplier), nil
	case models.Day:
		return fmt.Sprintf("%dd", multiplier), nil
	case models.Week:
		if multiplier == 1 {
			return "1w", nil
		}

		return "", fmt.Errorf("unsupported weekly multiplier for Binance: %d", multiplier)
	case models.Month:
		if multiplier == 1 {
			return "1M", nil
		}

		return "", fmt.Errorf("unsupported monthly multiplier for Binance: %d", multiplier)
	default:
		return "", fmt.Errorf("unsupported timespan for Binance: %s", timespan)
	}
}

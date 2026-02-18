package provider

import (
	"context"
	"fmt"
	goiter "iter"
	"log"
	"os"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	polygon "github.com/polygon-io/client-go/rest"
	"github.com/polygon-io/client-go/rest/iter"
	"github.com/polygon-io/client-go/rest/models"
	polygonws "github.com/polygon-io/client-go/websocket"
	polygonmodels "github.com/polygon-io/client-go/websocket/models"
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

// PolygonWebSocketService defines the interface for Polygon WebSocket operations.
// This abstraction enables testing with mock implementations.
type PolygonWebSocketService interface {
	// Connect establishes the WebSocket connection to Polygon.
	Connect() error

	// Subscribe subscribes to a topic for the given tickers.
	Subscribe(topic polygonws.Topic, tickers ...string) error

	// Unsubscribe unsubscribes from a topic for the given tickers.
	Unsubscribe(topic polygonws.Topic, tickers ...string) error

	// Output returns a channel that receives incoming messages.
	// Messages can be of type: models.EquityAgg, models.EquityTrade, etc.
	Output() <-chan any

	// Error returns a channel that receives error messages.
	Error() <-chan error

	// Close gracefully closes the WebSocket connection.
	Close()
}

// polygonWebSocketServiceWrapper wraps the real Polygon WebSocket client.
type polygonWebSocketServiceWrapper struct {
	client *polygonws.Client
}

func newPolygonWebSocketService(apiKey string, feed polygonws.Feed) (PolygonWebSocketService, error) {
	//nolint:exhaustruct // third-party struct with many optional fields
	config := polygonws.Config{
		APIKey: apiKey,
		Feed:   feed,
		Market: polygonws.Stocks,
	}

	client, err := polygonws.New(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create polygon websocket client: %w", err)
	}

	return &polygonWebSocketServiceWrapper{client: client}, nil
}

func (w *polygonWebSocketServiceWrapper) Connect() error {
	return w.client.Connect()
}

func (w *polygonWebSocketServiceWrapper) Subscribe(topic polygonws.Topic, tickers ...string) error {
	return w.client.Subscribe(topic, tickers...)
}

func (w *polygonWebSocketServiceWrapper) Unsubscribe(topic polygonws.Topic, tickers ...string) error {
	return w.client.Unsubscribe(topic, tickers...)
}

func (w *polygonWebSocketServiceWrapper) Output() <-chan any {
	return w.client.Output()
}

func (w *polygonWebSocketServiceWrapper) Error() <-chan error {
	return w.client.Error()
}

func (w *polygonWebSocketServiceWrapper) Close() {
	w.client.Close()
}

type PolygonClient struct {
	apiClient           PolygonAPIClient
	wsServiceForTesting PolygonWebSocketService // WebSocket service for testing
	apiKey              string
	writer              writer.MarketDataWriter
	onStatusChange      OnStatusChange
	symbols             []string
	interval            string
}

func NewPolygonClient(config *PolygonStreamConfig) (Provider, error) {
	if config == nil {
		return nil, fmt.Errorf("config is required for polygon provider")
	}

	if config.ApiKey == "" {
		return nil, fmt.Errorf("apiKey is required")
	}

	client := polygon.New(config.ApiKey)

	return &PolygonClient{
		apiClient:           &polygonClientWrapper{client: client},
		wsServiceForTesting: nil,
		apiKey:              config.ApiKey,
		writer:              nil,
		onStatusChange:      nil,
		symbols:             config.Symbols,
		interval:            config.Interval,
	}, nil
}

// NewPolygonClientWithAPI creates a PolygonClient with a custom API client (for testing).
func NewPolygonClientWithAPI(apiClient PolygonAPIClient, symbols []string, interval string) *PolygonClient {
	return &PolygonClient{
		apiClient:           apiClient,
		wsServiceForTesting: nil,
		apiKey:              "",
		writer:              nil,
		onStatusChange:      nil,
		symbols:             symbols,
		interval:            interval,
	}
}

// NewPolygonClientWithWebSocket creates a PolygonClient with custom WebSocket service (for testing).
func NewPolygonClientWithWebSocket(apiKey string, wsService PolygonWebSocketService, symbols []string, interval string) *PolygonClient {
	return &PolygonClient{
		apiClient:           nil,
		wsServiceForTesting: wsService,
		apiKey:              apiKey,
		writer:              nil,
		onStatusChange:      nil,
		symbols:             symbols,
		interval:            interval,
	}
}

// GetSymbols returns the list of symbols configured for streaming.
func (c *PolygonClient) GetSymbols() []string {
	return c.symbols
}

// GetInterval returns the candlestick interval configured for streaming.
func (c *PolygonClient) GetInterval() string {
	return c.interval
}

// Ensure iter.Iter[models.Agg] implements PolygonAggsIterator.
var _ PolygonAggsIterator = (*iter.Iter[models.Agg])(nil)

func (c *PolygonClient) ConfigWriter(w writer.MarketDataWriter) {
	c.writer = w
}

// SetOnStatusChange sets a callback that will be called when the WebSocket connection
// status changes (connected/disconnected).
func (c *PolygonClient) SetOnStatusChange(callback OnStatusChange) {
	c.onStatusChange = callback
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

	// Calculate total time range in milliseconds for progress tracking
	totalTimeRange := endDate.Sub(startDate).Milliseconds()
	startTimeMillis := startDate.UnixMilli()

	// Progress bar based on days for visual display
	totalDays := int(endDate.Sub(startDate).Hours()/24) + 1
	bar := progressbar.NewOptions(totalDays, progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", ticker)), progressbar.OptionShowCount())

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

		agg := aggsIter.Item()
		currentTimeMillis := time.Time(agg.Timestamp).UnixMilli()
		timeElapsed := currentTimeMillis - startTimeMillis

		// Report progress based on time elapsed vs total time range
		onProgress(float64(timeElapsed), float64(totalTimeRange), fmt.Sprintf("Downloading %s", ticker))

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

// Stream implements Provider.Stream for real-time WebSocket market data from Polygon.
// It subscribes to aggregate streams for all specified symbols and yields data as it arrives.
// The iterator terminates when the context is cancelled or an unrecoverable error occurs.
func (c *PolygonClient) Stream(ctx context.Context) goiter.Seq2[types.MarketData, error] {
	return func(yield func(types.MarketData, error) bool) {
		symbols := c.symbols
		interval := c.interval

		// Validate inputs
		if len(symbols) == 0 {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, fmt.Errorf("no symbols provided for streaming"))

			return
		}

		topic, err := convertIntervalToPolygonTopic(interval)
		if err != nil {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, err)

			return
		}

		// Create or use WebSocket service
		var wsService PolygonWebSocketService
		if c.wsServiceForTesting != nil {
			// Use injected service for testing
			wsService = c.wsServiceForTesting
		} else {
			// Create production WebSocket service
			wsService, err = newPolygonWebSocketService(c.apiKey, polygonws.RealTime)
			if err != nil {
				//nolint:exhaustruct // empty struct for error case
				yield(types.MarketData{}, fmt.Errorf("failed to create websocket service: %w", err))

				return
			}
			defer wsService.Close()
		}

		// Connect to WebSocket
		if err := wsService.Connect(); err != nil {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, fmt.Errorf("failed to connect to polygon websocket: %w", err))

			// Emit disconnected status on connection failure
			c.emitStatus(types.ProviderStatusDisconnected)

			return
		}

		// Emit connected status when WebSocket connection is established
		c.emitStatus(types.ProviderStatusConnected)

		// Subscribe to aggregate topic for all symbols
		if err := wsService.Subscribe(topic, symbols...); err != nil {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, fmt.Errorf("failed to subscribe to symbols: %w", err))

			// Emit disconnected status on subscription failure
			c.emitStatus(types.ProviderStatusDisconnected)

			return
		}

		// Ensure disconnected status is emitted when the stream ends
		defer c.emitStatus(types.ProviderStatusDisconnected)

		// Main message loop
		for {
			select {
			case <-ctx.Done():
				return

			case err := <-wsService.Error():
				//nolint:exhaustruct // empty struct for error case
				if !yield(types.MarketData{}, fmt.Errorf("websocket error: %w", err)) {
					return
				}

			case msg := <-wsService.Output():
				switch agg := msg.(type) {
				case polygonmodels.EquityAgg:
					marketData := convertEquityAggToMarketData(&agg)
					if !yield(marketData, nil) {
						return
					}
				// Ignore other message types (trades, quotes, control messages)
				default:
					// Skip non-aggregate messages
				}
			}
		}
	}
}

// emitStatus emits a status change if a callback is registered.
func (c *PolygonClient) emitStatus(status types.ProviderConnectionStatus) {
	if c.onStatusChange != nil {
		c.onStatusChange(status)
	}
}

// convertIntervalToPolygonTopic converts an interval string to a Polygon WebSocket topic.
func convertIntervalToPolygonTopic(interval string) (polygonws.Topic, error) {
	switch interval {
	case "1s":
		return polygonws.StocksSecAggs, nil
	case "1m":
		return polygonws.StocksMinAggs, nil
	default:
		// For other intervals, use minute aggregates and aggregate locally
		// This is a limitation of the Polygon WebSocket API which only supports
		// second and minute aggregates natively
		return polygonws.StocksMinAggs, nil
	}
}

// convertEquityAggToMarketData converts a Polygon EquityAgg to our internal MarketData type.
func convertEquityAggToMarketData(agg *polygonmodels.EquityAgg) types.MarketData {
	return types.MarketData{
		Id:     "",
		Symbol: agg.Symbol,
		Time:   time.UnixMilli(agg.StartTimestamp),
		Open:   agg.Open,
		High:   agg.High,
		Low:    agg.Low,
		Close:  agg.Close,
		Volume: agg.Volume,
	}
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

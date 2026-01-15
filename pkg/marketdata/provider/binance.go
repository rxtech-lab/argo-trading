package provider

import (
	"context"
	"fmt"
	"iter"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	binance "github.com/adshao/go-binance/v2"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

// BinanceKlinesService defines the interface for fetching klines from Binance.
type BinanceKlinesService interface {
	Symbol(symbol string) BinanceKlinesService
	Interval(interval string) BinanceKlinesService
	StartTime(startTime int64) BinanceKlinesService
	EndTime(endTime int64) BinanceKlinesService
	Do(ctx context.Context) ([]*binance.Kline, error)
}

// BinanceAPIClient defines the interface for the Binance API client.
type BinanceAPIClient interface {
	NewKlinesService() BinanceKlinesService
	NewListPricesService() BinanceListPricesService
}

// SymbolPrice represents the price of a symbol.
type SymbolPrice struct {
	Symbol string
	Price  string
}

// BinanceListPricesService defines the interface for fetching symbol prices.
type BinanceListPricesService interface {
	Symbols(symbols []string) BinanceListPricesService
	Do(ctx context.Context) ([]*SymbolPrice, error)
}

// binanceClientWrapper wraps the real binance.Client to implement BinanceAPIClient.
type binanceClientWrapper struct {
	client *binance.Client
}

func (w *binanceClientWrapper) NewKlinesService() BinanceKlinesService {
	return &binanceKlinesServiceWrapper{service: w.client.NewKlinesService()}
}

func (w *binanceClientWrapper) NewListPricesService() BinanceListPricesService {
	return &binanceListPricesServiceWrapper{service: w.client.NewListPricesService()}
}

// binanceKlinesServiceWrapper wraps the real binance.KlinesService.
type binanceKlinesServiceWrapper struct {
	service *binance.KlinesService
}

func (w *binanceKlinesServiceWrapper) Symbol(symbol string) BinanceKlinesService {
	w.service = w.service.Symbol(symbol)

	return w
}

func (w *binanceKlinesServiceWrapper) Interval(interval string) BinanceKlinesService {
	w.service = w.service.Interval(interval)

	return w
}

func (w *binanceKlinesServiceWrapper) StartTime(startTime int64) BinanceKlinesService {
	w.service = w.service.StartTime(startTime)

	return w
}

func (w *binanceKlinesServiceWrapper) EndTime(endTime int64) BinanceKlinesService {
	w.service = w.service.EndTime(endTime)

	return w
}

func (w *binanceKlinesServiceWrapper) Do(ctx context.Context) ([]*binance.Kline, error) {
	return w.service.Do(ctx)
}

// binanceListPricesServiceWrapper wraps the real binance.ListPricesService.
type binanceListPricesServiceWrapper struct {
	service *binance.ListPricesService
}

func (w *binanceListPricesServiceWrapper) Symbols(symbols []string) BinanceListPricesService {
	w.service = w.service.Symbols(symbols)

	return w
}

func (w *binanceListPricesServiceWrapper) Do(ctx context.Context) ([]*SymbolPrice, error) {
	prices, err := w.service.Do(ctx)
	if err != nil {
		return nil, err
	}

	result := make([]*SymbolPrice, len(prices))
	for i, p := range prices {
		result[i] = &SymbolPrice{
			Symbol: p.Symbol,
			Price:  p.Price,
		}
	}

	return result, nil
}

// BinanceWsKline represents the kline data within a WebSocket event.
type BinanceWsKline struct {
	StartTime int64
	EndTime   int64
	Symbol    string
	Interval  string
	Open      string
	Close     string
	High      string
	Low       string
	Volume    string
	IsFinal   bool
}

// BinanceWsKlineEvent represents a WebSocket kline event.
type BinanceWsKlineEvent struct {
	Symbol string
	Kline  BinanceWsKline
}

// WsKlineHandler is a handler function for WebSocket kline events.
type WsKlineHandler func(event *BinanceWsKlineEvent)

// WsErrorHandler is a handler function for WebSocket errors.
type WsErrorHandler func(err error)

// BinanceWebSocketService defines the interface for Binance WebSocket operations.
type BinanceWebSocketService interface {
	// WsKlineServe starts a WebSocket connection for kline data.
	// Returns: doneC (closes when connection ends), stopC (send to stop), error
	WsKlineServe(symbol string, interval string, handler WsKlineHandler, errHandler WsErrorHandler) (doneC chan struct{}, stopC chan struct{}, err error)
}

// binanceWebSocketServiceWrapper wraps the real binance WebSocket functions.
type binanceWebSocketServiceWrapper struct{}

func (w *binanceWebSocketServiceWrapper) WsKlineServe(symbol string, interval string, handler WsKlineHandler, errHandler WsErrorHandler) (chan struct{}, chan struct{}, error) {
	// Convert our handler types to binance handler types
	binanceHandler := func(event *binance.WsKlineEvent) {
		handler(&BinanceWsKlineEvent{
			Symbol: event.Symbol,
			Kline: BinanceWsKline{
				StartTime: event.Kline.StartTime,
				EndTime:   event.Kline.EndTime,
				Symbol:    event.Kline.Symbol,
				Interval:  event.Kline.Interval,
				Open:      event.Kline.Open,
				Close:     event.Kline.Close,
				High:      event.Kline.High,
				Low:       event.Kline.Low,
				Volume:    event.Kline.Volume,
				IsFinal:   event.Kline.IsFinal,
			},
		})
	}

	binanceErrHandler := func(err error) {
		errHandler(err)
	}

	return binance.WsKlineServe(symbol, interval, binanceHandler, binanceErrHandler)
}

type BinanceClient struct {
	apiClient      BinanceAPIClient
	wsService      BinanceWebSocketService
	writer         writer.MarketDataWriter
	onStatusChange OnStatusChange
}

func NewBinanceClient() (Provider, error) {
	client := binance.NewClient("", "")

	return &BinanceClient{
		apiClient:      &binanceClientWrapper{client: client},
		wsService:      &binanceWebSocketServiceWrapper{},
		writer:         nil,
		onStatusChange: nil,
	}, nil
}

// NewBinanceClientWithAPI creates a BinanceClient with a custom API client (for testing).
func NewBinanceClientWithAPI(apiClient BinanceAPIClient) *BinanceClient {
	return &BinanceClient{
		apiClient:      apiClient,
		wsService:      &binanceWebSocketServiceWrapper{},
		writer:         nil,
		onStatusChange: nil,
	}
}

// NewBinanceClientWithWebSocket creates a BinanceClient with custom API and WebSocket services (for testing).
func NewBinanceClientWithWebSocket(apiClient BinanceAPIClient, wsService BinanceWebSocketService) *BinanceClient {
	return &BinanceClient{
		apiClient:      apiClient,
		wsService:      wsService,
		writer:         nil,
		onStatusChange: nil,
	}
}

// BinanceEndpointConfig holds custom endpoint configuration for Binance client.
type BinanceEndpointConfig struct {
	RestBaseURL string // Custom REST API base URL (e.g., "http://localhost:8080")
	WsBaseURL   string // Custom WebSocket base URL (e.g., "ws://localhost:8080/ws")
}

// NewBinanceClientWithEndpoints creates a BinanceClient with custom API endpoints.
// This is useful for testing with mock servers or using alternative endpoints.
// If WsBaseURL is set, it overrides the global binance.BaseWsMainURL.
func NewBinanceClientWithEndpoints(config BinanceEndpointConfig) (Provider, error) {
	// Set WebSocket endpoint before creating any connections
	// Note: This is a global variable change that affects all WebSocket connections
	if config.WsBaseURL != "" {
		binance.BaseWsMainURL = config.WsBaseURL
	}

	client := binance.NewClient("", "")
	if config.RestBaseURL != "" {
		client.BaseURL = config.RestBaseURL
	}

	return &BinanceClient{
		apiClient:      &binanceClientWrapper{client: client},
		wsService:      &binanceWebSocketServiceWrapper{},
		writer:         nil,
		onStatusChange: nil,
	}, nil
}

func (c *BinanceClient) ConfigWriter(w writer.MarketDataWriter) {
	c.writer = w
}

// SetOnStatusChange sets a callback that will be called when the WebSocket connection
// status changes (connected/disconnected).
func (c *BinanceClient) SetOnStatusChange(callback OnStatusChange) {
	c.onStatusChange = callback
}

// emitStatus emits a status change if a callback is registered.
func (c *BinanceClient) emitStatus(status types.ProviderConnectionStatus) {
	if c.onStatusChange != nil {
		c.onStatusChange(status)
	}
}

// ValidateSymbols checks if all provided symbols are valid Binance trading pairs.
// It uses the price ticker API to verify symbols exist and are actively trading.
// Returns an error listing any invalid symbols.
func (c *BinanceClient) ValidateSymbols(ctx context.Context, symbols []string) error {
	if len(symbols) == 0 {
		return nil
	}

	prices, err := c.apiClient.NewListPricesService().
		Symbols(symbols).
		Do(ctx)
	if err != nil {
		return fmt.Errorf("failed to validate symbols: %w", err)
	}

	// Build a map of returned symbols for quick lookup
	priceMap := make(map[string]bool, len(prices))
	for _, p := range prices {
		priceMap[p.Symbol] = true
	}

	// Check that we got prices for all requested symbols
	var invalid []string
	for _, sym := range symbols {
		if !priceMap[sym] {
			invalid = append(invalid, sym)
		}
	}

	if len(invalid) > 0 {
		return fmt.Errorf("invalid symbols: %v", invalid)
	}

	return nil
}

// Download downloads the historical klines data for the given ticker and date range from Binance.
// It converts the binance kline format to our internal MarketData format and writes it using the configured writer.
// The context can be used to cancel the download operation.
func (c *BinanceClient) Download(ctx context.Context, ticker string, startDate time.Time, endDate time.Time, multiplier int, timespan models.Timespan, onProgress OnDownloadProgress) (path string, err error) {
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
	totalRecordsWritten := 0

	for {
		// Check for cancellation
		select {
		case <-ctx.Done():
			if totalRecordsWritten == 0 {
				c.cleanupFileIfExists()
			}

			return "", ctx.Err()
		default:
		}

		klines, err := c.apiClient.NewKlinesService().
			Symbol(ticker).
			Interval(interval).
			StartTime(currentStartTime).
			EndTime(endTimeMillis).
			Do(ctx)
		if err != nil {
			// Attempt to finalize/close even if fetch fails
			_, finalizeErr := c.writer.Finalize()

			// Cleanup file if no data was written
			if totalRecordsWritten == 0 {
				c.cleanupFileIfExists()
			}

			if finalizeErr != nil {
				return "", fmt.Errorf("failed to fetch klines from Binance: %w; also failed to finalize writer: %v", err, finalizeErr)
			}

			return "", fmt.Errorf("failed to fetch klines from Binance: %w", err)
		}

		// Calculate relative progress (time elapsed vs total time range)
		onProgress(
			float64(currentStartTime-startTimeMillis),
			float64(endTimeMillis-startTimeMillis),
			fmt.Sprintf("Downloading %s klines from Binance", ticker),
		)

		// Break conditions: no data or less than 500 records (last page)
		if len(klines) == 0 || len(klines) < 500 {
			// Process the remaining klines if any
			if err := processKlines(c.writer, ticker, klines); err != nil {
				// Attempt to finalize/close even if processing fails
				_, finalizeErr := c.writer.Finalize()

				// Cleanup file if no data was written
				if totalRecordsWritten == 0 {
					c.cleanupFileIfExists()
				}

				if finalizeErr != nil {
					return "", fmt.Errorf("failed to process klines: %w; also failed to finalize writer: %v", err, finalizeErr)
				}

				return "", fmt.Errorf("failed to process klines: %w", err)
			}

			totalRecordsWritten += len(klines)

			break
		}

		// Process current page of klines
		if err := processKlines(c.writer, ticker, klines); err != nil {
			// Attempt to finalize/close even if processing fails
			_, finalizeErr := c.writer.Finalize()

			// Cleanup file if no data was written
			if totalRecordsWritten == 0 {
				c.cleanupFileIfExists()
			}

			if finalizeErr != nil {
				return "", fmt.Errorf("failed to process klines: %w; also failed to finalize writer: %v", err, finalizeErr)
			}

			return "", fmt.Errorf("failed to process klines: %w", err)
		}

		totalRecordsWritten += len(klines)

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

// Stream implements Provider.Stream for real-time WebSocket market data.
// It subscribes to kline streams for all specified symbols and yields data as it arrives.
// The iterator terminates when the context is cancelled or an unrecoverable error occurs.
func (c *BinanceClient) Stream(ctx context.Context, symbols []string, interval string) iter.Seq2[types.MarketData, error] {
	return func(yield func(types.MarketData, error) bool) {
		if len(symbols) == 0 {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, fmt.Errorf("no symbols provided for streaming"))

			return
		}

		if !isValidBinanceInterval(interval) {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, fmt.Errorf("invalid interval: %s", interval))

			return
		}

		// Validate that all symbols are valid Binance trading pairs
		if err := c.ValidateSymbols(ctx, symbols); err != nil {
			//nolint:exhaustruct // empty struct for error case
			yield(types.MarketData{}, err)

			return
		}

		// Channel for market data from all WebSocket connections
		dataChan := make(chan types.MarketData, 100)
		errChan := make(chan error, len(symbols)*2)

		// Track all stop channels for cleanup with their close state
		type stopChanEntry struct {
			ch     chan struct{}
			closed bool
		}

		var stopChannels []*stopChanEntry

		var mu sync.Mutex

		var wg sync.WaitGroup

		// Helper to safely stop a channel
		safeStop := func(entry *stopChanEntry) {
			mu.Lock()
			defer mu.Unlock()

			if !entry.closed {
				entry.closed = true
				close(entry.ch)
			}
		}

		// Start WebSocket connection for each symbol
		for _, symbol := range symbols {
			wg.Add(1)

			go func(sym string) {
				defer wg.Done()

				handler := func(event *BinanceWsKlineEvent) {
					// Only emit finalized candles (IsFinal=true)
					// This ensures we only persist and process complete candle data
					if !event.Kline.IsFinal {
						return
					}

					marketData := convertWsKlineToMarketData(event)

					select {
					case dataChan <- marketData:
					case <-ctx.Done():
						return
					}
				}

				errHandler := func(err error) {
					select {
					case errChan <- fmt.Errorf("websocket error for %s: %w", sym, err):
					case <-ctx.Done():
					default:
					}
				}

				doneC, stopC, err := c.wsService.WsKlineServe(sym, interval, handler, errHandler)
				if err != nil {
					select {
					case errChan <- fmt.Errorf("failed to start websocket for %s: %w", sym, err):
					default:
					}

					// Emit disconnected status on connection failure
					c.emitStatus(types.ProviderStatusDisconnected)

					return
				}

				// Emit connected status when WebSocket connection is established
				c.emitStatus(types.ProviderStatusConnected)

				entry := &stopChanEntry{ch: stopC, closed: false}

				mu.Lock()
				stopChannels = append(stopChannels, entry)
				mu.Unlock()

				// Wait for context cancellation or connection close
				select {
				case <-ctx.Done():
					safeStop(entry)
					// Emit disconnected status when connection is closed
					c.emitStatus(types.ProviderStatusDisconnected)
				case <-doneC:
					// Emit disconnected status when connection is closed
					c.emitStatus(types.ProviderStatusDisconnected)
				}
			}(symbol)
		}

		// Cleanup function - stops all connections and closes channels
		cleanup := func() {
			mu.Lock()
			channels := make([]*stopChanEntry, len(stopChannels))
			copy(channels, stopChannels)
			mu.Unlock()

			for _, entry := range channels {
				safeStop(entry)
			}

			// Wait for all WebSocket goroutines to finish
			wg.Wait()

			// Close channels to signal completion
			close(dataChan)
			close(errChan)
		}

		// Cleanup goroutine - ensures all connections are stopped when context is cancelled
		var cleanupWg sync.WaitGroup

		cleanupWg.Add(1)

		go func() {
			defer cleanupWg.Done()
			<-ctx.Done()

			// Wait for all symbol goroutines to finish registering their stop channels
			wg.Wait()

			mu.Lock()
			channels := make([]*stopChanEntry, len(stopChannels))
			copy(channels, stopChannels)
			mu.Unlock()

			for _, entry := range channels {
				safeStop(entry)
			}
		}()

		// Main loop - yield data to the iterator
		defer func() {
			// If we exit early (yield returned false), trigger cleanup
			// Cancel context would have already triggered cleanup goroutine
			select {
			case <-ctx.Done():
				// Context cancelled, cleanup goroutine handles it
				cleanupWg.Wait()
			default:
				// Early exit from iterator, need to cleanup manually
				cleanup()
			}
		}()

		for {
			select {
			case <-ctx.Done():
				return

			case err, ok := <-errChan:
				if !ok {
					return
				}
				//nolint:exhaustruct // empty struct for error case
				if !yield(types.MarketData{}, err) {
					return
				}

			case data, ok := <-dataChan:
				if !ok {
					return
				}

				if !yield(data, nil) {
					return
				}
			}
		}
	}
}

// convertWsKlineToMarketData converts a WebSocket kline event to MarketData.
func convertWsKlineToMarketData(event *BinanceWsKlineEvent) types.MarketData {
	open, _ := strconv.ParseFloat(event.Kline.Open, 64)
	high, _ := strconv.ParseFloat(event.Kline.High, 64)
	low, _ := strconv.ParseFloat(event.Kline.Low, 64)
	closePrice, _ := strconv.ParseFloat(event.Kline.Close, 64)
	volume, _ := strconv.ParseFloat(event.Kline.Volume, 64)

	return types.MarketData{
		Id:     "",
		Symbol: event.Symbol,
		Time:   time.UnixMilli(event.Kline.StartTime),
		Open:   open,
		High:   high,
		Low:    low,
		Close:  closePrice,
		Volume: volume,
	}
}

// isValidBinanceInterval validates that the interval is supported by Binance.
func isValidBinanceInterval(interval string) bool {
	validIntervals := map[string]bool{
		"1s": true,
		"1m": true, "3m": true, "5m": true, "15m": true, "30m": true,
		"1h": true, "2h": true, "4h": true, "6h": true, "8h": true, "12h": true,
		"1d": true, "3d": true, "1w": true, "1M": true,
	}

	return validIntervals[interval]
}

// cleanupFileIfExists removes the output file if it exists.
// This is used to clean up when download fails and no data was written.
func (c *BinanceClient) cleanupFileIfExists() {
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

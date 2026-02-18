package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// mockStreamAPIClient implements BinanceAPIClient for stream testing.
// It returns valid prices for all symbols by default.
type mockStreamAPIClient struct {
	prices    []*SymbolPrice
	pricesErr error
}

func (m *mockStreamAPIClient) NewKlinesService() BinanceKlinesService {
	return nil // Not used in stream tests
}

func (m *mockStreamAPIClient) NewListPricesService() BinanceListPricesService {
	return &mockStreamListPricesService{client: m}
}

type mockStreamListPricesService struct {
	client  *mockStreamAPIClient
	symbols []string
}

func (m *mockStreamListPricesService) Symbols(symbols []string) BinanceListPricesService {
	m.symbols = symbols
	return m
}

func (m *mockStreamListPricesService) Do(_ context.Context) ([]*SymbolPrice, error) {
	if m.client.pricesErr != nil {
		return nil, m.client.pricesErr
	}
	if m.client.prices != nil {
		return m.client.prices, nil
	}
	// Default: return prices for all requested symbols (valid symbols)
	result := make([]*SymbolPrice, len(m.symbols))
	for i, sym := range m.symbols {
		result[i] = &SymbolPrice{Symbol: sym, Price: "100.00"}
	}
	return result, nil
}

// mockBinanceWebSocketService implements BinanceWebSocketService for testing.
type mockBinanceWebSocketService struct {
	events     []*BinanceWsKlineEvent // Events to emit
	errors     []error                // Errors to emit
	startError error                  // Error on WsKlineServe call
	eventDelay time.Duration          // Delay between events
}

func (m *mockBinanceWebSocketService) WsKlineServe(
	symbol string,
	interval string,
	handler WsKlineHandler,
	errHandler WsErrorHandler,
) (doneC chan struct{}, stopC chan struct{}, err error) {
	if m.startError != nil {
		return nil, nil, m.startError
	}

	doneC = make(chan struct{})
	stopC = make(chan struct{})

	go func() {
		defer close(doneC)

		// Emit configured events
		for _, event := range m.events {
			select {
			case <-stopC:
				return
			default:
				if m.eventDelay > 0 {
					time.Sleep(m.eventDelay)
				}
				handler(event)
			}
		}

		// Emit configured errors
		for _, err := range m.errors {
			errHandler(err)
		}

		// Wait for stop signal, but avoid blocking forever in tests
		select {
		case <-stopC:
		case <-time.After(5 * time.Second):
		}
	}()

	return doneC, stopC, nil
}

type BinanceStreamTestSuite struct {
	suite.Suite
}

func TestBinanceStreamSuite(t *testing.T) {
	suite.Run(t, new(BinanceStreamTestSuite))
}

func (suite *BinanceStreamTestSuite) TestStreamSingleSymbol() {
	// Note: Stream only emits finalized candles (IsFinal=true)
	// Both events must have IsFinal=true to be yielded
	events := []*BinanceWsKlineEvent{
		{
			Symbol: "BTCUSDT",
			Kline: BinanceWsKline{
				StartTime: 1704067200000,
				Open:      "42000.50",
				High:      "42500.00",
				Low:       "41800.00",
				Close:     "42300.00",
				Volume:    "1000.5",
				IsFinal:   true, // Must be true to be yielded
			},
		},
		{
			Symbol: "BTCUSDT",
			Kline: BinanceWsKline{
				StartTime: 1704067260000,
				Open:      "42300.00",
				High:      "42600.00",
				Low:       "42200.00",
				Close:     "42550.00",
				Volume:    "800.25",
				IsFinal:   true,
			},
		},
	}

	mockWs := &mockBinanceWebSocketService{events: events}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var received []struct {
		symbol string
		open   float64
		close  float64
	}

	for data, err := range client.Stream(ctx) {
		if err != nil {
			break
		}
		received = append(received, struct {
			symbol string
			open   float64
			close  float64
		}{
			symbol: data.Symbol,
			open:   data.Open,
			close:  data.Close,
		})
	}

	suite.Len(received, 2)
	suite.Equal("BTCUSDT", received[0].symbol)
	suite.InDelta(42000.50, received[0].open, 0.01)
	suite.InDelta(42300.00, received[0].close, 0.01)
	suite.Equal("BTCUSDT", received[1].symbol)
	suite.InDelta(42300.00, received[1].open, 0.01)
	suite.InDelta(42550.00, received[1].close, 0.01)
}

func (suite *BinanceStreamTestSuite) TestStreamMultipleSymbols() {
	// Create a mock that returns events for both symbols
	// Note: Stream only emits finalized candles (IsFinal=true)
	mockWs := &mockBinanceWebSocketService{
		events: []*BinanceWsKlineEvent{
			{
				Symbol: "BTCUSDT",
				Kline: BinanceWsKline{
					StartTime: 1704067200000,
					Open:      "42000.00",
					High:      "42500.00",
					Low:       "41800.00",
					Close:     "42300.00",
					Volume:    "1000.0",
					IsFinal:   true, // Must be true to be yielded
				},
			},
		},
	}

	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT", "ETHUSDT"}, "1m")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var received int
	for _, err := range client.Stream(ctx) {
		if err != nil {
			break
		}
		received++
	}

	// Should receive at least some data (exact count depends on timing)
	suite.GreaterOrEqual(received, 1)
}

func (suite *BinanceStreamTestSuite) TestStreamInvalidInterval() {
	mockWs := &mockBinanceWebSocketService{}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "2m")

	ctx := context.Background()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx) {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "invalid interval")
}

func (suite *BinanceStreamTestSuite) TestStreamEmptySymbols() {
	mockWs := &mockBinanceWebSocketService{}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{}, "1m")

	ctx := context.Background()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx) {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "no symbols provided")
}

func (suite *BinanceStreamTestSuite) TestStreamContextCancellation() {
	// Create events with a delay to ensure context cancellation happens during streaming
	events := []*BinanceWsKlineEvent{
		{
			Symbol: "BTCUSDT",
			Kline: BinanceWsKline{
				StartTime: 1704067200000,
				Open:      "42000.00",
				High:      "42500.00",
				Low:       "41800.00",
				Close:     "42300.00",
				Volume:    "1000.0",
			},
		},
	}

	mockWs := &mockBinanceWebSocketService{
		events:     events,
		eventDelay: 50 * time.Millisecond,
	}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	ctx, cancel := context.WithCancel(context.Background())

	// Cancel after a short delay
	go func() {
		time.Sleep(100 * time.Millisecond)
		cancel()
	}()

	iterationCount := 0
	for range client.Stream(ctx) {
		iterationCount++
		if iterationCount > 10 {
			break // Safety break
		}
	}

	// Stream should have stopped due to context cancellation
	suite.LessOrEqual(iterationCount, 10)
}

func (suite *BinanceStreamTestSuite) TestStreamConnectionError_TerminatesWithoutTimeout() {
	// This test verifies that the stream terminates on its own when all websocket
	// connections fail, WITHOUT relying on a context timeout. Before the fix,
	// the stream would hang forever because errChan and dataChan were never closed.
	mockWs := &mockBinanceWebSocketService{
		startError: errors.New("bad handshake"),
	}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	// Use a background context with NO timeout - the stream must terminate on its own
	ctx := context.Background()

	streamDone := make(chan struct{})
	var gotError bool
	var errorMsg string

	go func() {
		defer close(streamDone)
		for _, err := range client.Stream(ctx) {
			if err != nil {
				gotError = true
				errorMsg = err.Error()
			}
		}
	}()

	// Stream should terminate within a reasonable time without any context cancellation
	select {
	case <-streamDone:
		// Stream terminated on its own - correct behavior
	case <-time.After(5 * time.Second):
		suite.Fail("Stream hung - did not terminate after all connections failed")
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "failed to start websocket")
	suite.Contains(errorMsg, "bad handshake")
}

func (suite *BinanceStreamTestSuite) TestStreamConnectionError() {
	mockWs := &mockBinanceWebSocketService{
		startError: errors.New("connection refused"),
	}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx) {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "failed to start websocket")
	suite.Contains(errorMsg, "connection refused")
}

func (suite *BinanceStreamTestSuite) TestStreamWebSocketError() {
	mockWs := &mockBinanceWebSocketService{
		events: []*BinanceWsKlineEvent{},
		errors: []error{errors.New("websocket disconnected")},
	}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx) {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "websocket error")
	suite.Contains(errorMsg, "websocket disconnected")
}

func (suite *BinanceStreamTestSuite) TestConvertWsKlineToMarketData() {
	event := &BinanceWsKlineEvent{
		Symbol: "ETHUSDT",
		Kline: BinanceWsKline{
			StartTime: 1704067200000,
			Open:      "2300.50",
			High:      "2350.00",
			Low:       "2280.00",
			Close:     "2340.00",
			Volume:    "500.25",
		},
	}

	data := convertWsKlineToMarketData(event)

	suite.Equal("ETHUSDT", data.Symbol)
	suite.Equal(time.UnixMilli(1704067200000), data.Time)
	suite.InDelta(2300.50, data.Open, 0.01)
	suite.InDelta(2350.00, data.High, 0.01)
	suite.InDelta(2280.00, data.Low, 0.01)
	suite.InDelta(2340.00, data.Close, 0.01)
	suite.InDelta(500.25, data.Volume, 0.01)
}

func (suite *BinanceStreamTestSuite) TestIsValidBinanceInterval() {
	// Valid intervals
	suite.True(isValidBinanceInterval("1s"))
	suite.True(isValidBinanceInterval("1m"))
	suite.True(isValidBinanceInterval("3m"))
	suite.True(isValidBinanceInterval("5m"))
	suite.True(isValidBinanceInterval("15m"))
	suite.True(isValidBinanceInterval("30m"))
	suite.True(isValidBinanceInterval("1h"))
	suite.True(isValidBinanceInterval("2h"))
	suite.True(isValidBinanceInterval("4h"))
	suite.True(isValidBinanceInterval("6h"))
	suite.True(isValidBinanceInterval("8h"))
	suite.True(isValidBinanceInterval("12h"))
	suite.True(isValidBinanceInterval("1d"))
	suite.True(isValidBinanceInterval("3d"))
	suite.True(isValidBinanceInterval("1w"))
	suite.True(isValidBinanceInterval("1M"))

	// Invalid intervals
	suite.False(isValidBinanceInterval("2m"))
	suite.False(isValidBinanceInterval("7m"))
	suite.False(isValidBinanceInterval("3h"))
	suite.False(isValidBinanceInterval("2d"))
	suite.False(isValidBinanceInterval("2w"))
	suite.False(isValidBinanceInterval("2M"))
	suite.False(isValidBinanceInterval("invalid"))
	suite.False(isValidBinanceInterval(""))
}

func (suite *BinanceStreamTestSuite) TestValidateSymbolsValid() {
	mockAPI := &mockStreamAPIClient{
		prices: []*SymbolPrice{
			{Symbol: "BTCUSDT", Price: "42000.00"},
			{Symbol: "ETHUSDT", Price: "2300.00"},
		},
	}
	client := NewBinanceClientWithWebSocket(mockAPI, nil, []string{"BTCUSDT", "ETHUSDT"}, "1m")

	err := client.ValidateSymbols(context.Background(), []string{"BTCUSDT", "ETHUSDT"})
	suite.NoError(err)
}

func (suite *BinanceStreamTestSuite) TestValidateSymbolsInvalid() {
	mockAPI := &mockStreamAPIClient{
		// Return prices only for BTCUSDT, not INVALID
		prices: []*SymbolPrice{
			{Symbol: "BTCUSDT", Price: "42000.00"},
		},
	}
	client := NewBinanceClientWithWebSocket(mockAPI, nil, []string{"BTCUSDT", "INVALID"}, "1m")

	err := client.ValidateSymbols(context.Background(), []string{"BTCUSDT", "INVALID"})
	suite.Error(err)
	suite.Contains(err.Error(), "invalid symbols")
	suite.Contains(err.Error(), "INVALID")
}

func (suite *BinanceStreamTestSuite) TestValidateSymbolsAPIError() {
	mockAPI := &mockStreamAPIClient{
		pricesErr: errors.New("API error: rate limited"),
	}
	client := NewBinanceClientWithWebSocket(mockAPI, nil, []string{"BTCUSDT"}, "1m")

	err := client.ValidateSymbols(context.Background(), []string{"BTCUSDT"})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to validate symbols")
	suite.Contains(err.Error(), "rate limited")
}

func (suite *BinanceStreamTestSuite) TestValidateSymbolsEmpty() {
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, nil, []string{}, "1m")

	err := client.ValidateSymbols(context.Background(), []string{})
	suite.NoError(err)
}

func (suite *BinanceStreamTestSuite) TestStreamWithInvalidSymbol() {
	mockAPI := &mockStreamAPIClient{
		// Return empty prices - simulating invalid symbol
		prices: []*SymbolPrice{},
	}
	mockWs := &mockBinanceWebSocketService{}
	client := NewBinanceClientWithWebSocket(mockAPI, mockWs, []string{"INVALID"}, "1m")

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx) {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "invalid symbols")
}

func (suite *BinanceStreamTestSuite) TestStreamOneSecondInterval() {
	// Test streaming with 1s interval - Binance supports 1 second candles
	events := []*BinanceWsKlineEvent{
		{
			Symbol: "BTCUSDT",
			Kline: BinanceWsKline{
				StartTime: 1704067200000,
				Open:      "42000.50",
				High:      "42001.00",
				Low:       "41999.00",
				Close:     "42000.75",
				Volume:    "10.5",
				IsFinal:   true,
			},
		},
	}

	mockWs := &mockBinanceWebSocketService{events: events}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1s")

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var received int
	for data, err := range client.Stream(ctx) {
		if err != nil {
			suite.Fail("unexpected error: %v", err)
			break
		}
		suite.Equal("BTCUSDT", data.Symbol)
		suite.InDelta(42000.50, data.Open, 0.01)
		suite.InDelta(42000.75, data.Close, 0.01)
		received++
	}

	suite.Equal(1, received)
}

func (suite *BinanceStreamTestSuite) TestSetOnStatusChange() {
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, nil, []string{"BTCUSDT"}, "1m")

	client.SetOnStatusChange(func(_ types.ProviderConnectionStatus) {
		// Callback registered successfully
	})

	// Verify callback is set
	suite.NotNil(client.onStatusChange)
}

func (suite *BinanceStreamTestSuite) TestStreamEmitsConnectedStatus() {
	events := []*BinanceWsKlineEvent{
		{
			Symbol: "BTCUSDT",
			Kline: BinanceWsKline{
				StartTime: 1704067200000,
				Open:      "42000.50",
				High:      "42500.00",
				Low:       "41800.00",
				Close:     "42300.00",
				Volume:    "1000.5",
				IsFinal:   true,
			},
		},
	}

	mockWs := &mockBinanceWebSocketService{events: events}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	var statusChanges []types.ProviderConnectionStatus
	client.SetOnStatusChange(func(status types.ProviderConnectionStatus) {
		statusChanges = append(statusChanges, status)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Consume the stream to trigger the connection
	for range client.Stream(ctx) {
		// Just iterate through to completion
	}

	// Should have received at least one connected and one disconnected status
	suite.GreaterOrEqual(len(statusChanges), 1, "Should have received at least one status change")
	suite.Contains(statusChanges, types.ProviderStatusConnected, "Should have received connected status")
}

func (suite *BinanceStreamTestSuite) TestStreamEmitsDisconnectedOnError() {
	mockWs := &mockBinanceWebSocketService{
		startError: errors.New("connection refused"),
	}
	client := NewBinanceClientWithWebSocket(&mockStreamAPIClient{}, mockWs, []string{"BTCUSDT"}, "1m")

	var statusChanges []types.ProviderConnectionStatus
	client.SetOnStatusChange(func(status types.ProviderConnectionStatus) {
		statusChanges = append(statusChanges, status)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Stream should fail to connect
	for range client.Stream(ctx) {
		// Just iterate through to completion
	}

	// Should have received disconnected status on failure
	suite.Contains(statusChanges, types.ProviderStatusDisconnected, "Should have received disconnected status on connection error")
}

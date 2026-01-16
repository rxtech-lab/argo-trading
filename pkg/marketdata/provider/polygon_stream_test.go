package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	polygonws "github.com/polygon-io/client-go/websocket"
	"github.com/polygon-io/client-go/websocket/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// mockPolygonWebSocketService implements PolygonWebSocketService for testing.
type mockPolygonWebSocketService struct {
	events       []any   // Events to emit (models.EquityAgg, etc.)
	errors       []error // Errors to emit
	connectError error   // Error on Connect() call
	outputChan   chan any
	errorChan    chan error
	closed       bool
}

func newMockPolygonWebSocketService() *mockPolygonWebSocketService {
	return &mockPolygonWebSocketService{
		outputChan: make(chan any, 100),
		errorChan:  make(chan error, 10),
	}
}

func (m *mockPolygonWebSocketService) Connect() error {
	if m.connectError != nil {
		return m.connectError
	}

	// Start emitting events in background
	go func() {
		for _, event := range m.events {
			m.outputChan <- event
		}
		for _, err := range m.errors {
			m.errorChan <- err
		}
	}()

	return nil
}

func (m *mockPolygonWebSocketService) Subscribe(topic polygonws.Topic, tickers ...string) error {
	return nil
}

func (m *mockPolygonWebSocketService) Unsubscribe(topic polygonws.Topic, tickers ...string) error {
	return nil
}

func (m *mockPolygonWebSocketService) Output() <-chan any {
	return m.outputChan
}

func (m *mockPolygonWebSocketService) Error() <-chan error {
	return m.errorChan
}

func (m *mockPolygonWebSocketService) Close() {
	if !m.closed {
		m.closed = true
		close(m.outputChan)
		close(m.errorChan)
	}
}

type PolygonStreamTestSuite struct {
	suite.Suite
}

func TestPolygonStreamSuite(t *testing.T) {
	suite.Run(t, new(PolygonStreamTestSuite))
}

func (suite *PolygonStreamTestSuite) TestStreamSingleSymbol() {
	events := []any{
		models.EquityAgg{
			EventType: models.EventType{
				EventType: "AM",
			},
			Symbol:         "AAPL",
			Open:           150.00,
			High:           152.00,
			Low:            149.50,
			Close:          151.50,
			Volume:         1000000,
			StartTimestamp: 1704067200000,
		},
		models.EquityAgg{
			EventType: models.EventType{
				EventType: "AM",
			},
			Symbol:         "AAPL",
			Open:           151.50,
			High:           153.00,
			Low:            151.00,
			Close:          152.75,
			Volume:         800000,
			StartTimestamp: 1704067260000,
		},
	}

	mockWs := newMockPolygonWebSocketService()
	mockWs.events = events

	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var received []types.MarketData
	for data, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
		if err != nil {
			break
		}
		received = append(received, data)
	}

	suite.Len(received, 2)
	suite.Equal("AAPL", received[0].Symbol)
	suite.InDelta(150.00, received[0].Open, 0.01)
	suite.InDelta(151.50, received[0].Close, 0.01)
}

func (suite *PolygonStreamTestSuite) TestStreamMultipleSymbols() {
	events := []any{
		models.EquityAgg{
			Symbol:         "AAPL",
			Open:           150.00,
			Close:          151.50,
			High:           152.00,
			Low:            149.50,
			Volume:         1000000,
			StartTimestamp: 1704067200000,
		},
		models.EquityAgg{
			Symbol:         "GOOGL",
			Open:           140.00,
			Close:          141.50,
			High:           142.00,
			Low:            139.50,
			Volume:         500000,
			StartTimestamp: 1704067200000,
		},
	}

	mockWs := newMockPolygonWebSocketService()
	mockWs.events = events

	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	symbolsSeen := make(map[string]bool)
	for data, err := range client.Stream(ctx, []string{"AAPL", "GOOGL"}, "1m") {
		if err != nil {
			break
		}
		symbolsSeen[data.Symbol] = true
	}

	suite.True(symbolsSeen["AAPL"])
	suite.True(symbolsSeen["GOOGL"])
}

func (suite *PolygonStreamTestSuite) TestStreamConnectionError() {
	mockWs := newMockPolygonWebSocketService()
	mockWs.connectError = errors.New("authentication failed")

	client := NewPolygonClientWithWebSocket("invalid-api-key", mockWs)

	ctx := context.Background()
	var gotError bool
	var errorMsg string

	for _, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
		if err != nil {
			gotError = true
			errorMsg = err.Error()
			break
		}
	}

	suite.True(gotError)
	suite.Contains(errorMsg, "failed to connect")
}

func (suite *PolygonStreamTestSuite) TestStreamEmptySymbols() {
	mockWs := newMockPolygonWebSocketService()
	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	ctx := context.Background()
	var gotError bool

	for _, err := range client.Stream(ctx, []string{}, "1m") {
		if err != nil {
			gotError = true
			break
		}
	}

	suite.True(gotError)
}

func (suite *PolygonStreamTestSuite) TestStreamContextCancellation() {
	mockWs := newMockPolygonWebSocketService()
	// Don't add events - let the context cancellation terminate

	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	ctx, cancel := context.WithCancel(context.Background())

	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	iterCount := 0
	for range client.Stream(ctx, []string{"AAPL"}, "1m") {
		iterCount++
		if iterCount > 10 {
			break
		}
	}

	suite.LessOrEqual(iterCount, 10)
}

func (suite *PolygonStreamTestSuite) TestStreamWebSocketError() {
	mockWs := newMockPolygonWebSocketService()
	mockWs.errors = []error{errors.New("websocket disconnected")}

	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	var gotError bool
	var errorMsg string
	for _, err := range client.Stream(ctx, []string{"AAPL"}, "1m") {
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

func (suite *PolygonStreamTestSuite) TestConvertEquityAggToMarketData() {
	agg := &models.EquityAgg{
		Symbol:         "MSFT",
		Open:           380.50,
		High:           385.00,
		Low:            378.00,
		Close:          383.75,
		Volume:         500000,
		StartTimestamp: 1704067200000,
	}

	data := convertEquityAggToMarketData(agg)

	suite.Equal("MSFT", data.Symbol)
	suite.Equal(time.UnixMilli(1704067200000), data.Time)
	suite.InDelta(380.50, data.Open, 0.01)
	suite.InDelta(385.00, data.High, 0.01)
	suite.InDelta(378.00, data.Low, 0.01)
	suite.InDelta(383.75, data.Close, 0.01)
	suite.InDelta(500000, data.Volume, 0.01)
}

func (suite *PolygonStreamTestSuite) TestConvertIntervalToPolygonTopic() {
	topic, err := convertIntervalToPolygonTopic("1s")
	suite.NoError(err)
	suite.Equal(polygonws.StocksSecAggs, topic)

	topic, err = convertIntervalToPolygonTopic("1m")
	suite.NoError(err)
	suite.Equal(polygonws.StocksMinAggs, topic)

	// Other intervals should default to minute aggregates
	topic, err = convertIntervalToPolygonTopic("5m")
	suite.NoError(err)
	suite.Equal(polygonws.StocksMinAggs, topic)
}

func (suite *PolygonStreamTestSuite) TestSetOnStatusChange() {
	client := NewPolygonClientWithWebSocket("test-api-key", nil)

	client.SetOnStatusChange(func(_ types.ProviderConnectionStatus) {
		// Callback registered successfully
	})

	// Verify callback is set
	suite.NotNil(client.onStatusChange)
}

func (suite *PolygonStreamTestSuite) TestStreamEmitsConnectedStatus() {
	events := []any{
		models.EquityAgg{
			Symbol:         "AAPL",
			Open:           150.00,
			High:           152.00,
			Low:            149.50,
			Close:          151.50,
			Volume:         1000000,
			StartTimestamp: 1704067200000,
		},
	}

	mockWs := newMockPolygonWebSocketService()
	mockWs.events = events

	client := NewPolygonClientWithWebSocket("test-api-key", mockWs)

	var statusChanges []types.ProviderConnectionStatus
	client.SetOnStatusChange(func(status types.ProviderConnectionStatus) {
		statusChanges = append(statusChanges, status)
	})

	ctx, cancel := context.WithTimeout(context.Background(), 300*time.Millisecond)
	defer cancel()

	// Consume the stream to trigger the connection
	for range client.Stream(ctx, []string{"AAPL"}, "1m") {
		// Just iterate through to completion
	}

	// Should have received connected and disconnected status
	suite.GreaterOrEqual(len(statusChanges), 1, "Should have received at least one status change")
	suite.Contains(statusChanges, types.ProviderStatusConnected, "Should have received connected status")
}

func (suite *PolygonStreamTestSuite) TestStreamEmitsDisconnectedOnError() {
	mockWs := newMockPolygonWebSocketService()
	mockWs.connectError = errors.New("authentication failed")

	client := NewPolygonClientWithWebSocket("invalid-api-key", mockWs)

	var statusChanges []types.ProviderConnectionStatus
	client.SetOnStatusChange(func(status types.ProviderConnectionStatus) {
		statusChanges = append(statusChanges, status)
	})

	ctx := context.Background()

	// Stream should fail to connect
	for range client.Stream(ctx, []string{"AAPL"}, "1m") {
		// Just iterate through to completion
	}

	// Should have received disconnected status on failure
	suite.Contains(statusChanges, types.ProviderStatusDisconnected, "Should have received disconnected status on connection error")
}

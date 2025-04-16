package wasm

import (
	"context"
	"testing"
	"time"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

// StrategyApiTestSuite is a test suite for testing the strategy API
type StrategyApiTestSuite struct {
	suite.Suite
	ctrl           *gomock.Controller
	mockTrading    *mocks.MockTradingSystem
	mockIndicators *mocks.MockIndicatorRegistry
	mockDataSource *mocks.MockDataSource
	mockCache      *mocks.MockCache
	mockMarker     *mocks.MockMarker
	runtimeContext *runtime.RuntimeContext
	api            strategy.StrategyApi
}

// SetupTest runs before each test
func (suite *StrategyApiTestSuite) SetupTest() {
	suite.ctrl = gomock.NewController(suite.T())
	suite.mockTrading = mocks.NewMockTradingSystem(suite.ctrl)
	suite.mockIndicators = mocks.NewMockIndicatorRegistry(suite.ctrl)
	suite.mockDataSource = mocks.NewMockDataSource(suite.ctrl)
	suite.mockCache = mocks.NewMockCache(suite.ctrl)
	suite.mockMarker = mocks.NewMockMarker(suite.ctrl)

	suite.runtimeContext = &runtime.RuntimeContext{
		TradingSystem:     suite.mockTrading,
		IndicatorRegistry: suite.mockIndicators,
		DataSource:        suite.mockDataSource,
		Cache:             suite.mockCache,
		Marker:            suite.mockMarker,
	}

	suite.api = NewWasmStrategyApi(suite.runtimeContext)
}

// TearDownTest runs after each test
func (suite *StrategyApiTestSuite) TearDownTest() {
	suite.ctrl.Finish()
}

// TestStrategyApiSuite runs the test suite
func TestStrategyApiSuite(t *testing.T) {
	suite.Run(t, new(StrategyApiTestSuite))
}

// TestCancelAllOrders tests the CancelAllOrders method
func (suite *StrategyApiTestSuite) TestCancelAllOrders() {
	// Setup expectations
	suite.mockTrading.EXPECT().
		CancelAllOrders().
		Return(nil)

	// Execute test
	_, err := suite.api.CancelAllOrders(context.Background(), &emptypb.Empty{})
	suite.NoError(err)
}

// TestCancelOrder tests the CancelOrder method
func (suite *StrategyApiTestSuite) TestCancelOrder() {
	orderID := "test-order-id"

	// Setup expectations
	suite.mockTrading.EXPECT().
		CancelOrder(orderID).
		Return(nil)

	// Execute test
	_, err := suite.api.CancelOrder(context.Background(), &strategy.CancelOrderRequest{
		OrderId: orderID,
	})
	suite.NoError(err)
}

// TestGetOrderStatus tests the GetOrderStatus method
func (suite *StrategyApiTestSuite) TestGetOrderStatus() {
	orderID := "test-order-id"

	// Setup expectations
	suite.mockTrading.EXPECT().
		GetOrderStatus(orderID).
		Return(types.OrderStatusFilled, nil)

	// Execute test
	response, err := suite.api.GetOrderStatus(context.Background(), &strategy.GetOrderStatusRequest{
		OrderId: orderID,
	})

	suite.NoError(err)
	suite.Equal(strategy.OrderStatus_ORDER_STATUS_FILLED, response.Status)
}

// TestGetPosition tests the GetPosition method
func (suite *StrategyApiTestSuite) TestGetPosition() {
	symbol := "BTCUSDT"
	position := types.Position{
		Symbol:                       symbol,
		TotalLongPositionQuantity:    1.0,
		TotalLongInPositionQuantity:  1.0,
		TotalLongOutPositionQuantity: 0.0,
		TotalLongInPositionAmount:    50000.0,
		TotalLongOutPositionAmount:   0.0,
		TotalLongInFee:               0.1,
		TotalLongOutFee:              0.0,
		OpenTimestamp:                time.Now(),
		StrategyName:                 "test-strategy",
	}

	// Setup expectations
	suite.mockTrading.EXPECT().
		GetPosition(symbol).
		Return(position, nil)

	// Execute test
	response, err := suite.api.GetPosition(context.Background(), &strategy.GetPositionRequest{
		Symbol: symbol,
	})

	suite.NoError(err)
	suite.Equal(symbol, response.Symbol)
	suite.Equal(position.TotalLongPositionQuantity, response.Quantity)
	suite.Equal(position.StrategyName, response.StrategyName)
}

// TestGetPositions tests the GetPositions method
func (suite *StrategyApiTestSuite) TestGetPositions() {
	positions := []types.Position{
		{
			Symbol:                       "BTCUSDT",
			TotalLongPositionQuantity:    1.0,
			TotalLongInPositionQuantity:  1.0,
			TotalLongOutPositionQuantity: 0.0,
			TotalLongInPositionAmount:    50000.0,
			TotalLongOutPositionAmount:   0.0,
			TotalLongInFee:               0.1,
			TotalLongOutFee:              0.0,
			OpenTimestamp:                time.Now(),
			StrategyName:                 "test-strategy",
		},
	}

	// Setup expectations
	suite.mockTrading.EXPECT().
		GetPositions().
		Return(positions, nil)

	// Execute test
	response, err := suite.api.GetPositions(context.Background(), &emptypb.Empty{})

	suite.NoError(err)
	suite.Len(response.Positions, 1)
	suite.Equal(positions[0].Symbol, response.Positions[0].Symbol)
	suite.Equal(positions[0].TotalLongPositionQuantity, response.Positions[0].Quantity)
}

// TestReadLastData tests the ReadLastData method
func (suite *StrategyApiTestSuite) TestReadLastData() {
	symbol := "BTCUSDT"
	data := types.MarketData{
		Symbol: symbol,
		High:   50000.0,
		Low:    49000.0,
		Open:   49500.0,
		Close:  49800.0,
		Volume: 100.0,
		Time:   time.Now(),
	}

	// Setup expectations
	suite.mockDataSource.EXPECT().
		ReadLastData(symbol).
		Return(data, nil)

	// Execute test
	response, err := suite.api.ReadLastData(context.Background(), &strategy.ReadLastDataRequest{
		Symbol: symbol,
	})

	suite.NoError(err)
	suite.Equal(data.Symbol, response.Symbol)
	suite.Equal(data.High, response.High)
	suite.Equal(data.Low, response.Low)
}

// TestSetCache tests the SetCache method
func (suite *StrategyApiTestSuite) TestSetCache() {
	tests := []struct {
		name        string
		key         string
		inputValue  string
		expectError bool
	}{
		{
			name:        "String value",
			key:         "string-key",
			inputValue:  "test-value",
			expectError: false,
		},
		{
			name:        "Number as string",
			key:         "number-key",
			inputValue:  "42.5",
			expectError: false,
		},
		{
			name:        "JSON object",
			key:         "object-key",
			inputValue:  `{"name":"test","price":100.5,"tags":["crypto","trading"]}`,
			expectError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Setup expectations
			suite.mockCache.EXPECT().
				Set(tc.key, tc.inputValue).
				Return(nil)

			// Execute test
			_, err := suite.api.SetCache(context.Background(), &strategy.SetRequest{
				Key:   tc.key,
				Value: tc.inputValue,
			})

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
			}
		})
	}
}

// TestGetCache tests the GetCache method
func (suite *StrategyApiTestSuite) TestGetCache() {
	tests := []struct {
		name        string
		key         string
		value       interface{}
		expectError bool
		expectedRes string
	}{
		{
			name:        "String value",
			key:         "string-key",
			value:       "test-value",
			expectError: false,
			expectedRes: "test-value",
		},
		{
			name:        "Number value",
			key:         "number-key",
			value:       42.5,
			expectError: false,
			expectedRes: "42.5",
		},
		{
			name: "Object value",
			key:  "object-key",
			value: map[string]interface{}{
				"name":  "test",
				"price": 100.5,
				"tags":  []string{"crypto", "trading"},
			},
			expectError: false,
			expectedRes: `{"name":"test","price":100.5,"tags":["crypto","trading"]}`,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			// Setup expectations
			suite.mockCache.EXPECT().
				Get(tc.key).
				Return(tc.value, true)

			// Execute test
			response, err := suite.api.GetCache(context.Background(), &strategy.GetRequest{
				Key: tc.key,
			})

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectedRes, response.Value)
			}
		})
	}
}

// TestGetMarkers tests the GetMarkers method
func (suite *StrategyApiTestSuite) TestGetMarkers() {
	now := time.Now()
	markers := []types.Mark{
		{
			Signal: types.Signal{
				Symbol: "BTCUSDT",
				Type:   types.SignalTypeBuyLong,
				Time:   now,
			},
			Reason: "test reason",
		},
	}

	// Setup expectations
	suite.mockMarker.EXPECT().
		GetMarks().
		Return(markers, nil)

	// Execute test
	response, err := suite.api.GetMarkers(context.Background(), &emptypb.Empty{})

	suite.NoError(err)
	suite.Len(response.Markers, 1)
	suite.Equal(markers[0].Signal.Symbol, response.Markers[0].MarketData.Symbol)
	suite.Equal(strategy.SignalType_SIGNAL_TYPE_BUY_LONG, response.Markers[0].Signal)
}

func (suite *StrategyApiTestSuite) TestCount() {
	startTime := time.Now().UTC()
	endTime := startTime.Add(time.Hour)
	count := int(100)

	suite.mockDataSource.EXPECT().Count(
		optional.Some(startTime),
		optional.Some(endTime),
	).Return(count, nil)

	response, err := suite.api.Count(context.Background(), &strategy.CountRequest{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
	})
	suite.NoError(err)
	suite.Equal(int32(count), response.Count)
}

func (suite *StrategyApiTestSuite) TestGetRange() {
	startTime := time.Now().UTC()
	endTime := startTime.Add(time.Hour)
	interval := datasource.Interval1m
	data := []types.MarketData{
		{
			Symbol: "BTCUSDT",
			High:   50000.0,
			Low:    49000.0,
			Open:   49500.0,
			Close:  49800.0,
			Volume: 100.0,
			Time:   startTime,
		},
	}

	suite.mockDataSource.EXPECT().GetRange(
		startTime,
		endTime,
		optional.Some(interval),
	).Return(data, nil)

	response, err := suite.api.GetRange(context.Background(), &strategy.GetRangeRequest{
		StartTime: timestamppb.New(startTime),
		EndTime:   timestamppb.New(endTime),
		Interval:  strategy.Interval_INTERVAL_1M,
	})
	suite.NoError(err)
	suite.Len(response.Data, 1)
	suite.Equal("BTCUSDT", response.Data[0].Symbol)
	suite.Equal(50000.0, response.Data[0].High)
}

func (suite *StrategyApiTestSuite) TestGetSignal() {
	api := NewWasmStrategyApi(suite.runtimeContext)

	// Test case 1: Get signal successfully
	indicatorType := strategy.IndicatorType_INDICATOR_RSI
	marketData := &strategy.MarketData{
		Symbol: "BTCUSDT",
		High:   50000.0,
		Low:    49000.0,
		Open:   49500.0,
		Close:  49800.0,
		Volume: 100.0,
		Time:   timestamppb.Now(),
	}

	expectedSignal := types.Signal{
		Time:      time.Now(),
		Type:      types.SignalTypeBuyLong,
		Name:      "test",
		Reason:    "test reason",
		RawValue:  map[string]float64{"rsi": 30.0},
		Symbol:    "BTCUSDT",
		Indicator: types.IndicatorTypeRSI,
	}

	mockRSI := mocks.NewMockIndicator(suite.ctrl)
	mockRSI.EXPECT().
		GetSignal(gomock.Any(), gomock.Any()).
		Return(expectedSignal, nil).
		AnyTimes()
	mockRSI.EXPECT().
		Name().
		Return(types.IndicatorTypeRSI).
		AnyTimes()

	suite.mockIndicators.EXPECT().
		GetIndicator(gomock.Any()).
		Return(mockRSI, nil).
		AnyTimes()

	response, err := api.GetSignal(context.Background(), &strategy.GetSignalRequest{
		IndicatorType: indicatorType,
		MarketData:    marketData,
	})

	suite.NoError(err)
	suite.Equal(strategy.SignalType_SIGNAL_TYPE_BUY_LONG, response.Type)
	suite.Equal(expectedSignal.Name, response.Name)
	suite.Equal(expectedSignal.Reason, response.Reason)
	suite.Equal(expectedSignal.Symbol, response.Symbol)
}

func (suite *StrategyApiTestSuite) TestPlaceOrder() {
	order := &strategy.ExecuteOrder{
		Id:           "test-order",
		Symbol:       "BTCUSDT",
		Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
		OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
		Price:        50000.0,
		StrategyName: "test-strategy",
		Quantity:     1.0,
		Reason: &strategy.Reason{
			Reason:  "test reason",
			Message: "test message",
		},
	}

	expectedOrder := types.ExecuteOrder{
		ID:           order.Id,
		Symbol:       order.Symbol,
		Side:         types.PurchaseTypeBuy,
		OrderType:    types.OrderTypeLimit,
		Price:        order.Price,
		StrategyName: order.StrategyName,
		Quantity:     order.Quantity,
		PositionType: types.PositionTypeLong,
		Reason: types.Reason{
			Reason:  order.Reason.Reason,
			Message: order.Reason.Message,
		},
	}

	suite.mockTrading.EXPECT().PlaceOrder(expectedOrder).Return(nil)

	_, err := suite.api.PlaceOrder(context.Background(), order)
	suite.NoError(err)
}

func (suite *StrategyApiTestSuite) TestPlaceMultipleOrders() {
	orders := []*strategy.ExecuteOrder{
		{
			Id:           "test-order-1",
			Symbol:       "BTCUSDT",
			Side:         strategy.PurchaseType_PURCHASE_TYPE_BUY,
			OrderType:    strategy.OrderType_ORDER_TYPE_LIMIT,
			Price:        50000.0,
			StrategyName: "test-strategy",
			Quantity:     1.0,
			Reason: &strategy.Reason{
				Reason:  "test reason",
				Message: "test message",
			},
		},
	}

	expectedOrders := []types.ExecuteOrder{
		{
			ID:           orders[0].Id,
			Symbol:       orders[0].Symbol,
			Side:         types.PurchaseTypeBuy,
			OrderType:    types.OrderTypeLimit,
			Price:        orders[0].Price,
			StrategyName: orders[0].StrategyName,
			Quantity:     orders[0].Quantity,
			PositionType: types.PositionTypeLong,
			Reason: types.Reason{
				Reason:  orders[0].Reason.Reason,
				Message: orders[0].Reason.Message,
			},
		},
	}

	suite.mockTrading.EXPECT().PlaceMultipleOrders(expectedOrders).Return(nil)

	_, err := suite.api.PlaceMultipleOrders(context.Background(), &strategy.PlaceMultipleOrdersRequest{
		Orders: orders,
	})
	suite.NoError(err)
}

func (suite *StrategyApiTestSuite) TestNewStrategyApi() {
	api := NewWasmStrategyApi(suite.runtimeContext)
	suite.NotNil(api)
}

// TestMark tests the Mark method
func (suite *StrategyApiTestSuite) TestMark() {
	// Create test data
	now := time.Now()
	protoTime := timestamppb.New(now)
	
	marketData := &strategy.MarketData{
		Symbol: "BTCUSDT",
		Time:   protoTime,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	reason := "Test mark reason"
	
	// Expected internal types
	expectedMarketData := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   now,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	expectedSignal := types.Signal{
		Time:   now,
		Type:   types.SignalTypeBuyLong,
		Symbol: "BTCUSDT",
		Name:   string(types.SignalTypeBuyLong),
		Reason: reason,
	}
	
	// Setup expectations
	suite.mockMarker.EXPECT().
		Mark(gomock.Any(), gomock.Any(), gomock.Eq(reason)).
		DoAndReturn(func(md types.MarketData, signal types.Signal, r string) error {
			// Verify the market data
			suite.Equal(expectedMarketData.Symbol, md.Symbol)
			suite.Equal(expectedMarketData.Time.Unix(), md.Time.Unix())
			suite.Equal(expectedMarketData.Open, md.Open)
			suite.Equal(expectedMarketData.High, md.High)
			suite.Equal(expectedMarketData.Low, md.Low)
			suite.Equal(expectedMarketData.Close, md.Close)
			suite.Equal(expectedMarketData.Volume, md.Volume)
			
			// Verify the signal
			suite.Equal(expectedSignal.Type, signal.Type)
			suite.Equal(expectedSignal.Symbol, signal.Symbol)
			suite.Equal(expectedSignal.Reason, signal.Reason)
			
			return nil
		})
	
	// Execute test
	_, err := suite.api.Mark(context.Background(), &strategy.MarkRequest{
		MarketData: marketData,
		Signal:     strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
		Reason:     reason,
	})
	
	suite.NoError(err)
}

// TestMarkWithNilMarker tests the Mark method when the marker is nil
func (suite *StrategyApiTestSuite) TestMarkWithNilMarker() {
	// Save the original marker
	originalMarker := suite.runtimeContext.Marker
	
	// Set the marker to nil
	suite.runtimeContext.Marker = nil
	
	// Create test data
	now := time.Now()
	protoTime := timestamppb.New(now)
	
	marketData := &strategy.MarketData{
		Symbol: "BTCUSDT",
		Time:   protoTime,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	// Execute test
	_, err := suite.api.Mark(context.Background(), &strategy.MarkRequest{
		MarketData: marketData,
		Signal:     strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
		Reason:     "Test mark reason",
	})
	
	// Verify error
	suite.Error(err)
	suite.Contains(err.Error(), "marker is not available")
	
	// Restore the original marker
	suite.runtimeContext.Marker = originalMarker
}

// TestMarkWithNilMarketData tests the Mark method when market data is nil
func (suite *StrategyApiTestSuite) TestMarkWithNilMarketData() {
	// Execute test
	_, err := suite.api.Mark(context.Background(), &strategy.MarkRequest{
		MarketData: nil,
		Signal:     strategy.SignalType_SIGNAL_TYPE_BUY_LONG,
		Reason:     "Test mark reason",
	})
	
	// Verify error
	suite.Error(err)
	suite.Contains(err.Error(), "market data is required")
}

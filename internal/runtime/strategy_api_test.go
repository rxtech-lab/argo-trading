package runtime

import (
	"context"
	"testing"
	"time"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/knqyf263/go-plugin/types/known/timestamppb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
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
	runtimeContext *RuntimeContext
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

	suite.runtimeContext = &RuntimeContext{
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
		Symbol:           symbol,
		Quantity:         1.0,
		TotalInQuantity:  1.0,
		TotalOutQuantity: 0.0,
		TotalInAmount:    50000.0,
		TotalOutAmount:   0.0,
		TotalInFee:       0.1,
		TotalOutFee:      0.0,
		OpenTimestamp:    time.Now(),
		StrategyName:     "test-strategy",
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
	suite.Equal(position.Quantity, response.Quantity)
	suite.Equal(position.StrategyName, response.StrategyName)
}

// TestGetPositions tests the GetPositions method
func (suite *StrategyApiTestSuite) TestGetPositions() {
	positions := []types.Position{
		{
			Symbol:           "BTCUSDT",
			Quantity:         1.0,
			TotalInQuantity:  1.0,
			TotalOutQuantity: 0.0,
			TotalInAmount:    50000.0,
			TotalOutAmount:   0.0,
			TotalInFee:       0.1,
			TotalOutFee:      0.0,
			OpenTimestamp:    time.Now(),
			StrategyName:     "test-strategy",
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
	suite.Equal(positions[0].Quantity, response.Positions[0].Quantity)
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
	key := "test-key"
	value := "test-value"

	// Setup expectations
	suite.mockCache.EXPECT().
		Set(key, value).
		Return(nil)

	// Execute test
	_, err := suite.api.SetCache(context.Background(), &strategy.SetRequest{
		Key:   key,
		Value: value,
	})

	suite.NoError(err)
}

// TestGetCache tests the GetCache method
func (suite *StrategyApiTestSuite) TestGetCache() {
	key := "test-key"
	value := "test-value"

	// Setup expectations
	suite.mockCache.EXPECT().
		Get(key).
		Return(value, true)

	// Execute test
	response, err := suite.api.GetCache(context.Background(), &strategy.GetRequest{
		Key: key,
	})

	suite.NoError(err)
	suite.Contains(response.Value, value)
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
		GetMarkers().
		Return(markers, nil)

	// Execute test
	response, err := suite.api.GetMarkers(context.Background(), &emptypb.Empty{})

	suite.NoError(err)
	suite.Len(response.Markers, 1)
	suite.Equal(markers[0].Signal.Symbol, response.Markers[0].MarketData.Symbol)
	suite.Equal(strategy.SignalType_SIGNAL_TYPE_BUY_LONG, response.Markers[0].Signal)
}

func (suite *StrategyApiTestSuite) TestCount() {
	startTime := time.Now()
	endTime := startTime.Add(time.Hour)
	count := int64(100)

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
	startTime := time.Now()
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

	suite.mockDataSource.EXPECT().GetRange(startTime, endTime, interval).Return(data, nil)

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
		RawValue:  "test value",
		Symbol:    "BTCUSDT",
		Indicator: types.IndicatorTypeRSI,
	}

	suite.mockIndicators.EXPECT().
		GetIndicator(types.IndicatorTypeRSI).
		Return(expectedSignal, nil)

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

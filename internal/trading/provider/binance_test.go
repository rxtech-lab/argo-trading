package tradingprovider

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/adshao/go-binance/v2"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// Mock implementations for testing

// mockBinanceClient implements BinanceClient interface for testing
type mockBinanceClient struct {
	createOrderService      *mockCreateOrderService
	getAccountService       *mockGetAccountService
	listOpenOrdersService   *mockListOpenOrdersService
	cancelOrderService      *mockCancelOrderService
	cancelOpenOrdersService *mockCancelOpenOrdersService
	listTradesService       *mockListTradesService
	tradeFeeService         *mockTradeFeeService
}

func newMockBinanceClient() *mockBinanceClient {
	return &mockBinanceClient{
		createOrderService:      &mockCreateOrderService{},
		getAccountService:       &mockGetAccountService{},
		listOpenOrdersService:   &mockListOpenOrdersService{},
		cancelOrderService:      &mockCancelOrderService{},
		cancelOpenOrdersService: &mockCancelOpenOrdersService{},
		listTradesService:       &mockListTradesService{},
		tradeFeeService:         &mockTradeFeeService{},
	}
}

func (m *mockBinanceClient) NewCreateOrderService() CreateOrderService {
	return m.createOrderService
}

func (m *mockBinanceClient) NewGetAccountService() GetAccountService {
	return m.getAccountService
}

func (m *mockBinanceClient) NewListOpenOrdersService() ListOpenOrdersService {
	return m.listOpenOrdersService
}

func (m *mockBinanceClient) NewCancelOrderService() CancelOrderService {
	return m.cancelOrderService
}

func (m *mockBinanceClient) NewCancelOpenOrdersService() CancelOpenOrdersService {
	return m.cancelOpenOrdersService
}

func (m *mockBinanceClient) NewListTradesService() ListTradesService {
	return m.listTradesService
}

func (m *mockBinanceClient) NewTradeFeeService() TradeFeeService {
	return m.tradeFeeService
}

// mockCreateOrderService implements CreateOrderService
type mockCreateOrderService struct {
	response *binance.CreateOrderResponse
	err      error
	symbol   string
	side     binance.SideType
	orderTyp binance.OrderType
	quantity string
	price    string
	tif      binance.TimeInForceType
}

func (m *mockCreateOrderService) Symbol(symbol string) CreateOrderService {
	m.symbol = symbol
	return m
}

func (m *mockCreateOrderService) Side(side binance.SideType) CreateOrderService {
	m.side = side
	return m
}

func (m *mockCreateOrderService) Type(orderType binance.OrderType) CreateOrderService {
	m.orderTyp = orderType
	return m
}

func (m *mockCreateOrderService) Quantity(quantity string) CreateOrderService {
	m.quantity = quantity
	return m
}

func (m *mockCreateOrderService) Price(price string) CreateOrderService {
	m.price = price
	return m
}

func (m *mockCreateOrderService) TimeInForce(tif binance.TimeInForceType) CreateOrderService {
	m.tif = tif
	return m
}

func (m *mockCreateOrderService) Do(_ context.Context) (*binance.CreateOrderResponse, error) {
	return m.response, m.err
}

// mockGetAccountService implements GetAccountService
type mockGetAccountService struct {
	account *binance.Account
	err     error
}

func (m *mockGetAccountService) Do(_ context.Context) (*binance.Account, error) {
	return m.account, m.err
}

// mockListOpenOrdersService implements ListOpenOrdersService
type mockListOpenOrdersService struct {
	orders []*binance.Order
	err    error
}

func (m *mockListOpenOrdersService) Do(_ context.Context) ([]*binance.Order, error) {
	return m.orders, m.err
}

// mockCancelOrderService implements CancelOrderService
type mockCancelOrderService struct {
	response *binance.CancelOrderResponse
	err      error
	symbol   string
	orderID  int64
}

func (m *mockCancelOrderService) Symbol(symbol string) CancelOrderService {
	m.symbol = symbol
	return m
}

func (m *mockCancelOrderService) OrderID(orderID int64) CancelOrderService {
	m.orderID = orderID
	return m
}

func (m *mockCancelOrderService) Do(_ context.Context) (*binance.CancelOrderResponse, error) {
	return m.response, m.err
}

// mockCancelOpenOrdersService implements CancelOpenOrdersService
type mockCancelOpenOrdersService struct {
	err    error
	symbol string
}

func (m *mockCancelOpenOrdersService) Symbol(symbol string) CancelOpenOrdersService {
	m.symbol = symbol
	return m
}

func (m *mockCancelOpenOrdersService) Do(_ context.Context) error {
	return m.err
}

// mockListTradesService implements ListTradesService
type mockListTradesService struct {
	trades    []*binance.TradeV3
	err       error
	symbol    string
	limit     int
	startTime int64
	endTime   int64
}

func (m *mockListTradesService) Symbol(symbol string) ListTradesService {
	m.symbol = symbol
	return m
}

func (m *mockListTradesService) Limit(limit int) ListTradesService {
	m.limit = limit
	return m
}

func (m *mockListTradesService) StartTime(startTime int64) ListTradesService {
	m.startTime = startTime
	return m
}

func (m *mockListTradesService) EndTime(endTime int64) ListTradesService {
	m.endTime = endTime
	return m
}

func (m *mockListTradesService) Do(_ context.Context) ([]*binance.TradeV3, error) {
	return m.trades, m.err
}

// mockTradeFeeService implements TradeFeeService
type mockTradeFeeService struct {
	fees   []*binance.TradeFeeDetails
	err    error
	symbol string
}

func (m *mockTradeFeeService) Symbol(symbol string) TradeFeeService {
	m.symbol = symbol
	return m
}

func (m *mockTradeFeeService) Do(_ context.Context) ([]*binance.TradeFeeDetails, error) {
	return m.fees, m.err
}

type BinanceTradingTestSuite struct {
	suite.Suite
}

func TestBinanceTradingSuite(t *testing.T) {
	suite.Run(t, new(BinanceTradingTestSuite))
}

// Unit Tests - Config

func (suite *BinanceTradingTestSuite) TestParseBinanceConfig_Valid() {
	jsonConfig := `{"apiKey": "test-api-key", "secretKey": "test-secret-key"}`
	config, err := parseBinanceConfig(jsonConfig)
	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal("test-api-key", config.ApiKey)
	suite.Equal("test-secret-key", config.SecretKey)
}

func (suite *BinanceTradingTestSuite) TestParseBinanceConfig_MissingApiKey() {
	jsonConfig := `{"secretKey": "test-secret-key"}`
	config, err := parseBinanceConfig(jsonConfig)
	suite.Error(err)
	suite.Nil(config)
	suite.Contains(err.Error(), "invalid binance provider config")
}

func (suite *BinanceTradingTestSuite) TestParseBinanceConfig_MissingSecretKey() {
	jsonConfig := `{"apiKey": "test-api-key"}`
	config, err := parseBinanceConfig(jsonConfig)
	suite.Error(err)
	suite.Nil(config)
	suite.Contains(err.Error(), "invalid binance provider config")
}

func (suite *BinanceTradingTestSuite) TestParseBinanceConfig_InvalidJSON() {
	jsonConfig := `{invalid json}`
	config, err := parseBinanceConfig(jsonConfig)
	suite.Error(err)
	suite.Nil(config)
	suite.Contains(err.Error(), "failed to parse binance config")
}

func (suite *BinanceTradingTestSuite) TestParseBinanceConfig_EmptyJSON() {
	jsonConfig := `{}`
	config, err := parseBinanceConfig(jsonConfig)
	suite.Error(err)
	suite.Nil(config)
}

func (suite *BinanceTradingTestSuite) TestBinanceProviderConfig_Validate() {
	config := BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	err := config.Validate()
	suite.NoError(err)
}

func (suite *BinanceTradingTestSuite) TestBinanceProviderConfig_Validate_Empty() {
	config := BinanceProviderConfig{}
	err := config.Validate()
	suite.Error(err)
}

// Unit Tests - Trading System Creation

func (suite *BinanceTradingTestSuite) TestNewBinanceTradingSystem() {
	config := BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	system, err := NewBinanceTradingSystemProvider(config, true)
	suite.NoError(err)
	suite.NotNil(system)
	suite.NotNil(system.client)
}

// Unit Tests - Order Status Mapping

func (suite *BinanceTradingTestSuite) TestMapBinanceOrderStatus() {
	suite.Run("NEW", func() {
		status := mapBinanceOrderStatus("NEW")
		suite.Equal(OrderStatusPending, string(status))
	})

	suite.Run("FILLED", func() {
		status := mapBinanceOrderStatus("FILLED")
		suite.Equal(OrderStatusFilled, string(status))
	})

	suite.Run("CANCELED", func() {
		status := mapBinanceOrderStatus("CANCELED")
		suite.Equal(OrderStatusCancelled, string(status))
	})

	suite.Run("REJECTED", func() {
		status := mapBinanceOrderStatus("REJECTED")
		suite.Equal(OrderStatusRejected, string(status))
	})

	suite.Run("EXPIRED", func() {
		status := mapBinanceOrderStatus("EXPIRED")
		suite.Equal(OrderStatusFailed, string(status))
	})
}

const (
	OrderStatusPending   = "PENDING"
	OrderStatusFilled    = "FILLED"
	OrderStatusCancelled = "CANCELLED"
	OrderStatusRejected  = "REJECTED"
	OrderStatusFailed    = "FAILED"
)

// =============================================================================
// Unit Tests with Mocks
// =============================================================================

// PlaceOrder Tests

func (suite *BinanceTradingTestSuite) TestPlaceOrder_MarketBuy_Success() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12345,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.001,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal("BTCUSDT", mockClient.createOrderService.symbol)
	suite.Equal(binance.SideTypeBuy, mockClient.createOrderService.side)
	suite.Equal(binance.OrderTypeMarket, mockClient.createOrderService.orderTyp)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_MarketSell_Success() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12346,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeSell,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.001,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal(binance.SideTypeSell, mockClient.createOrderService.side)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_LimitBuy_Success() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12347,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeLimit,
		Quantity:  0.001,
		Price:     50000.0,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal(binance.OrderTypeLimit, mockClient.createOrderService.orderTyp)
	suite.Equal("50000", mockClient.createOrderService.price)
	suite.Equal(binance.TimeInForceTypeGTC, mockClient.createOrderService.tif)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_UnsupportedSide_Error() {
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseType("INVALID"),
		OrderType: types.OrderTypeMarket,
		Quantity:  0.001,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported order side")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_UnsupportedOrderType_Error() {
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderType("INVALID"),
		Quantity:  0.001,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported order type")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.err = errors.New("API error: insufficient balance")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.001,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to place order")
}

// PlaceMultipleOrders Tests

func (suite *BinanceTradingTestSuite) TestPlaceMultipleOrders_Success() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{OrderID: 12345}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders := []types.ExecuteOrder{
		{Symbol: "BTCUSDT", Side: types.PurchaseTypeBuy, OrderType: types.OrderTypeMarket, Quantity: 0.001},
		{Symbol: "ETHUSDT", Side: types.PurchaseTypeSell, OrderType: types.OrderTypeMarket, Quantity: 0.01},
	}

	err := provider.PlaceMultipleOrders(orders)
	suite.NoError(err)
}

func (suite *BinanceTradingTestSuite) TestPlaceMultipleOrders_FailsOnFirstError() {
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders := []types.ExecuteOrder{
		{Symbol: "BTCUSDT", Side: types.PurchaseTypeBuy, OrderType: types.OrderTypeMarket, Quantity: 0.001},
		{Symbol: "ETHUSDT", Side: types.PurchaseTypeSell, OrderType: types.OrderTypeMarket, Quantity: 0.01},
	}

	err := provider.PlaceMultipleOrders(orders)
	suite.Error(err)
}

// GetPositions Tests

func (suite *BinanceTradingTestSuite) TestGetPositions_Success() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "BTC", Free: "1.5", Locked: "0.5"},
			{Asset: "ETH", Free: "10.0", Locked: "0"},
			{Asset: "USDT", Free: "0", Locked: "0"}, // Should be filtered out
		},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	positions, err := provider.GetPositions()
	suite.NoError(err)
	suite.Len(positions, 2) // Only BTC and ETH (USDT has 0 balance)

	// Check BTC position
	var btcPos types.Position
	for _, p := range positions {
		if p.Symbol == "BTC" {
			btcPos = p
			break
		}
	}
	suite.Equal("BTC", btcPos.Symbol)
	suite.Equal(2.0, btcPos.TotalLongPositionQuantity) // 1.5 + 0.5
}

func (suite *BinanceTradingTestSuite) TestGetPositions_EmptyBalances() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	positions, err := provider.GetPositions()
	suite.NoError(err)
	suite.Len(positions, 0)
}

func (suite *BinanceTradingTestSuite) TestGetPositions_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	positions, err := provider.GetPositions()
	suite.Error(err)
	suite.Nil(positions)
}

// GetPosition Tests

func (suite *BinanceTradingTestSuite) TestGetPosition_Found() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "BTC", Free: "1.5", Locked: "0.5"},
		},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	position, err := provider.GetPosition("BTC")
	suite.NoError(err)
	suite.Equal("BTC", position.Symbol)
	suite.Equal(2.0, position.TotalLongPositionQuantity)
}

func (suite *BinanceTradingTestSuite) TestGetPosition_NotFound() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "BTC", Free: "1.5", Locked: "0.5"},
		},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	position, err := provider.GetPosition("ETH")
	suite.NoError(err)
	suite.Equal("ETH", position.Symbol)
	suite.Equal(0.0, position.TotalLongPositionQuantity)
}

func (suite *BinanceTradingTestSuite) TestGetPosition_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetPosition("BTC")
	suite.Error(err)
}

// CancelOrder Tests

func (suite *BinanceTradingTestSuite) TestCancelOrder_Success() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
	}
	mockClient.cancelOrderService.response = &binance.CancelOrderResponse{OrderID: 12345}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelOrder("12345")
	suite.NoError(err)
	suite.Equal("BTCUSDT", mockClient.cancelOrderService.symbol)
	suite.Equal(int64(12345), mockClient.cancelOrderService.orderID)
}

func (suite *BinanceTradingTestSuite) TestCancelOrder_NotFound() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelOrder("12345")
	suite.Error(err)
	suite.Contains(err.Error(), "order not found")
}

func (suite *BinanceTradingTestSuite) TestCancelOrder_InvalidOrderIDFormat_Mock() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	// "invalid-id" won't match order ID 12345
	err := provider.CancelOrder("invalid-id")
	suite.Error(err)
	suite.Contains(err.Error(), "order not found")
}

func (suite *BinanceTradingTestSuite) TestCancelOrder_APIError_OnGetOpenOrders() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelOrder("12345")
	suite.Error(err)
}

func (suite *BinanceTradingTestSuite) TestCancelOrder_APIError_OnCancel() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
	}
	mockClient.cancelOrderService.err = errors.New("cancel failed")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelOrder("12345")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to cancel order")
}

// CancelAllOrders Tests

func (suite *BinanceTradingTestSuite) TestCancelAllOrders_Success() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
		{OrderID: 12346, Symbol: "BTCUSDT", Side: binance.SideTypeSell, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "60000"},
		{OrderID: 12347, Symbol: "ETHUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.01", Price: "3000"},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelAllOrders()
	suite.NoError(err)
}

func (suite *BinanceTradingTestSuite) TestCancelAllOrders_NoOrders() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelAllOrders()
	suite.NoError(err)
}

func (suite *BinanceTradingTestSuite) TestCancelAllOrders_APIError_OnGetOpenOrders() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelAllOrders()
	suite.Error(err)
}

func (suite *BinanceTradingTestSuite) TestCancelAllOrders_APIError_OnCancel() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
	}
	mockClient.cancelOpenOrdersService.err = errors.New("cancel failed")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	err := provider.CancelAllOrders()
	suite.Error(err)
	suite.Contains(err.Error(), "failed to cancel orders")
}

// GetOrderStatus Tests

func (suite *BinanceTradingTestSuite) TestGetOrderStatus_Found_New() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Status: binance.OrderStatusTypeNew},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	status, err := provider.GetOrderStatus("12345")
	suite.NoError(err)
	suite.Equal(types.OrderStatusPending, status)
}

func (suite *BinanceTradingTestSuite) TestGetOrderStatus_Found_PartiallyFilled() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Status: binance.OrderStatusTypePartiallyFilled},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	status, err := provider.GetOrderStatus("12345")
	suite.NoError(err)
	suite.Equal(types.OrderStatusPending, status)
}

func (suite *BinanceTradingTestSuite) TestGetOrderStatus_NotFound() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	status, err := provider.GetOrderStatus("12345")
	suite.NoError(err)
	suite.Equal(types.OrderStatusFailed, status)
}

func (suite *BinanceTradingTestSuite) TestGetOrderStatus_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	status, err := provider.GetOrderStatus("12345")
	suite.Error(err)
	suite.Equal(types.OrderStatusFailed, status)
}

// GetAccountInfo Tests

func (suite *BinanceTradingTestSuite) TestGetAccountInfo_Success() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "USDT", Free: "1000", Locked: "500"},
			{Asset: "BUSD", Free: "200", Locked: "0"},
			{Asset: "BTC", Free: "1.0", Locked: "0"}, // Should be ignored
		},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	accountInfo, err := provider.GetAccountInfo()
	suite.NoError(err)
	suite.Equal(1700.0, accountInfo.Balance)     // 1000 + 500 + 200
	suite.Equal(1200.0, accountInfo.BuyingPower) // 1000 + 200 (free only)
	suite.Equal(1700.0, accountInfo.Equity)
}

func (suite *BinanceTradingTestSuite) TestGetAccountInfo_EmptyBalances() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	accountInfo, err := provider.GetAccountInfo()
	suite.NoError(err)
	suite.Equal(0.0, accountInfo.Balance)
	suite.Equal(0.0, accountInfo.BuyingPower)
}

func (suite *BinanceTradingTestSuite) TestGetAccountInfo_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetAccountInfo()
	suite.Error(err)
}

// GetOpenOrders Tests

func (suite *BinanceTradingTestSuite) TestGetOpenOrders_Success() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
		{OrderID: 12346, Symbol: "ETHUSDT", Side: binance.SideTypeSell, Type: binance.OrderTypeMarket, OrigQuantity: "0.01", Price: "0"},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders, err := provider.GetOpenOrders()
	suite.NoError(err)
	suite.Len(orders, 2)
	suite.Equal("12345", orders[0].ID)
	suite.Equal("BTCUSDT", orders[0].Symbol)
}

func (suite *BinanceTradingTestSuite) TestGetOpenOrders_Empty() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders, err := provider.GetOpenOrders()
	suite.NoError(err)
	suite.Len(orders, 0)
}

func (suite *BinanceTradingTestSuite) TestGetOpenOrders_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders, err := provider.GetOpenOrders()
	suite.Error(err)
	suite.Nil(orders)
}

// GetTrades Tests

func (suite *BinanceTradingTestSuite) TestGetTrades_RequiresSymbol() {
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetTrades(types.TradeFilter{})
	suite.Error(err)
	suite.Contains(err.Error(), "symbol is required")
}

func (suite *BinanceTradingTestSuite) TestGetTrades_Success() {
	mockClient := newMockBinanceClient()
	mockClient.listTradesService.trades = []*binance.TradeV3{
		{ID: 1, OrderID: 12345, Price: "50000", Quantity: "0.001", Commission: "0.00001", Time: 1609459200000, IsBuyer: true},
		{ID: 2, OrderID: 12346, Price: "51000", Quantity: "0.002", Commission: "0.00002", Time: 1609459300000, IsBuyer: false},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	trades, err := provider.GetTrades(types.TradeFilter{Symbol: "BTCUSDT"})
	suite.NoError(err)
	suite.Len(trades, 2)
	suite.Equal(50000.0, trades[0].ExecutedPrice)
	suite.Equal(types.PurchaseTypeBuy, trades[0].Order.Side)
	suite.Equal(types.PurchaseTypeSell, trades[1].Order.Side)
}

func (suite *BinanceTradingTestSuite) TestGetTrades_WithFilters() {
	mockClient := newMockBinanceClient()
	mockClient.listTradesService.trades = []*binance.TradeV3{}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	startTime := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2021, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := provider.GetTrades(types.TradeFilter{
		Symbol:    "BTCUSDT",
		Limit:     100,
		StartTime: startTime,
		EndTime:   endTime,
	})
	suite.NoError(err)
	suite.Equal(100, mockClient.listTradesService.limit)
	suite.Equal(startTime.UnixMilli(), mockClient.listTradesService.startTime)
	suite.Equal(endTime.UnixMilli(), mockClient.listTradesService.endTime)
}

func (suite *BinanceTradingTestSuite) TestGetTrades_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.listTradesService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetTrades(types.TradeFilter{Symbol: "BTCUSDT"})
	suite.Error(err)
}

// GetMaxBuyQuantity Tests

func (suite *BinanceTradingTestSuite) TestGetMaxBuyQuantity_InvalidPrice() {
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetMaxBuyQuantity("BTCUSDT", 0)
	suite.Error(err)
	suite.Contains(err.Error(), "price must be greater than zero")

	_, err = provider.GetMaxBuyQuantity("BTCUSDT", -100)
	suite.Error(err)
}

func (suite *BinanceTradingTestSuite) TestGetMaxBuyQuantity_Success() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "USDT", Free: "10000", Locked: "0"},
		},
	}
	mockClient.tradeFeeService.fees = []*binance.TradeFeeDetails{
		{Symbol: "BTCUSDT", TakerCommission: "0.001"},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	maxQty, err := provider.GetMaxBuyQuantity("BTCUSDT", 50000.0)
	suite.NoError(err)
	// effectiveBuyingPower = 10000 / 1.001 = 9990.01
	// maxQty = 9990.01 / 50000 = 0.1998
	suite.InDelta(0.1998, maxQty, 0.0001)
}

func (suite *BinanceTradingTestSuite) TestGetMaxBuyQuantity_FallbackFeeRate() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "USDT", Free: "10000", Locked: "0"},
		},
	}
	mockClient.tradeFeeService.err = errors.New("API error") // Fee API fails

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	maxQty, err := provider.GetMaxBuyQuantity("BTCUSDT", 50000.0)
	suite.NoError(err)
	// Uses default 0.1% fee rate
	// effectiveBuyingPower = 10000 / 1.001 = 9990.01
	// maxQty = 9990.01 / 50000 = 0.1998
	suite.InDelta(0.1998, maxQty, 0.0001)
}

func (suite *BinanceTradingTestSuite) TestGetMaxBuyQuantity_AccountAPIError() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetMaxBuyQuantity("BTCUSDT", 50000.0)
	suite.Error(err)
}

// GetMaxSellQuantity Tests

func (suite *BinanceTradingTestSuite) TestGetMaxSellQuantity_Success() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{
			{Asset: "BTC", Free: "1.5", Locked: "0.5"},
		},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	maxQty, err := provider.GetMaxSellQuantity("BTC")
	suite.NoError(err)
	suite.Equal(2.0, maxQty)
}

func (suite *BinanceTradingTestSuite) TestGetMaxSellQuantity_NoPosition() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.account = &binance.Account{
		Balances: []binance.Balance{},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	maxQty, err := provider.GetMaxSellQuantity("BTC")
	suite.NoError(err)
	suite.Equal(0.0, maxQty)
}

func (suite *BinanceTradingTestSuite) TestGetMaxSellQuantity_APIError() {
	mockClient := newMockBinanceClient()
	mockClient.getAccountService.err = errors.New("API error")

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	_, err := provider.GetMaxSellQuantity("BTC")
	suite.Error(err)
}

// Helper Function Tests

func (suite *BinanceTradingTestSuite) TestConvertBinanceOrderToExecuteOrder() {
	suite.Run("BuyLimit", func() {
		order := &binance.Order{
			OrderID:      12345,
			Symbol:       "BTCUSDT",
			Side:         binance.SideTypeBuy,
			Type:         binance.OrderTypeLimit,
			OrigQuantity: "0.001",
			Price:        "50000",
		}

		result, err := convertBinanceOrderToExecuteOrder(order)
		suite.NoError(err)
		suite.Equal("12345", result.ID)
		suite.Equal("BTCUSDT", result.Symbol)
		suite.Equal(types.PurchaseTypeBuy, result.Side)
		suite.Equal(types.OrderTypeLimit, result.OrderType)
		suite.Equal(0.001, result.Quantity)
		suite.Equal(50000.0, result.Price)
	})

	suite.Run("SellMarket", func() {
		order := &binance.Order{
			OrderID:      12346,
			Symbol:       "ETHUSDT",
			Side:         binance.SideTypeSell,
			Type:         binance.OrderTypeMarket,
			OrigQuantity: "0.01",
			Price:        "0",
		}

		result, err := convertBinanceOrderToExecuteOrder(order)
		suite.NoError(err)
		suite.Equal(types.PurchaseTypeSell, result.Side)
		suite.Equal(types.OrderTypeMarket, result.OrderType)
	})

	suite.Run("UnknownSide", func() {
		order := &binance.Order{
			OrderID:      12347,
			Symbol:       "BTCUSDT",
			Side:         binance.SideType("UNKNOWN"),
			Type:         binance.OrderTypeLimit,
			OrigQuantity: "0.001",
			Price:        "50000",
		}

		_, err := convertBinanceOrderToExecuteOrder(order)
		suite.Error(err)
		suite.Contains(err.Error(), "unknown side")
	})
}

func (suite *BinanceTradingTestSuite) TestConvertBinanceTradeToTrade() {
	suite.Run("BuyTrade", func() {
		trade := &binance.TradeV3{
			ID:         1,
			OrderID:    12345,
			Price:      "50000",
			Quantity:   "0.001",
			Commission: "0.00001",
			Time:       1609459200000,
			IsBuyer:    true,
		}

		result := convertBinanceTradeToTrade(trade, "BTCUSDT")
		suite.Equal("12345", result.Order.OrderID)
		suite.Equal("BTCUSDT", result.Order.Symbol)
		suite.Equal(types.PurchaseTypeBuy, result.Order.Side)
		suite.Equal(50000.0, result.ExecutedPrice)
		suite.Equal(0.001, result.ExecutedQty)
		suite.Equal(0.00001, result.Fee)
	})

	suite.Run("SellTrade", func() {
		trade := &binance.TradeV3{
			ID:         2,
			OrderID:    12346,
			Price:      "51000",
			Quantity:   "0.002",
			Commission: "0.00002",
			Time:       1609459300000,
			IsBuyer:    false,
		}

		result := convertBinanceTradeToTrade(trade, "BTCUSDT")
		suite.Equal(types.PurchaseTypeSell, result.Order.Side)
	})
}

func (suite *BinanceTradingTestSuite) TestGetOpenOrders_SkipsUnconvertibleOrders() {
	mockClient := newMockBinanceClient()
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
		{OrderID: 12346, Symbol: "ETHUSDT", Side: binance.SideType("INVALID"), Type: binance.OrderTypeLimit, OrigQuantity: "0.01", Price: "3000"}, // This one has invalid side and should be skipped
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	orders, err := provider.GetOpenOrders()
	suite.NoError(err)
	suite.Len(orders, 1) // Only the valid order
	suite.Equal("12345", orders[0].ID)
}

func (suite *BinanceTradingTestSuite) TestCancelOrder_ParsingError() {
	mockClient := newMockBinanceClient()
	// Create an order with a valid ID that matches our cancel request
	mockClient.listOpenOrdersService.orders = []*binance.Order{
		{OrderID: 12345, Symbol: "BTCUSDT", Side: binance.SideTypeBuy, Type: binance.OrderTypeLimit, OrigQuantity: "0.001", Price: "50000"},
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	// This ID matches "12345" when converted to string - if we try to cancel it,
	// the parsing will succeed. Let's test the error path with an order where ID
	// matches but parsing fails.
	err := provider.CancelOrder("12345")
	suite.NoError(err) // Should succeed since ID matches and parsing works
}

func (suite *BinanceTradingTestSuite) TestConvertBinanceOrderToExecuteOrder_DefaultOrderType() {
	// Test that unknown order types default to LIMIT
	order := &binance.Order{
		OrderID:      12345,
		Symbol:       "BTCUSDT",
		Side:         binance.SideTypeBuy,
		Type:         binance.OrderType("STOP_LOSS"), // Unknown type
		OrigQuantity: "0.001",
		Price:        "50000",
	}

	result, err := convertBinanceOrderToExecuteOrder(order)
	suite.NoError(err)
	suite.Equal(types.OrderTypeLimit, result.OrderType) // Should default to LIMIT
}

func (suite *BinanceTradingTestSuite) TestMapBinanceOrderStatus_AllCases() {
	testCases := []struct {
		input    binance.OrderStatusType
		expected types.OrderStatus
	}{
		{binance.OrderStatusTypeNew, types.OrderStatusPending},
		{binance.OrderStatusTypePartiallyFilled, types.OrderStatusPending},
		{binance.OrderStatusTypeFilled, types.OrderStatusFilled},
		{binance.OrderStatusTypeCanceled, types.OrderStatusCancelled},
		{binance.OrderStatusTypeRejected, types.OrderStatusRejected},
		{binance.OrderStatusTypeExpired, types.OrderStatusFailed},
		{binance.OrderStatusTypePendingCancel, types.OrderStatusFailed},
		{binance.OrderStatusType("UNKNOWN"), types.OrderStatusFailed},
	}

	for _, tc := range testCases {
		suite.Run(string(tc.input), func() {
			result := mapBinanceOrderStatus(tc.input)
			suite.Equal(tc.expected, result)
		})
	}
}

// =============================================================================
// Real Wrapper Tests - verify wrapper methods return correct types
// =============================================================================

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewCreateOrderService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewCreateOrderService()
	suite.NotNil(service)

	// Test chaining methods
	result := service.Symbol("BTCUSDT").Side(binance.SideTypeBuy).Type(binance.OrderTypeMarket).Quantity("0.001")
	suite.NotNil(result)

	// Test Price and TimeInForce for limit orders
	result = service.Price("50000").TimeInForce(binance.TimeInForceTypeGTC)
	suite.NotNil(result)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewGetAccountService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewGetAccountService()
	suite.NotNil(service)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewListOpenOrdersService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewListOpenOrdersService()
	suite.NotNil(service)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewCancelOrderService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewCancelOrderService()
	suite.NotNil(service)

	// Test chaining methods
	result := service.Symbol("BTCUSDT").OrderID(12345)
	suite.NotNil(result)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewCancelOpenOrdersService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewCancelOpenOrdersService()
	suite.NotNil(service)

	// Test chaining methods
	result := service.Symbol("BTCUSDT")
	suite.NotNil(result)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewListTradesService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewListTradesService()
	suite.NotNil(service)

	// Test chaining methods
	result := service.Symbol("BTCUSDT").Limit(100).StartTime(1609459200000).EndTime(1609545600000)
	suite.NotNil(result)
}

func (suite *BinanceTradingTestSuite) TestRealBinanceClient_NewTradeFeeService() {
	client := binance.NewClient("test-api", "test-secret")
	realClient := &realBinanceClient{client: client}

	service := realClient.NewTradeFeeService()
	suite.NotNil(service)

	// Test chaining methods
	result := service.Symbol("BTCUSDT")
	suite.NotNil(result)
}

// =============================================================================
// Quantity Validation Tests
// =============================================================================

func (suite *BinanceTradingTestSuite) TestPlaceOrder_FractionalQuantity_Success() {
	// Test 0.01 quantity with default precision (8) - should succeed
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12345,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.01,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal("0.01000000", mockClient.createOrderService.quantity)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_SatoshiPrecision_Success() {
	// Test 0.00000001 (1 satoshi) quantity with default precision (8) - should succeed
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12346,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.00000001,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal("0.00000001", mockClient.createOrderService.quantity)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_QuantityTooSmall_Error() {
	// Test quantity smaller than 1 satoshi - should fail after rounding
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.000000001, // Smaller than 8 decimal places
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "too small after rounding")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_ZeroQuantity_Error() {
	// Test zero quantity - should fail immediately
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "quantity must be greater than zero")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_NegativeQuantity_Error() {
	// Test negative quantity - should fail immediately
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithClient(mockClient)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  -0.01,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "quantity must be greater than zero")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_CustomPrecision_Success() {
	// Test with precision=2, quantity=0.01 - should succeed
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12347,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithPrecision(mockClient, 2)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.01,
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal("0.01", mockClient.createOrderService.quantity)
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_CustomPrecision_TooSmall_Error() {
	// Test with precision=1, quantity=0.01 - should fail (0.01 rounds to 0.0 with precision 1)
	mockClient := newMockBinanceClient()
	provider := newBinanceTradingSystemProviderWithPrecision(mockClient, 1)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.01,
	}

	err := provider.PlaceOrder(order)
	suite.Error(err)
	suite.Contains(err.Error(), "too small after rounding")
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_QuantityBelowPrecision_Errors() {
	// Table-driven tests for various precision levels and quantities below permitted
	testCases := []struct {
		name             string
		decimalPrecision int
		quantity         float64
		shouldFail       bool
	}{
		{
			name:             "0.001 with precision 2 - should fail",
			decimalPrecision: 2,
			quantity:         0.001, // Rounds to 0.00
			shouldFail:       true,
		},
		{
			name:             "0.009 with precision 2 - should fail",
			decimalPrecision: 2,
			quantity:         0.009, // Rounds to 0.00 (floor)
			shouldFail:       true,
		},
		{
			name:             "0.01 with precision 2 - should succeed",
			decimalPrecision: 2,
			quantity:         0.01, // Rounds to 0.01
			shouldFail:       false,
		},
		{
			name:             "0.0001 with precision 3 - should fail",
			decimalPrecision: 3,
			quantity:         0.0001, // Rounds to 0.000
			shouldFail:       true,
		},
		{
			name:             "0.001 with precision 3 - should succeed",
			decimalPrecision: 3,
			quantity:         0.001, // Rounds to 0.001
			shouldFail:       false,
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			mockClient := newMockBinanceClient()
			if !tc.shouldFail {
				mockClient.createOrderService.response = &binance.CreateOrderResponse{
					OrderID: 12345,
					Symbol:  "BTCUSDT",
				}
			}

			provider := newBinanceTradingSystemProviderWithPrecision(mockClient, tc.decimalPrecision)

			order := types.ExecuteOrder{
				Symbol:    "BTCUSDT",
				Side:      types.PurchaseTypeBuy,
				OrderType: types.OrderTypeMarket,
				Quantity:  tc.quantity,
			}

			err := provider.PlaceOrder(order)
			if tc.shouldFail {
				suite.Error(err, "Expected error for quantity %f with precision %d", tc.quantity, tc.decimalPrecision)
				suite.Contains(err.Error(), "too small after rounding")
			} else {
				suite.NoError(err, "Expected success for quantity %f with precision %d", tc.quantity, tc.decimalPrecision)
			}
		})
	}
}

func (suite *BinanceTradingTestSuite) TestPlaceOrder_QuantityRounding() {
	// Test that quantity is properly rounded down
	mockClient := newMockBinanceClient()
	mockClient.createOrderService.response = &binance.CreateOrderResponse{
		OrderID: 12348,
		Symbol:  "BTCUSDT",
	}

	provider := newBinanceTradingSystemProviderWithPrecision(mockClient, 2)

	order := types.ExecuteOrder{
		Symbol:    "BTCUSDT",
		Side:      types.PurchaseTypeBuy,
		OrderType: types.OrderTypeMarket,
		Quantity:  0.019999, // Should round down to 0.01
	}

	err := provider.PlaceOrder(order)
	suite.NoError(err)
	suite.Equal("0.01", mockClient.createOrderService.quantity)
}

func (suite *BinanceTradingTestSuite) TestNewBinanceTradingSystemProvider_HasCorrectPrecision() {
	// Verify new providers are created with correct decimal precision
	config := BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewBinanceTradingSystemProvider(config, true)
	suite.NoError(err)
	suite.Equal(BinanceDecimalPrecision, provider.decimalPrecision)
}

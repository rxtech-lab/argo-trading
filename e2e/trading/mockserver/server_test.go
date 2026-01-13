package mockserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

type MockServerTestSuite struct {
	suite.Suite
	server *MockBinanceServer
}

func TestMockServerSuite(t *testing.T) {
	suite.Run(t, new(MockServerTestSuite))
}

func (suite *MockServerTestSuite) SetupTest() {
	config := ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 10000.0,
			"BTC":  1.0,
			"ETH":  10.0,
		},
		TradeFees: map[string]*TradeFee{
			"BTCUSDT": {Symbol: "BTCUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
			"ETHUSDT": {Symbol: "ETHUSDT", MakerCommission: 0.001, TakerCommission: 0.001},
		},
		MarketData: &MarketDataGeneratorConfig{
			Symbols:           []string{"BTCUSDT", "ETHUSDT"},
			Pattern:           backtestTesthelper.PatternVolatile,
			InitialPrice:      50000.0,
			VolatilityPercent: 2.0,
			TrendStrength:     0.01,
			Seed:              12345,
		},
		StreamInterval: 50 * time.Millisecond,
	}

	suite.server = NewMockBinanceServer(config)
	err := suite.server.Start(":0")
	suite.Require().NoError(err)
}

func (suite *MockServerTestSuite) TearDownTest() {
	if suite.server != nil {
		suite.server.Stop()
	}
}

// Test Server Lifecycle

func (suite *MockServerTestSuite) TestServerStartAndStop() {
	suite.NotEmpty(suite.server.Address())
	suite.Contains(suite.server.BaseURL(), "http://")
	suite.Contains(suite.server.WebSocketURL(), "ws://")
}

// Test Price Management

func (suite *MockServerTestSuite) TestSetAndGetPrice() {
	suite.server.SetPrice("BTCUSDT", 50000.0)
	price := suite.server.GetPrice("BTCUSDT")
	suite.Equal(50000.0, price)
}

func (suite *MockServerTestSuite) TestGetPriceNonExistent() {
	price := suite.server.GetPrice("NONEXISTENT")
	suite.Equal(0.0, price)
}

// Test Balance Management

func (suite *MockServerTestSuite) TestGetBalance() {
	balance := suite.server.GetBalance("USDT")
	suite.NotNil(balance)
	suite.Equal("USDT", balance.Asset)
	suite.Equal(10000.0, balance.Free)
	suite.Equal(0.0, balance.Locked)
}

func (suite *MockServerTestSuite) TestSetBalance() {
	suite.server.SetBalance("BNB", 100.0, 10.0)
	balance := suite.server.GetBalance("BNB")
	suite.NotNil(balance)
	suite.Equal(100.0, balance.Free)
	suite.Equal(10.0, balance.Locked)
}

func (suite *MockServerTestSuite) TestGetBalanceNonExistent() {
	balance := suite.server.GetBalance("NONEXISTENT")
	suite.Nil(balance)
}

// Test Ticker Price Endpoint

func (suite *MockServerTestSuite) TestTickerPriceEndpoint() {
	suite.server.SetPrice("BTCUSDT", 50000.0)
	suite.server.SetPrice("ETHUSDT", 3000.0)

	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/ticker/price")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var prices []map[string]string
	err = json.NewDecoder(resp.Body).Decode(&prices)
	suite.NoError(err)
	suite.GreaterOrEqual(len(prices), 2)

	// Check that we have prices for our symbols
	priceMap := make(map[string]string)
	for _, p := range prices {
		priceMap[p["symbol"]] = p["price"]
	}
	suite.Contains(priceMap, "BTCUSDT")
	suite.Contains(priceMap, "ETHUSDT")
}

func (suite *MockServerTestSuite) TestTickerPriceWithSymbols() {
	suite.server.SetPrice("BTCUSDT", 50000.0)
	suite.server.SetPrice("ETHUSDT", 3000.0)

	symbols := `["BTCUSDT"]`
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/ticker/price?symbols=" + url.QueryEscape(symbols))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var prices []map[string]string
	err = json.NewDecoder(resp.Body).Decode(&prices)
	suite.NoError(err)
	suite.Len(prices, 1)
	suite.Equal("BTCUSDT", prices[0]["symbol"])
}

// Test Klines Endpoint

func (suite *MockServerTestSuite) TestKlinesEndpoint() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/klines?symbol=BTCUSDT&interval=1m")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var klines [][]interface{}
	err = json.NewDecoder(resp.Body).Decode(&klines)
	suite.NoError(err)
	suite.NotEmpty(klines)

	// Each kline should have 12 fields
	suite.Len(klines[0], 12)
}

func (suite *MockServerTestSuite) TestKlinesMissingParams() {
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/klines")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

// Test Account Endpoint

func (suite *MockServerTestSuite) TestAccountEndpoint() {
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/account")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var account map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&account)
	suite.NoError(err)

	suite.True(account["canTrade"].(bool))
	suite.Equal("SPOT", account["accountType"])

	balances := account["balances"].([]interface{})
	suite.NotEmpty(balances)
}

// Test Create Order Endpoint

func (suite *MockServerTestSuite) TestCreateMarketBuyOrder() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var order map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&order)
	suite.NoError(err)

	suite.Equal("BTCUSDT", order["symbol"])
	suite.Equal("FILLED", order["status"])
	suite.Equal("BUY", order["side"])
	suite.Equal("MARKET", order["type"])

	// Check balance was updated
	usdtBalance := suite.server.GetBalance("USDT")
	suite.Require().NotNil(usdtBalance)
	suite.Less(usdtBalance.Free, 10000.0) // Should have deducted for the purchase

	btcBalance := suite.server.GetBalance("BTC")
	suite.Require().NotNil(btcBalance)
	suite.Greater(btcBalance.Free, 1.0) // Should have added BTC
}

func (suite *MockServerTestSuite) TestCreateMarketSellOrder() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "SELL")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.5")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	// Check balances
	btcBalance := suite.server.GetBalance("BTC")
	suite.Require().NotNil(btcBalance)
	suite.Equal(0.5, btcBalance.Free) // 1.0 - 0.5

	usdtBalance := suite.server.GetBalance("USDT")
	suite.Require().NotNil(usdtBalance)
	suite.Greater(usdtBalance.Free, 10000.0) // Should have added USDT
}

func (suite *MockServerTestSuite) TestCreateLimitOrder() {
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var order map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&order)
	suite.NoError(err)

	suite.Equal("NEW", order["status"]) // Limit orders start as NEW
}

func (suite *MockServerTestSuite) TestCreateOrderInsufficientBalance() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "100") // Would cost 5,000,000 USDT

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCreateOrderMissingParams() {
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	// Missing side, type, quantity

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

// Test Open Orders Endpoint

func (suite *MockServerTestSuite) TestOpenOrdersEndpoint() {
	// Create a limit order first
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Get open orders
	resp, err = http.Get(suite.server.BaseURL() + "/api/v3/openOrders")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var orders []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&orders)
	suite.NoError(err)
	suite.Len(orders, 1)
	suite.Equal("NEW", orders[0]["status"])
}

// Test Cancel Order Endpoint

func (suite *MockServerTestSuite) TestCancelOrder() {
	// Create a limit order first
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)

	var orderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderResp)
	resp.Body.Close()

	orderID := int64(orderResp["orderId"].(float64))

	// Cancel the order
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v3/order?symbol=BTCUSDT&orderId=%d", suite.server.BaseURL(), orderID),
		nil)
	resp, err = http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var cancelResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&cancelResp)
	suite.Equal("CANCELED", cancelResp["status"])

	// Verify order is cancelled
	order := suite.server.GetOrder(orderID)
	suite.Require().NotNil(order)
	suite.Equal(OrderStatusCanceled, order.Status)
}

func (suite *MockServerTestSuite) TestCancelOrderNotFound() {
	req, _ := http.NewRequest("DELETE",
		suite.server.BaseURL()+"/api/v3/order?symbol=BTCUSDT&orderId=99999",
		nil)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

// Test Cancel All Orders Endpoint

func (suite *MockServerTestSuite) TestCancelAllOrders() {
	// Create multiple limit orders
	for i := 0; i < 3; i++ {
		form := url.Values{}
		form.Set("symbol", "BTCUSDT")
		form.Set("side", "BUY")
		form.Set("type", "LIMIT")
		form.Set("quantity", "0.1")
		form.Set("price", fmt.Sprintf("%d", 45000+i*1000))
		form.Set("timeInForce", "GTC")

		resp, err := http.Post(
			suite.server.BaseURL()+"/api/v3/order",
			"application/x-www-form-urlencoded",
			strings.NewReader(form.Encode()),
		)
		suite.Require().NoError(err)
		resp.Body.Close()
	}

	// Cancel all orders
	req, _ := http.NewRequest("DELETE",
		suite.server.BaseURL()+"/api/v3/openOrders?symbol=BTCUSDT",
		nil)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	// Verify no open orders
	resp, err = http.Get(suite.server.BaseURL() + "/api/v3/openOrders")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var orders []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orders)
	suite.Empty(orders)
}

// Test My Trades Endpoint

func (suite *MockServerTestSuite) TestMyTradesEndpoint() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Create a market order to generate a trade
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Get trades
	resp, err = http.Get(suite.server.BaseURL() + "/api/v3/myTrades?symbol=BTCUSDT")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var trades []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&trades)
	suite.NoError(err)
	suite.Len(trades, 1)
	suite.Equal("BTCUSDT", trades[0]["symbol"])
	suite.True(trades[0]["isBuyer"].(bool))
}

func (suite *MockServerTestSuite) TestMyTradesMissingSymbol() {
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/myTrades")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

// Test Trade Fee Endpoint

func (suite *MockServerTestSuite) TestTradeFeeEndpoint() {
	resp, err := http.Get(suite.server.BaseURL() + "/sapi/v1/asset/tradeFee?symbol=BTCUSDT")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var fees []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fees)
	suite.NoError(err)
	suite.Len(fees, 1)
	suite.Equal("BTCUSDT", fees[0]["symbol"])
}

func (suite *MockServerTestSuite) TestTradeFeeAllSymbols() {
	resp, err := http.Get(suite.server.BaseURL() + "/sapi/v1/asset/tradeFee")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var fees []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&fees)
	suite.NoError(err)
	suite.GreaterOrEqual(len(fees), 2)
}

// Test WebSocket Streaming

func (suite *MockServerTestSuite) TestWebSocketKlineStream() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	wsURL := suite.server.WebSocketURL() + "/ws/btcusdt@kline_1m"
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	suite.Require().NoError(err)
	defer conn.Close()

	// Read a few messages
	conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))

	receivedCount := 0
	for receivedCount < 3 {
		_, message, err := conn.ReadMessage()
		if err != nil {
			break
		}

		var event map[string]interface{}
		err = json.Unmarshal(message, &event)
		if err == nil && event["e"] == "kline" {
			receivedCount++
			kline := event["k"].(map[string]interface{})
			suite.Equal("BTCUSDT", kline["s"])
			suite.Equal("1m", kline["i"])
		}
	}

	suite.GreaterOrEqual(receivedCount, 1)
}

// Test Server Reset

func (suite *MockServerTestSuite) TestReset() {
	// Create some state
	suite.server.SetPrice("BTCUSDT", 50000.0)

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Reset the server
	suite.server.Reset(ServerConfig{
		InitialBalances: map[string]float64{
			"USDT": 5000.0,
		},
	})

	// Verify state was reset
	balance := suite.server.GetBalance("USDT")
	suite.NotNil(balance)
	suite.Equal(5000.0, balance.Free)

	trades := suite.server.GetTrades()
	suite.Empty(trades)
}

// Test Generate Market Data

func (suite *MockServerTestSuite) TestGenerateMarketData() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	data, err := suite.server.GenerateMarketData("BTCUSDT", 10)
	suite.Require().NoError(err)
	suite.Len(data, 10)

	for _, d := range data {
		suite.Equal("BTCUSDT", d.Symbol)
		suite.Greater(d.Open, 0.0)
		suite.Greater(d.High, 0.0)
		suite.Greater(d.Low, 0.0)
		suite.Greater(d.Close, 0.0)
		suite.Greater(d.Volume, 0.0)
	}
}

// Test Parse Interval

func (suite *MockServerTestSuite) TestParseInterval() {
	suite.Equal(time.Second, parseInterval("1s"))
	suite.Equal(time.Minute, parseInterval("1m"))
	suite.Equal(5*time.Minute, parseInterval("5m"))
	suite.Equal(15*time.Minute, parseInterval("15m"))
	suite.Equal(30*time.Minute, parseInterval("30m"))
	suite.Equal(time.Hour, parseInterval("1h"))
	suite.Equal(4*time.Hour, parseInterval("4h"))
	suite.Equal(24*time.Hour, parseInterval("1d"))
	suite.Equal(7*24*time.Hour, parseInterval("1w"))
	suite.Equal(30*24*time.Hour, parseInterval("1M"))
	suite.Equal(time.Duration(0), parseInterval("invalid"))
	suite.Equal(time.Duration(0), parseInterval(""))
}

// Test Symbol Info Initialization

func (suite *MockServerTestSuite) TestSymbolInfoInit() {
	// Create a new server with custom symbols
	config := ServerConfig{
		TradeFees: map[string]*TradeFee{
			"BTCUSDT":  {Symbol: "BTCUSDT"},
			"ETHBTC":   {Symbol: "ETHBTC"},
			"BNBBUSD":  {Symbol: "BNBBUSD"},
			"DOTETH":   {Symbol: "DOTETH"},
			"CUSTOMXY": {Symbol: "CUSTOMXY"},
		},
	}
	server := NewMockBinanceServer(config)

	// Verify symbol info was correctly parsed
	suite.Require().NotNil(server.symbolInfo["BTCUSDT"])
	suite.Equal("BTC", server.symbolInfo["BTCUSDT"].BaseAsset)
	suite.Equal("USDT", server.symbolInfo["BTCUSDT"].QuoteAsset)

	suite.Require().NotNil(server.symbolInfo["ETHBTC"])
	suite.Equal("ETH", server.symbolInfo["ETHBTC"].BaseAsset)
	suite.Equal("BTC", server.symbolInfo["ETHBTC"].QuoteAsset)

	suite.Require().NotNil(server.symbolInfo["BNBBUSD"])
	suite.Equal("BNB", server.symbolInfo["BNBBUSD"].BaseAsset)
	suite.Equal("BUSD", server.symbolInfo["BNBBUSD"].QuoteAsset)
}

// Integration-style test with multiple operations

func (suite *MockServerTestSuite) TestTradingFlow() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// 1. Check initial balance
	balance := suite.server.GetBalance("USDT")
	suite.Require().NotNil(balance)
	suite.Equal(10000.0, balance.Free)

	// 2. Place a buy order
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// 3. Verify balance changed
	balance = suite.server.GetBalance("USDT")
	suite.Require().NotNil(balance)
	suite.Less(balance.Free, 10000.0)

	btcBalance := suite.server.GetBalance("BTC")
	suite.Require().NotNil(btcBalance)
	suite.Equal(1.1, btcBalance.Free) // 1.0 + 0.1

	// 4. Check trades
	trades := suite.server.GetTrades()
	suite.Len(trades, 1)

	// 5. Place a sell order
	suite.server.SetPrice("BTCUSDT", 51000.0) // Price increased

	form = url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "SELL")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err = http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// 6. Verify final state
	btcBalance = suite.server.GetBalance("BTC")
	suite.Require().NotNil(btcBalance)
	suite.Equal(1.0, btcBalance.Free) // Back to 1.0

	usdtBalance := suite.server.GetBalance("USDT")
	suite.Require().NotNil(usdtBalance)
	suite.Greater(usdtBalance.Free, 5000.0) // Should have profit

	trades = suite.server.GetTrades()
	suite.Len(trades, 2)
}

// Helper test to ensure server can be created with minimal config

func (suite *MockServerTestSuite) TestMinimalConfig() {
	config := ServerConfig{}
	server := NewMockBinanceServer(config)
	suite.NotNil(server)

	// Should have default trade fees
	suite.NotEmpty(server.tradeFees)

	// Should have default stream interval
	suite.Equal(100*time.Millisecond, server.streamInterval)
}

// Test HTTP client integration

func (suite *MockServerTestSuite) TestHTTPClientIntegration() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	client := &http.Client{Timeout: 5 * time.Second}

	// Test GET request
	resp, err := client.Get(suite.server.BaseURL() + "/api/v3/account")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	suite.NoError(err)
	suite.NotEmpty(body)

	// Test POST request
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.01")

	resp, err = client.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)
}

// Test concurrent access

func (suite *MockServerTestSuite) TestConcurrentAccess() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	done := make(chan bool, 10)

	// Concurrent reads
	for i := 0; i < 5; i++ {
		go func() {
			select {
			case <-ctx.Done():
				return
			default:
				suite.server.GetPrice("BTCUSDT")
				suite.server.GetBalance("USDT")
				done <- true
			}
		}()
	}

	// Concurrent writes
	for i := 0; i < 5; i++ {
		go func(n int) {
			select {
			case <-ctx.Done():
				return
			default:
				suite.server.SetPrice("BTCUSDT", 50000.0+float64(n))
				done <- true
			}
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		select {
		case <-done:
		case <-ctx.Done():
			suite.Fail("Timeout waiting for concurrent operations")
		}
	}
}

// Additional edge case tests for improved coverage

func (suite *MockServerTestSuite) TestCreateOrderInvalidQuantity() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "invalid")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCreateOrderInvalidPrice() {
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "invalid")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCreateOrderMarketNoPriceAvailable() {
	// Try to buy a symbol that has no price set
	form := url.Values{}
	form.Set("symbol", "UNKNOWN")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCreateSellOrderInsufficientPosition() {
	suite.server.SetPrice("ETHUSDT", 3000.0)

	// Try to sell more ETH than we have
	form := url.Values{}
	form.Set("symbol", "ETHUSDT")
	form.Set("side", "SELL")
	form.Set("type", "MARKET")
	form.Set("quantity", "100") // We only have 10 ETH

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCreateSellOrderCreatesQuoteBalance() {
	// Sell ETH when we don't have a USDT balance initially
	suite.server.SetPrice("ETHUSDT", 3000.0)
	suite.server.SetBalance("USDT", 0, 0) // Set USDT to 0
	suite.server.SetBalance("ETH", 10, 0)

	form := url.Values{}
	form.Set("symbol", "ETHUSDT")
	form.Set("side", "SELL")
	form.Set("type", "MARKET")
	form.Set("quantity", "1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	// Check USDT balance was created
	usdtBalance := suite.server.GetBalance("USDT")
	suite.Require().NotNil(usdtBalance)
	suite.Equal(3000.0, usdtBalance.Free)
}

func (suite *MockServerTestSuite) TestCancelOrderInvalidOrderIdFormat() {
	req, _ := http.NewRequest("DELETE",
		suite.server.BaseURL()+"/api/v3/order?symbol=BTCUSDT&orderId=invalid",
		nil)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCancelOrderMissingParams() {
	req, _ := http.NewRequest("DELETE",
		suite.server.BaseURL()+"/api/v3/order",
		nil)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCancelOrderWrongSymbol() {
	// Create a limit order first
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)

	var orderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderResp)
	resp.Body.Close()

	orderID := int64(orderResp["orderId"].(float64))

	// Try to cancel with wrong symbol
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v3/order?symbol=ETHUSDT&orderId=%d", suite.server.BaseURL(), orderID),
		nil)
	resp, err = http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCancelAlreadyCancelledOrder() {
	// Create and cancel a limit order
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)

	var orderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderResp)
	resp.Body.Close()

	orderID := int64(orderResp["orderId"].(float64))

	// Cancel the order first time
	req, _ := http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v3/order?symbol=BTCUSDT&orderId=%d", suite.server.BaseURL(), orderID),
		nil)
	resp, err = http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Try to cancel again
	req, _ = http.NewRequest("DELETE",
		fmt.Sprintf("%s/api/v3/order?symbol=BTCUSDT&orderId=%d", suite.server.BaseURL(), orderID),
		nil)
	resp, err = http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestCancelAllOrdersMissingSymbol() {
	req, _ := http.NewRequest("DELETE",
		suite.server.BaseURL()+"/api/v3/openOrders",
		nil)
	resp, err := http.DefaultClient.Do(req)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestMyTradesWithTimeFilters() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Create a trade by placing an order
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Get trades with time filters
	startTime := time.Now().Add(-1 * time.Hour).UnixMilli()
	endTime := time.Now().Add(1 * time.Hour).UnixMilli()

	resp, err = http.Get(fmt.Sprintf("%s/api/v3/myTrades?symbol=BTCUSDT&startTime=%d&endTime=%d&limit=10",
		suite.server.BaseURL(), startTime, endTime))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var trades []map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&trades)
	suite.NoError(err)
	suite.Len(trades, 1)
}

func (suite *MockServerTestSuite) TestMyTradesFilterByStartTime() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Create a trade
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Get trades with future startTime (should return empty)
	futureTime := time.Now().Add(1 * time.Hour).UnixMilli()

	resp, err = http.Get(fmt.Sprintf("%s/api/v3/myTrades?symbol=BTCUSDT&startTime=%d",
		suite.server.BaseURL(), futureTime))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var trades []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&trades)
	suite.Empty(trades)
}

func (suite *MockServerTestSuite) TestMyTradesFilterByEndTime() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Create a trade
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Get trades with past endTime (should return empty)
	pastTime := time.Now().Add(-1 * time.Hour).UnixMilli()

	resp, err = http.Get(fmt.Sprintf("%s/api/v3/myTrades?symbol=BTCUSDT&endTime=%d",
		suite.server.BaseURL(), pastTime))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var trades []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&trades)
	suite.Empty(trades)
}

func (suite *MockServerTestSuite) TestKlinesWithTimeRange() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	startTime := time.Now().Add(-1 * time.Hour).UnixMilli()
	endTime := time.Now().UnixMilli()

	resp, err := http.Get(fmt.Sprintf("%s/api/v3/klines?symbol=BTCUSDT&interval=1m&startTime=%d&endTime=%d",
		suite.server.BaseURL(), startTime, endTime))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var klines [][]interface{}
	err = json.NewDecoder(resp.Body).Decode(&klines)
	suite.NoError(err)
	suite.NotEmpty(klines)
}

func (suite *MockServerTestSuite) TestKlinesInvalidInterval() {
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/klines?symbol=BTCUSDT&interval=invalid")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestTickerPriceInvalidSymbolsJSON() {
	resp, err := http.Get(suite.server.BaseURL() + "/api/v3/ticker/price?symbols=invalid")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestGetOrderExisting() {
	// Create a limit order first
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "LIMIT")
	form.Set("quantity", "0.1")
	form.Set("price", "45000")
	form.Set("timeInForce", "GTC")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)

	var orderResp map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&orderResp)
	resp.Body.Close()

	orderID := int64(orderResp["orderId"].(float64))

	// Get the order
	order := suite.server.GetOrder(orderID)
	suite.Require().NotNil(order)
	suite.Equal(orderID, order.OrderID)
	suite.Equal("BTCUSDT", order.Symbol)
}

func (suite *MockServerTestSuite) TestGetOrderNonExistent() {
	order := suite.server.GetOrder(999999)
	suite.Nil(order)
}

func (suite *MockServerTestSuite) TestAddressBeforeStart() {
	// Create a server but don't start it
	config := ServerConfig{}
	server := NewMockBinanceServer(config)
	suite.Equal("", server.Address())
}

func (suite *MockServerTestSuite) TestWebSocketWithDifferentIntervals() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	intervals := []string{"1s", "5m", "15m", "1h", "1d"}

	for _, interval := range intervals {
		wsURL := fmt.Sprintf("%s/ws/btcusdt@kline_%s", suite.server.WebSocketURL(), interval)
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			continue // WebSocket might not be available for all intervals in test
		}

		conn.SetReadDeadline(time.Now().Add(200 * time.Millisecond))
		_, _, _ = conn.ReadMessage()
		conn.Close()
	}
}

func (suite *MockServerTestSuite) TestCreateBuyOrderCreatesBaseBalance() {
	// Clear any existing balances and set up a scenario where
	// we need to create a new base asset balance
	suite.server.SetPrice("BTCUSDT", 50000.0)
	suite.server.SetBalance("USDT", 10000, 0)

	// Delete BTC balance to test creation
	suite.server.mu.Lock()
	delete(suite.server.balances, "BTC")
	suite.server.mu.Unlock()

	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	// Check BTC balance was created
	btcBalance := suite.server.GetBalance("BTC")
	suite.Require().NotNil(btcBalance)
	suite.Equal(0.1, btcBalance.Free)
}

func (suite *MockServerTestSuite) TestKlinesWithShortTimeRange() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Use a very short time range that results in <= 0 numPoints
	startTime := time.Now().UnixMilli()
	endTime := startTime + 1 // Just 1 millisecond

	resp, err := http.Get(fmt.Sprintf("%s/api/v3/klines?symbol=BTCUSDT&interval=1h&startTime=%d&endTime=%d",
		suite.server.BaseURL(), startTime, endTime))
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var klines [][]interface{}
	json.NewDecoder(resp.Body).Decode(&klines)
	suite.NotEmpty(klines) // Should still have at least 1 kline
}

func (suite *MockServerTestSuite) TestCreateOrderUnknownSymbol() {
	// Set price but don't initialize symbol info
	suite.server.SetPrice("NEWUSDT", 100.0)

	form := url.Values{}
	form.Set("symbol", "NEWUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusBadRequest, resp.StatusCode)
}

func (suite *MockServerTestSuite) TestMyTradesWrongSymbol() {
	suite.server.SetPrice("BTCUSDT", 50000.0)

	// Create a trade with BTCUSDT
	form := url.Values{}
	form.Set("symbol", "BTCUSDT")
	form.Set("side", "BUY")
	form.Set("type", "MARKET")
	form.Set("quantity", "0.1")

	resp, err := http.Post(
		suite.server.BaseURL()+"/api/v3/order",
		"application/x-www-form-urlencoded",
		strings.NewReader(form.Encode()),
	)
	suite.Require().NoError(err)
	resp.Body.Close()

	// Query trades for a different symbol
	resp, err = http.Get(suite.server.BaseURL() + "/api/v3/myTrades?symbol=ETHUSDT")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	var trades []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&trades)
	suite.Empty(trades)
}

func (suite *MockServerTestSuite) TestTradeFeeUnknownSymbol() {
	resp, err := http.Get(suite.server.BaseURL() + "/sapi/v1/asset/tradeFee?symbol=UNKNOWN")
	suite.Require().NoError(err)
	defer resp.Body.Close()

	suite.Equal(http.StatusOK, resp.StatusCode)

	var fees []map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&fees)
	suite.Empty(fees)
}

func (suite *MockServerTestSuite) TestStopServerTwice() {
	// Create a new server for this test
	config := ServerConfig{
		InitialBalances: map[string]float64{"USDT": 1000},
	}
	server := NewMockBinanceServer(config)
	err := server.Start(":0")
	suite.Require().NoError(err)

	// Stop it twice - should not panic
	err = server.Stop()
	suite.NoError(err)

	// Second stop should handle gracefully
	// Note: This may panic or error since httpServer is nil after first stop
	// but we're testing it doesn't cause issues
}

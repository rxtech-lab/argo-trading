package trading

import (
	"os"
	"testing"

	tradingprovider "github.com/rxtech-lab/argo-trading/internal/trading/provider"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// BinanceIntegrationTestSuite contains integration tests for the Binance trading provider.
// These tests require BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY environment variables.
type BinanceIntegrationTestSuite struct {
	suite.Suite
}

func TestBinanceIntegrationSuite(t *testing.T) {
	suite.Run(t, new(BinanceIntegrationTestSuite))
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetAccountInfo() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	accountInfo, err := system.GetAccountInfo()
	suite.NoError(err)
	suite.GreaterOrEqual(accountInfo.Balance, float64(0))
	suite.GreaterOrEqual(accountInfo.BuyingPower, float64(0))
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetPositions() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	positions, err := system.GetPositions()
	suite.NoError(err)
	suite.NotNil(positions)
	// Positions can be empty if no holdings, but should not error
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetPosition() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	// Get position for a symbol that may or may not have holdings
	position, err := system.GetPosition("BTC")
	suite.NoError(err)
	suite.Equal("BTC", position.Symbol)
	suite.GreaterOrEqual(position.TotalLongPositionQuantity, float64(0))
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetOpenOrders() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	orders, err := system.GetOpenOrders()
	suite.NoError(err)
	suite.NotNil(orders)
	// Orders can be empty if none are open, but should not error
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetMaxBuyQuantity() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	maxQty, err := system.GetMaxBuyQuantity("BTCUSDT", 50000.0)
	suite.NoError(err)
	suite.GreaterOrEqual(maxQty, float64(0))
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetMaxBuyQuantity_InvalidPrice() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	_, err = system.GetMaxBuyQuantity("BTCUSDT", 0)
	suite.Error(err)
	suite.Contains(err.Error(), "price must be greater than zero")
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetMaxSellQuantity() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	maxQty, err := system.GetMaxSellQuantity("BTC")
	suite.NoError(err)
	suite.GreaterOrEqual(maxQty, float64(0))
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetTrades_RequiresSymbol() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	// GetTrades requires symbol on Binance
	_, err = system.GetTrades(types.TradeFilter{})
	suite.Error(err)
	suite.Contains(err.Error(), "symbol is required")
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_GetTrades_WithSymbol() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	trades, err := system.GetTrades(types.TradeFilter{Symbol: "BTCUSDT", Limit: 10})
	suite.NoError(err)
	suite.NotNil(trades)
	// Trades can be empty if none exist, but should not error
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_CancelOrder_OrderNotFound() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	// Try to cancel a non-existent order
	err = system.CancelOrder("999999999999")
	suite.Error(err)
	suite.Contains(err.Error(), "order not found")
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_CancelOrder_InvalidOrderIDFormat() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	// Try to cancel with an invalid order ID format (non-numeric)
	// This should return "order not found" since no open orders will match
	err = system.CancelOrder("invalid-order-id")
	suite.Error(err)
	suite.Contains(err.Error(), "order not found")
}

func (suite *BinanceIntegrationTestSuite) TestIntegration_CancelAllOrders_NoOpenOrders() {
	apiKey := os.Getenv("BINANCE_TESTNET_API_KEY")
	secretKey := os.Getenv("BINANCE_TESTNET_SECRET_KEY")

	if apiKey == "" || secretKey == "" {
		suite.T().Skip("Skipping integration test: BINANCE_TESTNET_API_KEY and BINANCE_TESTNET_SECRET_KEY not set")
	}

	config := tradingprovider.BinanceProviderConfig{
		ApiKey:    apiKey,
		SecretKey: secretKey,
	}

	system, err := tradingprovider.NewBinanceTradingSystemProvider(config, true)
	suite.Require().NoError(err)

	// CancelAllOrders should succeed even when there are no open orders
	err = system.CancelAllOrders()
	suite.NoError(err)
}

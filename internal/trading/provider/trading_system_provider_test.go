package tradingprovider

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TradingSystemProviderTestSuite struct {
	suite.Suite
}

func TestTradingSystemProviderSuite(t *testing.T) {
	suite.Run(t, new(TradingSystemProviderTestSuite))
}

// Unit Tests - Provider Registry

func (suite *TradingSystemProviderTestSuite) TestGetSupportedProviders() {
	providers := GetSupportedProviders()
	suite.NotEmpty(providers)
	suite.Contains(providers, "binance-paper")
	suite.Contains(providers, "binance-live")
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderInfo_BinancePaper() {
	info, err := GetProviderInfo("binance-paper")
	suite.NoError(err)
	suite.Equal("binance-paper", info.Name)
	suite.Equal("Binance Testnet", info.DisplayName)
	suite.True(info.IsPaperTrading)
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderInfo_BinanceLive() {
	info, err := GetProviderInfo("binance-live")
	suite.NoError(err)
	suite.Equal("binance-live", info.Name)
	suite.Equal("Binance Live", info.DisplayName)
	suite.False(info.IsPaperTrading)
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderInfo_Unsupported() {
	_, err := GetProviderInfo("unsupported-provider")
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderConfigSchema_Binance() {
	schema, err := GetProviderConfigSchema("binance-paper")
	suite.NoError(err)
	suite.NotEmpty(schema)
	suite.Contains(schema, "apiKey")
	suite.Contains(schema, "secretKey")
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderConfigSchema_BinanceLive() {
	schema, err := GetProviderConfigSchema("binance-live")
	suite.NoError(err)
	suite.NotEmpty(schema)
	suite.Contains(schema, "apiKey")
	suite.Contains(schema, "secretKey")
}

func (suite *TradingSystemProviderTestSuite) TestGetProviderConfigSchema_Unsupported() {
	_, err := GetProviderConfigSchema("unsupported-provider")
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_Binance() {
	jsonConfig := `{"apiKey": "test-api-key", "secretKey": "test-secret-key"}`
	config, err := ParseProviderConfig("binance-paper", jsonConfig)
	suite.NoError(err)
	suite.NotNil(config)

	binanceConfig, ok := config.(*BinanceProviderConfig)
	suite.True(ok)
	suite.Equal("test-api-key", binanceConfig.ApiKey)
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_BinanceLive() {
	jsonConfig := `{"apiKey": "test-api-key", "secretKey": "test-secret-key"}`
	config, err := ParseProviderConfig("binance-live", jsonConfig)
	suite.NoError(err)
	suite.NotNil(config)

	binanceConfig, ok := config.(*BinanceProviderConfig)
	suite.True(ok)
	suite.Equal("test-api-key", binanceConfig.ApiKey)
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_Unsupported() {
	_, err := ParseProviderConfig("unsupported-provider", "{}")
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_InvalidJSON() {
	_, err := ParseProviderConfig("binance-paper", "{invalid json}")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse binance config")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_EmptyJSON() {
	_, err := ParseProviderConfig("binance-paper", "{}")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid binance provider config")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_MissingApiKey() {
	jsonConfig := `{"secretKey": "test-secret-key"}`
	_, err := ParseProviderConfig("binance-paper", jsonConfig)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid binance provider config")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_MissingSecretKey() {
	jsonConfig := `{"apiKey": "test-api-key"}`
	_, err := ParseProviderConfig("binance-paper", jsonConfig)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid binance provider config")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_EmptyProviderName() {
	jsonConfig := `{"apiKey": "test-api-key", "secretKey": "test-secret-key"}`
	_, err := ParseProviderConfig("", jsonConfig)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

func (suite *TradingSystemProviderTestSuite) TestParseProviderConfig_BinanceLive_InvalidJSON() {
	_, err := ParseProviderConfig("binance-live", "{invalid json}")
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse binance config")
}

// Unit Tests - NewTradingSystemProvider

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_BinancePaper() {
	config := &BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewTradingSystemProvider(ProviderBinancePaper, config)
	suite.NoError(err)
	suite.NotNil(provider)
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_BinanceLive() {
	config := &BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	provider, err := NewTradingSystemProvider(ProviderBinanceLive, config)
	suite.NoError(err)
	suite.NotNil(provider)
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_InvalidConfigType_BinancePaper() {
	// Pass wrong config type
	config := "invalid config"
	_, err := NewTradingSystemProvider(ProviderBinancePaper, config)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid config type")
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_InvalidConfigType_BinanceLive() {
	// Pass wrong config type
	config := "invalid config"
	_, err := NewTradingSystemProvider(ProviderBinanceLive, config)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid config type")
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_NilConfig() {
	_, err := NewTradingSystemProvider(ProviderBinancePaper, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid config type")
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_Unsupported() {
	config := &BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	_, err := NewTradingSystemProvider("unsupported", config)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

func (suite *TradingSystemProviderTestSuite) TestNewTradingSystemProvider_EmptyProviderType() {
	config := &BinanceProviderConfig{
		ApiKey:    "test-api-key",
		SecretKey: "test-secret-key",
	}
	_, err := NewTradingSystemProvider("", config)
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported trading provider")
}

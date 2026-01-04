package marketdata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProviderRegistryTestSuite struct {
	suite.Suite
}

func TestProviderRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(ProviderRegistryTestSuite))
}

func (suite *ProviderRegistryTestSuite) TestGetSupportedProviders() {
	providers := GetSupportedProviders()

	suite.NotEmpty(providers)
	suite.Contains(providers, "polygon")
	suite.Contains(providers, "binance")
	suite.Len(providers, 2)
}

func (suite *ProviderRegistryTestSuite) TestGetProviderInfo_Polygon() {
	info, err := GetProviderInfo("polygon")

	suite.NoError(err)
	suite.Equal("polygon", info.Name)
	suite.Equal("Polygon.io", info.DisplayName)
	suite.True(info.RequiresAuth)
	suite.NotEmpty(info.Description)
}

func (suite *ProviderRegistryTestSuite) TestGetProviderInfo_Binance() {
	info, err := GetProviderInfo("binance")

	suite.NoError(err)
	suite.Equal("binance", info.Name)
	suite.Equal("Binance", info.DisplayName)
	suite.False(info.RequiresAuth)
	suite.NotEmpty(info.Description)
}

func (suite *ProviderRegistryTestSuite) TestGetProviderInfo_InvalidProvider() {
	_, err := GetProviderInfo("invalid")

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")
}

func (suite *ProviderRegistryTestSuite) TestGetDownloadConfigSchema_Polygon() {
	schema, err := GetDownloadConfigSchema("polygon")

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Verify schema has expected structure
	suite.Contains(schemaMap, "properties")
	suite.Contains(schemaMap, "type")
	suite.Equal("object", schemaMap["type"])
}

func (suite *ProviderRegistryTestSuite) TestGetDownloadConfigSchema_Binance() {
	schema, err := GetDownloadConfigSchema("binance")

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Verify schema has expected structure
	suite.Contains(schemaMap, "properties")
	suite.Contains(schemaMap, "type")
	suite.Equal("object", schemaMap["type"])
}

func (suite *ProviderRegistryTestSuite) TestGetDownloadConfigSchema_InvalidProvider() {
	schema, err := GetDownloadConfigSchema("invalid")

	suite.Error(err)
	suite.Empty(schema)
	suite.Contains(err.Error(), "unsupported provider")
}

func (suite *ProviderRegistryTestSuite) TestParseDownloadConfig_Polygon() {
	jsonConfig := `{
		"ticker": "SPY",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1d",
		"dataPath": "/tmp/data",
		"apiKey": "test-api-key"
	}`

	config, err := ParseDownloadConfig("polygon", jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)

	polygonConfig, ok := config.(*PolygonDownloadConfig)
	suite.True(ok)
	suite.Equal("SPY", polygonConfig.Ticker)
	suite.Equal("test-api-key", polygonConfig.ApiKey)
}

func (suite *ProviderRegistryTestSuite) TestParseDownloadConfig_Binance() {
	jsonConfig := `{
		"ticker": "BTCUSDT",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1h",
		"dataPath": "/tmp/data"
	}`

	config, err := ParseDownloadConfig("binance", jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)

	binanceConfig, ok := config.(*BinanceDownloadConfig)
	suite.True(ok)
	suite.Equal("BTCUSDT", binanceConfig.Ticker)
}

func (suite *ProviderRegistryTestSuite) TestParseDownloadConfig_InvalidProvider() {
	jsonConfig := `{"ticker": "SPY"}`

	_, err := ParseDownloadConfig("invalid", jsonConfig)

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported provider")
}

func (suite *ProviderRegistryTestSuite) TestParseDownloadConfig_InvalidJSON() {
	jsonConfig := `{invalid json}`

	_, err := ParseDownloadConfig("polygon", jsonConfig)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse JSON")
}

func (suite *ProviderRegistryTestSuite) TestParseDownloadConfig_MissingRequiredFields() {
	jsonConfig := `{
		"ticker": "SPY"
	}`

	_, err := ParseDownloadConfig("polygon", jsonConfig)

	suite.Error(err)
}

package provider

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type StreamRegistryTestSuite struct {
	suite.Suite
}

func TestStreamRegistryTestSuite(t *testing.T) {
	suite.Run(t, new(StreamRegistryTestSuite))
}

func (suite *StreamRegistryTestSuite) TestGetStreamConfigSchema_Polygon() {
	schema, err := GetStreamConfigSchema("polygon")

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]any
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Verify schema has expected structure
	suite.Contains(schemaMap, "properties")
	suite.Contains(schemaMap, "type")
	suite.Equal("object", schemaMap["type"])

	// Verify apiKey property exists
	properties, ok := schemaMap["properties"].(map[string]any)
	suite.True(ok)
	suite.Contains(properties, "apiKey")
	suite.Contains(properties, "symbols")
	suite.Contains(properties, "interval")
}

func (suite *StreamRegistryTestSuite) TestGetStreamConfigSchema_Binance() {
	schema, err := GetStreamConfigSchema("binance")

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]any
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Verify schema has expected structure
	suite.Contains(schemaMap, "properties")
	suite.Contains(schemaMap, "type")
	suite.Equal("object", schemaMap["type"])

	// Verify symbols and interval exist but no apiKey
	properties, ok := schemaMap["properties"].(map[string]any)
	suite.True(ok)
	suite.Contains(properties, "symbols")
	suite.Contains(properties, "interval")
	suite.NotContains(properties, "apiKey")
}

func (suite *StreamRegistryTestSuite) TestGetStreamConfigSchema_InvalidProvider() {
	schema, err := GetStreamConfigSchema("invalid")

	suite.Error(err)
	suite.Empty(schema)
	suite.Contains(err.Error(), "unsupported market data provider")
}

func (suite *StreamRegistryTestSuite) TestGetStreamKeychainFields_Polygon() {
	fields, err := GetStreamKeychainFields("polygon")

	suite.NoError(err)
	suite.Equal([]string{"apiKey"}, fields)
}

func (suite *StreamRegistryTestSuite) TestGetStreamKeychainFields_Binance() {
	fields, err := GetStreamKeychainFields("binance")

	suite.NoError(err)
	suite.Empty(fields)
}

func (suite *StreamRegistryTestSuite) TestGetStreamKeychainFields_InvalidProvider() {
	_, err := GetStreamKeychainFields("invalid")

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported market data provider")
}

func (suite *StreamRegistryTestSuite) TestParseStreamConfig_Polygon() {
	jsonConfig := `{
		"symbols": ["SPY", "AAPL"],
		"interval": "1m",
		"apiKey": "test-api-key"
	}`

	config, err := ParseStreamConfig("polygon", jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)

	polygonConfig, ok := config.(*PolygonStreamConfig)
	suite.True(ok)
	suite.Equal([]string{"SPY", "AAPL"}, polygonConfig.Symbols)
	suite.Equal("1m", polygonConfig.Interval)
	suite.Equal("test-api-key", polygonConfig.ApiKey)
}

func (suite *StreamRegistryTestSuite) TestParseStreamConfig_Binance() {
	jsonConfig := `{
		"symbols": ["BTCUSDT"],
		"interval": "1h"
	}`

	config, err := ParseStreamConfig("binance", jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)

	binanceConfig, ok := config.(*BinanceStreamConfig)
	suite.True(ok)
	suite.Equal([]string{"BTCUSDT"}, binanceConfig.Symbols)
	suite.Equal("1h", binanceConfig.Interval)
}

func (suite *StreamRegistryTestSuite) TestParseStreamConfig_InvalidProvider() {
	_, err := ParseStreamConfig("invalid", `{}`)

	suite.Error(err)
	suite.Contains(err.Error(), "unsupported market data provider")
}

func (suite *StreamRegistryTestSuite) TestParseStreamConfig_InvalidJSON() {
	_, err := ParseStreamConfig("polygon", `{invalid json}`)

	suite.Error(err)
}

func (suite *StreamRegistryTestSuite) TestParseStreamConfig_MissingRequiredFields() {
	jsonConfig := `{
		"symbols": ["SPY"]
	}`

	_, err := ParseStreamConfig("polygon", jsonConfig)

	suite.Error(err)
}

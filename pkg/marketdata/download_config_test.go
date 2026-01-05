package marketdata

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type DownloadConfigTestSuite struct {
	suite.Suite
}

func TestDownloadConfigTestSuite(t *testing.T) {
	suite.Run(t, new(DownloadConfigTestSuite))
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigValidation_Valid() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "SPY",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1d",
		},
		ApiKey: "test-api-key",
	}

	err := config.Validate()
	suite.NoError(err)
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigValidation_MissingTicker() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1d",
		},
		ApiKey: "test-api-key",
	}

	err := config.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "Ticker")
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigValidation_MissingApiKey() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "SPY",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1d",
		},
		ApiKey: "",
	}

	err := config.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "ApiKey")
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigValidation_InvalidInterval() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "SPY",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "invalid",
		},
		ApiKey: "test-api-key",
	}

	err := config.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "Interval")
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigValidation_InvalidDateFormat() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "SPY",
			StartDate: "2024-01-01", // Missing time component
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1d",
		},
		ApiKey: "test-api-key",
	}

	err := config.Validate()
	suite.Error(err)
	suite.Contains(err.Error(), "startDate")
}

func (suite *DownloadConfigTestSuite) TestBinanceConfigValidation_Valid() {
	config := &BinanceDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "BTCUSDT",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1h",
		},
	}

	err := config.Validate()
	suite.NoError(err)
}

func (suite *DownloadConfigTestSuite) TestBinanceConfigValidation_MissingFields() {
	config := &BinanceDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1h",
		},
	}

	err := config.Validate()
	suite.Error(err)
}

func (suite *DownloadConfigTestSuite) TestParsePolygonConfig_Valid() {
	jsonConfig := `{
		"ticker": "SPY",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1d",
		"apiKey": "test-api-key"
	}`

	config, err := ParsePolygonConfig(jsonConfig)
	suite.NoError(err)
	suite.Equal("SPY", config.Ticker)
	suite.Equal("test-api-key", config.ApiKey)
}

func (suite *DownloadConfigTestSuite) TestParsePolygonConfig_InvalidJSON() {
	jsonConfig := `{invalid json}`

	_, err := ParsePolygonConfig(jsonConfig)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse JSON")
}

func (suite *DownloadConfigTestSuite) TestParsePolygonConfig_MissingRequiredField() {
	jsonConfig := `{
		"ticker": "SPY",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1d"
	}`

	_, err := ParsePolygonConfig(jsonConfig)
	suite.Error(err)
	suite.Contains(err.Error(), "ApiKey")
}

func (suite *DownloadConfigTestSuite) TestParseBinanceConfig_Valid() {
	jsonConfig := `{
		"ticker": "BTCUSDT",
		"startDate": "2024-01-01T00:00:00Z",
		"endDate": "2024-12-31T23:59:59Z",
		"interval": "1h"
	}`

	config, err := ParseBinanceConfig(jsonConfig)
	suite.NoError(err)
	suite.Equal("BTCUSDT", config.Ticker)
	suite.Equal("1h", config.Interval)
}

func (suite *DownloadConfigTestSuite) TestToDownloadParams() {
	config := &BaseDownloadConfig{
		Ticker:    "SPY",
		StartDate: "2024-01-01T00:00:00Z",
		EndDate:   "2024-12-31T23:59:59Z",
		Interval:  "1d",
	}

	params, err := config.ToDownloadParams()
	suite.NoError(err)
	suite.Equal("SPY", params.Ticker)
	suite.Equal(1, params.Multiplier)
}

func (suite *DownloadConfigTestSuite) TestPolygonToClientConfig() {
	config := &PolygonDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "SPY",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1d",
		},
		ApiKey: "test-api-key",
	}

	clientConfig := config.ToClientConfig("/tmp/data")
	suite.Equal(ProviderPolygon, clientConfig.ProviderType)
	suite.Equal(WriterDuckDB, clientConfig.WriterType)
	suite.Equal("/tmp/data", clientConfig.DataPath)
	suite.Equal("test-api-key", clientConfig.PolygonApiKey)
}

func (suite *DownloadConfigTestSuite) TestBinanceToClientConfig() {
	config := &BinanceDownloadConfig{
		BaseDownloadConfig: BaseDownloadConfig{
			Ticker:    "BTCUSDT",
			StartDate: "2024-01-01T00:00:00Z",
			EndDate:   "2024-12-31T23:59:59Z",
			Interval:  "1h",
		},
	}

	clientConfig := config.ToClientConfig("/tmp/data")
	suite.Equal(ProviderBinance, clientConfig.ProviderType)
	suite.Equal(WriterDuckDB, clientConfig.WriterType)
	suite.Equal("/tmp/data", clientConfig.DataPath)
}

func (suite *DownloadConfigTestSuite) TestAllIntervals() {
	intervals := []string{"1s", "1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w", "1M"}

	for _, interval := range intervals {
		config := &BinanceDownloadConfig{
			BaseDownloadConfig: BaseDownloadConfig{
				Ticker:    "BTCUSDT",
				StartDate: "2024-01-01T00:00:00Z",
				EndDate:   "2024-12-31T23:59:59Z",
				Interval:  interval,
			},
		}

		err := config.Validate()
		suite.NoError(err, "interval %s should be valid", interval)
	}
}

func (suite *DownloadConfigTestSuite) TestPolygonConfigJSONSchema() {
	schema, err := GetDownloadConfigSchema("polygon")
	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check that required fields are in the schema
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok, "schema should have properties")
	suite.Contains(properties, "ticker")
	suite.Contains(properties, "startDate")
	suite.Contains(properties, "endDate")
	suite.Contains(properties, "interval")
	suite.NotContains(properties, "dataPath") // dataPath is now a separate parameter
	suite.Contains(properties, "apiKey")
}

func (suite *DownloadConfigTestSuite) TestBinanceConfigJSONSchema() {
	schema, err := GetDownloadConfigSchema("binance")
	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var schemaMap map[string]interface{}
	err = json.Unmarshal([]byte(schema), &schemaMap)
	suite.NoError(err)

	// Check that required fields are in the schema
	properties, ok := schemaMap["properties"].(map[string]interface{})
	suite.True(ok, "schema should have properties")
	suite.Contains(properties, "ticker")
	suite.Contains(properties, "startDate")
	suite.Contains(properties, "endDate")
	suite.Contains(properties, "interval")
	suite.NotContains(properties, "dataPath") // dataPath is now a separate parameter

	// Binance should not have apiKey in schema
	suite.NotContains(properties, "apiKey")
}

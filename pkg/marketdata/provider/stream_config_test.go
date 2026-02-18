package provider

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
)

type StreamConfigTestSuite struct {
	suite.Suite
}

func TestStreamConfigTestSuite(t *testing.T) {
	suite.Run(t, new(StreamConfigTestSuite))
}

func (suite *StreamConfigTestSuite) TestBaseStreamConfig_Validate_Valid() {
	config := &BaseStreamConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "1m",
	}

	err := config.Validate()
	suite.NoError(err)
}

func (suite *StreamConfigTestSuite) TestBaseStreamConfig_Validate_MissingSymbols() {
	config := &BaseStreamConfig{
		Symbols:  nil,
		Interval: "1m",
	}

	err := config.Validate()
	suite.Error(err)
}

func (suite *StreamConfigTestSuite) TestBaseStreamConfig_Validate_MissingInterval() {
	config := &BaseStreamConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "",
	}

	err := config.Validate()
	suite.Error(err)
}

func (suite *StreamConfigTestSuite) TestBaseStreamConfig_Validate_InvalidInterval() {
	config := &BaseStreamConfig{
		Symbols:  []string{"BTCUSDT"},
		Interval: "2m",
	}

	err := config.Validate()
	suite.Error(err)
}

func (suite *StreamConfigTestSuite) TestPolygonStreamConfig_Validate_Valid() {
	config := &PolygonStreamConfig{
		BaseStreamConfig: BaseStreamConfig{
			Symbols:  []string{"SPY"},
			Interval: "1m",
		},
		ApiKey: "test-api-key",
	}

	err := config.Validate()
	suite.NoError(err)
}

func (suite *StreamConfigTestSuite) TestPolygonStreamConfig_Validate_MissingApiKey() {
	config := &PolygonStreamConfig{
		BaseStreamConfig: BaseStreamConfig{
			Symbols:  []string{"SPY"},
			Interval: "1m",
		},
		ApiKey: "",
	}

	err := config.Validate()
	suite.Error(err)
}

func (suite *StreamConfigTestSuite) TestBinanceStreamConfig_Validate_Valid() {
	config := &BinanceStreamConfig{
		BaseStreamConfig: BaseStreamConfig{
			Symbols:  []string{"BTCUSDT", "ETHUSDT"},
			Interval: "5m",
		},
	}

	err := config.Validate()
	suite.NoError(err)
}

func (suite *StreamConfigTestSuite) TestParsePolygonStreamConfig_Valid() {
	jsonConfig := `{
		"symbols": ["SPY", "AAPL"],
		"interval": "1m",
		"apiKey": "test-api-key"
	}`

	config, err := ParsePolygonStreamConfig(jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal([]string{"SPY", "AAPL"}, config.Symbols)
	suite.Equal("1m", config.Interval)
	suite.Equal("test-api-key", config.ApiKey)
}

func (suite *StreamConfigTestSuite) TestParsePolygonStreamConfig_InvalidJSON() {
	jsonConfig := `{invalid json}`

	_, err := ParsePolygonStreamConfig(jsonConfig)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse JSON config")
}

func (suite *StreamConfigTestSuite) TestParsePolygonStreamConfig_MissingRequiredFields() {
	jsonConfig := `{
		"symbols": ["SPY"]
	}`

	_, err := ParsePolygonStreamConfig(jsonConfig)

	suite.Error(err)
}

func (suite *StreamConfigTestSuite) TestParseBinanceStreamConfig_Valid() {
	jsonConfig := `{
		"symbols": ["BTCUSDT"],
		"interval": "1h"
	}`

	config, err := ParseBinanceStreamConfig(jsonConfig)

	suite.NoError(err)
	suite.NotNil(config)
	suite.Equal([]string{"BTCUSDT"}, config.Symbols)
	suite.Equal("1h", config.Interval)
}

func (suite *StreamConfigTestSuite) TestParseBinanceStreamConfig_InvalidJSON() {
	jsonConfig := `{invalid json}`

	_, err := ParseBinanceStreamConfig(jsonConfig)

	suite.Error(err)
	suite.Contains(err.Error(), "failed to parse JSON config")
}

func (suite *StreamConfigTestSuite) TestPolygonStreamConfig_KeychainFields() {
	//nolint:exhaustruct // Empty struct for field introspection
	fields := strategy.GetKeychainFields(PolygonStreamConfig{})

	suite.Equal([]string{"apiKey"}, fields)
}

func (suite *StreamConfigTestSuite) TestBinanceStreamConfig_KeychainFields() {
	//nolint:exhaustruct // Empty struct for field introspection
	fields := strategy.GetKeychainFields(BinanceStreamConfig{})

	suite.Empty(fields)
}

package engine

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type ConfigTestSuite struct {
	suite.Suite
}

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigTestSuite))
}

func (suite *ConfigTestSuite) TestEmptyConfig() {
	config := EmptyConfig()

	suite.Equal(0.0, config.InitialCapital)
	suite.Equal(commission_fee.BrokerInteractiveBroker, config.Broker)
	suite.True(config.StartTime.IsNone())
	suite.True(config.EndTime.IsNone())
	suite.Equal(1, config.DecimalPrecision)
	suite.Equal(1000, config.MarketDataCacheSize)
}

func (suite *ConfigTestSuite) TestTestConfig() {
	startTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)
	broker := commission_fee.BrokerZero

	config := TestConfig(startTime, endTime, broker)

	suite.Equal(10000.0, config.InitialCapital)
	suite.Equal(broker, config.Broker)
	suite.True(config.StartTime.IsSome())
	suite.True(config.EndTime.IsSome())
	suite.Equal(startTime, config.StartTime.Unwrap())
	suite.Equal(endTime, config.EndTime.Unwrap())
	suite.Equal(1, config.DecimalPrecision)
	suite.Equal(1000, config.MarketDataCacheSize)
}

func (suite *ConfigTestSuite) TestTestConfigWithInteractiveBroker() {
	startTime := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 12, 15, 0, 0, 0, 0, time.UTC)
	broker := commission_fee.BrokerInteractiveBroker

	config := TestConfig(startTime, endTime, broker)

	suite.Equal(broker, config.Broker)
	suite.Equal(startTime, config.StartTime.Unwrap())
	suite.Equal(endTime, config.EndTime.Unwrap())
}

func (suite *ConfigTestSuite) TestGenerateSchema() {
	config := &BacktestEngineV1Config{}
	schema, err := config.GenerateSchema()

	suite.NoError(err)
	suite.NotNil(schema)
	suite.Equal("backtest-engine-v1-config", schema.Title)
	suite.Equal("Configuration schema for BacktestEngineV1", schema.Description)
	suite.Equal("http://json-schema.org/draft-07/schema#", schema.Version)
}

func (suite *ConfigTestSuite) TestGenerateSchemaJSON() {
	config := &BacktestEngineV1Config{}
	schemaJSON, err := config.GenerateSchemaJSON()

	suite.NoError(err)
	suite.NotEmpty(schemaJSON)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(schemaJSON), &result)
	suite.NoError(err)

	// Check schema properties
	suite.Contains(result, "title")
	suite.Equal("backtest-engine-v1-config", result["title"])
}

func (suite *ConfigTestSuite) TestUnmarshalYAMLComplete() {
	yamlData := `
initial_capital: 50000
broker: interactive_broker
start_time: 2023-01-01T00:00:00Z
end_time: 2023-12-31T00:00:00Z
decimal_precision: 2
market_data_cache_size: 500
`

	var config BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(yamlData), &config)

	suite.NoError(err)
	suite.Equal(50000.0, config.InitialCapital)
	suite.Equal(commission_fee.BrokerInteractiveBroker, config.Broker)
	suite.True(config.StartTime.IsSome())
	suite.True(config.EndTime.IsSome())
	suite.Equal(2, config.DecimalPrecision)
	suite.Equal(500, config.MarketDataCacheSize)

	// Check dates
	startTime := config.StartTime.Unwrap()
	suite.Equal(2023, startTime.Year())
	suite.Equal(time.January, startTime.Month())
	suite.Equal(1, startTime.Day())

	endTime := config.EndTime.Unwrap()
	suite.Equal(2023, endTime.Year())
	suite.Equal(time.December, endTime.Month())
	suite.Equal(31, endTime.Day())
}

func (suite *ConfigTestSuite) TestUnmarshalYAMLWithoutTimes() {
	yamlData := `
initial_capital: 25000
broker: zero_commission
decimal_precision: 0
`

	var config BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(yamlData), &config)

	suite.NoError(err)
	suite.Equal(25000.0, config.InitialCapital)
	suite.Equal(commission_fee.BrokerZero, config.Broker)
	suite.True(config.StartTime.IsNone())
	suite.True(config.EndTime.IsNone())
	suite.Equal(0, config.DecimalPrecision)
}

func (suite *ConfigTestSuite) TestUnmarshalYAMLOnlyStartTime() {
	yamlData := `
initial_capital: 10000
broker: zero_commission
start_time: 2024-06-01T00:00:00Z
decimal_precision: 1
`

	var config BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(yamlData), &config)

	suite.NoError(err)
	suite.True(config.StartTime.IsSome())
	suite.True(config.EndTime.IsNone())
}

func (suite *ConfigTestSuite) TestUnmarshalYAMLOnlyEndTime() {
	yamlData := `
initial_capital: 10000
broker: zero_commission
end_time: 2024-12-01T00:00:00Z
decimal_precision: 1
`

	var config BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(yamlData), &config)

	suite.NoError(err)
	suite.True(config.StartTime.IsNone())
	suite.True(config.EndTime.IsSome())
}

func (suite *ConfigTestSuite) TestUnmarshalYAMLInvalid() {
	yamlData := `
initial_capital: not_a_number
`

	var config BacktestEngineV1Config
	err := yaml.Unmarshal([]byte(yamlData), &config)

	suite.Error(err)
}

func (suite *ConfigTestSuite) TestConfigStructFields() {
	config := BacktestEngineV1Config{
		InitialCapital:      100000.0,
		Broker:              commission_fee.BrokerInteractiveBroker,
		StartTime:           optional.Some(time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)),
		EndTime:             optional.Some(time.Date(2023, 12, 31, 0, 0, 0, 0, time.UTC)),
		DecimalPrecision:    3,
		MarketDataCacheSize: 2000,
	}

	suite.Equal(100000.0, config.InitialCapital)
	suite.Equal(commission_fee.BrokerInteractiveBroker, config.Broker)
	suite.Equal(3, config.DecimalPrecision)
	suite.Equal(2000, config.MarketDataCacheSize)
	suite.True(config.StartTime.IsSome())
	suite.True(config.EndTime.IsSome())
}

func (suite *ConfigTestSuite) TestGenerateSchemaWithValues() {
	config := &BacktestEngineV1Config{
		InitialCapital:      50000.0,
		Broker:              commission_fee.BrokerZero,
		DecimalPrecision:    2,
		MarketDataCacheSize: 500,
	}

	schema, err := config.GenerateSchema()
	suite.NoError(err)
	suite.NotNil(schema)
}

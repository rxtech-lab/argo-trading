package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
)

type BacktestEngineV1Config struct {
	InitialCapital float64                    `yaml:"initial_capital" json:"initial_capital" jsonschema:"title=Initial Capital,description=Starting capital for the backtest in USD,minimum=0"`
	Broker         commission_fee.Broker      `yaml:"broker" json:"broker" jsonschema:"title=Broker,description=The broker to use for commission calculations"`
	StartTime      optional.Option[time.Time] `yaml:"start_time" json:"start_time" jsonschema:"title=Start Time,description=Optional start time for the backtest period"`
	EndTime        optional.Option[time.Time] `yaml:"end_time" json:"end_time" jsonschema:"title=End Time,description=Optional end time for the backtest period"`
}

// UnmarshalYAML implements custom unmarshaling for BacktestEngineV1Config
func (c *BacktestEngineV1Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Config struct {
		InitialCapital float64               `yaml:"initial_capital"`
		Broker         commission_fee.Broker `yaml:"broker"`
		StartTime      *time.Time            `yaml:"start_time"`
		EndTime        *time.Time            `yaml:"end_time"`
	}

	var config Config
	if err := unmarshal(&config); err != nil {
		return err
	}

	c.InitialCapital = config.InitialCapital
	c.Broker = config.Broker
	if config.StartTime != nil {
		c.StartTime = optional.Some(*config.StartTime)
	}
	if config.EndTime != nil {
		c.EndTime = optional.Some(*config.EndTime)
	}

	return nil
}

// GenerateSchema generates a JSON schema for the BacktestEngineV1Config
func (c *BacktestEngineV1Config) GenerateSchema() (*jsonschema.Schema, error) {
	reflector := jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  false,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			fmt.Println("t", t.String())
			if t.String() == "optional.Option[time.Time]" {
				return &jsonschema.Schema{
					Type:   "string",
					Format: "date-time",
				}
			}
			if strings.Contains(t.String(), "commission_fee.Broker") {
				return &jsonschema.Schema{
					Type: "string",
					Enum: commission_fee.AllBrokers,
				}
			}
			return nil
		},
	}

	// Generate schema from BacktestEngineV1Config struct
	schema := reflector.Reflect(c)

	// Set schema metadata
	schema.Title = "backtest-engine-v1-config"
	schema.Description = "Configuration schema for BacktestEngineV1"
	schema.Version = "http://json-schema.org/draft-07/schema#"

	return schema, nil
}

// GenerateSchemaJSON generates a JSON schema string for the BacktestEngineV1Config
func (c *BacktestEngineV1Config) GenerateSchemaJSON() (string, error) {
	schema, err := c.GenerateSchema()
	if err != nil {
		return "", err
	}

	schemaBytes, err := json.MarshalIndent(schema, "", "  ")
	if err != nil {
		return "", err
	}

	return string(schemaBytes), nil
}

func TestConfig(startTime time.Time, endTime time.Time, broker commission_fee.Broker) BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital: 10000,
		Broker:         broker,
		StartTime:      optional.Some(startTime),
		EndTime:        optional.Some(endTime),
	}
}

// EmptyConfig returns a BacktestEngineV1Config with default values
func EmptyConfig() BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital: 0,
		Broker:         commission_fee.BrokerInteractiveBroker,
		StartTime:      optional.None[time.Time](),
		EndTime:        optional.None[time.Time](),
	}
}

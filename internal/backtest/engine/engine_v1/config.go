package engine

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"time"

	"github.com/invopop/jsonschema"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
)

// PortfolioCalculationStrategy selects how individual and cumulative PnL is
// calculated across entry/exit trades. FIFO matches each exit against the
// earliest unmatched entries; AverageCost uses the running weighted-average
// cost of the currently-open position.
type PortfolioCalculationStrategy string

const (
	// PortfolioCalculationFIFO computes PnL by first-in-first-out lot matching.
	PortfolioCalculationFIFO PortfolioCalculationStrategy = "fifo"
	// PortfolioCalculationAverageCost computes PnL using the running weighted
	// average cost of the currently-open position.
	PortfolioCalculationAverageCost PortfolioCalculationStrategy = "average_cost"
)

// AllPortfolioCalculationStrategies is the list of supported portfolio
// calculation strategies (used by schema generation).
var AllPortfolioCalculationStrategies = []any{
	string(PortfolioCalculationFIFO),
	string(PortfolioCalculationAverageCost),
}

type BacktestEngineV1Config struct {
	InitialCapital       float64                      `yaml:"initial_capital" json:"initial_capital" jsonschema:"title=Initial Capital,description=Starting capital for the backtest in USD,minimum=0"`
	Broker               commission_fee.Broker        `yaml:"broker" json:"broker" jsonschema:"title=Broker,description=The broker to use for commission calculations"`
	StartTime            optional.Option[time.Time]   `yaml:"start_time" json:"start_time" jsonschema:"title=Start Time,description=Optional start time for the backtest period"`
	EndTime              optional.Option[time.Time]   `yaml:"end_time" json:"end_time" jsonschema:"title=End Time,description=Optional end time for the backtest period"`
	DecimalPrecision     int                          `yaml:"decimal_precision" json:"decimal_precision" jsonschema:"title=Decimal Precision,description=The number of decimal places allowed for quantity (0 means integers only, higher values allow more decimal places),minimum=0,default=1"`
	MarketDataCacheSize  int                          `yaml:"market_data_cache_size" json:"market_data_cache_size" jsonschema:"title=Market Data Cache Size,description=The number of market data points to cache per symbol using sliding window algorithm. When data requests exceed cache size the system falls back to DuckDB. Set to 0 to disable caching.,minimum=0,default=1000"`
	PortfolioCalculation PortfolioCalculationStrategy `yaml:"portfolio_calculation" json:"portfolio_calculation" jsonschema:"title=Portfolio Calculation Strategy,description=How individual-trade and cumulative PnL are computed. 'fifo' matches exits against earliest entries; 'average_cost' uses the running weighted-average cost of the currently-open position. Defaults to 'average_cost' when unset.,default=average_cost"`
}

// UnmarshalYAML implements custom unmarshaling for BacktestEngineV1Config.
func (c *BacktestEngineV1Config) UnmarshalYAML(unmarshal func(interface{}) error) error {
	type Config struct {
		InitialCapital       float64                      `yaml:"initial_capital"`
		Broker               commission_fee.Broker        `yaml:"broker"`
		StartTime            *time.Time                   `yaml:"start_time"`
		EndTime              *time.Time                   `yaml:"end_time"`
		DecimalPrecision     int                          `yaml:"decimal_precision"`
		MarketDataCacheSize  int                          `yaml:"market_data_cache_size"`
		PortfolioCalculation PortfolioCalculationStrategy `yaml:"portfolio_calculation"`
	}

	var config Config
	if err := unmarshal(&config); err != nil {
		return err
	}

	c.InitialCapital = config.InitialCapital
	c.Broker = config.Broker
	c.DecimalPrecision = config.DecimalPrecision
	c.MarketDataCacheSize = config.MarketDataCacheSize
	c.PortfolioCalculation = config.PortfolioCalculation

	if config.StartTime != nil {
		c.StartTime = optional.Some(*config.StartTime)
	}

	if config.EndTime != nil {
		c.EndTime = optional.Some(*config.EndTime)
	}

	return nil
}

// GenerateSchema generates a JSON schema for the BacktestEngineV1Config.
func (c *BacktestEngineV1Config) GenerateSchema() (*jsonschema.Schema, error) {
	//nolint:exhaustruct // third-party struct with many optional fields
	reflector := jsonschema.Reflector{
		RequiredFromJSONSchemaTags: true,
		ExpandedStruct:             true,
		AllowAdditionalProperties:  false,
		Mapper: func(t reflect.Type) *jsonschema.Schema {
			fmt.Println("t", t.String())
			if t.String() == "optional.Option[time.Time]" {
				//nolint:exhaustruct // third-party struct with many optional fields
				return &jsonschema.Schema{
					Type:   "string",
					Format: "date-time",
				}
			}
			if strings.Contains(t.String(), "commission_fee.Broker") {
				//nolint:exhaustruct // third-party struct with many optional fields
				return &jsonschema.Schema{
					Type: "string",
					Enum: commission_fee.AllBrokers,
				}
			}
			if strings.Contains(t.String(), "PortfolioCalculationStrategy") {
				//nolint:exhaustruct // third-party struct with many optional fields
				return &jsonschema.Schema{
					Type: "string",
					Enum: AllPortfolioCalculationStrategies,
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

// GenerateSchemaJSON generates a JSON schema string for the BacktestEngineV1Config.
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
		InitialCapital:       10000,
		Broker:               broker,
		StartTime:            optional.Some(startTime),
		EndTime:              optional.Some(endTime),
		DecimalPrecision:     1,
		MarketDataCacheSize:  1000,
		PortfolioCalculation: PortfolioCalculationAverageCost,
	}
}

// EmptyConfig returns a BacktestEngineV1Config with default values.
func EmptyConfig() BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital:       0,
		Broker:               commission_fee.BrokerInteractiveBroker,
		StartTime:            optional.None[time.Time](),
		EndTime:              optional.None[time.Time](),
		DecimalPrecision:     1,
		MarketDataCacheSize:  1000,
		PortfolioCalculation: PortfolioCalculationAverageCost,
	}
}

// ResolvePortfolioCalculation returns the configured portfolio calculation
// strategy, defaulting to PortfolioCalculationAverageCost when the value is
// unset or unrecognised.
func ResolvePortfolioCalculation(s PortfolioCalculationStrategy) PortfolioCalculationStrategy {
	switch s {
	case PortfolioCalculationFIFO, PortfolioCalculationAverageCost:
		return s
	default:
		return PortfolioCalculationAverageCost
	}
}

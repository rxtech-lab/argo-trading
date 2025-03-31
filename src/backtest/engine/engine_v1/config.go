package engine

import (
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
)

type BacktestEngineV1Config struct {
	InitialCapital float64                    `yaml:"initial_capital"`
	Broker         commission_fee.Broker      `yaml:"broker"`
	StartTime      optional.Option[time.Time] `yaml:"start_time"`
	EndTime        optional.Option[time.Time] `yaml:"end_time"`
}

type BacktestEngineV1Fees struct {
	CommissionFormula string `yaml:"commission_formula"`
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

func TestConfig(startTime time.Time, endTime time.Time, broker commission_fee.Broker) BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital: 10000,
		Broker:         broker,
		StartTime:      optional.Some(startTime),
		EndTime:        optional.Some(endTime),
	}
}

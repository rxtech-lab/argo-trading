package engine

import (
	"time"

	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
)

type BacktestEngineV1Config struct {
	InitialCapital float64               `yaml:"initial_capital"`
	Broker         commission_fee.Broker `yaml:"broker"`
	ResultsFolder  string                `yaml:"results_folder"`
	StartTime      time.Time             `yaml:"start_time"`
	EndTime        time.Time             `yaml:"end_time"`
}

type BacktestEngineV1Fees struct {
	CommissionFormula string `yaml:"commission_formula"`
}

func TestConfig(startTime time.Time, endTime time.Time, broker commission_fee.Broker) BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital: 10000,
		Broker:         broker,
		ResultsFolder:  "results",
		StartTime:      startTime,
		EndTime:        endTime,
	}
}

package engine

import (
	"time"
)

type BacktestEngineV1Config struct {
	InitialCapital    float64   `yaml:"initial_capital"`
	CommissionFormula string    `yaml:"commission_formula"`
	ResultsFolder     string    `yaml:"results_folder"`
	StartTime         time.Time `yaml:"start_time"`
	EndTime           time.Time `yaml:"end_time"`
}

type BacktestEngineV1Fees struct {
	CommissionFormula string `yaml:"commission_formula"`
}

func TestConfig(startTime time.Time, endTime time.Time, commissionFormula string) BacktestEngineV1Config {
	return BacktestEngineV1Config{
		InitialCapital:    10000,
		CommissionFormula: commissionFormula,
		ResultsFolder:     "results",
		StartTime:         startTime,
		EndTime:           endTime,
	}
}

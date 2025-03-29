package engine

import (
	"path/filepath"

	"github.com/sirily11/argo-trading-go/src/engine"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/strategy"
	"go.uber.org/zap"
	"gopkg.in/yaml.v2"
)

type BacktestEngineV1 struct {
	config              BacktestEngineV1Config
	strategies          []strategy.TradingStrategy
	strategyConfigPaths []string
	dataPaths           []string
	resultsFolder       string
	log                 *logger.Logger
}

func NewBacktestEngineV1() engine.Engine {
	return &BacktestEngineV1{}
}

// Initialize implements engine.Engine.
func (b *BacktestEngineV1) Initialize(config string) error {
	// parse the config
	err := yaml.Unmarshal([]byte(config), &b.config)
	if err != nil {
		return err
	}

	// initialize the logger
	var loggerError error
	b.log, loggerError = logger.NewLogger()
	if loggerError != nil {
		return loggerError
	}

	b.log.Info("Backtest engine initialized",
		zap.String("config", config),
	)
	return nil
}

// LoadStrategy implements engine.Engine.
func (b *BacktestEngineV1) LoadStrategy(strategy strategy.TradingStrategy) error {
	b.strategies = append(b.strategies, strategy)
	b.log.Info("Strategy loaded",
		zap.Int("total_strategies", len(b.strategies)),
	)
	return nil
}

// SetConfigPath implements engine.Engine.
func (b *BacktestEngineV1) SetConfigPath(path string) error {
	// use glob to get all the files that match the path
	files, err := filepath.Glob(path)
	if err != nil {
		b.log.Error("Failed to set config path",
			zap.String("path", path),
			zap.Error(err),
		)
		return err
	}

	b.strategyConfigPaths = files
	b.log.Info("Config paths set",
		zap.Strings("files", files),
	)
	return nil
}

// SetDataPath implements engine.Engine.
func (b *BacktestEngineV1) SetDataPath(path string) error {
	// use glob to get all the files that match the path
	files, err := filepath.Glob(path)
	if err != nil {
		b.log.Error("Failed to set data path",
			zap.String("path", path),
			zap.Error(err),
		)
		return err
	}
	b.dataPaths = files
	b.log.Info("Data paths set",
		zap.Strings("files", files),
	)
	return nil
}

// SetResultsFolder implements engine.Engine.
func (b *BacktestEngineV1) SetResultsFolder(folder string) error {
	b.resultsFolder = folder
	b.log.Info("Results folder set",
		zap.String("folder", folder),
	)
	return nil
}

// Run implements engine.Engine.
func (b *BacktestEngineV1) Run() error {
	panic("unimplemented")
}

package engine

import (
	"errors"
	"fmt"
	"path/filepath"

	"github.com/schollz/progressbar/v3"
	"github.com/sirily11/argo-trading-go/src/engine"
	"github.com/sirily11/argo-trading-go/src/engine/engine_v1/datasource"
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

	b.log.Debug("Backtest engine initialized",
		zap.String("config", config),
	)
	return nil
}

// LoadStrategy implements engine.Engine.
func (b *BacktestEngineV1) LoadStrategy(strategy strategy.TradingStrategy) error {
	b.strategies = append(b.strategies, strategy)
	b.log.Debug("Strategy loaded",
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
	b.log.Debug("Config paths set",
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

	// Convert all paths to absolute paths
	absolutePaths := make([]string, len(files))
	for i, file := range files {
		absPath, err := filepath.Abs(file)
		if err != nil {
			b.log.Error("Failed to get absolute path",
				zap.String("path", file),
				zap.Error(err),
			)
			return err
		}
		absolutePaths[i] = absPath
	}

	b.dataPaths = absolutePaths
	b.log.Debug("Data paths set",
		zap.Strings("files", absolutePaths),
	)
	return nil
}

// SetResultsFolder implements engine.Engine.
func (b *BacktestEngineV1) SetResultsFolder(folder string) error {
	b.resultsFolder = folder
	b.log.Debug("Results folder set",
		zap.String("folder", folder),
	)
	return nil
}

func (b *BacktestEngineV1) preRunCheck() error {
	if len(b.strategies) == 0 {
		b.log.Error("No strategies loaded")
		return errors.New("no strategies loaded")
	}

	if len(b.strategyConfigPaths) == 0 {
		b.log.Error("No strategy config paths loaded")
		return errors.New("no strategy config paths loaded")
	}

	if len(b.dataPaths) == 0 {
		b.log.Error("No data paths loaded")
		return errors.New("no data paths loaded")
	}

	if b.resultsFolder == "" {
		b.log.Error("No results folder set")
		return errors.New("no results folder set")
	}
	return nil
}

// Run implements engine.Engine.
func (b *BacktestEngineV1) Run() error {
	if err := b.preRunCheck(); err != nil {
		return err
	}

	for _, strategy := range b.strategies {
		for _, configPath := range b.strategyConfigPaths {
			for _, dataPath := range b.dataPaths {
				resultFilePath := filepath.Join(b.resultsFolder, fmt.Sprintf("%s_%s_%s.csv", strategy.Name(), filepath.Base(configPath), filepath.Base(dataPath)))
				b.log.Debug("Running strategy",
					zap.String("strategy", strategy.Name()),
					zap.String("config", configPath),
					zap.String("data", dataPath),
					zap.String("result", resultFilePath),
				)

				// initialize the data source with in-memory database
				datasource, err := datasource.NewDataSource(":memory:", b.log)
				if err != nil {
					b.log.Error("Failed to create data source",
						zap.String("data", dataPath),
						zap.Error(err),
					)
					return err
				}

				// initialize the data source with the given data path
				err = datasource.Initialize(dataPath)
				if err != nil {
					b.log.Error("Failed to initialize data source",
						zap.String("data", dataPath),
						zap.Error(err),
					)
				}

				// get the count of the data source
				count, err := datasource.Count()
				if err != nil {
					b.log.Error("Failed to get count of data source",
						zap.String("data", dataPath),
						zap.Error(err),
					)
					return err
				}

				// Create progress bar
				bar := progressbar.Default(int64(count))
				bar.Describe(fmt.Sprintf("Processing %s with %s", filepath.Base(dataPath), strategy.Name()))

				for _, err := range datasource.ReadAll() {
					if err != nil {
						b.log.Error("Failed to read data",
							zap.Error(err),
						)
						return err
					}
					// Update progress bar
					bar.Add(1)
				}
			}
		}
	}
	return nil
}

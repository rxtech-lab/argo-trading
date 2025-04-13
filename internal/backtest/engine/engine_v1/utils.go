package engine

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime"
	"github.com/vbauerster/mpb/v8"
)

// runKey represents a unique key for a backtest run
type runKey struct {
	strategy, configPath, dataPath string
}

// ProgressBarConfig holds the configuration for creating progress bars
type ProgressBarConfig struct {
	Progress    *mpb.Progress
	Strategies  []runtime.StrategyRuntime
	ConfigPaths []string
	DataPaths   []string
	Logger      *logger.Logger
	StartTime   optional.Option[time.Time]
	EndTime     optional.Option[time.Time]
}

func getResultFolder(configPath string, dataPath string, b *BacktestEngineV1, strategy runtime.StrategyRuntime) string {
	// Create base folders for strategy and config
	strategyFolder := filepath.Join(b.resultsFolder, strategy.Name())
	configFolder := filepath.Join(strategyFolder, strings.TrimSuffix(filepath.Base(configPath), filepath.Ext(configPath)))

	// Create data folder with time range if specified
	var dataFolder string
	if b.config.StartTime.IsSome() || b.config.EndTime.IsSome() {
		startTimeStr := "all"
		endTimeStr := "all"
		if b.config.StartTime.IsSome() {
			startTimeStr = b.config.StartTime.Unwrap().Format("20060102")
		}
		if b.config.EndTime.IsSome() {
			endTimeStr = b.config.EndTime.Unwrap().Format("20060102")
		}
		timeRange := fmt.Sprintf("%s_%s", startTimeStr, endTimeStr)
		dataFolder = filepath.Join(configFolder, timeRange)
	} else {
		dataFolder = configFolder
	}

	// Add data file name as the final folder
	dataFileName := strings.TrimSuffix(filepath.Base(dataPath), filepath.Ext(dataPath))
	return filepath.Join(dataFolder, dataFileName)
}

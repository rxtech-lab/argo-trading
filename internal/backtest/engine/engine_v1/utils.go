package engine

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/rxtech-lab/argo-trading/internal/runtime"
)

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

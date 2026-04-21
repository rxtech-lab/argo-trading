package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

const wasmPath = "./portfolio_comparison_plugin.wasm"

// PortfolioComparisonTestSuite verifies that FIFO and average-cost portfolio
// calculation strategies produce the same final balance and commission fees
// when run against the same strategy and dataset.
type PortfolioComparisonTestSuite struct {
	suite.Suite
}

func TestPortfolioComparisonTestSuite(t *testing.T) {
	suite.Run(t, new(PortfolioComparisonTestSuite))
}

// runBacktest runs one backtest with the given engine config and returns the
// stats for the single symbol produced.
//
// This replicates testhelper.RunWasmStrategyTest but does NOT call
// backtest.Initialize("") a second time — that call in the helper would wipe
// our portfolio_calculation / broker / initial_capital config.
func (s *PortfolioComparisonTestSuite) runBacktest(configYAML, dataPath string) types.TradeStats {
	t := s.T()
	t.Helper()

	backtest, err := v1.NewBacktestEngineV1()
	require.NoError(t, err)

	require.NoError(t, backtest.Initialize(configYAML))

	log, err := logger.NewLogger()
	require.NoError(t, err)

	ds, err := datasource.NewDataSource(":memory:", log)
	require.NoError(t, err)
	require.NoError(t, backtest.SetDataSource(ds))

	runtime, err := wasm.NewStrategyWasmRuntime(wasmPath)
	require.NoError(t, err)

	_, err = runtime.GetConfigSchema()
	require.NoError(t, err)

	require.NoError(t, backtest.SetDataPath(dataPath))
	require.NoError(t, backtest.LoadStrategy(runtime))

	tmpFolder := t.TempDir()
	require.NoError(t, backtest.SetResultsFolder(filepath.Join(tmpFolder, "results")))

	// strategy config is trivially empty but the engine requires a path
	cfgPath := filepath.Join(tmpFolder, "config.json")
	cfgBytes, err := json.Marshal(struct{}{})
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(cfgPath, cfgBytes, 0644))
	require.NoError(t, backtest.SetConfigPath(cfgPath))

	require.NoError(t, backtest.Run(context.Background(), engine.LifecycleCallbacks{}))

	stats := s.readStatsFromFolder(tmpFolder)
	require.GreaterOrEqualf(t, len(stats), 1, "expected at least one stats entry")
	return stats[0]
}

// readStatsFromFolder walks the folder for stats.yaml files and unmarshals
// each one, returning the first entry from each.
func (s *PortfolioComparisonTestSuite) readStatsFromFolder(folder string) []types.TradeStats {
	t := s.T()
	t.Helper()

	var statsPaths []string
	err := filepath.Walk(folder, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !info.IsDir() && filepath.Base(path) == "stats.yaml" {
			statsPaths = append(statsPaths, path)
		}
		return nil
	})
	require.NoError(t, err)
	require.NotEmpty(t, statsPaths, "expected at least one stats.yaml")

	var out []types.TradeStats
	for _, p := range statsPaths {
		content, err := os.ReadFile(p)
		require.NoError(t, err)

		var entries []types.TradeStats
		require.NoError(t, yaml.Unmarshal(content, &entries))
		if len(entries) > 0 {
			out = append(out, entries[0])
		}
	}
	return out
}

// generateMockData writes 100 deterministic market data points to a parquet
// file and returns the path. Both backtest runs share this file.
func (s *PortfolioComparisonTestSuite) generateMockData() string {
	t := s.T()
	t.Helper()

	tmpDir := t.TempDir()
	mockPath := filepath.Join(tmpDir, testhelper.GenerateMockFilename("portfolio_comparison"))

	cfg := testhelper.MockDataConfig{
		Symbol:             "TESTSTOCK",
		StartTime:          time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:           time.Minute,
		NumDataPoints:      100,
		Pattern:            testhelper.PatternVolatile,
		InitialPrice:       100.0,
		MaxDrawdownPercent: 15.0,
		VolatilityPercent:  3.0,
		Seed:               42,
	}
	require.NoError(t, testhelper.GenerateAndWriteToParquet(cfg, mockPath))
	return mockPath
}

// TestFIFOAndAverageCostParity runs the same strategy + data twice, once per
// portfolio calculation method, and verifies:
//
//  1. FinalBalance matches across methods
//  2. TotalFees matches across methods
//  3. For each run: FinalBalance - RealizedPnL - UnrealizedPnL = InitialBalance
//     (fees are already embedded in RealizedPnL/UnrealizedPnL by the engine,
//     so they do not appear separately in the invariant)
func (s *PortfolioComparisonTestSuite) TestFIFOAndAverageCostParity() {
	dataPath := s.generateMockData()

	const (
		fifoConfig = `
initial_capital: 10000
broker: interactive_broker
portfolio_calculation: fifo
`
		avgConfig = `
initial_capital: 10000
broker: interactive_broker
portfolio_calculation: average_cost
`
	)

	fifoStats := s.runBacktest(fifoConfig, dataPath)
	avgStats := s.runBacktest(avgConfig, dataPath)

	s.Require().Equal("fifo", fifoStats.PortfolioCalculation)
	s.Require().Equal("average_cost", avgStats.PortfolioCalculation)
	s.Require().Equal("TESTSTOCK", fifoStats.Symbol)
	s.Require().Equal("TESTSTOCK", avgStats.Symbol)

	// All 10 scheduled orders should have been placed and filled.
	s.Require().Equal(10, fifoStats.TradeResult.NumberOfTrades,
		"FIFO run should have 10 trades (6 buys + 4 sells)")
	s.Require().Equal(10, avgStats.TradeResult.NumberOfTrades,
		"average_cost run should have 10 trades (6 buys + 4 sells)")

	// Parity: cash flows and commission fees do not depend on the PnL
	// accounting method; both runs must agree.
	s.Require().InDelta(fifoStats.FinalBalance, avgStats.FinalBalance, 1e-6,
		"FinalBalance must match across portfolio calculation methods")
	s.Require().InDelta(fifoStats.TotalFees, avgStats.TotalFees, 1e-6,
		"TotalFees must match across portfolio calculation methods")

	// Fees should be non-zero since we configured interactive_broker.
	s.Require().Greater(fifoStats.TotalFees, 0.0,
		"interactive_broker must produce non-zero fees")

	// Per-run accounting invariant:
	//   final_balance - realized_pnl - unrealized_pnl = initial_balance
	// Fees are already embedded in RealizedPnL/UnrealizedPnL by the engine.
	checkInvariant := func(label string, stats types.TradeStats) {
		lhs := stats.FinalBalance -
			stats.TradePnl.RealizedPnL - stats.TradePnl.UnrealizedPnL
		s.Require().InDelta(stats.InitialBalance, lhs, 1e-6,
			"%s: final_balance(%.6f) - realized(%.6f) - unrealized(%.6f) = %.6f; expected initial_balance=%.6f",
			label, stats.FinalBalance,
			stats.TradePnl.RealizedPnL, stats.TradePnl.UnrealizedPnL,
			lhs, stats.InitialBalance)
	}
	checkInvariant("fifo", fifoStats)
	checkInvariant("average_cost", avgStats)

	s.T().Logf("FIFO         | final=%.4f fees=%.4f realized=%.4f unrealized=%.4f",
		fifoStats.FinalBalance, fifoStats.TotalFees,
		fifoStats.TradePnl.RealizedPnL, fifoStats.TradePnl.UnrealizedPnL)
	s.T().Logf("average_cost | final=%.4f fees=%.4f realized=%.4f unrealized=%.4f",
		avgStats.FinalBalance, avgStats.TotalFees,
		avgStats.TradePnl.RealizedPnL, avgStats.TradePnl.UnrealizedPnL)
}

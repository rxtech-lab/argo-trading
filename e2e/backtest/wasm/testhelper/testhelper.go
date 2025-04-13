package testhelper

import (
	"os"
	"path/filepath"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

// E2ETestSuite is a base test suite for E2E tests
type E2ETestSuite struct {
	suite.Suite
	Backtest engine.Engine
}

// SetupTest initializes the backtest engine
func (s *E2ETestSuite) SetupTest(engineConfig string) {
	// initialize backtest engine
	backtest := v1.NewBacktestEngineV1()
	err := backtest.Initialize(engineConfig)
	s.Require().NoError(err)

	// initialize strategy api
	l, err := logger.NewLogger()
	s.Require().NoError(err)

	dataSource, err := datasource.NewDataSource(":memory:", l)
	s.Require().NoError(err)

	err = backtest.SetDataSource(dataSource)
	s.Require().NoError(err)

	s.Backtest = backtest
}

// RunWasmStrategyTest runs a test for a WASM strategy
func RunWasmStrategyTest(s *E2ETestSuite, strategyName string, wasmPath string) {
	type config struct {
		FastPeriod int    `yaml:"fastPeriod"`
		SlowPeriod int    `yaml:"slowPeriod"`
		Symbol     string `yaml:"symbol"`
	}

	cfg := config{
		FastPeriod: 10,
		SlowPeriod: 20,
		Symbol:     "BTCUSDT",
	}

	cfgBytes, err := yaml.Marshal(cfg)
	require.NoError(s.T(), err)

	// write config to file
	tmpFolder := s.T().TempDir()
	configPath := filepath.Join(tmpFolder, "config", "config.yaml")
	resultPath := filepath.Join(tmpFolder, "results")

	// create config folder
	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(s.T(), err)

	// write config to file
	err = os.WriteFile(configPath, cfgBytes, 0644)
	require.NoError(s.T(), err)

	err = s.Backtest.Initialize("")
	require.NoError(s.T(), err)

	runtime, err := wasm.NewStrategyWasmRuntime(wasmPath)
	require.NoError(s.T(), err)

	dataPath := "../../../../internal/indicator/test_data/test_data.parquet"
	err = s.Backtest.SetDataPath(dataPath)
	require.NoError(s.T(), err)

	err = s.Backtest.LoadStrategy(runtime)
	require.NoError(s.T(), err)

	err = s.Backtest.SetResultsFolder(resultPath)
	require.NoError(s.T(), err)

	// set config path
	err = s.Backtest.SetConfigPath(configPath)
	require.NoError(s.T(), err)

	err = s.Backtest.Run(optional.None[engine.OnProcessDataCallback]())
	require.NoError(s.T(), err)
}

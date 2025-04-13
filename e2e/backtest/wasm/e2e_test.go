package wasm

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v2"
)

type E2ETestSuite struct {
	suite.Suite
	backtest engine.Engine
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

func (s *E2ETestSuite) SetupTest() {
	// initialize backtest engine
	backtest := v1.NewBacktestEngineV1()

	// initialize strategy api
	l, err := logger.NewLogger()
	s.Require().NoError(err)

	dataSource, err := datasource.NewDataSource(":memory:", l)
	s.Require().NoError(err)

	err = backtest.SetDataSource(dataSource)
	s.Require().NoError(err)

	s.backtest = backtest
}

func (s *E2ETestSuite) TestSimpleMAStrategy() {
	s.Run("TestSimpleMAStrategy", func() {

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
		s.Require().NoError(err)

		// write config to file
		tmpFolder := s.T().TempDir()
		configPath := filepath.Join(tmpFolder, "config", "config.yaml")
		resultPath := filepath.Join(tmpFolder, "results")

		// create config folder
		err = os.MkdirAll(filepath.Dir(configPath), 0755)
		s.Require().NoError(err)

		// write config to file
		err = os.WriteFile(configPath, cfgBytes, 0644)
		s.Require().NoError(err)

		err = s.backtest.Initialize("")
		s.Require().NoError(err)

		runtime, err := wasm.NewStrategyWasmRuntime("./sma_plugin.wasm")
		s.Require().NoError(err)

		dataPath := "../../../internal/indicator/test_data/test_data.parquet"
		err = s.backtest.SetDataPath(dataPath)
		s.Require().NoError(err)

		err = s.backtest.LoadStrategy(runtime)
		s.Require().NoError(err)

		err = s.backtest.SetResultsFolder(resultPath)
		s.Require().NoError(err)

		// set config path
		err = s.backtest.SetConfigPath(configPath)
		s.Require().NoError(err)

		err = s.backtest.Run()
		s.Require().NoError(err)
	})
}

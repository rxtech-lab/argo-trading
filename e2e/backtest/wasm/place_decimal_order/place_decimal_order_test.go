package main

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine"
	v1 "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/runtime/wasm"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// engineConfigYAML is the backtest engine configuration used by every test in
// this suite. decimal_precision: 3 is the whole point of the suite — it lets
// the engine accept the 0.01 fractional quantities the strategy submits.
const engineConfigYAML = `
initial_capital: 10000
decimal_precision: 3
`

// PlaceDecimalOrderTestSuite exercises fractional-quantity market orders with
// decimal_precision: 3.
type PlaceDecimalOrderTestSuite struct {
	testhelper.E2ETestSuite
}

func TestPlaceDecimalOrderTestSuite(t *testing.T) {
	suite.Run(t, new(PlaceDecimalOrderTestSuite))
}

// SetupTest initializes the base test suite with our engine config.
func (s *PlaceDecimalOrderTestSuite) SetupTest() {
	s.E2ETestSuite.SetupTest(engineConfigYAML)
}

// runTest wires the WASM strategy into a fresh engine (so decimal_precision
// sticks) and runs the backtest end-to-end. Returns the temp folder containing
// the results parquet files.
func (s *PlaceDecimalOrderTestSuite) runTest() string {
	type strategyConfig struct {
		Symbol string `json:"symbol"`
	}

	cfgBytes, err := json.Marshal(strategyConfig{Symbol: "AAPL"})
	require.NoError(s.T(), err)

	tmpFolder := s.T().TempDir()
	configPath := filepath.Join(tmpFolder, "config", "config.json")
	resultPath := filepath.Join(tmpFolder, "results")

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(s.T(), err)

	err = os.WriteFile(configPath, cfgBytes, 0644)
	require.NoError(s.T(), err)

	backtest, err := v1.NewBacktestEngineV1()
	require.NoError(s.T(), err)

	err = backtest.Initialize(engineConfigYAML)
	require.NoError(s.T(), err)

	l, err := logger.NewLogger()
	require.NoError(s.T(), err)

	dataSource, err := datasource.NewDataSource(":memory:", l)
	require.NoError(s.T(), err)

	err = backtest.SetDataSource(dataSource)
	require.NoError(s.T(), err)

	s.Backtest = backtest

	runtime, err := wasm.NewStrategyWasmRuntime("./place_decimal_order_plugin.wasm")
	require.NoError(s.T(), err)

	_, err = runtime.GetConfigSchema()
	require.NoError(s.T(), err)

	err = s.Backtest.SetDataPath("../../../../internal/indicator/test_data/test_data.parquet")
	require.NoError(s.T(), err)

	err = s.Backtest.LoadStrategy(runtime)
	require.NoError(s.T(), err)

	err = s.Backtest.SetResultsFolder(resultPath)
	require.NoError(s.T(), err)

	err = s.Backtest.SetConfigPath(configPath)
	require.NoError(s.T(), err)

	err = s.Backtest.Run(context.Background(), engine.LifecycleCallbacks{})
	require.NoError(s.T(), err)

	return tmpFolder
}

// TestPlaceFractionalQuantityOrders verifies that a strategy submitting 0.01
// quantity market buys produces multiple filled trades when the engine is
// configured with decimal_precision: 3.
func (s *PlaceDecimalOrderTestSuite) TestPlaceFractionalQuantityOrders() {
	tmpFolder := s.runTest()

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	s.Require().Greater(len(orders), 1, "expected more than 1 order with fractional quantity")

	for _, order := range orders {
		s.Require().Equal(types.OrderStatusFilled, order.Status,
			"order %s should be FILLED, got %s (reason: %s)", order.OrderID, order.Status, order.Reason.Message)
		s.Require().InDelta(0.01, order.Quantity, 1e-9, "order quantity should be preserved at 0.01")
	}

	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	s.Require().Greater(len(trades), 1, "expected more than 1 trade with fractional quantity")

	for _, trade := range trades {
		s.Require().InDelta(0.01, trade.ExecutedQty, 1e-9, "trade executed qty should be 0.01")
		s.Require().Greater(trade.ExecutedPrice, 0.0, "trade should have a positive executed price")
	}
}

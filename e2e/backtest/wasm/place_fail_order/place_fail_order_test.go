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
	"github.com/rxtech-lab/argo-trading/internal/version"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// PlaceFailOrderTestSuite extends the base test suite
type PlaceFailOrderTestSuite struct {
	testhelper.E2ETestSuite
}

func TestPlaceFailOrderTestSuite(t *testing.T) {
	suite.Run(t, new(PlaceFailOrderTestSuite))
}

// SetupTest initializes the test with config
func (s *PlaceFailOrderTestSuite) SetupTest() {
	s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
}

// runTestWithConfig runs the backtest with a specific test case configuration
func (s *PlaceFailOrderTestSuite) runTestWithConfig(testCase string) string {
	// Set a compatible version for e2e tests (both engine and WASM strategy use "main" in dev)
	originalVersion := version.Version
	version.Version = "1.0.0"
	defer func() { version.Version = originalVersion }()

	type config struct {
		Symbol   string `json:"symbol"`
		TestCase string `json:"testCase"`
	}

	cfg := config{
		Symbol:   "AAPL",
		TestCase: testCase,
	}

	cfgBytes, err := json.Marshal(cfg)
	require.NoError(s.T(), err)

	tmpFolder := s.T().TempDir()
	configPath := filepath.Join(tmpFolder, "config", "config.json")
	resultPath := filepath.Join(tmpFolder, "results")

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	require.NoError(s.T(), err)

	err = os.WriteFile(configPath, cfgBytes, 0644)
	require.NoError(s.T(), err)

	// Re-initialize backtest engine for each test
	backtest, err := v1.NewBacktestEngineV1()
	require.NoError(s.T(), err)

	err = backtest.Initialize(`
initial_capital: 10000
`)
	require.NoError(s.T(), err)

	l, err := logger.NewLogger()
	require.NoError(s.T(), err)

	dataSource, err := datasource.NewDataSource(":memory:", l)
	require.NoError(s.T(), err)

	err = backtest.SetDataSource(dataSource)
	require.NoError(s.T(), err)

	s.Backtest = backtest

	runtime, err := wasm.NewStrategyWasmRuntime("./place_fail_order_plugin.wasm")
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

// countOrdersByStatus counts orders with a specific status
func countOrdersByStatus(orders []types.Order, status types.OrderStatus) int {
	count := 0
	for _, order := range orders {
		if order.Status == status {
			count++
		}
	}
	return count
}

// countOrdersByReason counts orders with a specific failure reason
func countOrdersByReason(orders []types.Order, reason string) int {
	count := 0
	for _, order := range orders {
		if order.Reason.Reason == reason {
			count++
		}
	}
	return count
}

// TestExceedBuyingPower tests that placing an order that exceeds buying power fails gracefully
func (s *PlaceFailOrderTestSuite) TestExceedBuyingPower() {
	tmpFolder := s.runTestWithConfig("exceed_buying_power")

	// Read all orders
	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Should have exactly 1 order
	s.Require().Equal(1, len(orders), "Expected 1 order")

	// The order should be failed
	s.Require().Equal(1, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 1 failed order")

	// The failure reason should be insufficient_buying_power
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientBuyPower),
		"Expected failure reason to be insufficient_buying_power")

	// Read trades - should have 0 trades (failed orders don't create trades)
	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(0, len(trades), "Expected 0 trades for failed order")
}

// TestExceedSellingPower tests that placing a sell order without holdings fails gracefully
func (s *PlaceFailOrderTestSuite) TestExceedSellingPower() {
	tmpFolder := s.runTestWithConfig("exceed_selling_power")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	s.Require().Equal(1, len(orders), "Expected 1 order")
	s.Require().Equal(1, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 1 failed order")
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientSellPower),
		"Expected failure reason to be insufficient_selling_power")

	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(0, len(trades), "Expected 0 trades for failed order")
}

// TestInvalidThenSuccess tests that after placing an invalid order, a valid order can still succeed
func (s *PlaceFailOrderTestSuite) TestInvalidThenSuccess() {
	tmpFolder := s.runTestWithConfig("invalid_then_success")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Should have 2 orders total
	s.Require().Equal(2, len(orders), "Expected 2 orders")

	// 1 failed (invalid quantity) and 1 successful
	s.Require().Equal(1, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 1 failed order")
	s.Require().Equal(1, countOrdersByStatus(orders, types.OrderStatusFilled), "Expected 1 filled order")

	// The failed order should have invalid_quantity reason
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInvalidQuantity),
		"Expected failure reason to be invalid_quantity")

	// Should have 1 trade (the successful order)
	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(1, len(trades), "Expected 1 trade for successful order")
}

// TestSuccessOrder tests that a valid order succeeds
func (s *PlaceFailOrderTestSuite) TestSuccessOrder() {
	tmpFolder := s.runTestWithConfig("success_order")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	s.Require().Equal(1, len(orders), "Expected 1 order")
	s.Require().Equal(1, countOrdersByStatus(orders, types.OrderStatusFilled), "Expected 1 filled order")
	s.Require().Equal(0, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 0 failed orders")

	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(1, len(trades), "Expected 1 trade")
	s.Require().Greater(trades[0].ExecutedPrice, 0.0, "Trade should have executed price")
}

// TestMixedOrders tests that multiple orders with mixed success/failure are handled correctly
func (s *PlaceFailOrderTestSuite) TestMixedOrders() {
	tmpFolder := s.runTestWithConfig("mixed_orders")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Should have 4 orders total
	s.Require().Equal(4, len(orders), "Expected 4 orders")

	// 2 failed and 2 successful
	s.Require().Equal(2, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 2 failed orders")
	s.Require().Equal(2, countOrdersByStatus(orders, types.OrderStatusFilled), "Expected 2 filled orders")

	// Check failure reasons
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientBuyPower),
		"Expected 1 order with insufficient_buying_power")
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientSellPower),
		"Expected 1 order with insufficient_selling_power")

	// Should have 2 trades
	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(2, len(trades), "Expected 2 trades for successful orders")
}

// TestMultipleFailedOrders tests that multiple failed orders are all recorded
func (s *PlaceFailOrderTestSuite) TestMultipleFailedOrders() {
	tmpFolder := s.runTestWithConfig("multiple_failed")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Should have 3 orders total, all failed
	s.Require().Equal(3, len(orders), "Expected 3 orders")
	s.Require().Equal(3, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 3 failed orders")
	s.Require().Equal(0, countOrdersByStatus(orders, types.OrderStatusFilled), "Expected 0 filled orders")

	// Check all three failure reasons are present
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientBuyPower),
		"Expected 1 order with insufficient_buying_power")
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInvalidQuantity),
		"Expected 1 order with invalid_quantity")
	s.Require().Equal(1, countOrdersByReason(orders, types.OrderReasonInsufficientSellPower),
		"Expected 1 order with insufficient_selling_power")

	// Should have 0 trades
	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().Equal(0, len(trades), "Expected 0 trades for all failed orders")
}

// TestMaxBuyOrder tests that using GetAccountInfo to calculate max buy quantity works
func (s *PlaceFailOrderTestSuite) TestMaxBuyOrder() {
	tmpFolder := s.runTestWithConfig("max_buy_order")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Should have at least 1 successful order (strategy places order on first tick when it can afford shares)
	s.Require().GreaterOrEqual(len(orders), 1, "Expected at least 1 order")
	s.Require().GreaterOrEqual(countOrdersByStatus(orders, types.OrderStatusFilled), 1, "Expected at least 1 filled order")
	s.Require().Equal(0, countOrdersByStatus(orders, types.OrderStatusFailed), "Expected 0 failed orders")

	// The order should use significant portion of buying power (price is ~247, so with 10000 we can buy ~40 shares)
	s.Require().Greater(orders[0].Quantity, 1.0, "Expected max buy order to have quantity > 1")

	// Should have at least 1 trade
	trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)
	s.Require().GreaterOrEqual(len(trades), 1, "Expected at least 1 trade")

	// Verify the trade details
	s.Require().Greater(trades[0].ExecutedPrice, 0.0, "Trade should have executed price")
	s.Require().Greater(trades[0].ExecutedQty, 1.0, "Trade should have quantity > 1")
}

// TestAllOrdersAreRecorded verifies that all orders (including failed ones) are recorded in the orders output
func (s *PlaceFailOrderTestSuite) TestAllOrdersAreRecorded() {
	// Run mixed orders test and verify the complete order history
	tmpFolder := s.runTestWithConfig("mixed_orders")

	orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
	s.Require().NoError(err)

	// Verify all 4 orders are present
	s.Require().Equal(4, len(orders), "All orders should be recorded including failed ones")

	// Verify order details
	for _, order := range orders {
		s.Require().NotEmpty(order.OrderID, "Order should have an ID")
		s.Require().NotEmpty(order.Symbol, "Order should have a symbol")
		s.Require().NotEmpty(order.StrategyName, "Order should have a strategy name")

		// Either filled or failed status
		s.Require().True(
			order.Status == types.OrderStatusFilled || order.Status == types.OrderStatusFailed,
			"Order status should be FILLED or Failed, got: %s", order.Status)

		// Failed orders should have a failure reason
		if order.Status == types.OrderStatusFailed {
			s.Require().NotEmpty(order.Reason.Reason, "Failed order should have a reason code")
			s.Require().NotEmpty(order.Reason.Message, "Failed order should have a reason message")
		}
	}
}

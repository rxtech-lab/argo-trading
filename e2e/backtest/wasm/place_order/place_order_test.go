package main

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// PlaceOrderTestSuite extends the base test suite
type PlaceOrderTestSuite struct {
	testhelper.E2ETestSuite
}

func TestPlaceOrderTestSuite(t *testing.T) {
	suite.Run(t, new(PlaceOrderTestSuite))
}

// SetupTest initializes the test with config
func (s *PlaceOrderTestSuite) SetupTest() {
	s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
}

func (s *PlaceOrderTestSuite) TestPlaceOrderStrategy() {
	s.Run("TestPlaceOrderStrategy", func() {
		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "PlaceOrderStrategy", "./place_order_plugin.wasm", "")
		// read stats
		stats, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Equal(stats[0].TradeResult.NumberOfTrades, 1)
		s.Require().Equal(stats[0].Symbol, "AAPL")

		// read trades
		trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Equal(len(trades), 1)
		s.Require().Greater(trades[0].ExecutedPrice, 0.0)
		s.Require().Equal(trades[0].ExecutedQty, 1.0)

		// check marker
		marker, err := testhelper.ReadMarker(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Greater(len(marker), 0)
		s.Require().Equal(marker[0].Signal.Type, types.SignalTypeBuyLong)
		s.Require().Equal(marker[0].Reason, "PlaceOrderStrategy")
	})
}

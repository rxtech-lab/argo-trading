package main

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite extends the base test suite
type E2ETestSuite struct {
	testhelper.E2ETestSuite
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// SetupTest initializes the test with config
func (s *E2ETestSuite) SetupTest() {
	s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
}

func (s *E2ETestSuite) TestPlaceOrderStrategy() {
	s.Run("TestPlaceOrderStrategy", func() {
		testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "PlaceOrderStrategy", "./place_order_plugin.wasm")
	})
}

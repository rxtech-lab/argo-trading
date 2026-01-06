package main

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

// PlaceOrderTestSuite extends the base test suite
type ComplexStrategyTestSuite struct {
	testhelper.E2ETestSuite
}

func TestComplexStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(ComplexStrategyTestSuite))
}

// SetupTest initializes the test with config
func (s *ComplexStrategyTestSuite) SetupTest() {

}

func (s *ComplexStrategyTestSuite) TestComplexStrategy() {
	s.Run("TestComplexStrategy", func() {
		s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "ComplexStrategy", "./complex_strategy_plugin.wasm", "")
		// read stats
		_, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
	})
}

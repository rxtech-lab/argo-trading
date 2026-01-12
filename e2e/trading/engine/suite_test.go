package engine_test

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	engine_v1 "github.com/rxtech-lab/argo-trading/internal/trading/engine/engine_v1"
	"github.com/stretchr/testify/suite"
)

// LiveTradingE2ETestSuite is the base test suite for live trading E2E tests.
type LiveTradingE2ETestSuite struct {
	suite.Suite
	engine engine.LiveTradingEngine
}

func TestLiveTradingE2E(t *testing.T) {
	suite.Run(t, new(LiveTradingE2ETestSuite))
}

// SetupTest initializes the engine for each test.
func (s *LiveTradingE2ETestSuite) SetupTest() {
	var err error
	s.engine, err = engine_v1.NewLiveTradingEngineV1()
	s.Require().NoError(err)
}

//go:build !wasip1

// TestStuckRepro pins the fix for the "stuck at first tick" bug observed
// in MultiConfirmStrategy. A one-shot init log gated on `tickCount == 0`
// fired on every warmup bar because `tickCount++` happened AFTER an early
// return that triggers while indicators (RSI) are still warming up. From
// a log-watcher's POV the strategy appeared frozen on the first tick.
//
// The minimal stuck_repro_strategy.go uses a dedicated `started` flag —
// this test asserts the init log fires exactly once.
package main

import (
	"strings"
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

type StuckReproTestSuite struct {
	testhelper.E2ETestSuite
}

func TestStuckReproTestSuite(t *testing.T) {
	suite.Run(t, new(StuckReproTestSuite))
}

func (s *StuckReproTestSuite) SetupTest() {}

func (s *StuckReproTestSuite) TestInitLogFiresOnce() {
	s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
	tmp := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "StuckReproStrategy", "./stuck_repro_plugin.wasm", "")

	logs, err := testhelper.ReadLogs(&s.E2ETestSuite, tmp)
	s.Require().NoError(err)

	started := 0
	hold := 0
	for _, l := range logs {
		switch {
		case strings.Contains(l.Message, "StuckReproStrategy started"):
			started++
		case strings.HasPrefix(l.Message, "HOLD tick="):
			hold++
		}
	}

	s.T().Logf("started=%d hold=%d total=%d", started, hold, len(logs))
	s.Require().Equal(1, started, "init log should fire exactly once, not per warmup bar")
	s.Require().Greater(hold, 0, "expected per-tick HOLD logs once RSI warmed up")
}

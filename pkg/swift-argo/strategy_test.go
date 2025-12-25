package swiftargo

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type StrategyTestSuite struct {
	suite.Suite
}

func TestStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(StrategyTestSuite))
}

func (suite *StrategyTestSuite) TestGetStrategyMetadata() {
	api := NewStrategyApi()
	metadata, err := api.GetStrategyMetadata("../../e2e/backtest/wasm/sma/sma_plugin.wasm")
	suite.NoError(err)
	suite.NotEmpty(metadata)
}

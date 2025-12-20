package engine

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type EngineTestSuite struct {
	suite.Suite
}

func TestEngineSuite(t *testing.T) {
	suite.Run(t, new(EngineTestSuite))
}

func (suite *EngineTestSuite) TestStrategyTypeConstants() {
	suite.Equal(StrategyType("wasm"), StrategyTypeWASM)
}

func (suite *EngineTestSuite) TestStrategyTypeAsString() {
	suite.Equal("wasm", string(StrategyTypeWASM))
}

func (suite *EngineTestSuite) TestOnProcessDataCallbackType() {
	// Test that the callback type works correctly
	var callback OnProcessDataCallback = func(current int, total int) error {
		return nil
	}

	suite.NotNil(callback)
	err := callback(1, 10)
	suite.NoError(err)
}

func (suite *EngineTestSuite) TestOnProcessDataCallbackWithProgress() {
	var progress []int
	callback := OnProcessDataCallback(func(current int, total int) error {
		progress = append(progress, current)
		return nil
	})

	for i := 1; i <= 5; i++ {
		err := callback(i, 5)
		suite.NoError(err)
	}

	suite.Equal([]int{1, 2, 3, 4, 5}, progress)
}

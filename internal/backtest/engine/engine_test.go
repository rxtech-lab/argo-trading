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
	var callback OnProcessDataCallback = func(info ProgressInfo) error {
		return nil
	}

	suite.NotNil(callback)
	err := callback(ProgressInfo{Current: 1, Total: 10})
	suite.NoError(err)
}

func (suite *EngineTestSuite) TestOnProcessDataCallbackWithProgress() {
	var progress []int
	callback := OnProcessDataCallback(func(info ProgressInfo) error {
		progress = append(progress, info.Current)
		return nil
	})

	for i := 1; i <= 5; i++ {
		err := callback(ProgressInfo{Current: i, Total: 5})
		suite.NoError(err)
	}

	suite.Equal([]int{1, 2, 3, 4, 5}, progress)
}

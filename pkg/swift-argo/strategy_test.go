package swiftargo

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type StrategyTestSuite struct {
	suite.Suite
}

func TestStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(StrategyTestSuite))
}

func TestNewStrategyApi(t *testing.T) {
	api := NewStrategyApi()
	assert.NotNil(t, api)
}

func TestStrategyApi_GetStrategyMetadata_InvalidPath(t *testing.T) {
	api := NewStrategyApi()
	metadata, err := api.GetStrategyMetadata("/nonexistent/path/strategy.wasm")
	assert.Error(t, err)
	assert.Nil(t, metadata)
}

package swiftargo_test

import (
	"testing"

	swiftargo "github.com/rxtech-lab/argo-trading/pkg/swift-argo"
	"github.com/stretchr/testify/assert"
)

func TestGetBacktestEngineConfigSchema(t *testing.T) {
	schema := swiftargo.GetBacktestEngineConfigSchema()
	assert.NotEmpty(t, schema)
}

package swiftargo_test

import (
	"encoding/json"
	"testing"

	swiftargo "github.com/rxtech-lab/argo-trading/pkg/swift-argo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetBacktestEngineConfigSchema(t *testing.T) {
	schema := swiftargo.GetBacktestEngineConfigSchema()
	assert.NotEmpty(t, schema)
}

func TestGetBacktestEngineConfigSchema_TimeFieldsAreStrings(t *testing.T) {
	schema := swiftargo.GetBacktestEngineConfigSchema()
	require.NotEmpty(t, schema)

	// Parse the schema JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	require.NoError(t, err)

	// Get properties
	properties, ok := schemaMap["properties"].(map[string]interface{})
	require.True(t, ok, "schema should have properties")

	// Check start_time field
	startTime, ok := properties["start_time"].(map[string]interface{})
	require.True(t, ok, "schema should have start_time property")
	assert.Equal(t, "string", startTime["type"], "start_time should be type string")
	assert.Equal(t, "date-time", startTime["format"], "start_time should have date-time format")

	// Check end_time field
	endTime, ok := properties["end_time"].(map[string]interface{})
	require.True(t, ok, "schema should have end_time property")
	assert.Equal(t, "string", endTime["type"], "end_time should be type string")
	assert.Equal(t, "date-time", endTime["format"], "end_time should have date-time format")
}

func TestGetBacktestEngineVersion(t *testing.T) {
	version := swiftargo.GetBacktestEngineVersion()
	assert.NotEmpty(t, version)
}

func TestGetBacktestEngineConfigSchema_EnumFields(t *testing.T) {
	schema := swiftargo.GetBacktestEngineConfigSchema()
	require.NotEmpty(t, schema)

	// Parse the schema JSON
	var schemaMap map[string]interface{}
	err := json.Unmarshal([]byte(schema), &schemaMap)
	require.NoError(t, err)

	// Get properties
	properties, ok := schemaMap["properties"].(map[string]interface{})
	require.True(t, ok, "schema should have properties")

	// Check broker field has enum
	broker, ok := properties["broker"].(map[string]interface{})
	require.True(t, ok, "schema should have broker property")
	brokerEnum, ok := broker["enum"].([]interface{})
	require.True(t, ok, "broker should have enum")
	assert.Len(t, brokerEnum, 2)
	assert.Contains(t, []string{brokerEnum[0].(string), brokerEnum[1].(string)}, "interactive_broker")
	assert.Contains(t, []string{brokerEnum[0].(string), brokerEnum[1].(string)}, "zero_commission")

	// Check portfolio_calculation field has enum
	portfolioCalc, ok := properties["portfolio_calculation"].(map[string]interface{})
	require.True(t, ok, "schema should have portfolio_calculation property")
	portfolioEnum, ok := portfolioCalc["enum"].([]interface{})
	require.True(t, ok, "portfolio_calculation should have enum")
	assert.Len(t, portfolioEnum, 2)
	assert.Contains(t, []string{portfolioEnum[0].(string), portfolioEnum[1].(string)}, "fifo")
	assert.Contains(t, []string{portfolioEnum[0].(string), portfolioEnum[1].(string)}, "average_cost")
}

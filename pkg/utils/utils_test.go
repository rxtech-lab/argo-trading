package utils

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
}

func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

// TestConfig is a sample config struct for testing
type TestConfig struct {
	Name        string `json:"name" jsonschema:"description=The name of the config"`
	Value       int    `json:"value" jsonschema:"description=A numeric value"`
	Enabled     bool   `json:"enabled"`
	Tags        []string `json:"tags,omitempty"`
}

// NestedConfig is a sample nested config struct for testing
type NestedConfig struct {
	ID     string     `json:"id"`
	Config TestConfig `json:"config"`
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigSimple() {
	config := TestConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(schema), &result)
	suite.NoError(err)

	// Check basic schema properties exist
	suite.Contains(result, "$schema")
	// Schema uses $ref to reference definitions in $defs
	suite.Contains(result, "$ref")
	suite.Contains(result, "$defs")
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigNested() {
	config := NestedConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(schema), &result)
	suite.NoError(err)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigPointer() {
	config := &TestConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigWithValues() {
	config := TestConfig{
		Name:    "test",
		Value:   42,
		Enabled: true,
		Tags:    []string{"tag1", "tag2"},
	}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)

	// Verify it's valid JSON
	var result map[string]interface{}
	err = json.Unmarshal([]byte(schema), &result)
	suite.NoError(err)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigEmptyStruct() {
	type EmptyConfig struct{}
	config := EmptyConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigPrimitiveTypes() {
	// Test with various primitive types
	schema, err := GetSchemaFromConfig("string")
	suite.NoError(err)
	suite.NotEmpty(schema)

	schema, err = GetSchemaFromConfig(42)
	suite.NoError(err)
	suite.NotEmpty(schema)

	schema, err = GetSchemaFromConfig(true)
	suite.NoError(err)
	suite.NotEmpty(schema)

	schema, err = GetSchemaFromConfig(3.14)
	suite.NoError(err)
	suite.NotEmpty(schema)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigSlice() {
	config := []TestConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)
}

func (suite *UtilsTestSuite) TestGetSchemaFromConfigMap() {
	config := map[string]TestConfig{}
	schema, err := GetSchemaFromConfig(config)

	suite.NoError(err)
	suite.NotEmpty(schema)
}

package strategy

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type JsonSchemaTestSuite struct {
	suite.Suite
}

func TestJsonSchemaTestSuite(t *testing.T) {
	suite.Run(t, new(JsonSchemaTestSuite))
}

func (suite *JsonSchemaTestSuite) TestToJSONSchema() {
	type TestConfig struct {
		FastPeriod int    `yaml:"fastPeriod" jsonschema:"title=Fast Period,description=The period for the fast moving average,minimum=1,default=5"`
		SlowPeriod int    `yaml:"slowPeriod" jsonschema:"title=Slow Period,description=The period for the slow moving average,minimum=1,default=20"`
		Symbol     string `yaml:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
	}

	schema, err := ToJSONSchema(TestConfig{})
	suite.NoError(err)
	suite.NotEmpty(schema)
}

func (suite *JsonSchemaTestSuite) TestToJSONSchema_EmptyStruct() {
	type EmptyConfig struct{}

	schema, err := ToJSONSchema(EmptyConfig{})
	suite.NoError(err)
	suite.NotEmpty(schema)
	suite.Contains(schema, "empty-config")
}

func (suite *JsonSchemaTestSuite) TestToJSONSchema_NestedStruct() {
	type InnerConfig struct {
		Value float64 `json:"value"`
	}
	type OuterConfig struct {
		Name  string      `json:"name"`
		Inner InnerConfig `json:"inner"`
	}

	schema, err := ToJSONSchema(OuterConfig{})
	suite.NoError(err)
	suite.NotEmpty(schema)
	suite.Contains(schema, "name")
	suite.Contains(schema, "inner")
}

func (suite *JsonSchemaTestSuite) TestToJSONSchema_WithSlice() {
	type ConfigWithSlice struct {
		Items []string `json:"items"`
	}

	schema, err := ToJSONSchema(ConfigWithSlice{})
	suite.NoError(err)
	suite.NotEmpty(schema)
	suite.Contains(schema, "items")
}

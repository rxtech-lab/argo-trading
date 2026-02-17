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

func (suite *JsonSchemaTestSuite) TestGetKeychainFields() {
	type Config struct {
		ApiKey    string `json:"apiKey" keychain:"true"`
		SecretKey string `json:"secretKey" keychain:"true"`
		BaseURL   string `json:"baseUrl,omitempty"`
	}

	fields := GetKeychainFields(Config{})
	suite.Equal([]string{"apiKey", "secretKey"}, fields)
}

func (suite *JsonSchemaTestSuite) TestGetKeychainFields_NoKeychainFields() {
	type Config struct {
		Name string `json:"name"`
	}

	fields := GetKeychainFields(Config{})
	suite.Nil(fields)
}

func (suite *JsonSchemaTestSuite) TestGetKeychainFields_NoJsonTag() {
	type Config struct {
		ApiKey string `keychain:"true"`
	}

	fields := GetKeychainFields(Config{})
	suite.Equal([]string{"ApiKey"}, fields)
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

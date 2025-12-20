package marketdata

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TypesTestSuite struct {
	suite.Suite
}

func TestTypesSuite(t *testing.T) {
	suite.Run(t, new(TypesTestSuite))
}

func (suite *TypesTestSuite) TestProviderTypeConstants() {
	suite.Equal(ProviderType("polygon"), ProviderPolygon)
	suite.Equal(ProviderType("binance"), ProviderBinance)
}

func (suite *TypesTestSuite) TestProviderTypeAsString() {
	suite.Equal("polygon", string(ProviderPolygon))
	suite.Equal("binance", string(ProviderBinance))
}

func (suite *TypesTestSuite) TestWriterTypeConstants() {
	suite.Equal(WriterType("duckdb"), WriterDuckDB)
}

func (suite *TypesTestSuite) TestWriterTypeAsString() {
	suite.Equal("duckdb", string(WriterDuckDB))
}

func (suite *TypesTestSuite) TestProviderTypeEquality() {
	provider1 := ProviderPolygon
	provider2 := ProviderType("polygon")

	suite.Equal(provider1, provider2)
}

func (suite *TypesTestSuite) TestProviderTypeInequality() {
	suite.NotEqual(ProviderPolygon, ProviderBinance)
}

func (suite *TypesTestSuite) TestWriterTypeEquality() {
	writer1 := WriterDuckDB
	writer2 := WriterType("duckdb")

	suite.Equal(writer1, writer2)
}

func (suite *TypesTestSuite) TestProviderTypeCount() {
	providers := []ProviderType{
		ProviderPolygon,
		ProviderBinance,
	}

	suite.Len(providers, 2)
}

func (suite *TypesTestSuite) TestClientConfigStructFields() {
	config := ClientConfig{
		ProviderType:  ProviderPolygon,
		WriterType:    WriterDuckDB,
		DataPath:      "/test/path",
		PolygonApiKey: "test-key",
	}

	suite.Equal(ProviderPolygon, config.ProviderType)
	suite.Equal(WriterDuckDB, config.WriterType)
	suite.Equal("/test/path", config.DataPath)
	suite.Equal("test-key", config.PolygonApiKey)
}

func (suite *TypesTestSuite) TestClientConfigEmptyStruct() {
	config := ClientConfig{}

	suite.Empty(string(config.ProviderType))
	suite.Empty(string(config.WriterType))
	suite.Empty(config.DataPath)
	suite.Empty(config.PolygonApiKey)
}

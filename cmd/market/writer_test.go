package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type WriterTestSuite struct {
	suite.Suite
}

func (suite *WriterTestSuite) TestMarketWriterConstants() {
	suite.Equal(MarketWriter("duckdb"), MarketWriterDuckDB)
}

func (suite *WriterTestSuite) TestMarketWriterValues() {
	suite.Equal("duckdb", string(MarketWriterDuckDB))
}

func (suite *WriterTestSuite) TestMarketWriterType() {
	var writer MarketWriter
	writer = MarketWriterDuckDB
	suite.Equal("duckdb", string(writer))
}

func TestWriterSuite(t *testing.T) {
	suite.Run(t, new(WriterTestSuite))
}

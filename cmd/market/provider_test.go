package main

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type ProviderTestSuite struct {
	suite.Suite
}

func (suite *ProviderTestSuite) TestMarketProviderConstants() {
	suite.Equal(MarketProvider("polygon"), MarketProviderPolygon)
	suite.Equal(MarketProvider("binance"), MarketProviderBinance)
}

func (suite *ProviderTestSuite) TestMarketProviderValues() {
	suite.Equal("polygon", string(MarketProviderPolygon))
	suite.Equal("binance", string(MarketProviderBinance))
}

func (suite *ProviderTestSuite) TestMarketProviderType() {
	var provider MarketProvider
	provider = MarketProviderPolygon
	suite.Equal("polygon", string(provider))

	provider = MarketProviderBinance
	suite.Equal("binance", string(provider))
}

func TestProviderSuite(t *testing.T) {
	suite.Run(t, new(ProviderTestSuite))
}

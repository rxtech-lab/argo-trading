package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type MarketTestSuite struct {
	suite.Suite
}

func TestMarketSuite(t *testing.T) {
	suite.Run(t, new(MarketTestSuite))
}

func (suite *MarketTestSuite) TestMarketDataStruct() {
	now := time.Now()
	data := MarketData{
		Id:     "test-id-123",
		Symbol: "AAPL",
		Time:   now,
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.5,
		Volume: 1000000.0,
	}

	suite.Equal("test-id-123", data.Id)
	suite.Equal("AAPL", data.Symbol)
	suite.Equal(now, data.Time)
	suite.Equal(150.0, data.Open)
	suite.Equal(155.0, data.High)
	suite.Equal(148.0, data.Low)
	suite.Equal(152.5, data.Close)
	suite.Equal(1000000.0, data.Volume)
}

func (suite *MarketTestSuite) TestMarketDataZeroValues() {
	data := MarketData{}

	suite.Empty(data.Id)
	suite.Empty(data.Symbol)
	suite.True(data.Time.IsZero())
	suite.Equal(0.0, data.Open)
	suite.Equal(0.0, data.High)
	suite.Equal(0.0, data.Low)
	suite.Equal(0.0, data.Close)
	suite.Equal(0.0, data.Volume)
}

func (suite *MarketTestSuite) TestMarketDataOHLCVRelationships() {
	// High should be >= all other prices, Low should be <= all other prices
	data := MarketData{
		Id:     "test-1",
		Symbol: "SPY",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   450.0,
		High:   455.0,
		Low:    448.0,
		Close:  452.0,
		Volume: 5000000.0,
	}

	suite.GreaterOrEqual(data.High, data.Open)
	suite.GreaterOrEqual(data.High, data.Close)
	suite.LessOrEqual(data.Low, data.Open)
	suite.LessOrEqual(data.Low, data.Close)
}

func (suite *MarketTestSuite) TestMarketDataMultipleSymbols() {
	spy := MarketData{
		Id:     "spy-1",
		Symbol: "SPY",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   450.0,
		High:   455.0,
		Low:    448.0,
		Close:  452.0,
		Volume: 5000000.0,
	}

	aapl := MarketData{
		Id:     "aapl-1",
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   180.0,
		High:   182.0,
		Low:    178.0,
		Close:  181.0,
		Volume: 2000000.0,
	}

	suite.NotEqual(spy.Id, aapl.Id)
	suite.NotEqual(spy.Symbol, aapl.Symbol)
	suite.Equal(spy.Time, aapl.Time)
}

func (suite *MarketTestSuite) TestMarketDataCryptoSymbol() {
	btc := MarketData{
		Id:     "btc-1",
		Symbol: "BTCUSD",
		Time:   time.Date(2023, 6, 15, 12, 0, 0, 0, time.UTC),
		Open:   26500.0,
		High:   27000.0,
		Low:    26000.0,
		Close:  26750.0,
		Volume: 100.5,
	}

	suite.Equal("BTCUSD", btc.Symbol)
	suite.Greater(btc.Open, 0.0)
	suite.Greater(btc.Volume, 0.0)
}

func (suite *MarketTestSuite) TestMarketDataNegativeVolume() {
	// Volume can technically be zero but typically not negative
	data := MarketData{
		Id:     "test-1",
		Symbol: "TEST",
		Volume: 0.0,
	}

	suite.Equal(0.0, data.Volume)
}

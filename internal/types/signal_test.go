package types

import (
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

type SignalTestSuite struct {
	suite.Suite
}

func TestSignalSuite(t *testing.T) {
	suite.Run(t, new(SignalTestSuite))
}

func (suite *SignalTestSuite) TestSignalTypeConstants() {
	suite.Equal(SignalType("buy_long"), SignalTypeBuyLong)
	suite.Equal(SignalType("sell_long"), SignalTypeSellLong)
	suite.Equal(SignalType("buy_short"), SignalTypeBuyShort)
	suite.Equal(SignalType("sell_short"), SignalTypeSellShort)
	suite.Equal(SignalType("no_action"), SignalTypeNoAction)
	suite.Equal(SignalType("close_position"), SignalTypeClosePosition)
	suite.Equal(SignalType("wait"), SignalTypeWait)
	suite.Equal(SignalType("abort"), SignalTypeAbort)
}

func (suite *SignalTestSuite) TestSignalStruct() {
	now := time.Now()
	signal := Signal{
		Time:      now,
		Type:      SignalTypeBuyLong,
		Name:      "RSI Oversold Signal",
		Reason:    "RSI crossed below 30",
		RawValue:  28.5,
		Symbol:    "AAPL",
		Indicator: IndicatorTypeRSI,
	}

	suite.Equal(now, signal.Time)
	suite.Equal(SignalTypeBuyLong, signal.Type)
	suite.Equal("RSI Oversold Signal", signal.Name)
	suite.Equal("RSI crossed below 30", signal.Reason)
	suite.Equal(28.5, signal.RawValue)
	suite.Equal("AAPL", signal.Symbol)
	suite.Equal(IndicatorTypeRSI, signal.Indicator)
}

func (suite *SignalTestSuite) TestSignalZeroValues() {
	signal := Signal{}

	suite.True(signal.Time.IsZero())
	suite.Empty(string(signal.Type))
	suite.Empty(signal.Name)
	suite.Empty(signal.Reason)
	suite.Nil(signal.RawValue)
	suite.Empty(signal.Symbol)
	suite.Empty(string(signal.Indicator))
}

func (suite *SignalTestSuite) TestSignalBuyLong() {
	signal := Signal{
		Time:      time.Date(2023, 6, 15, 10, 30, 0, 0, time.UTC),
		Type:      SignalTypeBuyLong,
		Name:      "MACD Crossover",
		Reason:    "MACD line crossed above signal line",
		RawValue:  map[string]float64{"macd": 0.5, "signal": 0.3},
		Symbol:    "SPY",
		Indicator: IndicatorTypeMACD,
	}

	suite.Equal(SignalTypeBuyLong, signal.Type)
	suite.Equal(IndicatorTypeMACD, signal.Indicator)
}

func (suite *SignalTestSuite) TestSignalSellLong() {
	signal := Signal{
		Type:      SignalTypeSellLong,
		Name:      "Overbought Exit",
		Reason:    "RSI exceeded 70",
		RawValue:  75.2,
		Symbol:    "AAPL",
		Indicator: IndicatorTypeRSI,
	}

	suite.Equal(SignalTypeSellLong, signal.Type)
}

func (suite *SignalTestSuite) TestSignalBuyShort() {
	signal := Signal{
		Type:      SignalTypeBuyShort,
		Name:      "Bearish Signal",
		Reason:    "Price broke support level",
		Symbol:    "TSLA",
		Indicator: IndicatorTypeBollingerBands,
	}

	suite.Equal(SignalTypeBuyShort, signal.Type)
}

func (suite *SignalTestSuite) TestSignalSellShort() {
	signal := Signal{
		Type:      SignalTypeSellShort,
		Name:      "Cover Short",
		Reason:    "Target reached",
		Symbol:    "META",
		Indicator: IndicatorTypeEMA,
	}

	suite.Equal(SignalTypeSellShort, signal.Type)
}

func (suite *SignalTestSuite) TestSignalNoAction() {
	signal := Signal{
		Type:   SignalTypeNoAction,
		Reason: "No clear signal",
	}

	suite.Equal(SignalTypeNoAction, signal.Type)
}

func (suite *SignalTestSuite) TestSignalClosePosition() {
	signal := Signal{
		Type:   SignalTypeClosePosition,
		Reason: "End of trading session",
		Symbol: "SPY",
	}

	suite.Equal(SignalTypeClosePosition, signal.Type)
}

func (suite *SignalTestSuite) TestSignalWait() {
	signal := Signal{
		Type:   SignalTypeWait,
		Reason: "Waiting for confirmation",
	}

	suite.Equal(SignalTypeWait, signal.Type)
}

func (suite *SignalTestSuite) TestSignalAbort() {
	signal := Signal{
		Type:   SignalTypeAbort,
		Reason: "Market conditions too volatile",
	}

	suite.Equal(SignalTypeAbort, signal.Type)
}

func (suite *SignalTestSuite) TestSignalWithComplexRawValue() {
	signal := Signal{
		Type: SignalTypeBuyLong,
		RawValue: struct {
			MACD       float64
			Signal     float64
			Histogram  float64
		}{
			MACD:      0.5,
			Signal:    0.3,
			Histogram: 0.2,
		},
		Indicator: IndicatorTypeMACD,
	}

	suite.NotNil(signal.RawValue)
}

func (suite *SignalTestSuite) TestSignalWithMultipleIndicators() {
	// Test different indicator types can be assigned
	indicators := []IndicatorType{
		IndicatorTypeRSI,
		IndicatorTypeMACD,
		IndicatorTypeBollingerBands,
		IndicatorTypeEMA,
		IndicatorTypeATR,
		IndicatorTypeRangeFilter,
		IndicatorTypeWaddahAttar,
	}

	for _, ind := range indicators {
		signal := Signal{
			Type:      SignalTypeBuyLong,
			Indicator: ind,
		}
		suite.Equal(ind, signal.Indicator)
	}
}

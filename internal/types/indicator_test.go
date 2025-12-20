package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type IndicatorTestSuite struct {
	suite.Suite
}

func TestIndicatorSuite(t *testing.T) {
	suite.Run(t, new(IndicatorTestSuite))
}

func (suite *IndicatorTestSuite) TestIndicatorTypeConstants() {
	suite.Equal(IndicatorType("rsi"), IndicatorTypeRSI)
	suite.Equal(IndicatorType("macd"), IndicatorTypeMACD)
	suite.Equal(IndicatorType("bollinger_bands"), IndicatorTypeBollingerBands)
	suite.Equal(IndicatorType("stochastic_oscillator"), IndicatorTypeStochasticOsciallator)
	suite.Equal(IndicatorType("williams_r"), IndicatorTypeWilliamsR)
	suite.Equal(IndicatorType("adx"), IndicatorTypeADX)
	suite.Equal(IndicatorType("cci"), IndicatorTypeCCI)
	suite.Equal(IndicatorType("ao"), IndicatorTypeAO)
	suite.Equal(IndicatorType("trend_strength"), IndicatorTypeTrendStrength)
	suite.Equal(IndicatorType("range_filter"), IndicatorTypeRangeFilter)
	suite.Equal(IndicatorType("ema"), IndicatorTypeEMA)
	suite.Equal(IndicatorType("waddah_attar"), IndicatorTypeWaddahAttar)
	suite.Equal(IndicatorType("atr"), IndicatorTypeATR)
	suite.Equal(IndicatorType("ma"), IndicatorTypeMA)
}

func (suite *IndicatorTestSuite) TestIndicatorTypeAsString() {
	suite.Equal("rsi", string(IndicatorTypeRSI))
	suite.Equal("macd", string(IndicatorTypeMACD))
	suite.Equal("bollinger_bands", string(IndicatorTypeBollingerBands))
	suite.Equal("stochastic_oscillator", string(IndicatorTypeStochasticOsciallator))
	suite.Equal("williams_r", string(IndicatorTypeWilliamsR))
	suite.Equal("adx", string(IndicatorTypeADX))
	suite.Equal("cci", string(IndicatorTypeCCI))
	suite.Equal("ao", string(IndicatorTypeAO))
	suite.Equal("trend_strength", string(IndicatorTypeTrendStrength))
	suite.Equal("range_filter", string(IndicatorTypeRangeFilter))
	suite.Equal("ema", string(IndicatorTypeEMA))
	suite.Equal("waddah_attar", string(IndicatorTypeWaddahAttar))
	suite.Equal("atr", string(IndicatorTypeATR))
	suite.Equal("ma", string(IndicatorTypeMA))
}

func (suite *IndicatorTestSuite) TestIndicatorTypeCount() {
	// Verify we have the expected number of indicator types defined
	indicators := []IndicatorType{
		IndicatorTypeRSI,
		IndicatorTypeMACD,
		IndicatorTypeBollingerBands,
		IndicatorTypeStochasticOsciallator,
		IndicatorTypeWilliamsR,
		IndicatorTypeADX,
		IndicatorTypeCCI,
		IndicatorTypeAO,
		IndicatorTypeTrendStrength,
		IndicatorTypeRangeFilter,
		IndicatorTypeEMA,
		IndicatorTypeWaddahAttar,
		IndicatorTypeATR,
		IndicatorTypeMA,
	}

	suite.Len(indicators, 14)
}

func (suite *IndicatorTestSuite) TestIndicatorTypeUniqueness() {
	// Ensure all indicator types have unique values
	indicators := []IndicatorType{
		IndicatorTypeRSI,
		IndicatorTypeMACD,
		IndicatorTypeBollingerBands,
		IndicatorTypeStochasticOsciallator,
		IndicatorTypeWilliamsR,
		IndicatorTypeADX,
		IndicatorTypeCCI,
		IndicatorTypeAO,
		IndicatorTypeTrendStrength,
		IndicatorTypeRangeFilter,
		IndicatorTypeEMA,
		IndicatorTypeWaddahAttar,
		IndicatorTypeATR,
		IndicatorTypeMA,
	}

	seen := make(map[IndicatorType]bool)
	for _, ind := range indicators {
		suite.False(seen[ind], "Duplicate indicator type found: %s", ind)
		seen[ind] = true
	}
}

func (suite *IndicatorTestSuite) TestIndicatorTypeEquality() {
	// Test that same indicator types are equal
	ind1 := IndicatorTypeRSI
	ind2 := IndicatorType("rsi")

	suite.Equal(ind1, ind2)
}

func (suite *IndicatorTestSuite) TestIndicatorTypeInequality() {
	// Test that different indicator types are not equal
	suite.NotEqual(IndicatorTypeRSI, IndicatorTypeMACD)
	suite.NotEqual(IndicatorTypeEMA, IndicatorTypeMA)
	suite.NotEqual(IndicatorTypeATR, IndicatorTypeADX)
}

func (suite *IndicatorTestSuite) TestMomentumIndicators() {
	// Test momentum indicators
	momentumIndicators := []IndicatorType{
		IndicatorTypeRSI,
		IndicatorTypeMACD,
		IndicatorTypeStochasticOsciallator,
		IndicatorTypeWilliamsR,
		IndicatorTypeCCI,
		IndicatorTypeAO,
	}

	for _, ind := range momentumIndicators {
		suite.NotEmpty(string(ind))
	}
}

func (suite *IndicatorTestSuite) TestTrendIndicators() {
	// Test trend indicators
	trendIndicators := []IndicatorType{
		IndicatorTypeADX,
		IndicatorTypeTrendStrength,
		IndicatorTypeEMA,
		IndicatorTypeMA,
	}

	for _, ind := range trendIndicators {
		suite.NotEmpty(string(ind))
	}
}

func (suite *IndicatorTestSuite) TestVolatilityIndicators() {
	// Test volatility indicators
	volatilityIndicators := []IndicatorType{
		IndicatorTypeBollingerBands,
		IndicatorTypeATR,
		IndicatorTypeRangeFilter,
	}

	for _, ind := range volatilityIndicators {
		suite.NotEmpty(string(ind))
	}
}

func (suite *IndicatorTestSuite) TestCustomIndicator() {
	// Test creating a custom indicator type
	customIndicator := IndicatorType("custom_indicator")

	suite.Equal("custom_indicator", string(customIndicator))
	suite.NotEqual(IndicatorTypeRSI, customIndicator)
}

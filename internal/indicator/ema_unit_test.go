package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type EMAUnitTestSuite struct {
	suite.Suite
}

func TestEMAUnitSuite(t *testing.T) {
	suite.Run(t, new(EMAUnitTestSuite))
}

func (suite *EMAUnitTestSuite) TestNewEMA() {
	ema := NewEMA()
	suite.NotNil(ema)

	// Cast to *EMA to check default values
	emaImpl := ema.(*EMA)
	suite.Equal(20, emaImpl.period)
}

func (suite *EMAUnitTestSuite) TestName() {
	ema := NewEMA()
	suite.Equal(types.IndicatorTypeEMA, ema.Name())
}

func (suite *EMAUnitTestSuite) TestConfigValid() {
	ema := NewEMA()
	emaImpl := ema.(*EMA)

	err := ema.Config(10)
	suite.NoError(err)
	suite.Equal(10, emaImpl.period)
}

func (suite *EMAUnitTestSuite) TestConfigInvalidParamCount() {
	ema := NewEMA()

	// No params
	err := ema.Config()
	suite.Error(err)
	suite.Contains(err.Error(), "expects 1 parameter")

	// Too many params
	err = ema.Config(10, 20)
	suite.Error(err)
}

func (suite *EMAUnitTestSuite) TestConfigInvalidPeriodType() {
	ema := NewEMA()
	err := ema.Config("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *EMAUnitTestSuite) TestConfigInvalidPeriodValue() {
	ema := NewEMA()

	err := ema.Config(0)
	suite.Error(err)
	suite.Contains(err.Error(), "must be a positive integer")

	err = ema.Config(-5)
	suite.Error(err)
}

func (suite *EMAUnitTestSuite) TestRawValueInvalidParams() {
	ema := NewEMA()

	// Too few params
	_, err := ema.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = ema.RawValue("symbol")
	suite.Error(err)

	_, err = ema.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = ema.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "expected string")

	// Invalid second param type
	_, err = ema.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "expected time.Time")
}

func (suite *EMAUnitTestSuite) TestConfigMultipleTimes() {
	ema := NewEMA()
	emaImpl := ema.(*EMA)

	// Configure multiple times
	err := ema.Config(10)
	suite.NoError(err)
	suite.Equal(10, emaImpl.period)

	err = ema.Config(30)
	suite.NoError(err)
	suite.Equal(30, emaImpl.period)
}

func (suite *EMAUnitTestSuite) TestCalculateSimpleMovingAverage() {
	data := []types.MarketData{
		{Close: 10.0},
		{Close: 20.0},
		{Close: 30.0},
		{Close: 40.0},
		{Close: 50.0},
	}

	result := calculateSimpleMovingAverage(data)
	suite.Equal(30.0, result) // (10+20+30+40+50)/5 = 30
}

func (suite *EMAUnitTestSuite) TestCalculateSimpleMovingAverageSingleValue() {
	data := []types.MarketData{
		{Close: 100.0},
	}

	result := calculateSimpleMovingAverage(data)
	suite.Equal(100.0, result)
}

func (suite *EMAUnitTestSuite) TestCalculateExponentialMovingAverageEmptyData() {
	data := []types.MarketData{}
	result := calculateExponentialMovingAverage(data, 5)
	suite.Equal(0.0, result)
}

func (suite *EMAUnitTestSuite) TestCalculateExponentialMovingAverageBasic() {
	data := []types.MarketData{
		{Close: 10.0},
		{Close: 11.0},
		{Close: 12.0},
		{Close: 13.0},
		{Close: 14.0},
		{Close: 15.0},
		{Close: 16.0},
	}

	result := calculateExponentialMovingAverage(data, 3)
	// Should return a valid EMA value
	suite.Greater(result, 0.0)
}

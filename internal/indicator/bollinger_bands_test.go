package indicator

import (
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/errors"
	"github.com/stretchr/testify/suite"
)

type BollingerBandsTestSuite struct {
	suite.Suite
}

func TestBollingerBandsSuite(t *testing.T) {
	suite.Run(t, new(BollingerBandsTestSuite))
}

func (suite *BollingerBandsTestSuite) TestNewBollingerBands() {
	bb := NewBollingerBands()
	suite.NotNil(bb)

	// Cast to *BollingerBands to check default values
	bbImpl := bb.(*BollingerBands)
	suite.Equal(20, bbImpl.period)
	suite.Equal(2.0, bbImpl.stdDev)
	suite.Equal(time.Hour*24, bbImpl.lookback)
}

func (suite *BollingerBandsTestSuite) TestName() {
	bb := NewBollingerBands()
	suite.Equal(types.IndicatorTypeBollingerBands, bb.Name())
}

func (suite *BollingerBandsTestSuite) TestConfigValid() {
	bb := NewBollingerBands()
	bbImpl := bb.(*BollingerBands)

	err := bb.Config(10, 1.5, time.Hour*12)
	suite.NoError(err)
	suite.Equal(10, bbImpl.period)
	suite.Equal(1.5, bbImpl.stdDev)
	suite.Equal(time.Hour*12, bbImpl.lookback)
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidParamCount() {
	bb := NewBollingerBands()

	// Too few params
	err := bb.Config(10, 1.5)
	suite.Error(err)
	suite.Contains(err.Error(), "expects 3 parameters")

	// Too many params
	err = bb.Config(10, 1.5, time.Hour, "extra")
	suite.Error(err)
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidPeriodType() {
	bb := NewBollingerBands()
	err := bb.Config("invalid", 1.5, time.Hour)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidPeriodValue() {
	bb := NewBollingerBands()
	err := bb.Config(0, 1.5, time.Hour)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")

	err = bb.Config(-5, 1.5, time.Hour)
	suite.Error(err)
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidStdDevType() {
	bb := NewBollingerBands()
	err := bb.Config(10, "invalid", time.Hour)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for stdDev")
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidStdDevValue() {
	bb := NewBollingerBands()
	err := bb.Config(10, 0.0, time.Hour)
	suite.Error(err)
	suite.Contains(err.Error(), "stdDev must be a positive number")

	err = bb.Config(10, -1.0, time.Hour)
	suite.Error(err)
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidLookbackType() {
	bb := NewBollingerBands()
	err := bb.Config(10, 1.5, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for lookback")
}

func (suite *BollingerBandsTestSuite) TestConfigInvalidLookbackValue() {
	bb := NewBollingerBands()
	err := bb.Config(10, 1.5, time.Duration(0))
	suite.Error(err)
	suite.Contains(err.Error(), "lookback must be a positive duration")

	err = bb.Config(10, 1.5, -time.Hour)
	suite.Error(err)
}

func (suite *BollingerBandsTestSuite) TestCalculateBands() {
	bb := &BollingerBands{
		period: 3,
		stdDev: 2.0,
	}

	// Create test market data
	data := []types.MarketData{
		{Close: 100.0},
		{Close: 102.0},
		{Close: 101.0},
	}

	upper, middle, lower, err := bb.calculateBands(data)
	suite.NoError(err)

	// Middle should be the average: (100 + 102 + 101) / 3 = 101.0
	suite.InDelta(101.0, middle, 0.001)

	// Upper and lower should be 2 std devs away
	suite.Greater(upper, middle)
	suite.Less(lower, middle)
}

func (suite *BollingerBandsTestSuite) TestCalculateBandsInsufficientData() {
	bb := &BollingerBands{
		period: 20,
		stdDev: 2.0,
	}

	// Only 5 data points, but period is 20
	data := []types.MarketData{
		{Close: 100.0},
		{Close: 102.0},
		{Close: 101.0},
		{Close: 103.0},
		{Close: 99.0},
	}

	_, _, _, err := bb.calculateBands(data)
	suite.Error(err)

	// Check it's an InsufficientDataError using the helper function
	suite.True(errors.IsInsufficientDataError(err))
	suite.Contains(err.Error(), "insufficient data points")

	// Verify the error contains the expected values
	var insufficientErr *errors.InsufficientDataError
	suite.Require().True(errors.As(err, &insufficientErr))
	suite.Require().NotNil(insufficientErr)
	suite.Equal(20, insufficientErr.Required)
	suite.Equal(5, insufficientErr.Actual)
}

func (suite *BollingerBandsTestSuite) TestRawValueInvalidParams() {
	bb := NewBollingerBands()

	// Too few params
	_, err := bb.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 2 parameters")

	// Invalid first param type
	_, err = bb.RawValue(123, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type types.MarketData")

	// Invalid second param type
	_, err = bb.RawValue(types.MarketData{}, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type IndicatorContext")
}

func (suite *BollingerBandsTestSuite) TestInsufficientDataError() {
	err := errors.NewInsufficientDataError(20, 5, "AAPL", "test error message")
	suite.Equal("test error message", err.Error())
	suite.True(errors.IsInsufficientDataError(err))
}

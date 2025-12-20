package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type MAUnitTestSuite struct {
	suite.Suite
}

func TestMAUnitSuite(t *testing.T) {
	suite.Run(t, new(MAUnitTestSuite))
}

func (suite *MAUnitTestSuite) TestNewMA() {
	ma := NewMA()
	suite.NotNil(ma)

	// Cast to *MA to check default values
	maImpl := ma.(*MA)
	suite.Equal(20, maImpl.period)
}

func (suite *MAUnitTestSuite) TestName() {
	ma := NewMA()
	suite.Equal(types.IndicatorTypeMA, ma.Name())
}

func (suite *MAUnitTestSuite) TestConfigValid() {
	ma := NewMA()
	maImpl := ma.(*MA)

	err := ma.Config(10)
	suite.NoError(err)
	suite.Equal(10, maImpl.period)
}

func (suite *MAUnitTestSuite) TestConfigWithFloat64() {
	ma := NewMA()
	maImpl := ma.(*MA)

	// MA supports float64 conversion
	err := ma.Config(15.0)
	suite.NoError(err)
	suite.Equal(15, maImpl.period)
}

func (suite *MAUnitTestSuite) TestConfigInvalidParamCount() {
	ma := NewMA()

	// No params
	err := ma.Config()
	suite.Error(err)
	suite.Contains(err.Error(), "expects 1 parameter")

	// Too many params
	err = ma.Config(10, 20)
	suite.Error(err)
}

func (suite *MAUnitTestSuite) TestConfigInvalidPeriodType() {
	ma := NewMA()
	err := ma.Config("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *MAUnitTestSuite) TestConfigInvalidPeriodValue() {
	ma := NewMA()

	err := ma.Config(0)
	suite.Error(err)
	suite.Contains(err.Error(), "must be a positive integer")

	err = ma.Config(-5)
	suite.Error(err)
}

func (suite *MAUnitTestSuite) TestRawValueInvalidParams() {
	ma := NewMA()

	// Too few params
	_, err := ma.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = ma.RawValue("symbol")
	suite.Error(err)

	_, err = ma.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = ma.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "expected string")

	// Invalid second param type
	_, err = ma.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "expected time.Time")
}

func (suite *MAUnitTestSuite) TestConfigMultipleTimes() {
	ma := NewMA()
	maImpl := ma.(*MA)

	// Configure multiple times
	err := ma.Config(10)
	suite.NoError(err)
	suite.Equal(10, maImpl.period)

	err = ma.Config(30)
	suite.NoError(err)
	suite.Equal(30, maImpl.period)
}

package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type ATRTestSuite struct {
	suite.Suite
}

func TestATRSuite(t *testing.T) {
	suite.Run(t, new(ATRTestSuite))
}

func (suite *ATRTestSuite) TestNewATR() {
	atr := NewATR()
	suite.NotNil(atr)

	// Cast to *ATR to check default values
	atrImpl := atr.(*ATR)
	suite.Equal(14, atrImpl.period)
}

func (suite *ATRTestSuite) TestName() {
	atr := NewATR()
	suite.Equal(types.IndicatorTypeATR, atr.Name())
}

func (suite *ATRTestSuite) TestConfigValid() {
	atr := NewATR()
	atrImpl := atr.(*ATR)

	err := atr.Config(20)
	suite.NoError(err)
	suite.Equal(20, atrImpl.period)
}

func (suite *ATRTestSuite) TestConfigInvalidParamCount() {
	atr := NewATR()

	// No params
	err := atr.Config()
	suite.Error(err)
	suite.Contains(err.Error(), "expects 1 parameter")

	// Too many params
	err = atr.Config(10, 20)
	suite.Error(err)
}

func (suite *ATRTestSuite) TestConfigInvalidPeriodType() {
	atr := NewATR()
	err := atr.Config("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *ATRTestSuite) TestConfigInvalidPeriodValue() {
	atr := NewATR()

	err := atr.Config(0)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")

	err = atr.Config(-5)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")
}

func (suite *ATRTestSuite) TestRawValueInvalidParams() {
	atr := NewATR()

	// Too few params
	_, err := atr.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = atr.RawValue("symbol")
	suite.Error(err)

	_, err = atr.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = atr.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	// Invalid second param type
	_, err = atr.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *ATRTestSuite) TestConfigMultipleTimes() {
	atr := NewATR()
	atrImpl := atr.(*ATR)

	// Configure multiple times
	err := atr.Config(10)
	suite.NoError(err)
	suite.Equal(10, atrImpl.period)

	err = atr.Config(30)
	suite.NoError(err)
	suite.Equal(30, atrImpl.period)
}

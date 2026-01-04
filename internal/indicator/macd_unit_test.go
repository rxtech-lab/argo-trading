package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type MACDUnitTestSuite struct {
	suite.Suite
}

func TestMACDUnitSuite(t *testing.T) {
	suite.Run(t, new(MACDUnitTestSuite))
}

func (suite *MACDUnitTestSuite) TestNewMACD() {
	macd := NewMACD()
	suite.NotNil(macd)

	// Cast to *MACD to check default values
	macdImpl := macd.(*MACD)
	suite.Equal(12, macdImpl.fastPeriod)
	suite.Equal(26, macdImpl.slowPeriod)
	suite.Equal(9, macdImpl.signalPeriod)
}

func (suite *MACDUnitTestSuite) TestName() {
	macd := NewMACD()
	suite.Equal(types.IndicatorTypeMACD, macd.Name())
}

func (suite *MACDUnitTestSuite) TestConfigValid() {
	macd := NewMACD()
	macdImpl := macd.(*MACD)

	err := macd.Config(10, 20, 5)
	suite.NoError(err)
	suite.Equal(10, macdImpl.fastPeriod)
	suite.Equal(20, macdImpl.slowPeriod)
	suite.Equal(5, macdImpl.signalPeriod)
}

func (suite *MACDUnitTestSuite) TestConfigInvalidParamCount() {
	macd := NewMACD()

	// Too few params
	err := macd.Config(10, 20)
	suite.Error(err)
	suite.Contains(err.Error(), "expects 3 parameters")

	// Too many params
	err = macd.Config(10, 20, 5, 10)
	suite.Error(err)
}

func (suite *MACDUnitTestSuite) TestConfigInvalidFastPeriodType() {
	macd := NewMACD()
	err := macd.Config("invalid", 20, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for fastPeriod")
}

func (suite *MACDUnitTestSuite) TestConfigInvalidFastPeriodValue() {
	macd := NewMACD()

	err := macd.Config(0, 20, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "fastPeriod must be a positive integer")

	err = macd.Config(-5, 20, 5)
	suite.Error(err)
}

func (suite *MACDUnitTestSuite) TestConfigInvalidSlowPeriodType() {
	macd := NewMACD()
	err := macd.Config(10, "invalid", 5)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for slowPeriod")
}

func (suite *MACDUnitTestSuite) TestConfigInvalidSlowPeriodValue() {
	macd := NewMACD()

	err := macd.Config(10, 0, 5)
	suite.Error(err)
	suite.Contains(err.Error(), "slowPeriod must be a positive integer")

	err = macd.Config(10, -5, 5)
	suite.Error(err)
}

func (suite *MACDUnitTestSuite) TestConfigInvalidSignalPeriodType() {
	macd := NewMACD()
	err := macd.Config(10, 20, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for signalPeriod")
}

func (suite *MACDUnitTestSuite) TestConfigInvalidSignalPeriodValue() {
	macd := NewMACD()

	err := macd.Config(10, 20, 0)
	suite.Error(err)
	suite.Contains(err.Error(), "signalPeriod must be a positive integer")

	err = macd.Config(10, 20, -5)
	suite.Error(err)
}

func (suite *MACDUnitTestSuite) TestRawValueInvalidParams() {
	macd := NewMACD()

	// Too few params
	_, err := macd.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = macd.RawValue("symbol")
	suite.Error(err)

	_, err = macd.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = macd.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	// Invalid second param type
	_, err = macd.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *MACDUnitTestSuite) TestConfigMultipleTimes() {
	macd := NewMACD()
	macdImpl := macd.(*MACD)

	// Configure multiple times
	err := macd.Config(8, 16, 5)
	suite.NoError(err)
	suite.Equal(8, macdImpl.fastPeriod)
	suite.Equal(16, macdImpl.slowPeriod)
	suite.Equal(5, macdImpl.signalPeriod)

	err = macd.Config(15, 30, 10)
	suite.NoError(err)
	suite.Equal(15, macdImpl.fastPeriod)
	suite.Equal(30, macdImpl.slowPeriod)
	suite.Equal(10, macdImpl.signalPeriod)
}

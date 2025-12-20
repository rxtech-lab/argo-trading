package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type WaddahAttarTestSuite struct {
	suite.Suite
}

func TestWaddahAttarSuite(t *testing.T) {
	suite.Run(t, new(WaddahAttarTestSuite))
}

func (suite *WaddahAttarTestSuite) TestNewWaddahAttar() {
	wa := NewWaddahAttar()
	suite.NotNil(wa)

	// Cast to *WaddahAttar to check default values
	waImpl := wa.(*WaddahAttar)
	suite.Equal(20, waImpl.fastPeriod)
	suite.Equal(40, waImpl.slowPeriod)
	suite.Equal(9, waImpl.signalPeriod)
	suite.Equal(14, waImpl.atrPeriod)
	suite.Equal(150.0, waImpl.multiplier)
}

func (suite *WaddahAttarTestSuite) TestName() {
	wa := NewWaddahAttar()
	suite.Equal(types.IndicatorTypeWaddahAttar, wa.Name())
}

func (suite *WaddahAttarTestSuite) TestConfigValid() {
	wa := NewWaddahAttar()
	waImpl := wa.(*WaddahAttar)

	err := wa.Config(12, 26, 9, 14, 100.0)
	suite.NoError(err)
	suite.Equal(12, waImpl.fastPeriod)
	suite.Equal(26, waImpl.slowPeriod)
	suite.Equal(9, waImpl.signalPeriod)
	suite.Equal(14, waImpl.atrPeriod)
	suite.Equal(100.0, waImpl.multiplier)
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidParamCount() {
	wa := NewWaddahAttar()

	// Too few params
	err := wa.Config(12, 26, 9, 14)
	suite.Error(err)
	suite.Contains(err.Error(), "expects 5 parameters")

	// Too many params
	err = wa.Config(12, 26, 9, 14, 100.0, "extra")
	suite.Error(err)
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidFastPeriodType() {
	wa := NewWaddahAttar()
	err := wa.Config("invalid", 26, 9, 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for fastPeriod")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidFastPeriodValue() {
	wa := NewWaddahAttar()

	err := wa.Config(0, 26, 9, 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "fastPeriod must be a positive integer")

	err = wa.Config(-5, 26, 9, 14, 100.0)
	suite.Error(err)
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidSlowPeriodType() {
	wa := NewWaddahAttar()
	err := wa.Config(12, "invalid", 9, 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for slowPeriod")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidSlowPeriodValue() {
	wa := NewWaddahAttar()

	err := wa.Config(12, 0, 9, 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "slowPeriod must be a positive integer")

	err = wa.Config(12, -10, 9, 14, 100.0)
	suite.Error(err)
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidSignalPeriodType() {
	wa := NewWaddahAttar()
	err := wa.Config(12, 26, "invalid", 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for signalPeriod")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidSignalPeriodValue() {
	wa := NewWaddahAttar()

	err := wa.Config(12, 26, 0, 14, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "signalPeriod must be a positive integer")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidATRPeriodType() {
	wa := NewWaddahAttar()
	err := wa.Config(12, 26, 9, "invalid", 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for atrPeriod")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidATRPeriodValue() {
	wa := NewWaddahAttar()

	err := wa.Config(12, 26, 9, 0, 100.0)
	suite.Error(err)
	suite.Contains(err.Error(), "atrPeriod must be a positive integer")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidMultiplierType() {
	wa := NewWaddahAttar()
	err := wa.Config(12, 26, 9, 14, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for multiplier")
}

func (suite *WaddahAttarTestSuite) TestConfigInvalidMultiplierValue() {
	wa := NewWaddahAttar()

	err := wa.Config(12, 26, 9, 14, 0.0)
	suite.Error(err)
	suite.Contains(err.Error(), "multiplier must be a positive number")

	err = wa.Config(12, 26, 9, 14, -50.0)
	suite.Error(err)
}

func (suite *WaddahAttarTestSuite) TestRawValueInvalidParams() {
	wa := NewWaddahAttar()

	// Too few params
	_, err := wa.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = wa.RawValue("symbol")
	suite.Error(err)

	_, err = wa.RawValue("symbol", "not-a-time")
	suite.Error(err)

	// Invalid first param type
	_, err = wa.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	// Invalid second param type
	_, err = wa.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *WaddahAttarTestSuite) TestWaddahAttarDataStruct() {
	data := WaddahAttarData{
		macd:        0.5,
		signal:      0.3,
		hist:        0.2,
		atr:         1.5,
		trend:       75.0,
		explosion:   225.0,
		initialized: true,
	}

	suite.Equal(0.5, data.macd)
	suite.Equal(0.3, data.signal)
	suite.Equal(0.2, data.hist)
	suite.Equal(1.5, data.atr)
	suite.Equal(75.0, data.trend)
	suite.Equal(225.0, data.explosion)
	suite.True(data.initialized)
}

func (suite *WaddahAttarTestSuite) TestConfigMultipleTimes() {
	wa := NewWaddahAttar()
	waImpl := wa.(*WaddahAttar)

	// Configure multiple times
	err := wa.Config(10, 20, 5, 10, 100.0)
	suite.NoError(err)
	suite.Equal(10, waImpl.fastPeriod)

	err = wa.Config(15, 30, 7, 12, 200.0)
	suite.NoError(err)
	suite.Equal(15, waImpl.fastPeriod)
	suite.Equal(30, waImpl.slowPeriod)
	suite.Equal(7, waImpl.signalPeriod)
	suite.Equal(12, waImpl.atrPeriod)
	suite.Equal(200.0, waImpl.multiplier)
}

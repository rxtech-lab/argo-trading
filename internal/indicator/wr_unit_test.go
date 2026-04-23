package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type WRUnitTestSuite struct {
	suite.Suite
}

func TestWRUnitSuite(t *testing.T) {
	suite.Run(t, new(WRUnitTestSuite))
}

func (suite *WRUnitTestSuite) TestNewWR() {
	wr := NewWR()
	suite.NotNil(wr)

	wrImpl := wr.(*WR)
	suite.Equal(14, wrImpl.period)
	suite.Equal(-20.0, wrImpl.overboughtThreshold)
	suite.Equal(-80.0, wrImpl.oversoldThreshold)
}

func (suite *WRUnitTestSuite) TestName() {
	wr := NewWR()
	suite.Equal(types.IndicatorTypeWilliamsR, wr.Name())
}

func (suite *WRUnitTestSuite) TestConfigValidPeriodOnly() {
	wr := NewWR()
	wrImpl := wr.(*WR)

	err := wr.Config(21)
	suite.NoError(err)
	suite.Equal(21, wrImpl.period)
	// Thresholds remain default.
	suite.Equal(-20.0, wrImpl.overboughtThreshold)
	suite.Equal(-80.0, wrImpl.oversoldThreshold)
}

func (suite *WRUnitTestSuite) TestConfigWithBothThresholds() {
	wr := NewWR()
	wrImpl := wr.(*WR)

	err := wr.Config(14, -10.0, -90.0)
	suite.NoError(err)
	suite.Equal(14, wrImpl.period)
	suite.Equal(-10.0, wrImpl.overboughtThreshold)
	suite.Equal(-90.0, wrImpl.oversoldThreshold)
}

func (suite *WRUnitTestSuite) TestConfigNoParams() {
	wr := NewWR()
	err := wr.Config()
	suite.Error(err)
	suite.Contains(err.Error(), "expects at least 1 parameter")
}

func (suite *WRUnitTestSuite) TestConfigInvalidPeriodType() {
	wr := NewWR()
	err := wr.Config("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *WRUnitTestSuite) TestConfigInvalidPeriodValue() {
	wr := NewWR()

	err := wr.Config(0)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")

	err = wr.Config(-5)
	suite.Error(err)
}

func (suite *WRUnitTestSuite) TestConfigInvalidOverboughtThresholdType() {
	wr := NewWR()
	err := wr.Config(14, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for overboughtThreshold")
}

func (suite *WRUnitTestSuite) TestConfigInvalidOversoldThresholdType() {
	wr := NewWR()
	err := wr.Config(14, -20.0, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for oversoldThreshold")
}

func (suite *WRUnitTestSuite) TestRawValueInvalidParams() {
	wr := NewWR()

	_, err := wr.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = wr.RawValue("symbol")
	suite.Error(err)

	_, err = wr.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	_, err = wr.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *WRUnitTestSuite) TestConfigMultipleTimes() {
	wr := NewWR()
	wrImpl := wr.(*WR)

	err := wr.Config(10)
	suite.NoError(err)
	suite.Equal(10, wrImpl.period)

	err = wr.Config(21, -15.0, -85.0)
	suite.NoError(err)
	suite.Equal(21, wrImpl.period)
	suite.Equal(-15.0, wrImpl.overboughtThreshold)
	suite.Equal(-85.0, wrImpl.oversoldThreshold)
}

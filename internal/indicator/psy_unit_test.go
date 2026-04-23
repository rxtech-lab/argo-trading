package indicator

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type PSYUnitTestSuite struct {
	suite.Suite
}

func TestPSYUnitSuite(t *testing.T) {
	suite.Run(t, new(PSYUnitTestSuite))
}

func (suite *PSYUnitTestSuite) TestNewPSY() {
	psy := NewPSY()
	suite.NotNil(psy)

	psyImpl := psy.(*PSY)
	suite.Equal(12, psyImpl.period)
	suite.Equal(75.0, psyImpl.upperThreshold)
	suite.Equal(25.0, psyImpl.lowerThreshold)
}

func (suite *PSYUnitTestSuite) TestName() {
	psy := NewPSY()
	suite.Equal(types.IndicatorTypePSY, psy.Name())
}

func (suite *PSYUnitTestSuite) TestConfigValidPeriodOnly() {
	psy := NewPSY()
	psyImpl := psy.(*PSY)

	err := psy.Config(24)
	suite.NoError(err)
	suite.Equal(24, psyImpl.period)
	// Thresholds remain default.
	suite.Equal(75.0, psyImpl.upperThreshold)
	suite.Equal(25.0, psyImpl.lowerThreshold)
}

func (suite *PSYUnitTestSuite) TestConfigWithBothThresholds() {
	psy := NewPSY()
	psyImpl := psy.(*PSY)

	err := psy.Config(12, 80.0, 20.0)
	suite.NoError(err)
	suite.Equal(12, psyImpl.period)
	suite.Equal(80.0, psyImpl.upperThreshold)
	suite.Equal(20.0, psyImpl.lowerThreshold)
}

func (suite *PSYUnitTestSuite) TestConfigNoParams() {
	psy := NewPSY()
	err := psy.Config()
	suite.Error(err)
	suite.Contains(err.Error(), "expects at least 1 parameter")
}

func (suite *PSYUnitTestSuite) TestConfigInvalidPeriodType() {
	psy := NewPSY()
	err := psy.Config("invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for period")
}

func (suite *PSYUnitTestSuite) TestConfigInvalidPeriodValue() {
	psy := NewPSY()

	err := psy.Config(0)
	suite.Error(err)
	suite.Contains(err.Error(), "period must be a positive integer")

	err = psy.Config(-5)
	suite.Error(err)
}

func (suite *PSYUnitTestSuite) TestConfigInvalidUpperThresholdType() {
	psy := NewPSY()
	err := psy.Config(12, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for upperThreshold")
}

func (suite *PSYUnitTestSuite) TestConfigInvalidLowerThresholdType() {
	psy := NewPSY()
	err := psy.Config(12, 75.0, "invalid")
	suite.Error(err)
	suite.Contains(err.Error(), "invalid type for lowerThreshold")
}

func (suite *PSYUnitTestSuite) TestRawValueInvalidParams() {
	psy := NewPSY()

	_, err := psy.RawValue()
	suite.Error(err)
	suite.Contains(err.Error(), "requires at least 3 parameters")

	_, err = psy.RawValue("symbol")
	suite.Error(err)

	_, err = psy.RawValue(123, nil, nil)
	suite.Error(err)
	suite.Contains(err.Error(), "first parameter must be of type string")

	_, err = psy.RawValue("symbol", "not-a-time", nil)
	suite.Error(err)
	suite.Contains(err.Error(), "second parameter must be of type time.Time")
}

func (suite *PSYUnitTestSuite) TestConfigMultipleTimes() {
	psy := NewPSY()
	psyImpl := psy.(*PSY)

	err := psy.Config(10)
	suite.NoError(err)
	suite.Equal(10, psyImpl.period)

	err = psy.Config(24, 70.0, 30.0)
	suite.NoError(err)
	suite.Equal(24, psyImpl.period)
	suite.Equal(70.0, psyImpl.upperThreshold)
	suite.Equal(30.0, psyImpl.lowerThreshold)
}

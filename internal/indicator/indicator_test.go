package indicator

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type IndicatorInterfaceTestSuite struct {
	suite.Suite
}

func TestIndicatorInterfaceSuite(t *testing.T) {
	suite.Run(t, new(IndicatorInterfaceTestSuite))
}

func (suite *IndicatorInterfaceTestSuite) TestIndicatorContextStruct() {
	ctx := IndicatorContext{}

	suite.Nil(ctx.DataSource)
	suite.Nil(ctx.IndicatorRegistry)
	suite.Nil(ctx.Cache)
}

func (suite *IndicatorInterfaceTestSuite) TestIndicatorContextWithValues() {
	// Create with nil values (just testing struct)
	ctx := IndicatorContext{
		DataSource:        nil,
		IndicatorRegistry: nil,
		Cache:             nil,
	}

	suite.Nil(ctx.DataSource)
	suite.Nil(ctx.IndicatorRegistry)
	suite.Nil(ctx.Cache)
}

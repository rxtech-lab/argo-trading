package tradingprovider

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/pkg/strategy"
	"github.com/stretchr/testify/suite"
)

type BinanceConfigTestSuite struct {
	suite.Suite
}

func TestBinanceConfigTestSuite(t *testing.T) {
	suite.Run(t, new(BinanceConfigTestSuite))
}

func (suite *BinanceConfigTestSuite) TestKeychainFields() {
	fields := strategy.GetKeychainFields(BinanceProviderConfig{})
	suite.Equal([]string{"apiKey", "secretKey"}, fields)
}

package commission_fee

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type CommissionFeeTestSuite struct {
	suite.Suite
}

func TestCommissionFeeSuite(t *testing.T) {
	suite.Run(t, new(CommissionFeeTestSuite))
}

func (suite *CommissionFeeTestSuite) TestZeroCommissionFee() {
	fee := NewZeroCommissionFee()
	suite.NotNil(fee)

	tests := []struct {
		name     string
		quantity float64
		expected float64
	}{
		{"zero quantity", 0, 0},
		{"small quantity", 10, 0},
		{"large quantity", 10000, 0},
		{"negative quantity", -100, 0},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := fee.Calculate(tc.quantity)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestInteractiveBrokerCommissionFee() {
	fee := NewInteractiveBrokerCommissionFee()
	suite.NotNil(fee)

	tests := []struct {
		name     string
		quantity float64
		expected float64
	}{
		{"zero quantity", 0, 1.0},        // minimum fee is 1.0
		{"small quantity - min fee", 10, 1.0},  // 0.005 * 10 = 0.05 < 1.0, so min fee applies
		{"quantity at threshold", 200, 1.0},    // 0.005 * 200 = 1.0, so exactly at threshold
		{"large quantity", 1000, 5.0},          // 0.005 * 1000 = 5.0 > 1.0
		{"very large quantity", 10000, 50.0},   // 0.005 * 10000 = 50.0
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := fee.Calculate(tc.quantity)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestGetCommissionFeeHandler() {
	tests := []struct {
		name           string
		broker         Broker
		expectedType   string
		testQuantity   float64
		expectedResult float64
	}{
		{
			name:           "interactive broker",
			broker:         BrokerInteractiveBroker,
			expectedType:   "*commission_fee.InteractiveBrokerCommissionFee",
			testQuantity:   1000,
			expectedResult: 5.0,
		},
		{
			name:           "zero commission",
			broker:         BrokerZero,
			expectedType:   "*commission_fee.ZeroCommissionFee",
			testQuantity:   1000,
			expectedResult: 0.0,
		},
		{
			name:           "unknown broker defaults to zero",
			broker:         Broker("unknown"),
			expectedType:   "*commission_fee.ZeroCommissionFee",
			testQuantity:   1000,
			expectedResult: 0.0,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			handler := GetCommissionFeeHandler(tc.broker)
			suite.NotNil(handler)
			result := handler.Calculate(tc.testQuantity)
			suite.Equal(tc.expectedResult, result)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestAllBrokers() {
	suite.Len(AllBrokers, 2)
	suite.Contains(AllBrokers, BrokerInteractiveBroker)
	suite.Contains(AllBrokers, BrokerZero)
}

func (suite *CommissionFeeTestSuite) TestBrokerConstants() {
	suite.Equal(Broker("interactive_broker"), BrokerInteractiveBroker)
	suite.Equal(Broker("zero_commission"), BrokerZero)
}

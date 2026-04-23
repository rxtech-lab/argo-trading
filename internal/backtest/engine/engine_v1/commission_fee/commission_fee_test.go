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
		price    float64
		expected float64
	}{
		{"zero quantity", 0, 100, 0},
		{"small quantity", 10, 100, 0},
		{"large quantity", 10000, 100, 0},
		{"negative quantity", -100, 100, 0},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := fee.Calculate(tc.quantity, tc.price)
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
		price    float64
		expected float64
	}{
		{"zero quantity", 0, 100, 1.0},             // minimum fee is 1.0
		{"small quantity - min fee", 10, 100, 1.0}, // 0.005 * 10 = 0.05 < 1.0, so min fee applies
		{"quantity at threshold", 200, 100, 1.0},   // 0.005 * 200 = 1.0, so exactly at threshold
		{"large quantity", 1000, 100, 5.0},         // 0.005 * 1000 = 5.0 > 1.0
		{"very large quantity", 10000, 100, 50.0},  // 0.005 * 10000 = 50.0
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := fee.Calculate(tc.quantity, tc.price)
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestBinanceCommissionFee() {
	fee := NewBinanceCommissionFee()
	suite.NotNil(fee)

	tests := []struct {
		name     string
		quantity float64
		price    float64
		expected float64
	}{
		{"zero quantity", 0, 30000, 0},
		{"zero price", 1, 0, 0},
		{"1 BTC at 30000", 1, 30000, 30.0},     // 0.001 * 1 * 30000
		{"0.5 BTC at 40000", 0.5, 40000, 20.0}, // 0.001 * 0.5 * 40000
		{"fractional notional", 0.1, 1234.5, 0.12345},
		{"negative quantity uses abs", -2, 100, 0.2}, // 0.001 * 2 * 100
		{"negative price uses abs", 2, -100, 0.2},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := fee.Calculate(tc.quantity, tc.price)
			suite.InDelta(tc.expected, result, 1e-9)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestBinanceCommissionFeeCustomRate() {
	// Simulate VIP tier / BNB-discount rate (e.g. 0.075%).
	fee := NewBinanceCommissionFeeWithRate(0.00075)
	suite.NotNil(fee)
	suite.InDelta(0.075, fee.Calculate(1, 100), 1e-9)

	// Negative rate should clamp to zero.
	clamped := NewBinanceCommissionFeeWithRate(-0.5)
	suite.Equal(0.0, clamped.Calculate(1, 100))
}

func (suite *CommissionFeeTestSuite) TestGetCommissionFeeHandler() {
	tests := []struct {
		name           string
		broker         Broker
		expectedType   string
		testQuantity   float64
		testPrice      float64
		expectedResult float64
	}{
		{
			name:           "interactive broker",
			broker:         BrokerInteractiveBroker,
			expectedType:   "*commission_fee.InteractiveBrokerCommissionFee",
			testQuantity:   1000,
			testPrice:      100,
			expectedResult: 5.0,
		},
		{
			name:           "zero commission",
			broker:         BrokerZero,
			expectedType:   "*commission_fee.ZeroCommissionFee",
			testQuantity:   1000,
			testPrice:      100,
			expectedResult: 0.0,
		},
		{
			name:           "binance",
			broker:         BrokerBinance,
			expectedType:   "*commission_fee.BinanceCommissionFee",
			testQuantity:   1,
			testPrice:      30000,
			expectedResult: 30.0, // 0.001 * 1 * 30000
		},
		{
			name:           "unknown broker defaults to zero",
			broker:         Broker("unknown"),
			expectedType:   "*commission_fee.ZeroCommissionFee",
			testQuantity:   1000,
			testPrice:      100,
			expectedResult: 0.0,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			handler := GetCommissionFeeHandler(tc.broker)
			suite.NotNil(handler)
			result := handler.Calculate(tc.testQuantity, tc.testPrice)
			suite.InDelta(tc.expectedResult, result, 1e-9)
		})
	}
}

func (suite *CommissionFeeTestSuite) TestAllBrokers() {
	suite.Len(AllBrokers, 3)
	suite.Contains(AllBrokers, BrokerInteractiveBroker)
	suite.Contains(AllBrokers, BrokerZero)
	suite.Contains(AllBrokers, BrokerBinance)
}

func (suite *CommissionFeeTestSuite) TestBrokerConstants() {
	suite.Equal(Broker("interactive_broker"), BrokerInteractiveBroker)
	suite.Equal(Broker("zero_commission"), BrokerZero)
	suite.Equal(Broker("binance"), BrokerBinance)
}

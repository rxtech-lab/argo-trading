package engine

import (
	"testing"

	"github.com/sirily11/argo-trading-go/src/backtest/engine/engine_v1/commission_fee"
	"github.com/stretchr/testify/suite"
)

// UtilsTestSuite is a test suite for utils package
type UtilsTestSuite struct {
	suite.Suite
}

// TestUtilsSuite runs the test suite
func TestUtilsSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (suite *UtilsTestSuite) TestCalculateMaxQuantity() {
	tests := []struct {
		name          string
		balance       float64
		price         float64
		commissionFee commission_fee.CommissionFee
		expectedQty   int
		expectedError bool
	}{
		{
			name:          "Simple case with no commission",
			balance:       1000.0,
			price:         100.0,
			commissionFee: commission_fee.NewZeroCommissionFee(),
			expectedQty:   10,
			expectedError: false,
		},
		{
			name:          "Case with commission",
			balance:       1000.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   9,
			expectedError: false,
		},
		{
			name:          "Zero balance",
			balance:       0.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
		{
			name:          "Zero price",
			balance:       1000.0,
			price:         0.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
		{
			name:          "Balance less than price",
			balance:       50.0,
			price:         100.0,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   0,
			expectedError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			qty := CalculateMaxQuantity(tc.balance, tc.price, tc.commissionFee)
			suite.Assert().Equal(tc.expectedQty, qty, "Quantity mismatch")
		})
	}
}

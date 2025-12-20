package utils

import (
	"testing"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
	"github.com/stretchr/testify/suite"
)

type UtilsTestSuite struct {
	suite.Suite
}

func TestUtilsTestSuite(t *testing.T) {
	suite.Run(t, new(UtilsTestSuite))
}

func (suite *UtilsTestSuite) TestCalculateMaxQuantity() {
	tests := []struct {
		name          string
		balance       float64
		price         float64
		commissionFee commission_fee.CommissionFee
		expectedQty   float64
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
			expectedQty:   9.99,
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
			expectedQty:   0.49,
			expectedError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			qty := CalculateMaxQuantity(tc.balance, tc.price, tc.commissionFee)
			suite.Assert().Equal(float64(tc.expectedQty), qty, "TotalLongPositionQuantity mismatch")
		})
	}
}

func (suite *UtilsTestSuite) TestCalculateOrderQuantityByPercentage() {
	tests := []struct {
		name          string
		balance       float64
		price         float64
		percentage    float64
		commissionFee commission_fee.CommissionFee
		expectedQty   float64
		expectedError bool
	}{
		{
			name:          "Simple case with no commission",
			balance:       1000.0,
			price:         100.0,
			percentage:    0.5,
			commissionFee: commission_fee.NewZeroCommissionFee(),
			expectedQty:   5.0,
			expectedError: false,
		},
		{
			name:          "Full balance",
			balance:       1000.0,
			price:         100.0,
			percentage:    1.0,
			commissionFee: commission_fee.NewZeroCommissionFee(),
			expectedQty:   10.0,
			expectedError: false,
		},
		{
			name:          "Small percentage with commission",
			balance:       1000.0,
			price:         100.0,
			percentage:    0.25,
			commissionFee: &commission_fee.InteractiveBrokerCommissionFee{},
			expectedQty:   2.49,
			expectedError: false,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			qty := CalculateOrderQuantityByPercentage(tc.balance, tc.price, tc.commissionFee, tc.percentage)
			suite.Assert().InDelta(tc.expectedQty, qty, 0.0001, "TotalLongPositionQuantity mismatch")
		})
	}
}

func (suite *UtilsTestSuite) TestRoundToDecimalPrecision() {
	tests := []struct {
		name             string
		quantity         float64
		decimalPrecision int
		expected         float64
	}{
		{
			name:             "Round to 0 decimals",
			quantity:         10.5678,
			decimalPrecision: 0,
			expected:         10.0,
		},
		{
			name:             "Round to 1 decimal",
			quantity:         10.5678,
			decimalPrecision: 1,
			expected:         10.5,
		},
		{
			name:             "Round to 2 decimals",
			quantity:         10.5678,
			decimalPrecision: 2,
			expected:         10.56,
		},
		{
			name:             "Round to 3 decimals",
			quantity:         10.5678,
			decimalPrecision: 3,
			expected:         10.567,
		},
		{
			name:             "Round to 4 decimals",
			quantity:         10.5678,
			decimalPrecision: 4,
			expected:         10.5678,
		},
		{
			name:             "Whole number with precision",
			quantity:         100.0,
			decimalPrecision: 2,
			expected:         100.0,
		},
		{
			name:             "Very small number",
			quantity:         0.0012345,
			decimalPrecision: 4,
			expected:         0.0012,
		},
		{
			name:             "Zero quantity",
			quantity:         0.0,
			decimalPrecision: 2,
			expected:         0.0,
		},
		{
			name:             "Negative number",
			quantity:         -10.5678,
			decimalPrecision: 2,
			expected:         -10.57, // math.Floor on negative rounds towards negative infinity
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			result := RoundToDecimalPrecision(tc.quantity, tc.decimalPrecision)
			suite.Assert().Equal(tc.expected, result, "Rounding mismatch")
		})
	}
}

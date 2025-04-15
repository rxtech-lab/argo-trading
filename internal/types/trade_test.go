package types

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type TradeTestSuite struct {
	suite.Suite
}

func TestTradeSuite(t *testing.T) {
	suite.Run(t, new(TradeTestSuite))
}

func (suite *TradeTestSuite) TestGetAverageEntryPrice() {
	tests := []struct {
		name          string
		position      Position
		expectedPrice float64
		expectedError bool
	}{
		{
			name: "Valid position with fees",
			position: Position{
				TotalLongInPositionQuantity: 100,
				TotalLongInPositionAmount:   10000,
				TotalLongInFee:              10,
			},
			expectedPrice: 100.1, // (10000 + 10) / 100
			expectedError: false,
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalLongInPositionQuantity: 0,
				TotalLongInPositionAmount:   10000,
				TotalLongInFee:              10,
			},
			expectedPrice: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageLongPositionEntryPrice()
			suite.Equal(tt.expectedPrice, result)
		})
	}
}

func (suite *TradeTestSuite) TestGetAverageExitPrice() {
	tests := []struct {
		name          string
		position      Position
		expectedPrice float64
		expectedError bool
	}{
		{
			name: "Valid position with fees",
			position: Position{
				TotalLongOutPositionQuantity: 100,
				TotalLongOutPositionAmount:   11000,
				TotalLongOutFee:              10,
			},
			expectedPrice: 109.9, // (11000 - 10) / 100
			expectedError: false,
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalLongOutPositionQuantity: 0,
				TotalLongOutPositionAmount:   11000,
				TotalLongOutFee:              10,
			},
			expectedPrice: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageLongPositionExitPrice()
			suite.Equal(tt.expectedPrice, result)
		})
	}
}

func (suite *TradeTestSuite) TestGetTotalPnL() {
	tests := []struct {
		name          string
		position      Position
		expectedPnL   float64
		expectedError bool
	}{
		{
			name: "Profitable trade",
			position: Position{
				TotalLongInPositionQuantity:  100,
				TotalLongInPositionAmount:    10000,
				TotalLongInFee:               10,
				TotalLongOutPositionQuantity: 100,
				TotalLongOutPositionAmount:   11000,
				TotalLongOutFee:              10,
			},
			expectedPnL:   980, // (11000 - 10) - (10000 + 10)
			expectedError: false,
		},
		{
			name: "Zero in quantity",
			position: Position{
				TotalLongInPositionQuantity:  0,
				TotalLongInPositionAmount:    10000,
				TotalLongInFee:               10,
				TotalLongOutPositionQuantity: 100,
				TotalLongOutPositionAmount:   11000,
				TotalLongOutFee:              10,
			},
			expectedPnL:   0,
			expectedError: false,
		},
		{
			name: "Zero out quantity",
			position: Position{
				TotalLongInPositionQuantity:  100,
				TotalLongInPositionAmount:    10000,
				TotalLongInFee:               10,
				TotalLongOutPositionQuantity: 0,
				TotalLongOutPositionAmount:   11000,
				TotalLongOutFee:              10,
			},
			expectedPnL:   0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetTotalPnL()
			suite.Equal(tt.expectedPnL, result)
		})
	}
}

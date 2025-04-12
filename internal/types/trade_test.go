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
				TotalInQuantity: 100,
				TotalInAmount:   10000,
				TotalInFee:      10,
			},
			expectedPrice: 100.1, // (10000 + 10) / 100
			expectedError: false,
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalInQuantity: 0,
				TotalInAmount:   10000,
				TotalInFee:      10,
			},
			expectedPrice: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageEntryPrice()
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
				TotalOutQuantity: 100,
				TotalOutAmount:   11000,
				TotalOutFee:      10,
			},
			expectedPrice: 109.9, // (11000 - 10) / 100
			expectedError: false,
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalOutQuantity: 0,
				TotalOutAmount:   11000,
				TotalOutFee:      10,
			},
			expectedPrice: 0,
			expectedError: false,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageExitPrice()
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
				TotalInQuantity:  100,
				TotalInAmount:    10000,
				TotalInFee:       10,
				TotalOutQuantity: 100,
				TotalOutAmount:   11000,
				TotalOutFee:      10,
			},
			expectedPnL:   980, // (11000 - 10) - (10000 + 10)
			expectedError: false,
		},
		{
			name: "Zero in quantity",
			position: Position{
				TotalInQuantity:  0,
				TotalInAmount:    10000,
				TotalInFee:       10,
				TotalOutQuantity: 100,
				TotalOutAmount:   11000,
				TotalOutFee:      10,
			},
			expectedPnL:   0,
			expectedError: false,
		},
		{
			name: "Zero out quantity",
			position: Position{
				TotalInQuantity:  100,
				TotalInAmount:    10000,
				TotalInFee:       10,
				TotalOutQuantity: 0,
				TotalOutAmount:   11000,
				TotalOutFee:      10,
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

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

func (suite *TradeTestSuite) TestGetAverageShortPositionEntryPrice() {
	tests := []struct {
		name          string
		position      Position
		expectedPrice float64
	}{
		{
			name: "Valid short position with fees",
			position: Position{
				TotalShortInPositionQuantity: 100,
				TotalShortInPositionAmount:   10000,
				TotalShortInFee:              10,
			},
			expectedPrice: 99.9, // (10000 - 10) / 100
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalShortInPositionQuantity: 0,
				TotalShortInPositionAmount:   10000,
				TotalShortInFee:              10,
			},
			expectedPrice: 0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageShortPositionEntryPrice()
			suite.Equal(tt.expectedPrice, result)
		})
	}
}

func (suite *TradeTestSuite) TestGetAverageShortPositionExitPrice() {
	tests := []struct {
		name          string
		position      Position
		expectedPrice float64
	}{
		{
			name: "Valid short position with fees",
			position: Position{
				TotalShortOutPositionQuantity: 100,
				TotalShortOutPositionAmount:   9000,
				TotalShortOutFee:              10,
			},
			expectedPrice: 90.1, // (9000 + 10) / 100
		},
		{
			name: "Zero quantity",
			position: Position{
				TotalShortOutPositionQuantity: 0,
				TotalShortOutPositionAmount:   9000,
				TotalShortOutFee:              10,
			},
			expectedPrice: 0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetAverageShortPositionExitPrice()
			suite.Equal(tt.expectedPrice, result)
		})
	}
}

func (suite *TradeTestSuite) TestGetTotalShortPositionPnl() {
	tests := []struct {
		name        string
		position    Position
		expectedPnL float64
	}{
		{
			name: "Profitable short trade - price went down",
			position: Position{
				TotalShortInPositionQuantity:  100,
				TotalShortInPositionAmount:    10000, // Entry at $100
				TotalShortInFee:               10,
				TotalShortOutPositionQuantity: 100,
				TotalShortOutPositionAmount:   9000, // Exit at $90
				TotalShortOutFee:              10,
			},
			expectedPnL: 980, // (entry - exit) = (10000 - 10) - (9000 + 10) = 9990 - 9010 = 980
		},
		{
			name: "Zero in quantity",
			position: Position{
				TotalShortInPositionQuantity:  0,
				TotalShortInPositionAmount:    10000,
				TotalShortInFee:               10,
				TotalShortOutPositionQuantity: 100,
				TotalShortOutPositionAmount:   9000,
				TotalShortOutFee:              10,
			},
			expectedPnL: 0,
		},
		{
			name: "Zero out quantity",
			position: Position{
				TotalShortInPositionQuantity:  100,
				TotalShortInPositionAmount:    10000,
				TotalShortInFee:               10,
				TotalShortOutPositionQuantity: 0,
				TotalShortOutPositionAmount:   9000,
				TotalShortOutFee:              10,
			},
			expectedPnL: 0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetTotalShortPositionPnl()
			pnl, _ := result.Float64()
			suite.Equal(tt.expectedPnL, pnl)
		})
	}
}

func (suite *TradeTestSuite) TestGetTotalLongPositionPnl() {
	tests := []struct {
		name        string
		position    Position
		expectedPnL float64
	}{
		{
			name: "Profitable long trade - price went up",
			position: Position{
				TotalLongInPositionQuantity:  100,
				TotalLongInPositionAmount:    10000, // Entry at $100
				TotalLongInFee:               10,
				TotalLongOutPositionQuantity: 100,
				TotalLongOutPositionAmount:   11000, // Exit at $110
				TotalLongOutFee:              10,
			},
			expectedPnL: 980, // (exit - entry) = (11000 - 10) - (10000 + 10) = 10990 - 10010 = 980
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
			expectedPnL: 0,
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
			expectedPnL: 0,
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			result := tt.position.GetTotalLongPositionPnl()
			pnl, _ := result.Float64()
			suite.Equal(tt.expectedPnL, pnl)
		})
	}
}

func (suite *TradeTestSuite) TestGetTotalPnLCombined() {
	// Test combining long and short PnL
	position := Position{
		// Long position: profit of 980
		TotalLongInPositionQuantity:  100,
		TotalLongInPositionAmount:    10000,
		TotalLongInFee:               10,
		TotalLongOutPositionQuantity: 100,
		TotalLongOutPositionAmount:   11000,
		TotalLongOutFee:              10,
		// Short position: profit of 980
		TotalShortInPositionQuantity:  100,
		TotalShortInPositionAmount:    10000,
		TotalShortInFee:               10,
		TotalShortOutPositionQuantity: 100,
		TotalShortOutPositionAmount:   9000,
		TotalShortOutFee:              10,
	}

	result := position.GetTotalPnL()
	suite.Equal(1960.0, result) // 980 + 980
}

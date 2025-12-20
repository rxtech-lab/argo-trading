package utils

import (
	"math"

	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"
)

// CalculateMaxQuantity calculates the maximum quantity that can be bought with the given balance and respecting decimal precision.
func CalculateMaxQuantity(balance float64, price float64, commissionFee commission_fee.CommissionFee) float64 {
	// Handle edge cases
	if price <= 0 || balance <= 0 {
		return 0
	}

	// Initial rough estimate (ignoring fees)
	maxQty := balance / price

	// Iteratively refine by accounting for fees
	for i := 0; i < 10; i++ { // Usually converges quickly, limit iterations
		totalCost := maxQty*price + commissionFee.Calculate(maxQty)
		if totalCost <= balance {
			break
		}
		// Adjust quantity down proportionally
		adjustment := balance / totalCost
		maxQty = maxQty * adjustment
	}

	return maxQty
}

// RoundToDecimalPrecision rounds the quantity to the specified decimal precision.
func RoundToDecimalPrecision(quantity float64, decimalPrecision int) float64 {
	multiplier := math.Pow10(decimalPrecision)

	return math.Floor(quantity*multiplier) / multiplier
}

// CalculateOrderQuantityByPercentage calculates the quantity of an order by the given percentage of the balance.
func CalculateOrderQuantityByPercentage(balance float64, price float64, commissionFee commission_fee.CommissionFee, percentage float64) float64 {
	quantity := balance * percentage

	return CalculateMaxQuantity(quantity, price, commissionFee)
}

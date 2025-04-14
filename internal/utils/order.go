package utils

import "github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/commission_fee"

// Calculate the maximum quantity that can be bought with the given balance. Returns the quantity in integer.
func CalculateMaxQuantity(balance float64, price float64, commissionFee commission_fee.CommissionFee) int {
	// Handle edge cases
	if price <= 0 || balance <= 0 {
		return 0
	}

	// Calculate the maximum quantity that can be bought with the given balance
	// We need to account for both the price and commission fee
	// We use binary search to find the maximum quantity that fits within the balance
	// The total cost should be: quantity * price + commissionFee.Calculate(quantity) <= balance
	left := 0
	right := int(balance / price) // Upper bound is balance/price
	maxQty := 0

	for left <= right {
		mid := (left + right) / 2
		totalCost := float64(mid)*price + commissionFee.Calculate(float64(mid))

		if totalCost <= balance {
			maxQty = mid
			left = mid + 1
		} else {
			right = mid - 1
		}
	}

	return maxQty
}

// Calculate the quantity of an order by the given percentage of the balance.
func CalculateOrderQuantityByPercentage(balance float64, price float64, commissionFee commission_fee.CommissionFee, percentage float64) int {
	quantity := balance * percentage

	return CalculateMaxQuantity(quantity, price, commissionFee)
}

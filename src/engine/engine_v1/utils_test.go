package engine

import (
	"testing"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/assert"
)

func TestCommissionFormula(t *testing.T) {
	// Create a new backtest engine
	backtest := NewBacktestEngineV1()

	// Initialize with a config that includes our commission formula
	startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	endTime := time.Date(2025, 2, 2, 23, 59, 59, 0, time.UTC)
	formula := `orderType == "BUY" ? max(1.0, 0.001 * total) : max(1.5, 0.0015 * total)`
	err := backtest.TestSetConfig(10000, 10000, "results", startTime, endTime, formula)
	assert.NoError(t, err, "Failed to configure test")

	// Define test cases
	testCases := []struct {
		name     string
		order    types.Order
		price    float64
		expected float64
	}{
		{
			name: "Small buy order (should use minimum commission)",
			order: types.Order{
				Symbol:       "AAPL",
				OrderType:    types.OrderTypeBuy,
				Quantity:     5,
				Price:        150,
				Timestamp:    time.Now(),
				OrderID:      "test-order-1",
				IsCompleted:  false,
				StrategyName: "test-strategy",
			},
			price:    150,
			expected: 1.0, // Minimum $1.0 commission
		},
		{
			name: "Large buy order",
			order: types.Order{
				Symbol:       "AAPL",
				OrderType:    types.OrderTypeBuy,
				Quantity:     100,
				Price:        150,
				Timestamp:    time.Now(),
				OrderID:      "test-order-2",
				IsCompleted:  false,
				StrategyName: "test-strategy",
			},
			price:    150,
			expected: 15.0, // 0.1% of $15,000 = $15.0
		},
		{
			name: "Small sell order (should use minimum commission)",
			order: types.Order{
				Symbol:       "AAPL",
				OrderType:    types.OrderTypeSell,
				Quantity:     5,
				Price:        150,
				Timestamp:    time.Now(),
				OrderID:      "test-order-3",
				IsCompleted:  false,
				StrategyName: "test-strategy",
			},
			price:    150,
			expected: 1.5, // Minimum $1.5 commission for sell orders
		},
		{
			name: "Large sell order",
			order: types.Order{
				Symbol:       "AAPL",
				OrderType:    types.OrderTypeSell,
				Quantity:     100,
				Price:        150,
				Timestamp:    time.Now(),
				OrderID:      "test-order-4",
				IsCompleted:  false,
				StrategyName: "test-strategy",
			},
			price:    150,
			expected: 22.5, // 0.15% of $15,000 = $22.5
		},
	}

	// Run tests
	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			commission, err := backtest.TestCalculateCommission(tc.order, tc.price)
			assert.NoError(t, err, "Failed to calculate commission")
			assert.Equal(t, tc.expected, commission, "Commission calculation mismatch")
		})
	}
}

func TestCommissionFormulaExpressions(t *testing.T) {
	// Test different formula expressions
	testFormulas := []struct {
		name     string
		formula  string
		order    types.Order
		price    float64
		expected float64
	}{
		{
			name:    "Percentage-based commission",
			formula: "0.001 * total",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  100,
				Price:     150,
			},
			price:    150,
			expected: 15.0, // 0.1% of $15,000 = $15.0
		},
		{
			name:    "Fixed fee plus percentage",
			formula: "0.0005 * total + 5",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  100,
				Price:     150,
			},
			price:    150,
			expected: 12.5, // 0.05% of $15,000 + $5 = $12.5
		},
		{
			name:    "Tiered commission structure",
			formula: "total < 1000 ? 0.002 * total : 0.001 * total",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  5,
				Price:     150,
			},
			price:    150,
			expected: 1.5, // 0.2% of $750 = $1.5
		},
		{
			name:    "Minimum commission",
			formula: "max(1.0, 0.001 * total)",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  5,
				Price:     150,
			},
			price:    150,
			expected: 1.0, // max(1.0, 0.001 * 750) = max(1.0, 0.75) = 1.0
		},
		{
			name:    "Symbol-specific fees",
			formula: `symbol == "AAPL" ? 0.0005 * total : 0.001 * total`,
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  100,
				Price:     150,
			},
			price:    150,
			expected: 7.5, // 0.05% of $15,000 = $7.5
		},
		{
			name:    "Complex tiered structure",
			formula: "total < 500 ? 0.003 * total : (total < 1000 ? 0.002 * total : 0.001 * total)",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  4,
				Price:     100,
			},
			price:    100,
			expected: 1.2, // 0.3% of $400 = $1.2
		},
		{
			name:    "Volume discount",
			formula: "quantity > 100 ? 0.0008 * total : 0.001 * total",
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  200,
				Price:     150,
			},
			price:    150,
			expected: 24.0, // 0.08% of $30,000 = $24.0
		},
		{
			name:    "Combined conditions",
			formula: `(orderType == "BUY" && total > 1000) ? 0.0008 * total : 0.001 * total`,
			order: types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  100,
				Price:     150,
			},
			price:    150,
			expected: 12.0, // 0.08% of $15,000 = $12.0
		},
	}

	// Run tests for each formula
	for _, tf := range testFormulas {
		t.Run(tf.name, func(t *testing.T) {
			// Create a new backtest engine for each test
			backtest := NewBacktestEngineV1()

			// Initialize with the test formula using TestSetConfig
			startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			endTime := time.Date(2025, 2, 2, 23, 59, 59, 0, time.UTC)
			err := backtest.TestSetConfig(10000, 10000, "results", startTime, endTime, tf.formula)
			assert.NoError(t, err, "Failed to configure test")

			// Calculate commission
			commission, err := backtest.TestCalculateCommission(tf.order, tf.price)
			assert.NoError(t, err, "Failed to calculate commission")
			assert.Equal(t, tf.expected, commission, "Commission calculation mismatch")
		})
	}
}

func TestInvalidFormulas(t *testing.T) {
	// Test invalid formulas
	testFormulas := []struct {
		name    string
		formula string
	}{
		{
			name:    "Syntax error",
			formula: "0.001 * total +",
		},
		{
			name:    "Unknown variable",
			formula: "0.001 * unknown_var",
		},
		{
			name:    "Invalid operation",
			formula: `"string" * total`,
		},
	}

	// Run tests for each invalid formula
	for _, tf := range testFormulas {
		t.Run(tf.name, func(t *testing.T) {
			// Create a new backtest engine for each test
			backtest := NewBacktestEngineV1()

			// Initialize with the test formula
			startTime := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
			endTime := time.Date(2025, 2, 2, 23, 59, 59, 0, time.UTC)
			err := backtest.TestSetConfig(10000, 10000, "results", startTime, endTime, tf.formula)
			assert.NoError(t, err, "Failed to configure test")

			// Calculate commission - should fail
			order := types.Order{
				Symbol:    "AAPL",
				OrderType: types.OrderTypeBuy,
				Quantity:  100,
				Price:     150,
			}
			_, err = backtest.TestCalculateCommission(order, 150)
			assert.Error(t, err, "Expected error for invalid formula")
		})
	}
}

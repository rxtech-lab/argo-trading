package engine

import (
	"fmt"
	"math"

	"github.com/expr-lang/expr"
	"github.com/sirily11/argo-trading-go/src/types"
)

func calculateCommissionWithExpression(formula string, order types.Order, executionPrice float64) (float64, error) {
	// Create an environment with variables and functions
	env := map[string]interface{}{
		"quantity":  order.Quantity,
		"price":     executionPrice,
		"total":     order.Quantity * executionPrice,
		"symbol":    order.Symbol,
		"orderType": string(order.OrderType),
		"max":       math.Max,
		"min":       math.Min,
		"abs":       math.Abs,
		"sqrt":      math.Sqrt,
		"pow":       math.Pow,
		"ceil":      math.Ceil,
		"floor":     math.Floor,
		"round":     math.Round,
	}

	// Compile the expression
	program, err := expr.Compile(formula, expr.Env(env), expr.AllowUndefinedVariables())
	if err != nil {
		return 0, fmt.Errorf("invalid commission formula: %w", err)
	}

	// Run the expression
	result, err := expr.Run(program, env)
	if err != nil {
		return 0, fmt.Errorf("error evaluating commission formula: %w", err)
	}

	// Convert the result to float64
	var commission float64
	switch v := result.(type) {
	case float64:
		commission = v
	case float32:
		commission = float64(v)
	case int:
		commission = float64(v)
	case int64:
		commission = float64(v)
	default:
		return 0, fmt.Errorf("commission formula did not evaluate to a number, got %T", result)
	}

	return commission, nil
}

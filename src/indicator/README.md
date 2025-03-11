# Technical Indicator System

This package provides a flexible and extensible technical indicator system for the Argo Trading platform. It allows strategies to use various technical indicators for market analysis and decision making.

## Overview

The indicator system consists of the following components:

1. **Indicator Interface**: Defines the contract that all indicators must implement
2. **IndicatorContext**: Provides data access methods for indicators
3. **Built-in Indicators**: Pre-implemented indicators like RSI, MACD, etc.

## Using Indicators in Strategies

Strategies can access indicators through the `StrategyContext` interface:

```go
// In your strategy's ProcessData method
rsiResult, err := ctx.GetIndicator("RSI")
if err != nil {
    return nil, fmt.Errorf("failed to get RSI indicator: %w", err)
}

// Cast to the expected type
rsiValues, ok := rsiResult.([]float64)
if !ok || len(rsiValues) == 0 {
    return nil, fmt.Errorf("invalid RSI result format")
}

// Get the latest RSI value
currentRSI := rsiValues[len(rsiValues)-1]

// Make trading decisions based on the indicator value
if currentRSI < 30 {
    // Oversold condition, consider buying
    // ...
} else if currentRSI > 70 {
    // Overbought condition, consider selling
    // ...
}
```

## Implementing Custom Indicators

You can create your own indicators by implementing the `Indicator` interface:

```go
type MyCustomIndicator struct {
    // Your indicator parameters
    period int
    params map[string]interface{}
}

func NewMyCustomIndicator(period int) *MyCustomIndicator {
    return &MyCustomIndicator{
        period: period,
        params: map[string]interface{}{
            "period": period,
        },
    }
}

// Name returns the name of the indicator
func (i *MyCustomIndicator) Name() string {
    return "MyCustomIndicator"
}

// SetParams allows setting parameters for the indicator
func (i *MyCustomIndicator) SetParams(params map[string]interface{}) error {
    // Implement parameter setting logic
    return nil
}

// GetParams returns the current parameters of the indicator
func (i *MyCustomIndicator) GetParams() map[string]interface{} {
    return i.params
}

// Calculate computes the indicator value using the provided context
func (i *MyCustomIndicator) Calculate(ctx indicator.IndicatorContext) (interface{}, error) {
    // Get data from context
    data := ctx.GetData()

    // Or get data for a specific time range
    // startTime := time.Now().Add(-30 * 24 * time.Hour) // 30 days ago
    // endTime := time.Now()
    // data := ctx.GetDataForTimeRange(startTime, endTime)

    // Implement your calculation logic
    // ...
    return result, nil
}
```

Then update the GetIndicator method in the strategyContext to include your new indicator:

```go
func (c strategyContext) GetIndicator(name string) (interface{}, error) {
    var ind indicator.Indicator

    switch name {
    case "RSI":
        ind = indicator.NewRSI(14)
    case "MACD":
        ind = indicator.NewMACD(12, 26, 9)
    case "MyCustomIndicator":
        ind = indicator.NewMyCustomIndicator(10)
    default:
        return nil, fmt.Errorf("indicator %s not found", name)
    }

    // Create indicator context and calculate
    ctx := indicator.NewIndicatorContext(c.engine.historicalData)
    return ind.Calculate(ctx)
}
```

## Available Indicators

The following indicators are available out of the box:

1. **RSI (Relative Strength Index)**: Measures the speed and change of price movements

   - Parameters: `period` (default: 14)
   - Return type: `[]float64`

2. **MACD (Moving Average Convergence Divergence)**: Trend-following momentum indicator
   - Parameters: `fastPeriod` (default: 12), `slowPeriod` (default: 26), `signalPeriod` (default: 9)
   - Return type: `indicator.MACDResult` containing MACD line, signal line, and histogram

## Extending the System

To add more indicators:

1. Create a new file in the `indicator` package
2. Implement the `Indicator` interface
3. Update the `GetIndicator` method in the strategyContext to include your new indicator

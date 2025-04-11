package main

import (
	"context"
	"fmt"

	"github.com/sirily11/argo-trading-go/pkg/strategy"
)

// RSIConfig contains the configuration for the RSI strategy
type RSIConfig struct {
	Period     int     `json:"period"`
	Overbought float64 `json:"overbought"`
	Oversold   float64 `json:"oversold"`
}

// RSIStrategy implements the TradingStrategy interface
type RSIStrategy struct {
	config RSIConfig
	prices []float64
}

// Initialize sets up the strategy with the given configuration
func (s *RSIStrategy) Initialize(ctx context.Context, req *strategy.InitializeRequest) (*strategy.InitializeResponse, error) {
	// Parse the configuration
	// In a real implementation, you would use a JSON parser here
	s.config = RSIConfig{
		Period:     14,
		Overbought: 70,
		Oversold:   30,
	}
	s.prices = make([]float64, 0)
	return &strategy.InitializeResponse{Success: true}, nil
}

// ProcessData processes new market data and generates signals
func (s *RSIStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*strategy.ProcessDataResponse, error) {
	// Add the new price to our price history
	s.prices = append(s.prices, req.Data.Price)

	// If we don't have enough data yet, return
	if len(s.prices) < s.config.Period {
		return &strategy.ProcessDataResponse{Success: true}, nil
	}

	// Calculate RSI
	rsi := s.calculateRSI()

	// Generate signals based on RSI
	if rsi > s.config.Overbought {
		// Overbought signal
		_, err := req.Context.HostFunctions.MarkSignal(ctx, &strategy.SignalRequest{
			Symbol:     req.Data.Symbol,
			SignalType: "SELL",
			Reason:     fmt.Sprintf("RSI %f is above overbought threshold %f", rsi, s.config.Overbought),
			Timestamp:  req.Data.Timestamp,
		})
		if err != nil {
			return &strategy.ProcessDataResponse{Success: false, Error: err.Error()}, nil
		}
	} else if rsi < s.config.Oversold {
		// Oversold signal
		_, err := req.Context.HostFunctions.MarkSignal(ctx, &strategy.SignalRequest{
			Symbol:     req.Data.Symbol,
			SignalType: "BUY",
			Reason:     fmt.Sprintf("RSI %f is below oversold threshold %f", rsi, s.config.Oversold),
			Timestamp:  req.Data.Timestamp,
		})
		if err != nil {
			return &strategy.ProcessDataResponse{Success: false, Error: err.Error()}, nil
		}
	}

	return &strategy.ProcessDataResponse{Success: true}, nil
}

// Name returns the name of the strategy
func (s *RSIStrategy) Name(ctx context.Context, req *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "RSI Strategy"}, nil
}

// calculateRSI calculates the Relative Strength Index
func (s *RSIStrategy) calculateRSI() float64 {
	if len(s.prices) < s.config.Period {
		return 0
	}

	// Calculate price changes
	gains := make([]float64, 0)
	losses := make([]float64, 0)

	for i := 1; i < len(s.prices); i++ {
		change := s.prices[i] - s.prices[i-1]
		if change >= 0 {
			gains = append(gains, change)
			losses = append(losses, 0)
		} else {
			gains = append(gains, 0)
			losses = append(losses, -change)
		}
	}

	// Calculate average gain and loss
	avgGain := average(gains[:s.config.Period])
	avgLoss := average(losses[:s.config.Period])

	// Calculate RS and RSI
	rs := avgGain / avgLoss
	rsi := 100 - (100 / (1 + rs))

	return rsi
}

// average calculates the average of a float64 slice
func average(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func main() {
	// This is required for the plugin to work
	// The plugin system will call the exported functions
}

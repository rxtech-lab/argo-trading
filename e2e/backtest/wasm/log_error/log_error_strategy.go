//go:build wasip1

package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/knqyf263/go-plugin/types/known/emptypb"
	"github.com/rxtech-lab/argo-trading/pkg/strategy"
)

// LogErrorStrategy implements a strategy that logs messages at different levels
// and throws an error to test error marker functionality
type LogErrorStrategy struct {
	config Config
}

// Config represents the configuration for the LogErrorStrategy
type Config struct {
	Symbol string `yaml:"symbol" jsonschema:"title=Symbol,description=The symbol to trade,default=AAPL"`
}

func main() {}

func init() {
	strategy.RegisterTradingStrategy(NewLogErrorStrategy())
}

func NewLogErrorStrategy() strategy.TradingStrategy {
	return &LogErrorStrategy{}
}

// Initialize implements strategy.TradingStrategy.
func (s *LogErrorStrategy) Initialize(_ context.Context, req *strategy.InitializeRequest) (*emptypb.Empty, error) {
	var config Config
	// unmarshal json to config
	if err := json.Unmarshal([]byte(req.Config), &config); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %w", err)
	}

	s.config = config
	return &emptypb.Empty{}, nil
}

// Name implements strategy.TradingStrategy.
func (s *LogErrorStrategy) Name(_ context.Context, _ *strategy.NameRequest) (*strategy.NameResponse, error) {
	return &strategy.NameResponse{Name: "LogErrorStrategy"}, nil
}

// GetDescription implements strategy.TradingStrategy.
func (s *LogErrorStrategy) GetDescription(_ context.Context, _ *strategy.GetDescriptionRequest) (*strategy.GetDescriptionResponse, error) {
	return &strategy.GetDescriptionResponse{Description: "A strategy that logs messages and throws errors for testing"}, nil
}

// ProcessData implements strategy.TradingStrategy.
// Logs at different levels and throws an error on a specific data point
func (s *LogErrorStrategy) ProcessData(ctx context.Context, req *strategy.ProcessDataRequest) (*emptypb.Empty, error) {
	// Get API for interacting with the host system
	api := strategy.NewStrategyApi()

	// Get the process count from cache to track which data point we're on
	key := "log_error_process_count"
	cache, err := api.GetCache(ctx, &strategy.GetRequest{Key: key})
	if err != nil {
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	// Parse count from cache (default to 0)
	count := 0
	if cache.Value != "" {
		fmt.Sscanf(cache.Value, "%d", &count)
	}

	// Log at different levels based on count
	switch count {
	case 0:
		_, err = api.Log(ctx, &strategy.LogRequest{
			Message: "Processing first data point",
			Level:   strategy.LogLevel_LOG_LEVEL_INFO,
			Fields:  map[string]string{"count": "0", "action": "start"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to log: %w", err)
		}
	case 1:
		_, err = api.Log(ctx, &strategy.LogRequest{
			Message: "Debug log message",
			Level:   strategy.LogLevel_LOG_LEVEL_DEBUG,
			Fields:  map[string]string{"count": "1"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to log: %w", err)
		}
	case 2:
		_, err = api.Log(ctx, &strategy.LogRequest{
			Message: "Warning before error",
			Level:   strategy.LogLevel_LOG_LEVEL_WARN,
			Fields:  map[string]string{"count": "2"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to log: %w", err)
		}
	case 3:
		// Increment count before returning error so next call continues
		_, err = api.SetCache(ctx, &strategy.SetRequest{
			Key:   key,
			Value: fmt.Sprintf("%d", count+1),
		})
		if err != nil {
			return nil, fmt.Errorf("failed to set cache: %w", err)
		}
		// Return an error to trigger error marker
		return nil, errors.New("simulated strategy error for testing")
	case 4:
		_, err = api.Log(ctx, &strategy.LogRequest{
			Message: "Recovered after error",
			Level:   strategy.LogLevel_LOG_LEVEL_INFO,
			Fields:  map[string]string{"count": "4", "status": "recovered"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to log: %w", err)
		}
	case 5:
		_, err = api.Log(ctx, &strategy.LogRequest{
			Message: "Error level log",
			Level:   strategy.LogLevel_LOG_LEVEL_ERROR,
			Fields:  map[string]string{"count": "5"},
		})
		if err != nil {
			return nil, fmt.Errorf("failed to log: %w", err)
		}
	}

	// Update count in cache
	_, err = api.SetCache(ctx, &strategy.SetRequest{
		Key:   key,
		Value: fmt.Sprintf("%d", count+1),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to set cache: %w", err)
	}

	return &emptypb.Empty{}, nil
}

// GetConfigSchema implements strategy.TradingStrategy.
func (s *LogErrorStrategy) GetConfigSchema(_ context.Context, _ *strategy.GetConfigSchemaRequest) (*strategy.GetConfigSchemaResponse, error) {
	schema, err := strategy.ToJSONSchema(Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to get schema: %w", err)
	}
	return &strategy.GetConfigSchemaResponse{Schema: schema}, nil
}

// GetIdentifier implements strategy.TradingStrategy.
func (s *LogErrorStrategy) GetIdentifier(_ context.Context, _ *strategy.GetIdentifierRequest) (*strategy.GetIdentifierResponse, error) {
	return &strategy.GetIdentifierResponse{
		Identifier: "com.argo-trading.e2e.log-error",
	}, nil
}

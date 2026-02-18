package testhelper

import (
	"context"
	"fmt"
	"iter"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	backtestTesthelper "github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/provider"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
)

// MockMarketDataConfig holds the configuration for generating mock market data for streaming.
type MockMarketDataConfig struct {
	// Symbol is the ticker symbol for this configuration
	Symbol string

	// Pattern determines price movement (increasing, decreasing, volatile)
	Pattern backtestTesthelper.SimulationPattern

	// InitialPrice is the starting price
	InitialPrice float64

	// NumDataPoints is the total number of data points to generate
	NumDataPoints int

	// TrendStrength controls how strong the trend is (0.0-1.0)
	TrendStrength float64

	// VolatilityPercent controls price volatility
	VolatilityPercent float64

	// MaxDrawdownPercent limits drawdown for volatile pattern
	MaxDrawdownPercent float64

	// Seed for reproducible random generation
	Seed int64

	// ErrorAfterN injects an error after N data points (0 = no error)
	ErrorAfterN int

	// ErrorToReturn is the error to inject
	ErrorToReturn error

	// Interval is the time interval between data points (default: 1 minute)
	Interval time.Duration

	// StartTime is the starting time for data generation (default: now)
	StartTime time.Time
}

// MockMarketDataProvider implements provider.Provider for testing.
// It generates mock market data using the existing MockDataGenerator
// and streams it via the iter.Seq2 pattern.
type MockMarketDataProvider struct {
	configs  map[string]MockMarketDataConfig
	symbols  []string
	interval string
}

// NewMockMarketDataProvider creates a new mock market data provider with the given configurations.
// Each configuration specifies how data should be generated for a specific symbol.
func NewMockMarketDataProvider(configs ...MockMarketDataConfig) *MockMarketDataProvider {
	p := &MockMarketDataProvider{
		configs:  make(map[string]MockMarketDataConfig),
		symbols:  make([]string, 0, len(configs)),
		interval: "1m",
	}

	for _, c := range configs {
		// Set defaults
		if c.Interval == 0 {
			c.Interval = time.Minute
		}

		if c.StartTime.IsZero() {
			c.StartTime = time.Now()
		}

		p.configs[c.Symbol] = c
		p.symbols = append(p.symbols, c.Symbol)
	}

	return p
}

// ConfigWriter implements provider.Provider.
// This is a no-op for mock provider since we don't write to files.
func (p *MockMarketDataProvider) ConfigWriter(_ writer.MarketDataWriter) {
	// No-op for mock provider
}

// Download implements provider.Provider.
// This is not supported for mock provider since we only do streaming.
func (p *MockMarketDataProvider) Download(
	_ context.Context,
	_ string,
	_ time.Time,
	_ time.Time,
	_ int,
	_ models.Timespan,
	_ provider.OnDownloadProgress,
) (string, error) {
	return "", fmt.Errorf("download not supported in mock provider")
}

// Stream implements provider.Provider.
// Yields generated market data as fast as possible for quick test execution.
// Data is generated using the MockDataGenerator from backtest testhelper.
func (p *MockMarketDataProvider) Stream(ctx context.Context) iter.Seq2[types.MarketData, error] {
	return func(yield func(types.MarketData, error) bool) {
		// Generate data for each symbol
		for _, symbol := range p.symbols {
			config, ok := p.configs[symbol]
			if !ok {
				yield(types.MarketData{}, fmt.Errorf("no config for symbol: %s", symbol)) //nolint:exhaustruct // error case
				return
			}

			// Create MockDataGenerator configuration
			generatorConfig := backtestTesthelper.MockDataConfig{
				Symbol:             config.Symbol,
				StartTime:          config.StartTime,
				EndTime:            time.Time{}, // Not used when NumDataPoints is set
				Interval:           config.Interval,
				NumDataPoints:      config.NumDataPoints,
				Pattern:            config.Pattern,
				InitialPrice:       config.InitialPrice,
				MaxDrawdownPercent: config.MaxDrawdownPercent,
				VolatilityPercent:  config.VolatilityPercent,
				TrendStrength:      config.TrendStrength,
				Seed:               config.Seed,
			}

			generator := backtestTesthelper.NewMockDataGenerator(generatorConfig)

			data, err := generator.Generate()
			if err != nil {
				yield(types.MarketData{}, err) //nolint:exhaustruct // error case
				return
			}

			for i, d := range data {
				// Check for context cancellation
				select {
				case <-ctx.Done():
					return
				default:
				}

				// Inject error if configured
				if config.ErrorAfterN > 0 && i >= config.ErrorAfterN {
					yield(types.MarketData{}, config.ErrorToReturn) //nolint:exhaustruct // error case
					return
				}

				if !yield(d, nil) {
					return
				}
			}
		}
	}
}

// GetSymbols implements provider.Provider.
func (p *MockMarketDataProvider) GetSymbols() []string {
	return p.symbols
}

// GetInterval implements provider.Provider.
func (p *MockMarketDataProvider) GetInterval() string {
	return p.interval
}

// SetOnStatusChange implements provider.Provider.
// This is a no-op for mock provider since it's always connected.
func (p *MockMarketDataProvider) SetOnStatusChange(_ provider.OnStatusChange) {
	// No-op for mock provider
}

// Verify MockMarketDataProvider implements provider.Provider interface.
var _ provider.Provider = (*MockMarketDataProvider)(nil)

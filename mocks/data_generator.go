package mocks

import (
	"math"
	"math/rand"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
)

// DataGenerator generates realistic market data for testing and benchmarking.
type DataGenerator struct {
	rng *rand.Rand
}

// NewDataGenerator creates a new DataGenerator with the given seed.
// Use a fixed seed for reproducible results in tests.
func NewDataGenerator(seed int64) *DataGenerator {
	return &DataGenerator{
		rng: rand.New(rand.NewSource(seed)),
	}
}

// GeneratorConfig configures how market data is generated.
type GeneratorConfig struct {
	// Symbol is the trading symbol (e.g., "AAPL", "SPY")
	Symbol string
	// StartTime is the beginning of the data series
	StartTime time.Time
	// Interval is the duration between each bar
	Interval time.Duration
	// Count is the number of data points to generate
	Count int
	// InitialPrice is the starting price
	InitialPrice float64
	// Volatility controls price movement (0.01 = 1% typical daily volatility)
	Volatility float64
	// Trend is the drift factor (-0.01 to 0.01 for bearish to bullish)
	Trend float64
	// VolumeBase is the average volume per bar
	VolumeBase float64
	// VolumeVariance is the variance in volume (0.0 to 1.0)
	VolumeVariance float64
}

// DefaultConfig returns a sensible default configuration.
func DefaultConfig() GeneratorConfig {
	return GeneratorConfig{
		Symbol:         "TEST",
		StartTime:      time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC),
		Interval:       time.Minute,
		Count:          10000,
		InitialPrice:   100.0,
		Volatility:     0.002, // 0.2% per bar
		Trend:          0.0,   // neutral
		VolumeBase:     10000,
		VolumeVariance: 0.3,
	}
}

// Generate creates a slice of MarketData based on the configuration.
// The generated data follows a geometric Brownian motion model for realistic price movements.
func (g *DataGenerator) Generate(config GeneratorConfig) []types.MarketData {
	data := make([]types.MarketData, config.Count)
	currentPrice := config.InitialPrice
	currentTime := config.StartTime

	for i := 0; i < config.Count; i++ {
		// Generate OHLCV using geometric Brownian motion
		open := currentPrice

		// Generate intra-bar price movements
		// Using Box-Muller transform for normal distribution
		u1 := g.rng.Float64()
		u2 := g.rng.Float64()
		z := math.Sqrt(-2*math.Log(u1)) * math.Cos(2*math.Pi*u2)

		// Price change with trend and volatility
		priceChange := config.Volatility * z
		drift := config.Trend / float64(config.Count) // Distribute trend across bars

		close := open * (1 + priceChange + drift)
		if close <= 0 {
			close = open * 0.99 // Prevent negative prices
		}

		// High and low are within the open-close range plus some extension
		highExtension := math.Abs(g.rng.Float64() * config.Volatility * open * 0.5)
		lowExtension := math.Abs(g.rng.Float64() * config.Volatility * open * 0.5)

		high := math.Max(open, close) + highExtension
		low := math.Min(open, close) - lowExtension
		if low <= 0 {
			low = math.Min(open, close) * 0.99
		}

		// Volume with variance
		volumeVariation := 1.0 + (g.rng.Float64()*2-1)*config.VolumeVariance
		volume := config.VolumeBase * volumeVariation
		if volume < 0 {
			volume = config.VolumeBase * 0.1
		}

		data[i] = types.MarketData{
			Id:     "",
			Symbol: config.Symbol,
			Time:   currentTime,
			Open:   roundToDecimals(open, 4),
			High:   roundToDecimals(high, 4),
			Low:    roundToDecimals(low, 4),
			Close:  roundToDecimals(close, 4),
			Volume: roundToDecimals(volume, 2),
		}

		// Update for next iteration
		currentPrice = close
		currentTime = currentTime.Add(config.Interval)
	}

	return data
}

// GenerateMultiSymbol generates data for multiple symbols.
func (g *DataGenerator) GenerateMultiSymbol(symbols []string, baseConfig GeneratorConfig) []types.MarketData {
	var allData []types.MarketData

	for _, symbol := range symbols {
		config := baseConfig
		config.Symbol = symbol
		// Vary initial price and volatility slightly per symbol
		config.InitialPrice = baseConfig.InitialPrice * (0.8 + g.rng.Float64()*0.4)
		config.Volatility = baseConfig.Volatility * (0.8 + g.rng.Float64()*0.4)

		symbolData := g.Generate(config)
		allData = append(allData, symbolData...)
	}

	return allData
}

// Generate10K is a convenience function to generate 10,000 data points
// with default settings for benchmarking.
func Generate10K(symbol string) []types.MarketData {
	gen := NewDataGenerator(42) // Fixed seed for reproducibility
	config := DefaultConfig()
	config.Symbol = symbol
	config.Count = 10000
	return gen.Generate(config)
}

// Generate10KMultiSymbol generates 10,000 data points for each symbol.
func Generate10KMultiSymbol(symbols []string) []types.MarketData {
	gen := NewDataGenerator(42)
	config := DefaultConfig()
	config.Count = 10000
	return gen.GenerateMultiSymbol(symbols, config)
}

// roundToDecimals rounds a float64 to the specified number of decimal places.
func roundToDecimals(val float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(val*pow) / pow
}

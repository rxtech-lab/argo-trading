package testhelper

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/types"
)

// SimulationPattern defines the type of price simulation pattern
type SimulationPattern string

const (
	// PatternIncreasing simulates a continuously increasing price trend
	PatternIncreasing SimulationPattern = "increasing"
	// PatternDecreasing simulates a continuously decreasing price trend
	PatternDecreasing SimulationPattern = "decreasing"
	// PatternVolatile simulates a volatile price with maximum drawdown constraint
	PatternVolatile SimulationPattern = "volatile"
)

// Default configuration constants
const (
	// DefaultMinimumPrice is the minimum price floor to prevent negative or zero prices
	DefaultMinimumPrice = 0.01
	// DefaultBaseVolume is the base volume for generating random volume data
	DefaultBaseVolume = 1000000.0
	// DefaultIncreasingNoiseBias is the bias factor for noise in increasing pattern (0.3 means slightly positive bias)
	DefaultIncreasingNoiseBias = 0.3
	// DefaultDecreasingNoiseBias is the bias factor for noise in decreasing pattern (0.7 means slightly negative bias)
	DefaultDecreasingNoiseBias = 0.7
	// DefaultVolatileUpwardBias is the bias factor for volatile pattern (0.45 means slight upward bias)
	DefaultVolatileUpwardBias = 0.45
)

// MockDataConfig holds the configuration for generating mock market data
type MockDataConfig struct {
	// Symbol is the ticker symbol for the generated data
	Symbol string
	// StartTime is the start time for the generated data
	StartTime time.Time
	// EndTime is the end time for the generated data
	EndTime time.Time
	// Interval is the time interval between data points (e.g., 1 minute, 1 hour)
	Interval time.Duration
	// NumDataPoints is the number of data points to generate.
	// If set, it takes precedence over EndTime calculation based on interval
	NumDataPoints int
	// Pattern is the simulation pattern to use
	Pattern SimulationPattern
	// InitialPrice is the starting price for the simulation
	InitialPrice float64
	// MaxDrawdownPercent is the maximum allowed drawdown percentage (only used with PatternVolatile)
	// This represents the maximum percentage the price can drop from its peak
	MaxDrawdownPercent float64
	// VolatilityPercent is the base volatility percentage for price changes (only used with PatternVolatile)
	VolatilityPercent float64
	// TrendStrength is the strength of the trend (0.0 to 1.0) for increasing/decreasing patterns
	// Higher values result in stronger trends
	TrendStrength float64
	// Seed is the random seed for reproducible results. If 0, uses current time
	Seed int64
}

// MockDataGenerator generates mock market data for e2e testing
type MockDataGenerator struct {
	config MockDataConfig
	rng    *rand.Rand
}

// NewMockDataGenerator creates a new MockDataGenerator with the given configuration
func NewMockDataGenerator(config MockDataConfig) *MockDataGenerator {
	seed := config.Seed
	if seed == 0 {
		seed = time.Now().UnixNano()
	}
	rng := rand.New(rand.NewSource(seed))

	// Set default values if not provided
	if config.InitialPrice <= 0 {
		config.InitialPrice = 100.0
	}
	if config.TrendStrength <= 0 {
		config.TrendStrength = 0.01 // 1% default trend per interval
	}
	if config.VolatilityPercent <= 0 {
		config.VolatilityPercent = 2.0 // 2% default volatility
	}
	if config.MaxDrawdownPercent <= 0 {
		config.MaxDrawdownPercent = 10.0 // 10% default max drawdown
	}

	return &MockDataGenerator{
		config: config,
		rng:    rng,
	}
}

// Generate generates mock market data based on the configuration
func (g *MockDataGenerator) Generate() ([]types.MarketData, error) {
	if g.config.Symbol == "" {
		return nil, fmt.Errorf("symbol is required")
	}
	if g.config.StartTime.IsZero() {
		return nil, fmt.Errorf("start time is required")
	}
	if g.config.Interval <= 0 {
		return nil, fmt.Errorf("interval must be positive")
	}
	if g.config.NumDataPoints <= 0 && g.config.EndTime.IsZero() {
		return nil, fmt.Errorf("either NumDataPoints or EndTime is required")
	}

	numPoints := g.config.NumDataPoints
	if numPoints <= 0 {
		// Calculate number of points from time range
		duration := g.config.EndTime.Sub(g.config.StartTime)
		numPoints = int(duration / g.config.Interval)
		if numPoints <= 0 {
			return nil, fmt.Errorf("end time must be after start time")
		}
	}

	data := make([]types.MarketData, numPoints)
	currentPrice := g.config.InitialPrice
	peakPrice := currentPrice
	currentTime := g.config.StartTime

	for i := 0; i < numPoints; i++ {
		var priceChange float64

		switch g.config.Pattern {
		case PatternIncreasing:
			priceChange = g.generateIncreasingChange(currentPrice)
		case PatternDecreasing:
			priceChange = g.generateDecreasingChange(currentPrice)
		case PatternVolatile:
			priceChange = g.generateVolatileChange(currentPrice, peakPrice)
		default:
			return nil, fmt.Errorf("unknown pattern: %s", g.config.Pattern)
		}

		newPrice := currentPrice + priceChange
		if newPrice <= 0 {
			newPrice = DefaultMinimumPrice // Prevent negative or zero prices
		}

		// Generate OHLCV data
		open := currentPrice
		closePrice := newPrice

		// Generate high and low within the range
		minPrice := math.Min(open, closePrice)
		maxPrice := math.Max(open, closePrice)
		volatilityRange := maxPrice * (g.config.VolatilityPercent / 100.0) * 0.5

		high := maxPrice + g.rng.Float64()*volatilityRange
		low := minPrice - g.rng.Float64()*volatilityRange
		if low <= 0 {
			low = DefaultMinimumPrice
		}

		// Ensure OHLC relationships are valid
		if high < math.Max(open, closePrice) {
			high = math.Max(open, closePrice)
		}
		if low > math.Min(open, closePrice) {
			low = math.Min(open, closePrice)
		}

		// Generate volume (random with some variance)
		volume := DefaultBaseVolume * (0.5 + g.rng.Float64())

		data[i] = types.MarketData{
			Id:     uuid.New().String(),
			Symbol: g.config.Symbol,
			Time:   currentTime,
			Open:   open,
			High:   high,
			Low:    low,
			Close:  closePrice,
			Volume: volume,
		}

		currentPrice = newPrice
		currentTime = currentTime.Add(g.config.Interval)

		// Update peak price for drawdown calculation
		if currentPrice > peakPrice {
			peakPrice = currentPrice
		}
	}

	return data, nil
}

// generateIncreasingChange generates a price change for an increasing trend
func (g *MockDataGenerator) generateIncreasingChange(currentPrice float64) float64 {
	// Base upward trend
	trend := currentPrice * g.config.TrendStrength

	// Add some randomness (can be slightly negative but overall positive)
	noise := currentPrice * (g.config.VolatilityPercent / 100.0) * (g.rng.Float64() - DefaultIncreasingNoiseBias)

	return trend + noise
}

// generateDecreasingChange generates a price change for a decreasing trend
func (g *MockDataGenerator) generateDecreasingChange(currentPrice float64) float64 {
	// Base downward trend
	trend := -currentPrice * g.config.TrendStrength

	// Add some randomness (can be slightly positive but overall negative)
	noise := currentPrice * (g.config.VolatilityPercent / 100.0) * (g.rng.Float64() - DefaultDecreasingNoiseBias)

	return trend + noise
}

// generateVolatileChange generates a volatile price change with drawdown constraint
func (g *MockDataGenerator) generateVolatileChange(currentPrice, peakPrice float64) float64 {
	// Random direction with slight upward bias
	direction := g.rng.Float64() - DefaultVolatileUpwardBias

	// Base change
	change := currentPrice * (g.config.VolatilityPercent / 100.0) * direction

	// Check if the new price would violate drawdown constraint
	newPrice := currentPrice + change
	maxDrawdown := peakPrice * (g.config.MaxDrawdownPercent / 100.0)
	drawdownFloor := peakPrice - maxDrawdown

	if newPrice < drawdownFloor {
		// Constrain to maximum drawdown
		newPrice = drawdownFloor + g.rng.Float64()*(g.config.VolatilityPercent/100.0)*currentPrice
		change = newPrice - currentPrice
	}

	return change
}

// WriteToParquet writes the generated market data to a parquet file
func WriteToParquet(data []types.MarketData, outputPath string) error {
	if len(data) == 0 {
		return fmt.Errorf("no data to write")
	}

	// Create output directory if it doesn't exist
	outputDir := filepath.Dir(outputPath)
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Open in-memory DuckDB database
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return fmt.Errorf("failed to open DuckDB connection: %w", err)
	}
	defer db.Close()

	// Create a table for market data
	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol VARCHAR,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create table: %w", err)
	}

	// Prepare insert statement
	stmt, err := db.Prepare(`
		INSERT INTO market_data (time, symbol, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	// Insert all data points
	for _, d := range data {
		_, err = stmt.Exec(d.Time, d.Symbol, d.Open, d.High, d.Low, d.Close, d.Volume)
		if err != nil {
			return fmt.Errorf("failed to insert data: %w", err)
		}
	}

	// Export to parquet file
	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, outputPath))
	if err != nil {
		return fmt.Errorf("failed to export to parquet: %w", err)
	}

	return nil
}

// GenerateAndWriteToParquet is a convenience function that generates mock data and writes it to a parquet file
func GenerateAndWriteToParquet(config MockDataConfig, outputPath string) error {
	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()
	if err != nil {
		return fmt.Errorf("failed to generate mock data: %w", err)
	}

	return WriteToParquet(data, outputPath)
}

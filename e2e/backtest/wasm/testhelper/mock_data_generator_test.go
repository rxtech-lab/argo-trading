package testhelper

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
)

// MockDataGeneratorTestSuite is the test suite for the MockDataGenerator
type MockDataGeneratorTestSuite struct {
	suite.Suite
}

func TestMockDataGeneratorSuite(t *testing.T) {
	suite.Run(t, new(MockDataGeneratorTestSuite))
}

func (suite *MockDataGeneratorTestSuite) TestGenerateMockFilename() {
	// Test that the filename is generated with mock_ prefix
	filename := GenerateMockFilename("test_data")
	suite.Equal("mock_test_data.parquet", filename)

	// Test with different base names
	suite.Equal("mock_btc_hourly.parquet", GenerateMockFilename("btc_hourly"))
	suite.Equal("mock_spy.parquet", GenerateMockFilename("spy"))
}

func (suite *MockDataGeneratorTestSuite) TestNewMockDataGenerator() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	suite.NotNil(generator)
	suite.Equal("TEST", generator.config.Symbol)
	suite.Equal(100.0, generator.config.InitialPrice) // Default value
}

func (suite *MockDataGeneratorTestSuite) TestGenerateIncreasingPattern() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
		InitialPrice:  100.0,
		TrendStrength: 0.02, // 2% increase per interval
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()

	suite.Require().NoError(err)
	suite.Require().Len(data, 100)

	// Check that the overall trend is increasing
	firstPrice := data[0].Close
	lastPrice := data[len(data)-1].Close
	suite.Greater(lastPrice, firstPrice, "Price should increase overall with increasing pattern")

	// Verify all data points have correct symbol
	for _, d := range data {
		suite.Equal("TEST", d.Symbol)
	}

	// Verify time progression
	for i := 1; i < len(data); i++ {
		suite.Equal(time.Minute, data[i].Time.Sub(data[i-1].Time))
	}

	// Verify OHLC relationships
	for _, d := range data {
		suite.GreaterOrEqual(d.High, d.Open, "High should be >= Open")
		suite.GreaterOrEqual(d.High, d.Close, "High should be >= Close")
		suite.LessOrEqual(d.Low, d.Open, "Low should be <= Open")
		suite.LessOrEqual(d.Low, d.Close, "Low should be <= Close")
		suite.Greater(d.Volume, 0.0, "Volume should be positive")
	}
}

func (suite *MockDataGeneratorTestSuite) TestGenerateDecreasingPattern() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternDecreasing,
		InitialPrice:  100.0,
		TrendStrength: 0.02, // 2% decrease per interval
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()

	suite.Require().NoError(err)
	suite.Require().Len(data, 100)

	// Check that the overall trend is decreasing
	firstPrice := data[0].Close
	lastPrice := data[len(data)-1].Close
	suite.Less(lastPrice, firstPrice, "Price should decrease overall with decreasing pattern")

	// Verify prices don't go negative
	for _, d := range data {
		suite.Greater(d.Close, 0.0, "Close price should be positive")
		suite.Greater(d.Low, 0.0, "Low price should be positive")
	}
}

func (suite *MockDataGeneratorTestSuite) TestGenerateVolatilePattern() {
	config := MockDataConfig{
		Symbol:             "TEST",
		StartTime:          time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:           time.Minute,
		NumDataPoints:      200,
		Pattern:            PatternVolatile,
		InitialPrice:       100.0,
		MaxDrawdownPercent: 10.0, // 10% max drawdown
		VolatilityPercent:  3.0,  // 3% volatility
		Seed:               42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()

	suite.Require().NoError(err)
	suite.Require().Len(data, 200)

	// Track peak price and verify drawdown constraint
	peakPrice := data[0].Close
	for _, d := range data {
		if d.Close > peakPrice {
			peakPrice = d.Close
		}

		// Calculate current drawdown from peak
		drawdownPercent := ((peakPrice - d.Close) / peakPrice) * 100

		// Allow small margin for floating point arithmetic
		suite.LessOrEqual(drawdownPercent, config.MaxDrawdownPercent+1.0,
			"Drawdown should not exceed max drawdown percentage plus margin")
	}
}

func (suite *MockDataGeneratorTestSuite) TestGenerateWithEndTime() {
	startTime := time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC)
	endTime := time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC) // 1 hour later

	config := MockDataConfig{
		Symbol:       "TEST",
		StartTime:    startTime,
		EndTime:      endTime,
		Interval:     time.Minute,
		Pattern:      PatternIncreasing,
		InitialPrice: 100.0,
		Seed:         42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()

	suite.Require().NoError(err)
	// 60 minutes = 60 data points
	suite.Len(data, 60)
}

func (suite *MockDataGeneratorTestSuite) TestGenerateValidation() {
	// Test missing symbol
	config := MockDataConfig{
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
	}
	generator := NewMockDataGenerator(config)
	_, err := generator.Generate()
	suite.Error(err)
	suite.Contains(err.Error(), "symbol is required")

	// Test missing start time
	config = MockDataConfig{
		Symbol:        "TEST",
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
	}
	generator = NewMockDataGenerator(config)
	_, err = generator.Generate()
	suite.Error(err)
	suite.Contains(err.Error(), "start time is required")

	// Test missing interval
	config = MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
	}
	generator = NewMockDataGenerator(config)
	_, err = generator.Generate()
	suite.Error(err)
	suite.Contains(err.Error(), "interval must be positive")

	// Test missing NumDataPoints and EndTime
	config = MockDataConfig{
		Symbol:    "TEST",
		StartTime: time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:  time.Minute,
		Pattern:   PatternIncreasing,
	}
	generator = NewMockDataGenerator(config)
	_, err = generator.Generate()
	suite.Error(err)
	suite.Contains(err.Error(), "either NumDataPoints or EndTime is required")

	// Test unknown pattern
	config = MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       "unknown",
	}
	generator = NewMockDataGenerator(config)
	_, err = generator.Generate()
	suite.Error(err)
	suite.Contains(err.Error(), "unknown pattern")
}

func (suite *MockDataGeneratorTestSuite) TestWriteToParquet() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 50,
		Pattern:       PatternIncreasing,
		InitialPrice:  100.0,
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()
	suite.Require().NoError(err)

	// Create temp file with mock_ prefix
	tmpDir := suite.T().TempDir()
	outputPath := filepath.Join(tmpDir, GenerateMockFilename("test_data"))

	// Write to parquet
	err = WriteToParquet(data, outputPath)
	suite.Require().NoError(err)

	// Verify file exists
	_, err = os.Stat(outputPath)
	suite.NoError(err)
}

func (suite *MockDataGeneratorTestSuite) TestGenerateAndWriteToParquet() {
	config := MockDataConfig{
		Symbol:        "BTCUSD",
		StartTime:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Interval:      time.Hour,
		NumDataPoints: 24,
		Pattern:       PatternVolatile,
		InitialPrice:  50000.0,
		Seed:          42,
	}

	tmpDir := suite.T().TempDir()
	outputPath := filepath.Join(tmpDir, GenerateMockFilename("btc_data"))

	err := GenerateAndWriteToParquet(config, outputPath)
	suite.Require().NoError(err)

	// Verify file exists
	_, err = os.Stat(outputPath)
	suite.NoError(err)
}

func (suite *MockDataGeneratorTestSuite) TestReproducibilityWithSeed() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternVolatile,
		InitialPrice:  100.0,
		Seed:          12345,
	}

	// Generate first dataset
	generator1 := NewMockDataGenerator(config)
	data1, err := generator1.Generate()
	suite.Require().NoError(err)

	// Generate second dataset with same config
	generator2 := NewMockDataGenerator(config)
	data2, err := generator2.Generate()
	suite.Require().NoError(err)

	// Both should be identical
	suite.Require().Len(data1, len(data2))
	for i := range data1 {
		suite.Equal(data1[i].Close, data2[i].Close, "Close prices should match for reproducibility")
		suite.Equal(data1[i].Open, data2[i].Open, "Open prices should match for reproducibility")
	}
}

func (suite *MockDataGeneratorTestSuite) TestDifferentIntervals() {
	// Test hourly interval
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		Interval:      time.Hour,
		NumDataPoints: 24,
		Pattern:       PatternIncreasing,
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()
	suite.Require().NoError(err)
	suite.Len(data, 24)

	// Verify hourly progression
	for i := 1; i < len(data); i++ {
		suite.Equal(time.Hour, data[i].Time.Sub(data[i-1].Time))
	}

	// Test 5-minute interval
	config.Interval = 5 * time.Minute
	config.NumDataPoints = 12

	generator = NewMockDataGenerator(config)
	data, err = generator.Generate()
	suite.Require().NoError(err)
	suite.Len(data, 12)

	// Verify 5-minute progression
	for i := 1; i < len(data); i++ {
		suite.Equal(5*time.Minute, data[i].Time.Sub(data[i-1].Time))
	}
}

func (suite *MockDataGeneratorTestSuite) TestWriteToParquetEmptyData() {
	tmpDir := suite.T().TempDir()
	outputPath := filepath.Join(tmpDir, GenerateMockFilename("empty"))

	err := WriteToParquet(nil, outputPath)
	suite.Error(err)
	suite.Contains(err.Error(), "no data to write")
}

func (suite *MockDataGeneratorTestSuite) TestUniqueIDs() {
	config := MockDataConfig{
		Symbol:        "TEST",
		StartTime:     time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:      time.Minute,
		NumDataPoints: 100,
		Pattern:       PatternIncreasing,
		Seed:          42,
	}

	generator := NewMockDataGenerator(config)
	data, err := generator.Generate()
	suite.Require().NoError(err)

	// Verify all IDs are unique
	ids := make(map[string]bool)
	for _, d := range data {
		suite.False(ids[d.Id], "IDs should be unique")
		ids[d.Id] = true
	}
}

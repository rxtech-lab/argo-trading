package mocks

import (
	"testing"
	"time"
)

func TestDataGenerator_Generate(t *testing.T) {
	gen := NewDataGenerator(42) // Fixed seed for reproducibility
	config := DefaultConfig()
	config.Count = 100

	data := gen.Generate(config)

	if len(data) != 100 {
		t.Errorf("expected 100 data points, got %d", len(data))
	}

	// Verify data is in chronological order
	for i := 1; i < len(data); i++ {
		if !data[i].Time.After(data[i-1].Time) {
			t.Errorf("data not in chronological order at index %d", i)
		}
	}

	// Verify symbol is set correctly
	for i, d := range data {
		if d.Symbol != config.Symbol {
			t.Errorf("expected symbol %s at index %d, got %s", config.Symbol, i, d.Symbol)
		}
	}

	// Verify OHLC values are positive
	for i, d := range data {
		if d.Open <= 0 || d.High <= 0 || d.Low <= 0 || d.Close <= 0 {
			t.Errorf("invalid OHLC values at index %d: O=%f H=%f L=%f C=%f",
				i, d.Open, d.High, d.Low, d.Close)
		}
	}

	// Verify High >= Low
	for i, d := range data {
		if d.High < d.Low {
			t.Errorf("High < Low at index %d: H=%f L=%f", i, d.High, d.Low)
		}
	}

	// Verify time intervals
	expectedInterval := config.Interval
	for i := 1; i < len(data); i++ {
		actualInterval := data[i].Time.Sub(data[i-1].Time)
		if actualInterval != expectedInterval {
			t.Errorf("unexpected interval at index %d: expected %v, got %v",
				i, expectedInterval, actualInterval)
		}
	}
}

func TestDataGenerator_Reproducibility(t *testing.T) {
	// Same seed should produce same results
	gen1 := NewDataGenerator(42)
	gen2 := NewDataGenerator(42)

	config := DefaultConfig()
	config.Count = 10

	data1 := gen1.Generate(config)
	data2 := gen2.Generate(config)

	for i := range data1 {
		if data1[i].Close != data2[i].Close {
			t.Errorf("data not reproducible at index %d: got %f and %f",
				i, data1[i].Close, data2[i].Close)
		}
	}
}

func TestDataGenerator_Different_Seeds(t *testing.T) {
	gen1 := NewDataGenerator(42)
	gen2 := NewDataGenerator(123)

	config := DefaultConfig()
	config.Count = 10

	data1 := gen1.Generate(config)
	data2 := gen2.Generate(config)

	// Different seeds should produce different results
	sameCount := 0
	for i := range data1 {
		if data1[i].Close == data2[i].Close {
			sameCount++
		}
	}

	if sameCount == len(data1) {
		t.Error("different seeds produced identical data")
	}
}

func TestGenerate10K(t *testing.T) {
	data := Generate10K("TEST")

	if len(data) != 10000 {
		t.Errorf("expected 10000 data points, got %d", len(data))
	}

	// Verify first data point
	if data[0].Symbol != "TEST" {
		t.Errorf("expected symbol TEST, got %s", data[0].Symbol)
	}

	// Verify chronological order
	for i := 1; i < 100; i++ { // Check first 100 for speed
		if !data[i].Time.After(data[i-1].Time) {
			t.Errorf("data not in chronological order at index %d", i)
		}
	}
}

func TestGenerateMultiSymbol(t *testing.T) {
	symbols := []string{"AAPL", "GOOG", "MSFT"}
	gen := NewDataGenerator(42)
	config := DefaultConfig()
	config.Count = 100

	data := gen.GenerateMultiSymbol(symbols, config)

	expectedTotal := len(symbols) * config.Count
	if len(data) != expectedTotal {
		t.Errorf("expected %d data points, got %d", expectedTotal, len(data))
	}

	// Verify each symbol has data
	symbolCounts := make(map[string]int)
	for _, d := range data {
		symbolCounts[d.Symbol]++
	}

	for _, symbol := range symbols {
		if symbolCounts[symbol] != config.Count {
			t.Errorf("expected %d data points for %s, got %d",
				config.Count, symbol, symbolCounts[symbol])
		}
	}
}

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Count != 10000 {
		t.Errorf("expected default count 10000, got %d", config.Count)
	}

	if config.Symbol != "TEST" {
		t.Errorf("expected default symbol TEST, got %s", config.Symbol)
	}

	if config.Interval != time.Minute {
		t.Errorf("expected default interval 1m, got %v", config.Interval)
	}

	if config.InitialPrice != 100.0 {
		t.Errorf("expected default initial price 100.0, got %f", config.InitialPrice)
	}
}

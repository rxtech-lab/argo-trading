package datasource

import (
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/assert"
)

func TestCSVIterator(t *testing.T) {
	// Get the path to the data.csv file
	dataFilePath := filepath.Join(".", "data.csv")

	// Create a temporary directory for the test
	tempDir := t.TempDir()
	tempFile := filepath.Join(tempDir, "test_data.csv")

	// Copy the content from the data.csv file to the temporary file
	sourceFile, err := os.Open(dataFilePath)
	assert.NoError(t, err, "Failed to open source data file")
	defer sourceFile.Close()

	destFile, err := os.Create(tempFile)
	assert.NoError(t, err, "Failed to create temporary test file")
	defer destFile.Close()

	_, err = io.Copy(destFile, sourceFile)
	assert.NoError(t, err, "Failed to copy data to temporary file")

	// Close the destination file to ensure all data is written
	destFile.Close()

	// Expected data based on the CSV content
	expectedData := []types.MarketData{
		{
			Time:   parseTime(t, "2025-01-01T08:03:00+08:00"),
			Open:   250.4216,
			High:   250.4216,
			Low:    250.4216,
			Close:  250.4216,
			Volume: 189,
		},
		{
			Time:   parseTime(t, "2025-01-01T08:11:00+08:00"),
			Open:   250.45,
			High:   250.45,
			Low:    250.45,
			Close:  250.45,
			Volume: 221,
		},
		{
			Time:   parseTime(t, "2025-01-01T08:14:00+08:00"),
			Open:   250.4208,
			High:   250.4208,
			Low:    250.4208,
			Close:  250.4208,
			Volume: 502,
		},
		{
			Time:   parseTime(t, "2025-01-01T08:19:00+08:00"),
			Open:   250.43,
			High:   250.43,
			Low:    250.42,
			Close:  250.42,
			Volume: 1246,
		},
	}

	// Test the iterator
	t.Run("Iterator should yield all market data records", func(t *testing.T) {
		// Create a new CSV iterator for this test case
		csvIterator := CSVIterator{
			FilePath: tempFile,
		}

		iterator := csvIterator.Iterator(parseTime(t, "2025-01-01T00:00:00+08:00"), parseTime(t, "2025-01-02T00:00:00+08:00"))

		var results []types.MarketData
		iterator(func(data types.MarketData) bool {
			results = append(results, data)
			return true // Continue iteration
		})

		// Verify the number of records
		assert.Equal(t, len(expectedData), len(results), "Number of records should match")

		// Verify each record
		for i, expected := range expectedData {
			assert.Equal(t, expected.Time.Format(time.RFC3339), results[i].Time.Format(time.RFC3339), "Time should match for record %d", i)
			assert.Equal(t, expected.Open, results[i].Open, "Open should match for record %d", i)
			assert.Equal(t, expected.High, results[i].High, "High should match for record %d", i)
			assert.Equal(t, expected.Low, results[i].Low, "Low should match for record %d", i)
			assert.Equal(t, expected.Close, results[i].Close, "Close should match for record %d", i)
			assert.Equal(t, expected.Volume, results[i].Volume, "Volume should match for record %d", i)
		}
	})

	t.Run("Iterator should stop when yield returns false", func(t *testing.T) {
		// Create a new CSV iterator for this test case
		csvIterator := CSVIterator{
			FilePath: tempFile,
		}

		iterator := csvIterator.Iterator(parseTime(t, "2025-01-01T00:00:00+08:00"), parseTime(t, "2025-01-02T00:00:00+08:00"))

		var results []types.MarketData
		iterator(func(data types.MarketData) bool {
			results = append(results, data)
			return len(results) < 2 // Stop after 2 records
		})

		// Verify we only got 2 records
		assert.Equal(t, 2, len(results), "Should only get 2 records when yield returns false")

		// Verify the records we did get
		for i := 0; i < 2; i++ {
			assert.Equal(t, expectedData[i].Time.Format(time.RFC3339), results[i].Time.Format(time.RFC3339), "Time should match for record %d", i)
			assert.Equal(t, expectedData[i].Open, results[i].Open, "Open should match for record %d", i)
			assert.Equal(t, expectedData[i].High, results[i].High, "High should match for record %d", i)
			assert.Equal(t, expectedData[i].Low, results[i].Low, "Low should match for record %d", i)
			assert.Equal(t, expectedData[i].Close, results[i].Close, "Close should match for record %d", i)
			assert.Equal(t, expectedData[i].Volume, results[i].Volume, "Volume should match for record %d", i)
		}
	})

	t.Run("Iterator should return empty slice if no data is found", func(t *testing.T) {
		// Create a new CSV iterator for this test case
		csvIterator := CSVIterator{
			FilePath: tempFile,
		}

		iterator := csvIterator.Iterator(parseTime(t, "2024-01-01T00:00:00+08:00"), parseTime(t, "2024-01-02T00:00:00+08:00"))

		var results []types.MarketData
		iterator(func(data types.MarketData) bool {
			results = append(results, data)
			return true
		})

		// Verify we only got 0 records
		assert.Equal(t, 0, len(results), "Should only get 0 records when no data is found")

	})

	t.Run("Iterator should yield data for a specific time range", func(t *testing.T) {
		// Create a new CSV iterator for this test case
		csvIterator := CSVIterator{
			FilePath: tempFile,
		}

		iterator := csvIterator.Iterator(parseTime(t, "2025-01-01T08:03:00+08:00"), parseTime(t, "2025-01-01T08:11:00+08:00"))

		var results []types.MarketData
		iterator(func(data types.MarketData) bool {
			results = append(results, data)
			return true // Continue iteration
		})

		assert.Equal(t, 2, len(results), "Should only get 2 records when yield returns false")

		// Verify the records we did get
		for i := 0; i < 2; i++ {
			assert.Equal(t, expectedData[i].Time.Format(time.RFC3339), results[i].Time.Format(time.RFC3339), "Time should match for record %d", i)
			assert.Equal(t, expectedData[i].Open, results[i].Open, "Open should match for record %d", i)
			assert.Equal(t, expectedData[i].High, results[i].High, "High should match for record %d", i)
			assert.Equal(t, expectedData[i].Low, results[i].Low, "Low should match for record %d", i)
			assert.Equal(t, expectedData[i].Close, results[i].Close, "Close should match for record %d", i)
			assert.Equal(t, expectedData[i].Volume, results[i].Volume, "Volume should match for record %d", i)
		}
	})
}

// Helper function to parse time strings
func parseTime(t *testing.T, timeStr string) time.Time {
	parsed, err := time.Parse(time.RFC3339, timeStr)
	assert.NoError(t, err, "Failed to parse time: %s", timeStr)
	return parsed
}

package datasource

import (
	"log"
	"os"
	"time"

	"github.com/gocarina/gocsv"
	"github.com/sirily11/argo-trading-go/src/types"
)

type CSVIterator struct {
	FilePath string
	cache    []types.MarketData
}

func NewCSVIterator(filePath string) types.MarketDataSource {
	return &CSVIterator{
		FilePath: filePath,
	}
}

func (i *CSVIterator) Iterator(startTime, endTime time.Time) func(yield func(types.MarketData) bool) {
	return func(yield func(types.MarketData) bool) {
		// Ensure cache is populated
		if i.cache == nil {
			// Load data into cache using GetDataForTimeRange with very wide time range
			// This will populate the cache with all data
			veryOldTime := time.Date(1900, 1, 1, 0, 0, 0, 0, time.UTC)
			veryFutureTime := time.Now().AddDate(100, 0, 0) // 100 years in the future
			i.GetDataForTimeRange(veryOldTime, veryFutureTime)
		}

		// Iterate through cached data and yield matching records
		for _, marketData := range i.cache {
			// Filter by time range
			if (marketData.Time.Equal(startTime) || marketData.Time.After(startTime)) &&
				(marketData.Time.Equal(endTime) || marketData.Time.Before(endTime)) {
				if !yield(marketData) {
					break
				}
			}
		}
	}
}

func (i *CSVIterator) GetDataForTimeRange(startTime, endTime time.Time) []types.MarketData {
	// Check if cache is already populated
	if i.cache == nil {
		// Cache is empty, load data from CSV file
		csvFile, err := os.Open(i.FilePath)
		if err != nil {
			log.Fatalf("failed to open CSV file: %v", err)
		}
		defer csvFile.Close()

		// Unmarshal the CSV file into the cache
		if err := gocsv.UnmarshalFile(csvFile, &i.cache); err != nil {
			log.Fatalf("failed to unmarshal CSV: %v", err)
		}

		log.Printf("Loaded %d market data points into cache", len(i.cache))
	}

	// Filter cached data by time range
	var filteredData []types.MarketData
	for _, data := range i.cache {
		if (data.Time.Equal(startTime) || data.Time.After(startTime)) &&
			(data.Time.Equal(endTime) || data.Time.Before(endTime)) {
			filteredData = append(filteredData, data)
		}
	}

	return filteredData
}

// ClearCache clears the in-memory cache to free up memory
func (i *CSVIterator) ClearCache() {
	i.cache = nil
	log.Printf("Cache cleared for CSV iterator: %s", i.FilePath)
}

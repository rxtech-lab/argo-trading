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
}

func NewCSVIterator(filePath string) types.MarketDataSource {
	return &CSVIterator{
		FilePath: filePath,
	}
}

func (i *CSVIterator) Iterator(startTime, endTime time.Time) func(yield func(types.MarketData) bool) {
	return func(yield func(types.MarketData) bool) {
		// read the file
		csvFile, err := os.Open(i.FilePath)
		if err != nil {
			log.Fatalf("failed to open CSV file: %v", err)
		}
		defer csvFile.Close()

		// read the file line by line using a channel
		marketDataChan := make(chan types.MarketData)

		// Start a goroutine to parse the CSV file
		go func() {
			// Use UnmarshalToChan to read the CSV file line by line
			if err := gocsv.UnmarshalToChan(csvFile, marketDataChan); err != nil {
				log.Fatalf("failed to unmarshal CSV: %v", err)
			}
			// No need to explicitly close the channel - UnmarshalToChan will close it
		}()

		// Process each record as it comes in - simpler and more efficient loop
		for marketData := range marketDataChan {
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
	// Read all data from CSV file
	csvFile, err := os.Open(i.FilePath)
	if err != nil {
		log.Fatalf("failed to open CSV file: %v", err)
	}
	defer csvFile.Close()

	var marketData []types.MarketData
	if err := gocsv.UnmarshalFile(csvFile, &marketData); err != nil {
		log.Fatalf("failed to unmarshal CSV: %v", err)
	}

	// Filter by time range
	var filteredData []types.MarketData
	for _, data := range marketData {
		if (data.Time.Equal(startTime) || data.Time.After(startTime)) &&
			(data.Time.Equal(endTime) || data.Time.Before(endTime)) {
			filteredData = append(filteredData, data)
		}
	}

	return filteredData
}

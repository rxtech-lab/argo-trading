package datasource

import (
	"log"
	"os"

	"github.com/gocarina/gocsv"
	"github.com/sirily11/argo-trading-go/src/types"
)

type CSVIterator struct {
	FilePath string
}

func (i *CSVIterator) Iterator() func(yield func(types.MarketData) bool) {
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
			if !yield(marketData) {
				break
			}
		}
	}
}

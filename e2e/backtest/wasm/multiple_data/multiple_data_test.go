package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

// MultipleDataTestSuite extends the base test suite
type MultipleDataTestSuite struct {
	testhelper.E2ETestSuite
}

func TestMultipleDataTestSuite(t *testing.T) {
	suite.Run(t, new(MultipleDataTestSuite))
}

// SetupTest initializes the test with config
func (s *MultipleDataTestSuite) SetupTest() {
	s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
}

// TestPlaceOrderStrategy tests the strategy's ability to process multiple data files concurrently.
// The strategy is designed to place one order per data file processed.
// This test verifies that when provided with 3 data files, the strategy:
// 1. Successfully processes all files
// 2. Places one order per file (3 total orders)
// 3. Generates corresponding statistics for each file (3 total stats)
func (s *MultipleDataTestSuite) TestPlaceOrderStrategyWithMultipleData() {
	s.Run("TestPlaceOrderStrategy", func() {
		// create multiple test data in tmp folder
		originalTestData := "../../../../internal/indicator/test_data/test_data.parquet"
		tempDataPath := filepath.Join(s.T().TempDir(), "data")
		testDataPattern := filepath.Join(tempDataPath, "*.parquet")

		// Create multiple test data files with different symbols using our helper
		err := os.MkdirAll(tempDataPath, 0755)
		s.Require().NoError(err)

		// Create three copies of the original test data with different symbols
		for i := 1; i <= 3; i++ {
			outputFile := filepath.Join(tempDataPath, fmt.Sprintf("test_data_%d.parquet", i))
			symbol := fmt.Sprintf("SYMBOL%d", i)

			err := testhelper.UpdateParquetSymbol(originalTestData, outputFile, symbol)
			s.Require().NoError(err)
		}

		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "PlaceOrderStrategy", "../place_order/place_order_plugin.wasm", testDataPattern)
		// read stats
		stats, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
		s.Require().Equal(len(stats), 3)

		// make sure each trade, num of trades > 0
		for _, stat := range stats {
			s.Require().Greater(stat.TradeResult.NumberOfTrades, 0)
		}
	})
}

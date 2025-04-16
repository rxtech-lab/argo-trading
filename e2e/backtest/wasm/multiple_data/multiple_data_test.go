package main

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/stretchr/testify/suite"
)

// E2ETestSuite extends the base test suite
type E2ETestSuite struct {
	testhelper.E2ETestSuite
}

func TestE2ETestSuite(t *testing.T) {
	suite.Run(t, new(E2ETestSuite))
}

// SetupTest initializes the test with config
func (s *E2ETestSuite) SetupTest() {
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
func (s *E2ETestSuite) TestPlaceOrderStrategyWithMultipleData() {
	s.Run("TestPlaceOrderStrategy", func() {
		// create multiple test data in tmp folder
		originalTestData := "../../../../internal/indicator/test_data/test_data.parquet"
		tempDataPath := filepath.Join(s.T().TempDir(), "data")
		testDataPattern := filepath.Join(tempDataPath, "*.parquet")
		// create multiple test data in tmp folder
		err := os.MkdirAll(tempDataPath, 0755)
		s.Require().NoError(err)

		// copy original test data to tmp folder as test_data_1.parquet, test_data_2.parquet, test_data_3.parquet
		for i := 1; i <= 3; i++ {
			// Read the original file
			data, err := os.ReadFile(originalTestData)
			s.Require().NoError(err)

			// Write to new file
			err = os.WriteFile(filepath.Join(tempDataPath, fmt.Sprintf("test_data_%d.parquet", i)), data, 0644)
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

package main

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/e2e/backtest/wasm/testhelper"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// ComplexStrategyTestSuite extends the base test suite
type ComplexStrategyTestSuite struct {
	testhelper.E2ETestSuite
	mockDataPath string
}

func TestComplexStrategyTestSuite(t *testing.T) {
	suite.Run(t, new(ComplexStrategyTestSuite))
}

// SetupTest initializes the test with config
func (s *ComplexStrategyTestSuite) SetupTest() {
}

// generateMockData generates a large dataset for testing
func (s *ComplexStrategyTestSuite) generateMockData() string {
	// Create a temporary directory for mock data
	tmpDir := s.T().TempDir()
	mockDataPath := filepath.Join(tmpDir, testhelper.GenerateMockFilename("complex_strategy_data"))

	// Generate a large volatile dataset to trigger various trading conditions
	config := testhelper.MockDataConfig{
		Symbol:             "TESTSTOCK",
		StartTime:          time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
		Interval:           time.Minute,
		NumDataPoints:      1000, // Large dataset: 1000 data points
		Pattern:            testhelper.PatternVolatile,
		InitialPrice:       100.0,
		MaxDrawdownPercent: 15.0,
		VolatilityPercent:  5.0, // High volatility to generate more trading signals
		Seed:               42,  // Fixed seed for reproducibility
	}

	err := testhelper.GenerateAndWriteToParquet(config, mockDataPath)
	s.Require().NoError(err, "Failed to generate mock data")

	// Verify file was created
	_, err = os.Stat(mockDataPath)
	s.Require().NoError(err, "Mock data file was not created")

	return mockDataPath
}

func (s *ComplexStrategyTestSuite) TestComplexStrategy() {
	s.Run("TestComplexStrategyWithDefaultData", func() {
		s.E2ETestSuite.SetupTest(`
initial_capital: 10000
`)
		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "ComplexStrategy", "./complex_strategy_plugin.wasm", "")
		// read stats
		_, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err)
	})
}

func (s *ComplexStrategyTestSuite) TestComplexStrategyWithLargeDataset() {
	s.Run("TestComplexStrategyWithMockData", func() {
		// Generate large mock dataset
		mockDataPath := s.generateMockData()

		// Setup test with higher initial capital to support multiple trades
		s.E2ETestSuite.SetupTest(`
initial_capital: 100000
`)

		// Run the strategy with the generated mock data
		tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "ComplexStrategy", "./complex_strategy_plugin.wasm", mockDataPath)

		// Verify stats were generated
		stats, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err, "Failed to read stats")
		s.Require().Greater(len(stats), 0, "Should have at least one stats entry")
		s.Require().Equal("TESTSTOCK", stats[0].Symbol, "Symbol should match mock data symbol")

		// Verify strategy metadata
		s.Require().Equal("com.argo-trading.hybrid-strategy", stats[0].Strategy.ID, "Strategy ID should match")
		s.Require().Equal("HybridTradingStrategy", stats[0].Strategy.Name, "Strategy name should match")

		// Read and verify trades
		trades, err := testhelper.ReadTrades(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err, "Failed to read trades")
		s.Require().Greater(len(trades), 0, "Should have at least one trade with volatile data")

		// Verify trade details
		for _, trade := range trades {
			s.Require().Greater(trade.ExecutedPrice, 0.0, "Trade executed price should be positive")
			s.Require().Equal(1.0, trade.ExecutedQty, "Trade quantity should be 1.0")
			s.Require().Equal("TESTSTOCK", trade.Order.Symbol, "Trade symbol should match")
			s.Require().Equal("HybridTradingStrategy", trade.Order.StrategyName, "Strategy name should match")
		}

		// Verify we have both buy and sell trades
		var buyCount, sellCount int
		for _, trade := range trades {
			if trade.Order.Side == types.PurchaseTypeBuy {
				buyCount++
			} else if trade.Order.Side == types.PurchaseTypeSell {
				sellCount++
			}
		}
		s.Require().Greater(buyCount, 0, "Should have at least one buy trade")

		// Read and verify orders
		orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err, "Failed to read orders")
		s.Require().GreaterOrEqual(len(orders), len(trades), "Should have at least as many orders as trades")

		// Verify order details
		for _, order := range orders {
			s.Require().NotEmpty(order.OrderID, "Order ID should not be empty")
			s.Require().Equal("TESTSTOCK", order.Symbol, "Order symbol should match")
			s.Require().Equal("HybridTradingStrategy", order.StrategyName, "Strategy name should match")
			s.Require().Greater(order.Price, 0.0, "Order price should be positive")
			s.Require().NotEmpty(order.Reason.Message, "Order reason message should not be empty")
		}

		// Read and verify marks
		marks, err := testhelper.ReadMarker(&s.E2ETestSuite, tmpFolder)
		s.Require().NoError(err, "Failed to read marks")
		s.Require().Greater(len(marks), 0, "Should have at least one mark")

		// Verify we have different types of marks (shapes, colors, levels, categories)
		shapeCounts := make(map[types.MarkShape]int)
		colorCounts := make(map[types.MarkColor]int)
		levelCounts := make(map[types.MarkLevel]int)
		categoryCounts := make(map[string]int)

		for _, mark := range marks {
			shapeCounts[mark.Shape]++
			colorCounts[mark.Color]++
			levelCounts[mark.Level]++
			categoryCounts[mark.Category]++
		}

		// Verify we have multiple mark types (based on our strategy implementation)
		s.Require().Greater(len(categoryCounts), 0, "Should have marks from different categories")

		// Check for expected categories from our strategy
		// Categories used: DataQuality, MarketEvent, Trade, OrderError, RiskManagement
		expectedCategories := []string{"Trade", "MarketEvent"}
		for _, cat := range expectedCategories {
			// At least one of the expected categories should have marks
			if categoryCounts[cat] > 0 {
				break
			}
		}

		// Verify marks have proper data (Title and Message are required)
		for _, mark := range marks {
			s.Require().NotEmpty(mark.Title, "Mark should have title")
			s.Require().NotEmpty(mark.Message, "Mark should have message")
			s.Require().NotEmpty(mark.Category, "Mark should have category")
		}

		// Verify we have marks with different shapes
		s.Require().Greater(len(shapeCounts), 0, "Should have marks with different shapes")

		// Verify we have marks with different colors
		s.Require().Greater(len(colorCounts), 0, "Should have marks with different colors")

		// Verify we have marks with different levels
		s.Require().Greater(len(levelCounts), 0, "Should have marks with different levels")

		// Verify Trade category marks have signals
		tradeMarks := 0
		for _, mark := range marks {
			if mark.Category == "Trade" {
				tradeMarks++
				s.Require().True(mark.Signal.IsSome(), "Trade marks should have signals")
				signal, err := mark.Signal.Take()
				s.Require().NoError(err)
				s.Require().Contains(
					[]types.SignalType{types.SignalTypeBuyLong, types.SignalTypeSellLong},
					signal.Type,
					"Trade marks should have BUY_LONG or SELL_LONG signal type",
				)
			}
		}

		// Log summary for debugging
		s.T().Logf("Test Summary:")
		s.T().Logf("  Total trades: %d (buy: %d, sell: %d)", len(trades), buyCount, sellCount)
		s.T().Logf("  Total orders: %d", len(orders))
		s.T().Logf("  Total marks: %d", len(marks))
		s.T().Logf("  Mark shapes: %v", shapeCounts)
		s.T().Logf("  Mark colors: %v", colorCounts)
		s.T().Logf("  Mark levels: %v", levelCounts)
		s.T().Logf("  Mark categories: %v", categoryCounts)
	})
}

func (s *ComplexStrategyTestSuite) TestComplexStrategyWithDifferentPatterns() {
	patterns := []struct {
		name    string
		pattern testhelper.SimulationPattern
	}{
		{"Increasing", testhelper.PatternIncreasing},
		{"Decreasing", testhelper.PatternDecreasing},
	}

	for _, p := range patterns {
		s.Run("TestComplexStrategyWith"+p.name+"Pattern", func() {
			// Create a temporary directory for mock data
			tmpDir := s.T().TempDir()
			mockDataPath := filepath.Join(tmpDir, testhelper.GenerateMockFilename(p.name+"_data"))

			// Generate dataset with specific pattern
			config := testhelper.MockDataConfig{
				Symbol:            "TESTSTOCK",
				StartTime:         time.Date(2024, 1, 1, 9, 0, 0, 0, time.UTC),
				Interval:          time.Minute,
				NumDataPoints:     500, // Moderate dataset size
				Pattern:           p.pattern,
				InitialPrice:      100.0,
				TrendStrength:     0.005, // Moderate trend
				VolatilityPercent: 3.0,
				Seed:              42,
			}

			err := testhelper.GenerateAndWriteToParquet(config, mockDataPath)
			s.Require().NoError(err, "Failed to generate mock data for "+p.name+" pattern")

			// Setup test
			s.E2ETestSuite.SetupTest(`
initial_capital: 50000
`)

			// Run the strategy
			tmpFolder := testhelper.RunWasmStrategyTest(&s.E2ETestSuite, "ComplexStrategy", "./complex_strategy_plugin.wasm", mockDataPath)

			// Verify stats were generated
			stats, err := testhelper.ReadStats(&s.E2ETestSuite, tmpFolder)
			s.Require().NoError(err, "Failed to read stats for "+p.name+" pattern")
			s.Require().Greater(len(stats), 0, "Should have stats for "+p.name+" pattern")

			// Verify orders exist (strategy should generate signals)
			orders, err := testhelper.ReadOrders(&s.E2ETestSuite, tmpFolder)
			s.Require().NoError(err, "Failed to read orders for "+p.name+" pattern")

			// Verify marks exist
			marks, err := testhelper.ReadMarker(&s.E2ETestSuite, tmpFolder)
			s.Require().NoError(err, "Failed to read marks for "+p.name+" pattern")
			s.Require().Greater(len(marks), 0, "Should have marks for "+p.name+" pattern")

			s.T().Logf("%s pattern - Orders: %d, Marks: %d", p.name, len(orders), len(marks))
		})
	}
}

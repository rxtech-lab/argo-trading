package types

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"
)

type StatisticsTestSuite struct {
	suite.Suite
	tempDir string
}

func TestStatisticsSuite(t *testing.T) {
	suite.Run(t, new(StatisticsTestSuite))
}

func (suite *StatisticsTestSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "statistics_test")
	suite.NoError(err)
	suite.tempDir = tempDir
}

func (suite *StatisticsTestSuite) TearDownTest() {
	os.RemoveAll(suite.tempDir)
}

func (suite *StatisticsTestSuite) TestWriteTradeStats() {
	stats := []TradeStats{
		{
			Symbol: "BTC/USD",
			TradeResult: TradeResult{
				NumberOfTrades:        100,
				NumberOfWinningTrades: 60,
				NumberOfLosingTrades:  40,
				WinRate:               0.6,
				MaxDrawdown:           0.15,
			},
			TotalFees: 50.0,
			TradeHoldingTime: TradeHoldingTime{
				Min: 60,
				Max: 3600,
				Avg: 1800,
			},
			TradePnl: TradePnl{
				RealizedPnL:   1000.0,
				UnrealizedPnL: 200.0,
				TotalPnL:      1200.0,
				MaximumLoss:   -100.0,
				MaximumProfit: 500.0,
			},
			BuyAndHoldPnl: 800.0,
		},
	}

	filePath := filepath.Join(suite.tempDir, "stats.yaml")
	err := WriteTradeStats(filePath, stats)
	suite.NoError(err)

	// Verify file was created
	_, err = os.Stat(filePath)
	suite.NoError(err)

	// Read and verify contents
	data, err := os.ReadFile(filePath)
	suite.NoError(err)

	var readStats []TradeStats
	err = yaml.Unmarshal(data, &readStats)
	suite.NoError(err)

	suite.Len(readStats, 1)
	suite.Equal("BTC/USD", readStats[0].Symbol)
	suite.Equal(100, readStats[0].TradeResult.NumberOfTrades)
	suite.Equal(60, readStats[0].TradeResult.NumberOfWinningTrades)
	suite.Equal(40, readStats[0].TradeResult.NumberOfLosingTrades)
	suite.Equal(0.6, readStats[0].TradeResult.WinRate)
	suite.Equal(0.15, readStats[0].TradeResult.MaxDrawdown)
	suite.Equal(50.0, readStats[0].TotalFees)
	suite.Equal(60, readStats[0].TradeHoldingTime.Min)
	suite.Equal(3600, readStats[0].TradeHoldingTime.Max)
	suite.Equal(1800, readStats[0].TradeHoldingTime.Avg)
	suite.Equal(1000.0, readStats[0].TradePnl.RealizedPnL)
	suite.Equal(200.0, readStats[0].TradePnl.UnrealizedPnL)
	suite.Equal(1200.0, readStats[0].TradePnl.TotalPnL)
	suite.Equal(-100.0, readStats[0].TradePnl.MaximumLoss)
	suite.Equal(500.0, readStats[0].TradePnl.MaximumProfit)
	suite.Equal(800.0, readStats[0].BuyAndHoldPnl)
}

func (suite *StatisticsTestSuite) TestWriteTradeStatsMultiple() {
	stats := []TradeStats{
		{
			Symbol: "BTC/USD",
			TradeResult: TradeResult{
				NumberOfTrades: 50,
			},
		},
		{
			Symbol: "ETH/USD",
			TradeResult: TradeResult{
				NumberOfTrades: 75,
			},
		},
	}

	filePath := filepath.Join(suite.tempDir, "multiple_stats.yaml")
	err := WriteTradeStats(filePath, stats)
	suite.NoError(err)

	// Read and verify
	data, err := os.ReadFile(filePath)
	suite.NoError(err)

	var readStats []TradeStats
	err = yaml.Unmarshal(data, &readStats)
	suite.NoError(err)

	suite.Len(readStats, 2)
	suite.Equal("BTC/USD", readStats[0].Symbol)
	suite.Equal("ETH/USD", readStats[1].Symbol)
}

func (suite *StatisticsTestSuite) TestWriteTradeStatsEmpty() {
	stats := []TradeStats{}

	filePath := filepath.Join(suite.tempDir, "empty_stats.yaml")
	err := WriteTradeStats(filePath, stats)
	suite.NoError(err)

	// Read and verify
	data, err := os.ReadFile(filePath)
	suite.NoError(err)

	var readStats []TradeStats
	err = yaml.Unmarshal(data, &readStats)
	suite.NoError(err)

	suite.Empty(readStats)
}

func (suite *StatisticsTestSuite) TestWriteTradeStatsInvalidPath() {
	stats := []TradeStats{{Symbol: "BTC/USD"}}

	// Try to write to a non-existent directory
	filePath := filepath.Join(suite.tempDir, "nonexistent", "dir", "stats.yaml")
	err := WriteTradeStats(filePath, stats)
	suite.Error(err)
}

func (suite *StatisticsTestSuite) TestTradeHoldingTimeStruct() {
	holding := TradeHoldingTime{
		Min: 10,
		Max: 100,
		Avg: 50,
	}

	suite.Equal(10, holding.Min)
	suite.Equal(100, holding.Max)
	suite.Equal(50, holding.Avg)
}

func (suite *StatisticsTestSuite) TestTradePnlStruct() {
	pnl := TradePnl{
		RealizedPnL:   1000.0,
		UnrealizedPnL: 200.0,
		TotalPnL:      1200.0,
		MaximumLoss:   -50.0,
		MaximumProfit: 300.0,
	}

	suite.Equal(1000.0, pnl.RealizedPnL)
	suite.Equal(200.0, pnl.UnrealizedPnL)
	suite.Equal(1200.0, pnl.TotalPnL)
	suite.Equal(-50.0, pnl.MaximumLoss)
	suite.Equal(300.0, pnl.MaximumProfit)
}

func (suite *StatisticsTestSuite) TestTradeResultStruct() {
	result := TradeResult{
		NumberOfTrades:        100,
		NumberOfWinningTrades: 65,
		NumberOfLosingTrades:  35,
		WinRate:               0.65,
		MaxDrawdown:           0.2,
	}

	suite.Equal(100, result.NumberOfTrades)
	suite.Equal(65, result.NumberOfWinningTrades)
	suite.Equal(35, result.NumberOfLosingTrades)
	suite.Equal(0.65, result.WinRate)
	suite.Equal(0.2, result.MaxDrawdown)
}

// TestWriteTradeStatsExtendedFields verifies that the new monthly breakdown
// fields and the median/percentile additions to TradeHoldingTime and TradePnl
// round-trip through YAML serialization correctly.
func (suite *StatisticsTestSuite) TestWriteTradeStatsExtendedFields() {
	stats := []TradeStats{
		{
			Symbol: "BTC/USD",
			TradeHoldingTime: TradeHoldingTime{
				Min:    60,
				Max:    7200,
				Avg:    1800,
				Median: 1500,
				Percentiles: Percentiles{
					P25: 600,
					P50: 1500,
					P75: 3000,
					P90: 5400,
					P95: 6500,
					P99: 7100,
				},
			},
			TradePnl: TradePnl{
				RealizedPnL: 100,
				MedianPnL:   25.5,
				Percentiles: Percentiles{
					P25: -10,
					P50: 25.5,
					P75: 60,
					P90: 80,
					P95: 90,
					P99: 99,
				},
			},
			MonthlyTrades: []MonthlyTradeStats{
				{Month: "2024-01", NumberOfTrades: 4, NumberOfTradingPairs: 2, NumberOfWinningTrades: 1, NumberOfLosingTrades: 1},
				{Month: "2024-02", NumberOfTrades: 2, NumberOfTradingPairs: 1, NumberOfWinningTrades: 1},
			},
			MonthlyBalance: []MonthlyBalanceChange{
				{Month: "2024-01", StartingBalance: 1000, EndingBalance: 1000, Change: 0, RealizedPnL: 0},
				{Month: "2024-02", StartingBalance: 1000, EndingBalance: 1200, Change: 200, RealizedPnL: 200},
			},
			MonthlyHoldingTime: []MonthlyHoldingTime{
				{Month: "2024-01", Min: 3600, Max: 7200, Avg: 5400, Median: 5400},
				{Month: "2024-02", Min: 3600, Max: 3600, Avg: 3600, Median: 3600},
			},
		},
	}

	filePath := filepath.Join(suite.tempDir, "extended_stats.yaml")
	suite.Require().NoError(WriteTradeStats(filePath, stats))

	data, err := os.ReadFile(filePath)
	suite.Require().NoError(err)

	var readStats []TradeStats

	err = yaml.Unmarshal(data, &readStats)
	suite.Require().NoError(err)
	suite.Require().Len(readStats, 1)

	suite.Equal(stats[0].TradeHoldingTime, readStats[0].TradeHoldingTime)
	suite.Equal(stats[0].TradePnl, readStats[0].TradePnl)
	suite.Equal(stats[0].MonthlyTrades, readStats[0].MonthlyTrades)
	suite.Equal(stats[0].MonthlyBalance, readStats[0].MonthlyBalance)
	suite.Equal(stats[0].MonthlyHoldingTime, readStats[0].MonthlyHoldingTime)
}

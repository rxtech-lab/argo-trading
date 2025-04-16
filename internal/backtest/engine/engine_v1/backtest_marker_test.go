package engine

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// BacktestMarkerTestSuite is a test suite for BacktestMarker
type BacktestMarkerTestSuite struct {
	suite.Suite
	marker *BacktestMarker
	logger *logger.Logger
	tempDir string
}

// SetupSuite runs once before all tests in the suite
func (suite *BacktestMarkerTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
	
	// Create a temporary directory for the test
	tempDir, err := os.MkdirTemp("", "backtest-marker-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

// TearDownSuite runs once after all tests in the suite
func (suite *BacktestMarkerTestSuite) TearDownSuite() {
	// Clean up the temporary directory
	os.RemoveAll(suite.tempDir)
}

// SetupTest runs before each test
func (suite *BacktestMarkerTestSuite) SetupTest() {
	// Create a new marker before each test
	marker, err := NewBacktestMarker(suite.logger)
	suite.Require().NoError(err)
	suite.marker = marker
}

// TearDownTest runs after each test
func (suite *BacktestMarkerTestSuite) TearDownTest() {
	// Close the marker after each test
	if suite.marker != nil {
		suite.marker.Close()
	}
}

// TestBacktestMarkerSuite runs the test suite
func TestBacktestMarkerSuite(t *testing.T) {
	suite.Run(t, new(BacktestMarkerTestSuite))
}

// TestMarkAndGetMarks tests marking a point and retrieving marks
func (suite *BacktestMarkerTestSuite) TestMarkAndGetMarks() {
	// Create test data
	now := time.Now()
	marketData := types.MarketData{
		Symbol: "BTC/USD",
		Time:   now,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	signal := types.Signal{
		Time:   now,
		Type:   types.SignalTypeBuyLong,
		Name:   "TestSignal",
		Reason: "Test signal reason",
		Symbol: "BTC/USD",
	}
	
	reason := "Testing marker"
	
	// Mark the point
	err := suite.marker.Mark(marketData, signal, reason)
	suite.Require().NoError(err)
	
	// Get the marks
	marks, err := suite.marker.GetMarks()
	suite.Require().NoError(err)
	suite.Require().Len(marks, 1)
	
	// Verify the mark
	mark := marks[0]
	suite.Equal(types.SignalTypeBuyLong, mark.Signal.Type)
	suite.Equal("TestSignal", mark.Signal.Name)
	suite.Equal(reason, mark.Reason)
}

// TestMultipleMarks tests marking multiple points
func (suite *BacktestMarkerTestSuite) TestMultipleMarks() {
	// Create test data for multiple marks
	now := time.Now()
	
	marketData1 := types.MarketData{
		Symbol: "BTC/USD",
		Time:   now,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	signal1 := types.Signal{
		Time:   now,
		Type:   types.SignalTypeBuyLong,
		Name:   "BuySignal",
		Reason: "Buy signal reason",
		Symbol: "BTC/USD",
	}
	
	marketData2 := types.MarketData{
		Symbol: "BTC/USD",
		Time:   now.Add(time.Hour),
		Open:   50500.0,
		High:   52000.0,
		Low:    50000.0,
		Close:  51500.0,
		Volume: 1500.0,
	}
	
	signal2 := types.Signal{
		Time:   now.Add(time.Hour),
		Type:   types.SignalTypeSellLong,
		Name:   "SellSignal",
		Reason: "Sell signal reason",
		Symbol: "BTC/USD",
	}
	
	// Mark the points
	err := suite.marker.Mark(marketData1, signal1, "First mark")
	suite.Require().NoError(err)
	
	err = suite.marker.Mark(marketData2, signal2, "Second mark")
	suite.Require().NoError(err)
	
	// Get the marks
	marks, err := suite.marker.GetMarks()
	suite.Require().NoError(err)
	suite.Require().Len(marks, 2)
	
	// Verify the marks are in order (by timestamp)
	suite.Equal(types.SignalTypeBuyLong, marks[0].Signal.Type)
	suite.Equal("BuySignal", marks[0].Signal.Name)
	suite.Equal("First mark", marks[0].Reason)
	
	suite.Equal(types.SignalTypeSellLong, marks[1].Signal.Type)
	suite.Equal("SellSignal", marks[1].Signal.Name)
	suite.Equal("Second mark", marks[1].Reason)
}

// TestCleanup tests the cleanup functionality
func (suite *BacktestMarkerTestSuite) TestCleanup() {
	// Create test data
	now := time.Now()
	marketData := types.MarketData{
		Symbol: "BTC/USD",
		Time:   now,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	signal := types.Signal{
		Time:   now,
		Type:   types.SignalTypeBuyLong,
		Name:   "TestSignal",
		Reason: "Test signal reason",
		Symbol: "BTC/USD",
	}
	
	// Mark a point
	err := suite.marker.Mark(marketData, signal, "Test cleanup")
	suite.Require().NoError(err)
	
	// Verify we have a mark
	marks, err := suite.marker.GetMarks()
	suite.Require().NoError(err)
	suite.Require().Len(marks, 1)
	
	// Cleanup the marker
	err = suite.marker.Cleanup()
	suite.Require().NoError(err)
	
	// Verify we have no marks
	marks, err = suite.marker.GetMarks()
	suite.Require().NoError(err)
	suite.Require().Len(marks, 0)
}

// TestWrite tests writing marks to a file
func (suite *BacktestMarkerTestSuite) TestWrite() {
	// Create test data
	now := time.Now()
	marketData := types.MarketData{
		Symbol: "BTC/USD",
		Time:   now,
		Open:   50000.0,
		High:   51000.0,
		Low:    49000.0,
		Close:  50500.0,
		Volume: 1000.0,
	}
	
	signal := types.Signal{
		Time:   now,
		Type:   types.SignalTypeBuyLong,
		Name:   "TestSignal",
		Reason: "Test signal reason",
		Symbol: "BTC/USD",
	}
	
	// Mark a point
	err := suite.marker.Mark(marketData, signal, "Test write")
	suite.Require().NoError(err)
	
	// Write the marks to a file
	outputPath := filepath.Join(suite.tempDir, "test-marks")
	err = suite.marker.Write(outputPath)
	suite.Require().NoError(err)
	
	// Verify the file exists
	_, err = os.Stat(filepath.Join(outputPath, "marks.parquet"))
	suite.Require().NoError(err)
} 
package engine

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// BacktestMarkerTestSuite is a test suite for BacktestMarker
type BacktestMarkerTestSuite struct {
	suite.Suite
	marker *BacktestMarker
	logger *logger.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *BacktestMarkerTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger

	marker, err := NewBacktestMarker(suite.logger)
	suite.Require().NoError(err)
	suite.marker = marker
}

// TearDownSuite runs once after all tests in the suite
func (suite *BacktestMarkerTestSuite) TearDownSuite() {
	if suite.marker != nil {
		suite.marker.Close()
	}
}

// SetupTest runs before each test
func (suite *BacktestMarkerTestSuite) SetupTest() {
	// Cleanup before running each test
	err := suite.marker.Cleanup()
	suite.Require().NoError(err)
}

// TestMarkAndGet tests the Mark and GetMarks methods
func (suite *BacktestMarkerTestSuite) TestMarkAndGet() {
	// Sample market data
	marketData := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
		Open:   150.0,
		High:   152.0,
		Low:    149.0,
		Close:  151.0,
		Volume: 1000.0,
	}

	// Test cases
	testCases := []struct {
		name string
		mark types.Mark
	}{
		{
			name: "Basic mark with signal",
			mark: types.Mark{
				MarketDataId: "md123",
				Color:        "green",
				Shape:        types.MarkShapeCircle,
				Level:        types.MarkLevelInfo,
				Title:        "Buy Signal",
				Message:      "Strong buy signal detected",
				Category:     "trade",
				Signal: optional.Some(types.Signal{
					Type:   types.SignalTypeBuyLong,
					Name:   "TestSignal",
					Time:   time.Date(2023, 1, 1, 10, 0, 0, 0, time.UTC),
					Symbol: "AAPL",
				}),
			},
		},
		{
			name: "Mark with different shape",
			mark: types.Mark{
				MarketDataId: "md456",
				Color:        "red",
				Shape:        types.MarkShapeTriangle,
				Level:        types.MarkLevelWarning,
				Title:        "Sell Signal",
				Message:      "Bearish pattern detected",
				Category:     "analysis",
				Signal: optional.Some(types.Signal{
					Type:   types.SignalTypeSellLong,
					Name:   "SellSignal",
					Time:   time.Date(2023, 1, 2, 11, 0, 0, 0, time.UTC),
					Symbol: "MSFT",
				}),
			},
		},
	}

	// Insert test marks
	for _, tc := range testCases {
		err := suite.marker.Mark(marketData, tc.mark)
		suite.Require().NoError(err, "Failed to insert mark for case: %s", tc.name)
	}

	// Get all marks
	marks, err := suite.marker.GetMarks()
	suite.Require().NoError(err, "Failed to get marks")
	suite.Require().Equal(len(testCases), len(marks), "Number of marks does not match test cases")

	// Verify marks
	for i, tc := range testCases {
		suite.Run(tc.name, func() {
			mark := marks[i]
			suite.Equal(tc.mark.MarketDataId, mark.MarketDataId)
			suite.Equal(tc.mark.Color, mark.Color)
			suite.Equal(tc.mark.Shape, mark.Shape)
			suite.Equal(tc.mark.Level, mark.Level)
			suite.Equal(tc.mark.Title, mark.Title)
			suite.Equal(tc.mark.Message, mark.Message)
			suite.Equal(tc.mark.Category, mark.Category)

			// Verify signal
			suite.True(mark.Signal.IsSome(), "Signal should be present")
			signalVal, err := mark.Signal.Take()
			suite.Require().NoError(err, "Taking signal value should not error")

			tcSignal, _ := tc.mark.Signal.Take()
			suite.Equal(string(tcSignal.Type), string(signalVal.Type))
			suite.Equal(tcSignal.Name, signalVal.Name)
			suite.Equal(tcSignal.Time.UTC(), signalVal.Time.UTC())
			suite.Equal(tcSignal.Symbol, signalVal.Symbol)
		})
	}
}

// TestWriteToParquet tests the Write method
func (suite *BacktestMarkerTestSuite) TestWriteToParquet() {
	// Sample market data
	marketData := types.MarketData{
		Symbol: "GOOG",
		Time:   time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
		Open:   120.0,
		High:   122.0,
		Low:    119.0,
		Close:  121.5,
		Volume: 2000.0,
	}

	// Create a mark
	mark := types.Mark{
		MarketDataId: "md789",
		Color:        "blue",
		Shape:        types.MarkShapeSquare,
		Level:        types.MarkLevelError,
		Title:        "Export Test",
		Message:      "Testing parquet export",
		Category:     "test",
		Signal: optional.Some(types.Signal{
			Type:   types.SignalTypeBuyLong,
			Name:   "ExportTest",
			Time:   time.Date(2023, 1, 3, 12, 0, 0, 0, time.UTC),
			Symbol: "GOOG",
		}),
	}

	// Add test mark
	err := suite.marker.Mark(marketData, mark)
	suite.Require().NoError(err)

	// Create a temporary directory for the test
	tempDir := suite.T().TempDir()

	// Export to parquet
	err = suite.marker.Write(tempDir)
	suite.Require().NoError(err, "Failed to write marks to parquet")
}

// TestBacktestMarkerSuite runs the test suite
func TestBacktestMarkerSuite(t *testing.T) {
	suite.Run(t, new(BacktestMarkerTestSuite))
}

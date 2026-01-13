package prefetch

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/logger"
	"github.com/rxtech-lab/argo-trading/internal/trading/engine"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/rxtech-lab/argo-trading/mocks"
	"github.com/rxtech-lab/argo-trading/pkg/marketdata/writer"
	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"
)

type PrefetchManagerTestSuite struct {
	suite.Suite
	logger *logger.Logger
}

func (s *PrefetchManagerTestSuite) SetupSuite() {
	log, err := logger.NewLogger()
	s.Require().NoError(err)
	s.logger = log
}

func TestPrefetchManagerTestSuite(t *testing.T) {
	suite.Run(t, new(PrefetchManagerTestSuite))
}

func (s *PrefetchManagerTestSuite) TestParseIntervalDuration() {
	testCases := []struct {
		interval string
		expected time.Duration
	}{
		{"1s", time.Second},
		{"1m", time.Minute},
		{"3m", 3 * time.Minute},
		{"5m", 5 * time.Minute},
		{"15m", 15 * time.Minute},
		{"30m", 30 * time.Minute},
		{"1h", time.Hour},
		{"2h", 2 * time.Hour},
		{"4h", 4 * time.Hour},
		{"6h", 6 * time.Hour},
		{"8h", 8 * time.Hour},
		{"12h", 12 * time.Hour},
		{"1d", 24 * time.Hour},
		{"3d", 72 * time.Hour},
		{"1w", 168 * time.Hour},
		{"unknown", time.Minute}, // Default
	}

	for _, tc := range testCases {
		result := parseIntervalDuration(tc.interval)
		s.Equal(tc.expected, result, "Failed for interval: %s", tc.interval)
	}
}

func (s *PrefetchManagerTestSuite) TestIntervalToMultiplier() {
	testCases := []struct {
		interval string
		expected int
	}{
		{"1s", 1},
		{"1m", 1},
		{"3m", 3},
		{"5m", 5},
		{"15m", 15},
		{"30m", 30},
		{"1h", 1},
		{"2h", 2},
		{"4h", 4},
		{"6h", 6},
		{"8h", 8},
		{"12h", 12},
		{"1d", 1},
		{"3d", 3},
		{"1w", 1},
		{"unknown", 1}, // Default
	}

	for _, tc := range testCases {
		result := intervalToMultiplier(tc.interval)
		s.Equal(tc.expected, result, "Failed for interval: %s", tc.interval)
	}
}

func (s *PrefetchManagerTestSuite) TestIsEnabled() {
	pm := NewPrefetchManager(s.logger)

	// Not initialized - should be disabled
	s.False(pm.IsEnabled())
}

func (s *PrefetchManagerTestSuite) TestCalculateStartTime_Days() {
	pm := NewPrefetchManager(s.logger)
	pm.config = engine.PrefetchConfig{
		Enabled:       true,
		StartTimeType: "days",
		Days:          30,
	}

	startTime := pm.calculateStartTime()
	expectedStart := time.Now().AddDate(0, 0, -30)

	// Allow 1 second tolerance
	diff := startTime.Sub(expectedStart)
	s.True(diff < time.Second && diff > -time.Second)
}

func (s *PrefetchManagerTestSuite) TestCalculateStartTime_Date() {
	pm := NewPrefetchManager(s.logger)

	specificDate := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	pm.config = engine.PrefetchConfig{
		Enabled:       true,
		StartTimeType: "date",
		StartTime:     specificDate,
	}

	startTime := pm.calculateStartTime()
	s.Equal(specificDate, startTime)
}

func (s *PrefetchManagerTestSuite) TestGetConfig() {
	pm := NewPrefetchManager(s.logger)
	pm.config = engine.PrefetchConfig{
		Enabled:       true,
		StartTimeType: "days",
		Days:          14,
	}

	config := pm.GetConfig()
	s.True(config.Enabled)
	s.Equal("days", config.StartTimeType)
	s.Equal(14, config.Days)
}

func (s *PrefetchManagerTestSuite) TestExecutePrefetch_Disabled() {
	pm := NewPrefetchManager(s.logger)
	pm.config = engine.PrefetchConfig{
		Enabled: false,
	}

	// Should return nil without doing anything
	err := pm.ExecutePrefetch(nil, []string{"BTCUSDT"})
	s.NoError(err)
}

// ============================================================================
// intervalToTimespan Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestIntervalToTimespan() {
	testCases := []struct {
		interval string
		expected models.Timespan
	}{
		{"1s", models.Second},
		{"1m", models.Minute},
		{"3m", models.Minute},
		{"5m", models.Minute},
		{"15m", models.Minute},
		{"30m", models.Minute},
		{"1h", models.Hour},
		{"2h", models.Hour},
		{"4h", models.Hour},
		{"6h", models.Hour},
		{"8h", models.Hour},
		{"12h", models.Hour},
		{"1d", models.Day},
		{"3d", models.Day},
		{"1w", models.Week},
		{"unknown", models.Minute}, // Default
	}

	for _, tc := range testCases {
		result := intervalToTimespan(tc.interval)
		s.Equal(tc.expected, result, "Failed for interval: %s", tc.interval)
	}
}

// ============================================================================
// ExecutePrefetch Tests with MockProvider
// ============================================================================

func (s *PrefetchManagerTestSuite) TestExecutePrefetch_Enabled_Success() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	// Create temp directory for test
	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	// Create streaming writer
	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")
	err = streamingWriter.Initialize()
	s.Require().NoError(err)
	defer streamingWriter.Close()

	// Create mock provider
	mockProvider := mocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().ConfigWriter(gomock.Any()).Times(2)
	mockProvider.EXPECT().Download(
		gomock.Any(),  // ctx
		"BTCUSDT",     // ticker
		gomock.Any(),  // startDate
		gomock.Any(),  // endDate
		1,             // multiplier
		models.Minute, // timespan
		nil,           // onProgress
	).Return("", nil)
	mockProvider.EXPECT().Download(
		gomock.Any(),
		"ETHUSDT",
		gomock.Any(),
		gomock.Any(),
		1,
		models.Minute,
		nil,
	).Return("", nil)

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		nil,
	)

	ctx := context.Background()
	err = pm.ExecutePrefetch(ctx, []string{"BTCUSDT", "ETHUSDT"})
	s.NoError(err)
}

func (s *PrefetchManagerTestSuite) TestExecutePrefetch_Enabled_PartialFailure() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")
	err = streamingWriter.Initialize()
	s.Require().NoError(err)
	defer streamingWriter.Close()

	mockProvider := mocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().ConfigWriter(gomock.Any()).Times(2)
	// First symbol succeeds
	mockProvider.EXPECT().Download(
		gomock.Any(),
		"BTCUSDT",
		gomock.Any(),
		gomock.Any(),
		1,
		models.Minute,
		nil,
	).Return("", nil)
	// Second symbol fails
	mockProvider.EXPECT().Download(
		gomock.Any(),
		"ETHUSDT",
		gomock.Any(),
		gomock.Any(),
		1,
		models.Minute,
		nil,
	).Return("", fmt.Errorf("download failed"))

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		nil,
	)

	ctx := context.Background()
	// Should not return error - continues with other symbols
	err = pm.ExecutePrefetch(ctx, []string{"BTCUSDT", "ETHUSDT"})
	s.NoError(err)
}

// ============================================================================
// HandleStreamStart Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestHandleStreamStart_Disabled() {
	pm := NewPrefetchManager(s.logger)
	pm.config = engine.PrefetchConfig{
		Enabled: false,
	}

	ctx := context.Background()
	err := pm.HandleStreamStart(ctx, time.Now(), []string{"BTCUSDT"})
	s.NoError(err)
}

func (s *PrefetchManagerTestSuite) TestHandleStreamStart_NoStoredData() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")

	mockProvider := mocks.NewMockProvider(ctrl)
	// No Download expected since there's no stored data to compare

	var statusUpdates []types.EngineStatus
	onStatusUpdate := engine.OnStatusUpdateCallback(func(status types.EngineStatus) error {
		statusUpdates = append(statusUpdates, status)
		return nil
	})

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		&onStatusUpdate,
	)

	ctx := context.Background()
	err = pm.HandleStreamStart(ctx, time.Now(), []string{"BTCUSDT"})
	s.NoError(err)

	// Should transition to Running status
	s.Contains(statusUpdates, types.EngineStatusRunning)
}

// ============================================================================
// GetLastStoredTimestamp Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestGetLastStoredTimestamp_FileNotExists() {
	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = "/nonexistent/path/data.parquet"

	_, err := pm.GetLastStoredTimestamp("BTCUSDT")
	s.Error(err)
	s.Contains(err.Error(), "parquet file does not exist")
}

func (s *PrefetchManagerTestSuite) TestGetLastStoredTimestamp_Success() {
	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	parquetPath := filepath.Join(tempDir, "test_data.parquet")

	// Create a parquet file with test data using DuckDB
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	// Create table and insert test data
	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT,
			symbol TEXT,
			time TIMESTAMP,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	s.Require().NoError(err)

	_, err = db.Exec(`
		INSERT INTO market_data VALUES
		('1', 'BTCUSDT', ?, 50000, 50100, 49900, 50050, 100),
		('2', 'BTCUSDT', ?, 50050, 50200, 50000, 50150, 150)
	`, testTime, testTime.Add(time.Minute))
	s.Require().NoError(err)

	// Export to parquet
	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, parquetPath))
	s.Require().NoError(err)

	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = parquetPath

	lastTime, err := pm.GetLastStoredTimestamp("BTCUSDT")
	s.NoError(err)
	s.Equal(testTime.Add(time.Minute), lastTime)
}

func (s *PrefetchManagerTestSuite) TestGetLastStoredTimestamp_NoDataForSymbol() {
	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	parquetPath := filepath.Join(tempDir, "test_data.parquet")

	// Create a parquet file with data for a different symbol
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	testTime := time.Date(2025, 1, 15, 10, 30, 0, 0, time.UTC)
	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT,
			symbol TEXT,
			time TIMESTAMP,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	s.Require().NoError(err)

	_, err = db.Exec(`
		INSERT INTO market_data VALUES
		('1', 'ETHUSDT', ?, 3000, 3010, 2990, 3005, 100)
	`, testTime)
	s.Require().NoError(err)

	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, parquetPath))
	s.Require().NoError(err)

	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = parquetPath

	_, err = pm.GetLastStoredTimestamp("BTCUSDT")
	s.Error(err)
	s.Contains(err.Error(), "no data found for symbol")
}

// ============================================================================
// DetectGap Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestDetectGap_NoStoredData() {
	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = "/nonexistent/path/data.parquet"
	pm.gapToleranceUnit = time.Minute

	gap, err := pm.DetectGap(time.Now(), "BTCUSDT")
	s.NoError(err) // No error when no stored data
	s.Equal(time.Duration(0), gap)
}

func (s *PrefetchManagerTestSuite) TestDetectGap_WithinTolerance() {
	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	parquetPath := filepath.Join(tempDir, "test_data.parquet")

	// Create parquet file with recent data
	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	storedTime := time.Now().Add(-30 * time.Second) // 30 seconds ago
	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT,
			symbol TEXT,
			time TIMESTAMP,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	s.Require().NoError(err)

	_, err = db.Exec(`INSERT INTO market_data VALUES ('1', 'BTCUSDT', ?, 50000, 50100, 49900, 50050, 100)`, storedTime)
	s.Require().NoError(err)

	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, parquetPath))
	s.Require().NoError(err)

	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = parquetPath
	pm.gapToleranceUnit = time.Minute // 2 * 1min = 2min tolerance

	// Stream time is 1 minute after stored - within tolerance
	streamTime := storedTime.Add(time.Minute)
	gap, err := pm.DetectGap(streamTime, "BTCUSDT")
	s.NoError(err)
	s.Equal(time.Duration(0), gap) // Within tolerance, no gap
}

func (s *PrefetchManagerTestSuite) TestDetectGap_ExceedsTolerance() {
	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	parquetPath := filepath.Join(tempDir, "test_data.parquet")

	db, err := sql.Open("duckdb", ":memory:")
	s.Require().NoError(err)
	defer db.Close()

	storedTime := time.Now().Add(-10 * time.Minute) // 10 minutes ago
	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT,
			symbol TEXT,
			time TIMESTAMP,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	s.Require().NoError(err)

	_, err = db.Exec(`INSERT INTO market_data VALUES ('1', 'BTCUSDT', ?, 50000, 50100, 49900, 50050, 100)`, storedTime)
	s.Require().NoError(err)

	_, err = db.Exec(fmt.Sprintf(`COPY market_data TO '%s' (FORMAT PARQUET)`, parquetPath))
	s.Require().NoError(err)

	pm := NewPrefetchManager(s.logger)
	pm.parquetPath = parquetPath
	pm.gapToleranceUnit = time.Minute // 2 * 1min = 2min tolerance

	// Stream time is now - 10 minutes gap exceeds 2 min tolerance
	gap, err := pm.DetectGap(time.Now(), "BTCUSDT")
	s.NoError(err)
	s.Greater(gap, time.Duration(0)) // Gap detected
}

// ============================================================================
// FillGap Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestFillGap_Success() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")
	err = streamingWriter.Initialize()
	s.Require().NoError(err)
	defer streamingWriter.Close()

	mockProvider := mocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().ConfigWriter(gomock.Any()).Times(1)
	mockProvider.EXPECT().Download(
		gomock.Any(),
		"BTCUSDT",
		gomock.Any(),
		gomock.Any(),
		1,
		models.Minute,
		nil,
	).Return("", nil)

	var statusUpdates []types.EngineStatus
	onStatusUpdate := engine.OnStatusUpdateCallback(func(status types.EngineStatus) error {
		statusUpdates = append(statusUpdates, status)
		return nil
	})

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		&onStatusUpdate,
	)

	ctx := context.Background()
	from := time.Now().Add(-time.Hour)
	to := time.Now()

	err = pm.FillGap(ctx, "BTCUSDT", from, to)
	s.NoError(err)

	// Should emit GapFilling status
	s.Contains(statusUpdates, types.EngineStatusGapFilling)
}

func (s *PrefetchManagerTestSuite) TestFillGap_DownloadFailure() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")
	err = streamingWriter.Initialize()
	s.Require().NoError(err)
	defer streamingWriter.Close()

	mockProvider := mocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().ConfigWriter(gomock.Any()).Times(1)
	mockProvider.EXPECT().Download(
		gomock.Any(),
		"BTCUSDT",
		gomock.Any(),
		gomock.Any(),
		1,
		models.Minute,
		nil,
	).Return("", fmt.Errorf("network error"))

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		nil,
	)

	ctx := context.Background()
	from := time.Now().Add(-time.Hour)
	to := time.Now()

	err = pm.FillGap(ctx, "BTCUSDT", from, to)
	s.Error(err)
	s.Contains(err.Error(), "failed to fill gap")
}

// ============================================================================
// Status Callback Tests
// ============================================================================

func (s *PrefetchManagerTestSuite) TestExecutePrefetch_EmitsStatus() {
	ctrl := gomock.NewController(s.T())
	defer ctrl.Finish()

	tempDir, err := os.MkdirTemp("", "prefetch_test_*")
	s.Require().NoError(err)
	defer os.RemoveAll(tempDir)

	streamingWriter := writer.NewStreamingDuckDBWriter(tempDir, "test", "1m")
	err = streamingWriter.Initialize()
	s.Require().NoError(err)
	defer streamingWriter.Close()

	mockProvider := mocks.NewMockProvider(ctrl)
	mockProvider.EXPECT().ConfigWriter(gomock.Any()).Times(1)
	mockProvider.EXPECT().Download(
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
		gomock.Any(),
	).Return("", nil)

	var statusUpdates []types.EngineStatus
	onStatusUpdate := engine.OnStatusUpdateCallback(func(status types.EngineStatus) error {
		statusUpdates = append(statusUpdates, status)
		return nil
	})

	pm := NewPrefetchManager(s.logger)
	pm.Initialize(
		engine.PrefetchConfig{
			Enabled:       true,
			StartTimeType: "days",
			Days:          7,
		},
		mockProvider,
		streamingWriter,
		"1m",
		&onStatusUpdate,
	)

	ctx := context.Background()
	err = pm.ExecutePrefetch(ctx, []string{"BTCUSDT"})
	s.NoError(err)

	// Should emit Prefetching status
	s.Contains(statusUpdates, types.EngineStatusPrefetching)
}

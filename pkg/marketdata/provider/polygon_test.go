package provider

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// mockPolygonAPIClient implements PolygonAPIClient for testing.
type mockPolygonAPIClient struct {
	iterator PolygonAggsIterator
}

func (m *mockPolygonAPIClient) ListAggs(_ context.Context, _ *models.ListAggsParams, _ ...models.RequestOption) PolygonAggsIterator {
	return m.iterator
}

// mockPolygonIterator implements PolygonAggsIterator for testing.
type mockPolygonIterator struct {
	aggs   []models.Agg
	index  int
	err    error
	errOnN int // Return error after N calls to Next() (0 means return error immediately on Err())
}

func (m *mockPolygonIterator) Next() bool {
	if m.index < len(m.aggs) {
		m.index++
		return true
	}
	return false
}

func (m *mockPolygonIterator) Item() models.Agg {
	if m.index > 0 && m.index <= len(m.aggs) {
		return m.aggs[m.index-1]
	}
	return models.Agg{}
}

func (m *mockPolygonIterator) Err() error {
	return m.err
}

type PolygonClientTestSuite struct {
	suite.Suite
}

func TestPolygonClientSuite(t *testing.T) {
	suite.Run(t, new(PolygonClientTestSuite))
}

func (suite *PolygonClientTestSuite) TestNewPolygonClient_ValidApiKey() {
	client, err := NewPolygonClient("test-api-key")
	suite.NoError(err)
	suite.NotNil(client)

	polygonClient, ok := client.(*PolygonClient)
	suite.True(ok)
	suite.NotNil(polygonClient.apiClient)
	suite.Nil(polygonClient.writer)
}

func (suite *PolygonClientTestSuite) TestNewPolygonClientWithAPI() {
	mockAPI := &mockPolygonAPIClient{}
	client := NewPolygonClientWithAPI(mockAPI)
	suite.NotNil(client)
	suite.Equal(mockAPI, client.apiClient)
	suite.Nil(client.writer)
}

func (suite *PolygonClientTestSuite) TestNewPolygonClient_EmptyApiKey() {
	client, err := NewPolygonClient("")
	suite.Error(err)
	suite.Nil(client)
	suite.Contains(err.Error(), "apiKey is required")
}

func (suite *PolygonClientTestSuite) TestPolygonClient_ConfigWriter() {
	client, err := NewPolygonClient("test-api-key")
	suite.Require().NoError(err)

	polygonClient := client.(*PolygonClient)
	suite.Nil(polygonClient.writer)

	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}
	polygonClient.ConfigWriter(mockW)
	suite.Equal(mockW, polygonClient.writer)
}

func (suite *PolygonClientTestSuite) TestPolygonClient_Download_WithoutWriter() {
	client, err := NewPolygonClient("test-api-key")
	suite.Require().NoError(err)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err = client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "no writer configured")
}

func (suite *PolygonClientTestSuite) TestPolygonClient_Download_WriterInitializeError() {
	client, err := NewPolygonClient("test-api-key")
	suite.Require().NoError(err)

	polygonClient := client.(*PolygonClient)
	mockW := &mockWriter{initializeErr: errors.New("initialization failed")}
	polygonClient.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err = client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to initialize writer")
}

// TestDownloadSuccess tests a successful download with mock API.
func (suite *PolygonClientTestSuite) TestDownloadSuccess() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 31, 0, 0, time.UTC)),
			Open:      100.5,
			High:      102.0,
			Low:       100.0,
			Close:     101.5,
			Volume:    1500000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	path, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/test.parquet", path)
	suite.Len(mockW.writtenData, 2)
	suite.True(mockW.initialized)

	// Verify first data point
	suite.Equal("SPY", mockW.writtenData[0].Symbol)
	suite.InDelta(100.0, mockW.writtenData[0].Open, 0.01)
	suite.InDelta(101.0, mockW.writtenData[0].High, 0.01)
	suite.InDelta(99.0, mockW.writtenData[0].Low, 0.01)
	suite.InDelta(100.5, mockW.writtenData[0].Close, 0.01)
	suite.InDelta(1000000, mockW.writtenData[0].Volume, 0.01)
}

// TestDownloadEmptyAggs tests download when API returns no data.
func (suite *PolygonClientTestSuite) TestDownloadEmptyAggs() {
	mockIter := &mockPolygonIterator{aggs: []models.Agg{}}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/empty.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	path, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/empty.parquet", path)
	suite.Len(mockW.writtenData, 0)
}

// TestDownloadIteratorError tests error handling when iterator returns an error.
func (suite *PolygonClientTestSuite) TestDownloadIteratorError() {
	mockIter := &mockPolygonIterator{
		aggs: []models.Agg{},
		err:  errors.New("API rate limit exceeded"),
	}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "error iterating polygon aggregates")
	suite.Contains(err.Error(), "API rate limit exceeded")
}

// TestDownloadWriteError tests error handling when writer returns an error.
func (suite *PolygonClientTestSuite) TestDownloadWriteError() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{writeErr: errors.New("disk full")}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to write data")
}

// TestDownloadFinalizeError tests error handling when finalize fails.
func (suite *PolygonClientTestSuite) TestDownloadFinalizeError() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{finalizeErr: errors.New("finalize failed")}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to finalize writer")
}

// TestDownloadCloseError tests error handling when writer close fails.
func (suite *PolygonClientTestSuite) TestDownloadCloseError() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{
		outputPath: "/tmp/test.parquet",
		closeErr:   errors.New("close failed"),
	}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "error closing writer")
}

// TestDownloadManyDataPoints tests downloading many data points with progress updates.
func (suite *PolygonClientTestSuite) TestDownloadManyDataPoints() {
	// Create 1500 data points to test the progress bar update logic (updates every 1000)
	aggs := make([]models.Agg, 1500)
	baseTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	for i := 0; i < 1500; i++ {
		aggs[i] = models.Agg{
			Timestamp: models.Millis(baseTime.Add(time.Duration(i) * time.Minute)),
			Open:      100.0 + float64(i)*0.01,
			High:      101.0 + float64(i)*0.01,
			Low:       99.0 + float64(i)*0.01,
			Close:     100.5 + float64(i)*0.01,
			Volume:    1000000,
		}
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/large.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	progressCalled := false
	path, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {
		progressCalled = true
	})
	suite.NoError(err)
	suite.Equal("/tmp/large.parquet", path)
	suite.Len(mockW.writtenData, 1500)
	suite.True(progressCalled)
}

// TestDownloadProgressCallback tests that progress callback is called with correct values.
func (suite *PolygonClientTestSuite) TestDownloadProgressCallback() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	progressCalled := false
	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {
		progressCalled = true
		suite.GreaterOrEqual(total, float64(0))
		suite.Contains(message, "SPY")
	})
	suite.NoError(err)
	suite.True(progressCalled)
}

// TestDownloadProgressPercentage tests that progress percentage is calculated correctly based on time.
// This test ensures the fix for the 3000% progress issue is working correctly.
func (suite *PolygonClientTestSuite) TestDownloadProgressPercentage() {
	// Create many data points (1500 minutes) across a 2-day period
	// This simulates downloading minute-level data which previously caused 3000% progress
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC) // 2 days
	baseTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)

	aggs := make([]models.Agg, 1500)
	for i := 0; i < 1500; i++ {
		aggs[i] = models.Agg{
			Timestamp: models.Millis(baseTime.Add(time.Duration(i) * time.Minute)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		}
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	var maxPercentage float64
	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {
		percentage := (current / total) * 100
		if percentage > maxPercentage {
			maxPercentage = percentage
		}
		// Progress should never exceed 100%
		suite.LessOrEqual(percentage, 100.0, "Progress percentage should never exceed 100%%")
		// Current should never exceed total
		suite.LessOrEqual(current, total, "Current progress should never exceed total")
	})
	suite.NoError(err)
	suite.Greater(maxPercentage, 0.0, "Progress should be reported")
}

// TestDownloadWriterCloseWithExistingError tests that close error is logged when another error exists.
func (suite *PolygonClientTestSuite) TestDownloadWriterCloseWithExistingError() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{
		writeErr: errors.New("write failed"),
		closeErr: errors.New("close also failed"),
	}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	// The error should be about the write failure, not the close failure
	suite.Contains(err.Error(), "failed to write data")
	// Close should have been called despite write error
	suite.Equal(1, mockW.closeCallCount)
}

// mockWriterForPolygon is a polygon-specific mock to use in tests that verifies data transformation.
type mockWriterForPolygon struct {
	mockWriter
	receivedData []types.MarketData
}

func (m *mockWriterForPolygon) Write(data types.MarketData) error {
	m.writeCallCount++
	if m.writeErr != nil {
		return m.writeErr
	}
	m.receivedData = append(m.receivedData, data)
	m.writtenData = append(m.writtenData, data)
	return nil
}

// TestDownloadDataTransformation tests that Polygon data is correctly transformed to MarketData.
func (suite *PolygonClientTestSuite) TestDownloadDataTransformation() {
	timestamp := time.Date(2024, 1, 15, 14, 30, 0, 0, time.UTC)
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(timestamp),
			Open:      150.25,
			High:      155.50,
			Low:       149.00,
			Close:     154.75,
			Volume:    2500000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriterForPolygon{
		mockWriter: mockWriter{outputPath: "/tmp/transformed.parquet"},
	}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(context.Background(), "AAPL", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Len(mockW.receivedData, 1)

	data := mockW.receivedData[0]
	suite.Equal("", data.Id)
	suite.Equal("AAPL", data.Symbol)
	suite.Equal(timestamp, data.Time)
	suite.InDelta(150.25, data.Open, 0.001)
	suite.InDelta(155.50, data.High, 0.001)
	suite.InDelta(149.00, data.Low, 0.001)
	suite.InDelta(154.75, data.Close, 0.001)
	suite.InDelta(2500000, data.Volume, 0.001)
}

// TestDownloadIteratorError_DeletesFileWhenNoData verifies that the output file is deleted
// when an iterator error occurs and no data was written.
func (suite *PolygonClientTestSuite) TestDownloadIteratorError_DeletesFileWhenNoData() {
	// Create a temporary file to simulate the output file
	tmpFile, err := os.CreateTemp("", "polygon_test_*.parquet")
	suite.Require().NoError(err)
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Verify temp file exists
	_, err = os.Stat(tmpPath)
	suite.Require().NoError(err, "temp file should exist before test")

	mockIter := &mockPolygonIterator{
		aggs: []models.Agg{},
		err:  errors.New("API rate limit exceeded"),
	}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: tmpPath}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err = client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "error iterating polygon aggregates")

	// Verify temp file was deleted
	_, err = os.Stat(tmpPath)
	suite.True(os.IsNotExist(err), "temp file should be deleted when error occurs with no data")
}

// TestDownloadWriteError_DeletesFileWhenNoData verifies that the output file is deleted
// when a write error occurs on the first record (no data written yet).
func (suite *PolygonClientTestSuite) TestDownloadWriteError_DeletesFileWhenNoData() {
	// Create a temporary file to simulate the output file
	tmpFile, err := os.CreateTemp("", "polygon_test_*.parquet")
	suite.Require().NoError(err)
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Verify temp file exists
	_, err = os.Stat(tmpPath)
	suite.Require().NoError(err, "temp file should exist before test")

	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{
		outputPath: tmpPath,
		writeErr:   errors.New("disk full"),
	}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err = client.Download(context.Background(), "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to write data")

	// Verify temp file was deleted
	_, err = os.Stat(tmpPath)
	suite.True(os.IsNotExist(err), "temp file should be deleted when write error occurs with no data")
}

// TestDownload_Cancellation tests that download can be cancelled via context.
func (suite *PolygonClientTestSuite) TestDownload_Cancellation() {
	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download(ctx, "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.ErrorIs(err, context.Canceled)
}

// TestDownload_CancellationCleansUpFile tests that file is deleted when cancelled with no data written.
func (suite *PolygonClientTestSuite) TestDownload_CancellationCleansUpFile() {
	// Create a temporary file to simulate the output file
	tmpFile, err := os.CreateTemp("", "polygon_cancel_test_*.parquet")
	suite.Require().NoError(err)
	tmpPath := tmpFile.Name()
	tmpFile.Close()

	// Verify temp file exists
	_, err = os.Stat(tmpPath)
	suite.Require().NoError(err, "temp file should exist before test")

	aggs := []models.Agg{
		{
			Timestamp: models.Millis(time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)),
			Open:      100.0,
			High:      101.0,
			Low:       99.0,
			Close:     100.5,
			Volume:    1000000,
		},
	}

	mockIter := &mockPolygonIterator{aggs: aggs}
	mockAPI := &mockPolygonAPIClient{iterator: mockIter}
	mockW := &mockWriter{outputPath: tmpPath}

	client := NewPolygonClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	// Create a cancelled context
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err = client.Download(ctx, "SPY", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.ErrorIs(err, context.Canceled)

	// Verify temp file was deleted
	_, err = os.Stat(tmpPath)
	suite.True(os.IsNotExist(err), "temp file should be deleted when cancelled with no data written")
}

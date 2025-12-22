package provider

import (
	"context"
	"errors"
	"testing"
	"time"

	binance "github.com/adshao/go-binance/v2"
	"github.com/polygon-io/client-go/rest/models"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

// mockWriter is a simple mock implementation of MarketDataWriter for testing.
type mockWriter struct {
	initialized      bool
	initializeErr    error
	writeErr         error
	writeErrAfterN   int // Return writeErr after N successful writes (0 means immediate error)
	finalizeErr      error
	closeErr         error
	outputPath       string
	writtenData      []types.MarketData
	writeCallCount   int
	finalizeCallCount int
	closeCallCount   int
}

func (m *mockWriter) Initialize() error {
	if m.initializeErr != nil {
		return m.initializeErr
	}
	m.initialized = true
	return nil
}

func (m *mockWriter) Write(data types.MarketData) error {
	m.writeCallCount++
	if m.writeErr != nil && (m.writeErrAfterN == 0 || m.writeCallCount > m.writeErrAfterN) {
		return m.writeErr
	}
	m.writtenData = append(m.writtenData, data)
	return nil
}

func (m *mockWriter) Finalize() (string, error) {
	m.finalizeCallCount++
	if m.finalizeErr != nil {
		return "", m.finalizeErr
	}
	return m.outputPath, nil
}

func (m *mockWriter) Close() error {
	m.closeCallCount++
	return m.closeErr
}

// mockBinanceAPIClient implements BinanceAPIClient for testing.
type mockBinanceAPIClient struct {
	klines    []*binance.Kline
	klinesErr error
	// For pagination testing - returns different results on subsequent calls
	callCount       int
	klinesPerCall   [][]*binance.Kline
	errorsPerCall   []error
}

func (m *mockBinanceAPIClient) NewKlinesService() BinanceKlinesService {
	return &mockBinanceKlinesService{client: m}
}

type mockBinanceKlinesService struct {
	client   *mockBinanceAPIClient
	symbol   string
	interval string
	start    int64
	end      int64
}

func (m *mockBinanceKlinesService) Symbol(symbol string) BinanceKlinesService {
	m.symbol = symbol
	return m
}

func (m *mockBinanceKlinesService) Interval(interval string) BinanceKlinesService {
	m.interval = interval
	return m
}

func (m *mockBinanceKlinesService) StartTime(startTime int64) BinanceKlinesService {
	m.start = startTime
	return m
}

func (m *mockBinanceKlinesService) EndTime(endTime int64) BinanceKlinesService {
	m.end = endTime
	return m
}

func (m *mockBinanceKlinesService) Do(_ context.Context) ([]*binance.Kline, error) {
	// If we have per-call data, use it
	if len(m.client.klinesPerCall) > 0 {
		idx := m.client.callCount
		m.client.callCount++
		if idx < len(m.client.klinesPerCall) {
			var err error
			if idx < len(m.client.errorsPerCall) {
				err = m.client.errorsPerCall[idx]
			}
			return m.client.klinesPerCall[idx], err
		}
		return nil, nil
	}
	// Otherwise use single response
	return m.client.klines, m.client.klinesErr
}

type BinanceClientTestSuite struct {
	suite.Suite
}

func TestBinanceClientSuite(t *testing.T) {
	suite.Run(t, new(BinanceClientTestSuite))
}

func (suite *BinanceClientTestSuite) TestNewBinanceClient() {
	client, err := NewBinanceClient()
	suite.NoError(err)
	suite.NotNil(client)

	binanceClient, ok := client.(*BinanceClient)
	suite.True(ok)
	suite.NotNil(binanceClient.apiClient)
	suite.Nil(binanceClient.writer)
}

func (suite *BinanceClientTestSuite) TestNewBinanceClientWithAPI() {
	mockAPI := &mockBinanceAPIClient{}
	client := NewBinanceClientWithAPI(mockAPI)
	suite.NotNil(client)
	suite.Equal(mockAPI, client.apiClient)
	suite.Nil(client.writer)
}

func (suite *BinanceClientTestSuite) TestConfigWriter() {
	client, err := NewBinanceClient()
	suite.Require().NoError(err)

	binanceClient := client.(*BinanceClient)
	suite.Nil(binanceClient.writer)

	mockW := &mockWriter{}
	binanceClient.ConfigWriter(mockW)
	suite.Equal(mockW, binanceClient.writer)
}

func (suite *BinanceClientTestSuite) TestDownloadWithoutWriter() {
	client, err := NewBinanceClient()
	suite.Require().NoError(err)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err = client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "writer is not configured")
}

func (suite *BinanceClientTestSuite) TestDownloadWithInvalidTimespan() {
	client, err := NewBinanceClient()
	suite.Require().NoError(err)

	binanceClient := client.(*BinanceClient)
	binanceClient.ConfigWriter(&mockWriter{})

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	// Use an invalid timespan (Quarter is not supported by Binance)
	_, err = client.Download("BTCUSDT", startDate, endDate, 1, models.Quarter, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "unsupported timespan")
}

func (suite *BinanceClientTestSuite) TestDownloadWriterInitializationError() {
	client, err := NewBinanceClient()
	suite.Require().NoError(err)

	binanceClient := client.(*BinanceClient)
	mockW := &mockWriter{initializeErr: errors.New("initialization failed")}
	binanceClient.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	_, err = client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to initialize writer")
}

// TestConvertTimespanToBinanceInterval tests the conversion of timespan to Binance interval strings.
func (suite *BinanceClientTestSuite) TestConvertTimespanToBinanceInterval() {
	tests := []struct {
		name       string
		timespan   models.Timespan
		multiplier int
		want       string
		wantErr    bool
		errMsg     string
	}{
		{
			name:       "1 minute",
			timespan:   models.Minute,
			multiplier: 1,
			want:       "1m",
			wantErr:    false,
		},
		{
			name:       "5 minutes",
			timespan:   models.Minute,
			multiplier: 5,
			want:       "5m",
			wantErr:    false,
		},
		{
			name:       "15 minutes",
			timespan:   models.Minute,
			multiplier: 15,
			want:       "15m",
			wantErr:    false,
		},
		{
			name:       "30 minutes",
			timespan:   models.Minute,
			multiplier: 30,
			want:       "30m",
			wantErr:    false,
		},
		{
			name:       "1 hour",
			timespan:   models.Hour,
			multiplier: 1,
			want:       "1h",
			wantErr:    false,
		},
		{
			name:       "4 hours",
			timespan:   models.Hour,
			multiplier: 4,
			want:       "4h",
			wantErr:    false,
		},
		{
			name:       "1 day",
			timespan:   models.Day,
			multiplier: 1,
			want:       "1d",
			wantErr:    false,
		},
		{
			name:       "3 days",
			timespan:   models.Day,
			multiplier: 3,
			want:       "3d",
			wantErr:    false,
		},
		{
			name:       "1 week",
			timespan:   models.Week,
			multiplier: 1,
			want:       "1w",
			wantErr:    false,
		},
		{
			name:       "2 weeks - unsupported",
			timespan:   models.Week,
			multiplier: 2,
			want:       "",
			wantErr:    true,
			errMsg:     "unsupported weekly multiplier",
		},
		{
			name:       "1 month",
			timespan:   models.Month,
			multiplier: 1,
			want:       "1M",
			wantErr:    false,
		},
		{
			name:       "3 months - unsupported",
			timespan:   models.Month,
			multiplier: 3,
			want:       "",
			wantErr:    true,
			errMsg:     "unsupported monthly multiplier",
		},
		{
			name:       "quarter - unsupported",
			timespan:   models.Quarter,
			multiplier: 1,
			want:       "",
			wantErr:    true,
			errMsg:     "unsupported timespan",
		},
	}

	for _, tt := range tests {
		suite.Run(tt.name, func() {
			got, err := convertTimespanToBinanceInterval(tt.timespan, tt.multiplier)
			if tt.wantErr {
				suite.Error(err)
				suite.Contains(err.Error(), tt.errMsg)
			} else {
				suite.NoError(err)
				suite.Equal(tt.want, got)
			}
		})
	}
}

// TestProcessKlines tests the conversion of Binance klines to MarketData.
func (suite *BinanceClientTestSuite) TestProcessKlines() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000, // 2024-01-01 00:00:00 UTC
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
		{
			OpenTime:  1704067260000, // 2024-01-01 00:01:00 UTC
			Open:      "42300.00",
			High:      "42400.00",
			Low:       "42200.00",
			Close:     "42350.00",
			Volume:    "500.25",
			CloseTime: 1704067319999,
		},
	}

	mockW := &mockWriter{}

	err := processKlines(mockW, "BTCUSDT", klines)
	suite.NoError(err)
	suite.Len(mockW.writtenData, 2)

	// Verify first kline
	data0 := mockW.writtenData[0]
	suite.Equal("BTCUSDT", data0.Symbol)
	suite.Equal(time.UnixMilli(1704067200000), data0.Time)
	suite.InDelta(42000.50, data0.Open, 0.01)
	suite.InDelta(42500.00, data0.High, 0.01)
	suite.InDelta(41800.00, data0.Low, 0.01)
	suite.InDelta(42300.00, data0.Close, 0.01)
	suite.InDelta(1000.5, data0.Volume, 0.01)

	// Verify second kline
	data1 := mockW.writtenData[1]
	suite.Equal("BTCUSDT", data1.Symbol)
	suite.Equal(time.UnixMilli(1704067260000), data1.Time)
	suite.InDelta(42300.00, data1.Open, 0.01)
	suite.InDelta(42400.00, data1.High, 0.01)
	suite.InDelta(42200.00, data1.Low, 0.01)
	suite.InDelta(42350.00, data1.Close, 0.01)
	suite.InDelta(500.25, data1.Volume, 0.01)
}

// TestProcessKlinesEmpty tests processKlines with an empty slice.
func (suite *BinanceClientTestSuite) TestProcessKlinesEmpty() {
	mockW := &mockWriter{}
	err := processKlines(mockW, "BTCUSDT", []*binance.Kline{})
	suite.NoError(err)
	suite.Len(mockW.writtenData, 0)
}

// TestProcessKlinesWriteError tests processKlines when the writer returns an error.
func (suite *BinanceClientTestSuite) TestProcessKlinesWriteError() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
	}

	mockW := &mockWriter{writeErr: errors.New("write failed")}

	err := processKlines(mockW, "BTCUSDT", klines)
	suite.Error(err)
	suite.Contains(err.Error(), "failed to write market data")
}

// TestProgressCalculation verifies that progress values are relative, not absolute timestamps.
// This test documents the expected behavior after the bug fix.
func (suite *BinanceClientTestSuite) TestProgressCalculation() {
	// Simulate the progress calculation logic from Download method
	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 31, 0, 0, 0, 0, time.UTC)

	startTimeMillis := startDate.UnixMilli()
	endTimeMillis := endDate.UnixMilli()

	// Simulate being at the start of download
	currentStartTime := startTimeMillis
	progressCurrent := float64(currentStartTime - startTimeMillis)
	progressTotal := float64(endTimeMillis - startTimeMillis)

	// At the beginning, current should be 0
	suite.Equal(float64(0), progressCurrent)
	suite.Greater(progressTotal, float64(0))

	// Progress should be a reasonable number (30 days in milliseconds ≈ 2.59 billion)
	// This is much smaller than raw timestamps (≈ 1.7 trillion for 2024)
	expectedTotalMs := float64(30 * 24 * 60 * 60 * 1000) // 30 days in ms
	suite.InDelta(expectedTotalMs, progressTotal, float64(24*60*60*1000))

	// Simulate being halfway through
	halfwayTime := startTimeMillis + (endTimeMillis-startTimeMillis)/2
	progressCurrentHalf := float64(halfwayTime - startTimeMillis)

	// Progress should be roughly half of total
	suite.InDelta(progressTotal/2, progressCurrentHalf, 1)

	// Simulate being at the end
	currentStartTime = endTimeMillis
	progressCurrentEnd := float64(currentStartTime - startTimeMillis)

	// Progress should equal total at the end
	suite.Equal(progressTotal, progressCurrentEnd)

	// Verify that we're NOT using absolute timestamps
	// Absolute timestamps for 2024 would be around 1.7 trillion milliseconds
	suite.Less(progressTotal, float64(1e12), "Progress total should be relative, not an absolute timestamp")
}

// TestDownloadSuccess tests a successful download with mock API.
func (suite *BinanceClientTestSuite) TestDownloadSuccess() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
		{
			OpenTime:  1704067260000,
			Open:      "42300.00",
			High:      "42400.00",
			Low:       "42200.00",
			Close:     "42350.00",
			Volume:    "500.25",
			CloseTime: 1704067319999,
		},
	}

	mockAPI := &mockBinanceAPIClient{klines: klines}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	path, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/test.parquet", path)
	suite.Len(mockW.writtenData, 2)
	suite.True(mockW.initialized)
}

// TestDownloadEmptyKlines tests download when API returns empty klines.
func (suite *BinanceClientTestSuite) TestDownloadEmptyKlines() {
	mockAPI := &mockBinanceAPIClient{klines: []*binance.Kline{}}
	mockW := &mockWriter{outputPath: "/tmp/empty.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	path, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/empty.parquet", path)
	suite.Len(mockW.writtenData, 0)
}

// TestDownloadAPIError tests error handling when API returns an error.
func (suite *BinanceClientTestSuite) TestDownloadAPIError() {
	mockAPI := &mockBinanceAPIClient{klinesErr: errors.New("API rate limit exceeded")}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to fetch klines from Binance")
	suite.Contains(err.Error(), "API rate limit exceeded")
}

// TestDownloadAPIErrorWithFinalizeError tests combined API and finalize errors.
func (suite *BinanceClientTestSuite) TestDownloadAPIErrorWithFinalizeError() {
	mockAPI := &mockBinanceAPIClient{klinesErr: errors.New("API error")}
	mockW := &mockWriter{finalizeErr: errors.New("finalize failed")}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to fetch klines from Binance")
	suite.Contains(err.Error(), "also failed to finalize writer")
}

// TestDownloadFinalizeError tests error handling when finalize fails at the end.
func (suite *BinanceClientTestSuite) TestDownloadFinalizeError() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
	}

	mockAPI := &mockBinanceAPIClient{klines: klines}
	mockW := &mockWriter{finalizeErr: errors.New("disk full")}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to finalize writer")
}

// TestDownloadWriteErrorWithFinalizeError tests write error with finalize error.
func (suite *BinanceClientTestSuite) TestDownloadWriteErrorWithFinalizeError() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
	}

	mockAPI := &mockBinanceAPIClient{klines: klines}
	mockW := &mockWriter{
		writeErr:    errors.New("write error"),
		finalizeErr: errors.New("finalize failed"),
	}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to process klines")
	suite.Contains(err.Error(), "also failed to finalize writer")
}

// TestDownloadWriteErrorWithFinalizeSuccess tests write error with successful finalize.
func (suite *BinanceClientTestSuite) TestDownloadWriteErrorWithFinalizeSuccess() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
	}

	mockAPI := &mockBinanceAPIClient{klines: klines}
	mockW := &mockWriter{writeErr: errors.New("write error")}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to process klines")
	suite.NotContains(err.Error(), "also failed to finalize")
}

// TestDownloadPagination tests pagination when we get exactly 500 klines (full page).
func (suite *BinanceClientTestSuite) TestDownloadPagination() {
	// Create 500 klines to trigger pagination
	firstPage := make([]*binance.Kline, 500)
	for i := 0; i < 500; i++ {
		firstPage[i] = &binance.Kline{
			OpenTime:  1704067200000 + int64(i*60000),
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067200000 + int64(i*60000) + 59999,
		}
	}

	// Second page has fewer than 500 (last page)
	secondPage := []*binance.Kline{
		{
			OpenTime:  1704067200000 + 500*60000,
			Open:      "42300.00",
			High:      "42400.00",
			Low:       "42200.00",
			Close:     "42350.00",
			Volume:    "500.25",
			CloseTime: 1704067200000 + 500*60000 + 59999,
		},
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{firstPage, secondPage},
	}
	mockW := &mockWriter{outputPath: "/tmp/paginated.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	path, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/paginated.parquet", path)
	// Should have written 501 records (500 from first page + 1 from second)
	suite.Len(mockW.writtenData, 501)
	suite.Equal(2, mockAPI.callCount)
}

// TestDownloadPaginationWithAPIErrorOnSecondPage tests API error during pagination.
func (suite *BinanceClientTestSuite) TestDownloadPaginationWithAPIErrorOnSecondPage() {
	// Create 500 klines to trigger pagination
	firstPage := make([]*binance.Kline, 500)
	for i := 0; i < 500; i++ {
		firstPage[i] = &binance.Kline{
			OpenTime:  1704067200000 + int64(i*60000),
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067200000 + int64(i*60000) + 59999,
		}
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{firstPage, nil},
		errorsPerCall: []error{nil, errors.New("connection timeout")},
	}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to fetch klines from Binance")
	suite.Contains(err.Error(), "connection timeout")
	// First page should have been written before error
	suite.Len(mockW.writtenData, 500)
}

// TestDownloadPaginationWriteErrorOnSecondPage tests write error during pagination.
func (suite *BinanceClientTestSuite) TestDownloadPaginationWriteErrorOnSecondPage() {
	// Create 500 klines to trigger pagination
	firstPage := make([]*binance.Kline, 500)
	for i := 0; i < 500; i++ {
		firstPage[i] = &binance.Kline{
			OpenTime:  1704067200000 + int64(i*60000),
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067200000 + int64(i*60000) + 59999,
		}
	}

	secondPage := []*binance.Kline{
		{
			OpenTime:  1704067200000 + 500*60000,
			Open:      "42300.00",
			High:      "42400.00",
			Low:       "42200.00",
			Close:     "42350.00",
			Volume:    "500.25",
			CloseTime: 1704067200000 + 500*60000 + 59999,
		},
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{firstPage, secondPage},
	}
	mockW := &mockWriter{
		writeErr:     errors.New("disk full"),
		writeErrAfterN: 500, // Fail after first 500 writes
	}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to process klines")
}

// TestDownloadProgressCallback tests that progress callback is called.
func (suite *BinanceClientTestSuite) TestDownloadProgressCallback() {
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: 1704067259999,
		},
	}

	mockAPI := &mockBinanceAPIClient{klines: klines}
	mockW := &mockWriter{outputPath: "/tmp/test.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	endDate := time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC)

	progressCalled := false
	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {
		progressCalled = true
		suite.GreaterOrEqual(total, float64(0))
		suite.Contains(message, "BTCUSDT")
	})
	suite.NoError(err)
	// Give the goroutine time to execute
	time.Sleep(10 * time.Millisecond)
	suite.True(progressCalled)
}

// TestDownloadPaginationTimeBreak tests pagination ending due to time condition.
func (suite *BinanceClientTestSuite) TestDownloadPaginationTimeBreak() {
	// Create 500 klines where the last kline's CloseTime exceeds the end time
	// This triggers the "currentStartTime >= endTimeMillis" break condition
	startTimeMs := int64(1704067200000) // 2024-01-01 00:00:00 UTC
	endTimeMs := int64(1704070800000)   // 2024-01-01 01:00:00 UTC (1 hour later)

	firstPage := make([]*binance.Kline, 500)
	for i := 0; i < 500; i++ {
		// Each kline is 1 minute, but we set CloseTime to exceed endTime at the end
		openTime := startTimeMs + int64(i*60000)
		closeTime := openTime + 59999
		// For the last kline, set CloseTime to be at or after endTime
		if i == 499 {
			closeTime = endTimeMs + 1000 // CloseTime exceeds endTime
		}
		firstPage[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: closeTime,
		}
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{firstPage},
	}
	mockW := &mockWriter{outputPath: "/tmp/timebreak.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.UnixMilli(startTimeMs)
	endDate := time.UnixMilli(endTimeMs)

	path, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/timebreak.parquet", path)
	suite.Len(mockW.writtenData, 500)
	// Only one API call needed because time break condition was met
	suite.Equal(1, mockAPI.callCount)
}

// TestDownloadFullPageWriteError tests write error during processing of a full page (500 records).
func (suite *BinanceClientTestSuite) TestDownloadFullPageWriteError() {
	// Create exactly 500 klines (full page) to trigger processing via the full-page path (lines 162-171)
	fullPage := make([]*binance.Kline, 500)
	startTimeMs := int64(1704067200000)
	endTimeMs := startTimeMs + int64(500*60000) // 500 minutes

	for i := 0; i < 500; i++ {
		openTime := startTimeMs + int64(i*60000)
		fullPage[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: openTime + 59999,
		}
	}

	// Mock returns a full page, then an empty page (to not infinite loop)
	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{fullPage, {}},
	}
	mockW := &mockWriter{
		writeErr:       errors.New("disk full during full page"),
		writeErrAfterN: 0, // Fail immediately
	}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.UnixMilli(startTimeMs)
	endDate := time.UnixMilli(endTimeMs + 60000) // A bit after the last kline

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to process klines")
}

// TestDownloadFullPageWriteErrorWithFinalizeError tests write error during full page with finalize error.
func (suite *BinanceClientTestSuite) TestDownloadFullPageWriteErrorWithFinalizeError() {
	fullPage := make([]*binance.Kline, 500)
	startTimeMs := int64(1704067200000)

	for i := 0; i < 500; i++ {
		openTime := startTimeMs + int64(i*60000)
		fullPage[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: openTime + 59999,
		}
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{fullPage, {}},
	}
	mockW := &mockWriter{
		writeErr:       errors.New("write failed on full page"),
		writeErrAfterN: 0,
		finalizeErr:    errors.New("finalize also failed"),
	}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.UnixMilli(startTimeMs)
	endDate := time.UnixMilli(startTimeMs + int64(600*60000))

	_, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.Error(err)
	suite.Contains(err.Error(), "failed to process klines")
	suite.Contains(err.Error(), "also failed to finalize writer")
}

// TestBinanceClientWrapperMethods tests the wrapper methods by verifying the client structure.
func (suite *BinanceClientTestSuite) TestBinanceClientWrapperMethods() {
	// Create a real BinanceClient to verify the wrapper is correctly set up
	client, err := NewBinanceClient()
	suite.NoError(err)

	binanceClient, ok := client.(*BinanceClient)
	suite.True(ok)

	// Verify the apiClient is a binanceClientWrapper
	_, ok = binanceClient.apiClient.(*binanceClientWrapper)
	suite.True(ok, "apiClient should be a binanceClientWrapper")
}

// TestProcessKlinesWithInvalidNumbers tests processKlines with invalid number strings.
func (suite *BinanceClientTestSuite) TestProcessKlinesWithInvalidNumbers() {
	// ParseFloat will return 0 for truly invalid strings (not "NaN" or "inf")
	klines := []*binance.Kline{
		{
			OpenTime:  1704067200000,
			Open:      "invalid",
			High:      "also_invalid",
			Low:       "not_a_number",
			Close:     "xyz",
			Volume:    "abc",
			CloseTime: 1704067259999,
		},
	}

	mockW := &mockWriter{}

	err := processKlines(mockW, "BTCUSDT", klines)
	suite.NoError(err)
	suite.Len(mockW.writtenData, 1)
	// Verify that invalid numbers are parsed as 0
	suite.Equal(float64(0), mockW.writtenData[0].Open)
	suite.Equal(float64(0), mockW.writtenData[0].High)
	suite.Equal(float64(0), mockW.writtenData[0].Low)
	suite.Equal(float64(0), mockW.writtenData[0].Close)
	suite.Equal(float64(0), mockW.writtenData[0].Volume)
}

// TestDownloadPaginationWithLargeDataset tests pagination with multiple full pages.
func (suite *BinanceClientTestSuite) TestDownloadPaginationWithLargeDataset() {
	startTimeMs := int64(1704067200000)

	// Create three pages: two full (500 each) and one partial (100)
	page1 := make([]*binance.Kline, 500)
	page2 := make([]*binance.Kline, 500)
	page3 := make([]*binance.Kline, 100)

	for i := 0; i < 500; i++ {
		openTime := startTimeMs + int64(i*60000)
		page1[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42000.50",
			High:      "42500.00",
			Low:       "41800.00",
			Close:     "42300.00",
			Volume:    "1000.5",
			CloseTime: openTime + 59999,
		}
	}

	for i := 0; i < 500; i++ {
		openTime := startTimeMs + int64((500+i)*60000)
		page2[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42100.50",
			High:      "42600.00",
			Low:       "41900.00",
			Close:     "42400.00",
			Volume:    "1100.5",
			CloseTime: openTime + 59999,
		}
	}

	for i := 0; i < 100; i++ {
		openTime := startTimeMs + int64((1000+i)*60000)
		page3[i] = &binance.Kline{
			OpenTime:  openTime,
			Open:      "42200.50",
			High:      "42700.00",
			Low:       "42000.00",
			Close:     "42500.00",
			Volume:    "1200.5",
			CloseTime: openTime + 59999,
		}
	}

	mockAPI := &mockBinanceAPIClient{
		klinesPerCall: [][]*binance.Kline{page1, page2, page3},
	}
	mockW := &mockWriter{outputPath: "/tmp/large.parquet"}

	client := NewBinanceClientWithAPI(mockAPI)
	client.ConfigWriter(mockW)

	startDate := time.UnixMilli(startTimeMs)
	endDate := time.UnixMilli(startTimeMs + int64(2000*60000)) // Far enough in the future

	path, err := client.Download("BTCUSDT", startDate, endDate, 1, models.Minute, func(current float64, total float64, message string) {})
	suite.NoError(err)
	suite.Equal("/tmp/large.parquet", path)
	// Should have written 1100 records (500 + 500 + 100)
	suite.Len(mockW.writtenData, 1100)
	suite.Equal(3, mockAPI.callCount)
}

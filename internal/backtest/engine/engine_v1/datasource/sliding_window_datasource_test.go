package datasource

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
)

// MockDataSource is a mock implementation of DataSource for testing
type MockDataSource struct {
	mock.Mock
}

func (m *MockDataSource) Initialize(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	args := m.Called(start, end)
	return args.Get(0).(func(yield func(types.MarketData, error) bool))
}

func (m *MockDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	args := m.Called(start, end, interval)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.MarketData), args.Error(1)
}

func (m *MockDataSource) GetPreviousNumberOfDataPoints(end time.Time, symbol string, count int) ([]types.MarketData, error) {
	args := m.Called(end, symbol, count)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]types.MarketData), args.Error(1)
}

func (m *MockDataSource) GetMarketData(symbol string, timestamp time.Time) (types.MarketData, error) {
	args := m.Called(symbol, timestamp)
	return args.Get(0).(types.MarketData), args.Error(1)
}

func (m *MockDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	args := m.Called(symbol)
	return args.Get(0).(types.MarketData), args.Error(1)
}

func (m *MockDataSource) ExecuteSQL(query string, params ...interface{}) ([]SQLResult, error) {
	args := m.Called(query, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]SQLResult), args.Error(1)
}

func (m *MockDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	args := m.Called(start, end)
	return args.Int(0), args.Error(1)
}

func (m *MockDataSource) Close() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockDataSource) GetAllSymbols() ([]string, error) {
	args := m.Called()
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]string), args.Error(1)
}

type SlidingWindowDataSourceTestSuite struct {
	suite.Suite
	mockDS *MockDataSource
	ds     *SlidingWindowDataSource
}

func TestSlidingWindowDataSourceTestSuite(t *testing.T) {
	suite.Run(t, new(SlidingWindowDataSourceTestSuite))
}

func (s *SlidingWindowDataSourceTestSuite) SetupTest() {
	s.mockDS = new(MockDataSource)
	s.ds = NewSlidingWindowDataSource(s.mockDS, 10)
}

func (s *SlidingWindowDataSourceTestSuite) createMarketData(symbol string, timeOffset int, close float64) types.MarketData {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	return types.MarketData{
		Id:     symbol + "_" + baseTime.Add(time.Duration(timeOffset)*time.Minute).Format("20060102150405"),
		Symbol: symbol,
		Time:   baseTime.Add(time.Duration(timeOffset) * time.Minute),
		Open:   close - 1,
		High:   close + 1,
		Low:    close - 2,
		Close:  close,
		Volume: 1000,
	}
}

func (s *SlidingWindowDataSourceTestSuite) TestNewSlidingWindowDataSource() {
	assert.NotNil(s.T(), s.ds)
	assert.NotNil(s.T(), s.ds.GetCache())
	assert.Equal(s.T(), 10, s.ds.GetCache().MaxSize())
}

func (s *SlidingWindowDataSourceTestSuite) TestAddToCache() {
	data := s.createMarketData("SPY", 0, 100)
	s.ds.AddToCache(data)

	assert.Equal(s.T(), 1, s.ds.GetCache().Size("SPY"))
}

func (s *SlidingWindowDataSourceTestSuite) TestGetRangeFromCache() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate cache
	for i := 0; i < 5; i++ {
		s.ds.AddToCache(s.createMarketData("SPY", i, float64(100+i)))
	}

	// Request data that's in cache - should NOT call underlying datasource
	result, err := s.ds.GetRange(baseTime, baseTime.Add(2*time.Minute), optional.None[Interval]())

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 3, len(result))
	assert.Equal(s.T(), 100.0, result[0].Close)
	assert.Equal(s.T(), 101.0, result[1].Close)
	assert.Equal(s.T(), 102.0, result[2].Close)

	// Mock should not have been called
	s.mockDS.AssertNotCalled(s.T(), "GetRange", mock.Anything, mock.Anything, mock.Anything)
}

func (s *SlidingWindowDataSourceTestSuite) TestGetRangeFallbackToUnderlying() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate cache with limited data
	s.ds.AddToCache(s.createMarketData("SPY", 2, 102))
	s.ds.AddToCache(s.createMarketData("SPY", 3, 103))

	// Request data that starts before cache coverage
	expectedResult := []types.MarketData{
		s.createMarketData("SPY", 0, 100),
		s.createMarketData("SPY", 1, 101),
		s.createMarketData("SPY", 2, 102),
	}
	s.mockDS.On("GetRange", baseTime, baseTime.Add(2*time.Minute), optional.None[Interval]()).Return(expectedResult, nil)

	result, err := s.ds.GetRange(baseTime, baseTime.Add(2*time.Minute), optional.None[Interval]())

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 3, len(result))
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestGetRangeWithIntervalBypassesCache() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate cache
	for i := 0; i < 5; i++ {
		s.ds.AddToCache(s.createMarketData("SPY", i, float64(100+i)))
	}

	// Request with interval - should bypass cache
	interval := optional.Some(Interval1h)
	expectedResult := []types.MarketData{s.createMarketData("SPY", 0, 100)}
	s.mockDS.On("GetRange", baseTime, baseTime.Add(2*time.Minute), interval).Return(expectedResult, nil)

	result, err := s.ds.GetRange(baseTime, baseTime.Add(2*time.Minute), interval)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(result))
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestGetPreviousNumberOfDataPointsFromCache() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate cache
	for i := 0; i < 5; i++ {
		s.ds.AddToCache(s.createMarketData("SPY", i, float64(100+i)))
	}

	// Request data from cache
	result, err := s.ds.GetPreviousNumberOfDataPoints(baseTime.Add(4*time.Minute), "SPY", 3)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 3, len(result))
	assert.Equal(s.T(), 102.0, result[0].Close)
	assert.Equal(s.T(), 103.0, result[1].Close)
	assert.Equal(s.T(), 104.0, result[2].Close)

	s.mockDS.AssertNotCalled(s.T(), "GetPreviousNumberOfDataPoints", mock.Anything, mock.Anything, mock.Anything)
}

func (s *SlidingWindowDataSourceTestSuite) TestGetPreviousNumberOfDataPointsFallback() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Populate cache with only 2 data points
	s.ds.AddToCache(s.createMarketData("SPY", 3, 103))
	s.ds.AddToCache(s.createMarketData("SPY", 4, 104))

	// Request 5 data points (more than cache has)
	expectedResult := []types.MarketData{
		s.createMarketData("SPY", 0, 100),
		s.createMarketData("SPY", 1, 101),
		s.createMarketData("SPY", 2, 102),
		s.createMarketData("SPY", 3, 103),
		s.createMarketData("SPY", 4, 104),
	}
	s.mockDS.On("GetPreviousNumberOfDataPoints", baseTime.Add(4*time.Minute), "SPY", 5).Return(expectedResult, nil)

	result, err := s.ds.GetPreviousNumberOfDataPoints(baseTime.Add(4*time.Minute), "SPY", 5)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 5, len(result))
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestGetMarketDataFromCache() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	s.ds.AddToCache(s.createMarketData("SPY", 0, 100))

	result, err := s.ds.GetMarketData("SPY", baseTime)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 100.0, result.Close)
	s.mockDS.AssertNotCalled(s.T(), "GetMarketData", mock.Anything, mock.Anything)
}

func (s *SlidingWindowDataSourceTestSuite) TestGetMarketDataFallback() {
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Cache is empty
	expectedData := s.createMarketData("SPY", 0, 100)
	s.mockDS.On("GetMarketData", "SPY", baseTime).Return(expectedData, nil)

	result, err := s.ds.GetMarketData("SPY", baseTime)

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 100.0, result.Close)
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestReadLastDataFromCache() {
	s.ds.AddToCache(s.createMarketData("SPY", 0, 100))
	s.ds.AddToCache(s.createMarketData("SPY", 1, 101))
	s.ds.AddToCache(s.createMarketData("SPY", 2, 102))

	result, err := s.ds.ReadLastData("SPY")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 102.0, result.Close)
	s.mockDS.AssertNotCalled(s.T(), "ReadLastData", mock.Anything)
}

func (s *SlidingWindowDataSourceTestSuite) TestReadLastDataFallback() {
	expectedData := s.createMarketData("SPY", 0, 100)
	s.mockDS.On("ReadLastData", "SPY").Return(expectedData, nil)

	result, err := s.ds.ReadLastData("SPY")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 100.0, result.Close)
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestExecuteSQLBypassesCache() {
	expectedResult := []SQLResult{{Values: map[string]interface{}{"count": 100}}}
	s.mockDS.On("ExecuteSQL", "SELECT COUNT(*) FROM market_data", []interface{}(nil)).Return(expectedResult, nil)

	result, err := s.ds.ExecuteSQL("SELECT COUNT(*) FROM market_data")

	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 1, len(result))
	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestClearCache() {
	s.ds.AddToCache(s.createMarketData("SPY", 0, 100))
	assert.Equal(s.T(), 1, s.ds.GetCache().TotalSize())

	s.ds.ClearCache()
	assert.Equal(s.T(), 0, s.ds.GetCache().TotalSize())
}

func (s *SlidingWindowDataSourceTestSuite) TestPassthroughMethods() {
	// Test Initialize
	s.mockDS.On("Initialize", "/path/to/data").Return(nil)
	err := s.ds.Initialize("/path/to/data")
	assert.NoError(s.T(), err)

	// Test Count
	s.mockDS.On("Count", optional.None[time.Time](), optional.None[time.Time]()).Return(100, nil)
	count, err := s.ds.Count(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 100, count)

	// Test GetAllSymbols
	s.mockDS.On("GetAllSymbols").Return([]string{"SPY", "AAPL"}, nil)
	symbols, err := s.ds.GetAllSymbols()
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), []string{"SPY", "AAPL"}, symbols)

	// Test Close
	s.mockDS.On("Close").Return(nil)
	err = s.ds.Close()
	assert.NoError(s.T(), err)

	s.mockDS.AssertExpectations(s.T())
}

func (s *SlidingWindowDataSourceTestSuite) TestZeroCacheSize() {
	ds := NewSlidingWindowDataSource(s.mockDS, 0)
	baseTime := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	// Adding to cache should be a no-op
	ds.AddToCache(s.createMarketData("SPY", 0, 100))
	assert.Equal(s.T(), 0, ds.GetCache().TotalSize())

	// All queries should go to underlying datasource
	expectedData := s.createMarketData("SPY", 0, 100)
	s.mockDS.On("GetMarketData", "SPY", baseTime).Return(expectedData, nil)

	result, err := ds.GetMarketData("SPY", baseTime)
	assert.NoError(s.T(), err)
	assert.Equal(s.T(), 100.0, result.Close)
	s.mockDS.AssertExpectations(s.T())
}

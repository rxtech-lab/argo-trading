package datasource

import (
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDataSource is a mock implementation of DataSource for testing
type MockDataSource struct {
	mock.Mock
	data []types.MarketData
}

func (m *MockDataSource) Initialize(path string) error {
	args := m.Called(path)
	return args.Error(0)
}

func (m *MockDataSource) ReadAll(start optional.Option[time.Time], end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return func(yield func(types.MarketData, error) bool) {
		for _, d := range m.data {
			if start.IsSome() && d.Time.Before(start.Unwrap()) {
				continue
			}
			if end.IsSome() && d.Time.After(end.Unwrap()) {
				continue
			}
			if !yield(d, nil) {
				return
			}
		}
	}
}

func (m *MockDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[Interval]) ([]types.MarketData, error) {
	args := m.Called(start, end, interval)
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
	return args.Get(0).([]SQLResult), args.Error(1)
}

func (m *MockDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	return len(m.data), nil
}

func (m *MockDataSource) Close() error {
	return nil
}

func (m *MockDataSource) GetAllSymbols() ([]string, error) {
	symbolMap := make(map[string]bool)
	for _, d := range m.data {
		symbolMap[d.Symbol] = true
	}
	symbols := make([]string, 0, len(symbolMap))
	for s := range symbolMap {
		symbols = append(symbols, s)
	}
	return symbols, nil
}

// Helper to generate test data
func generateTestData(symbol string, count int, startTime time.Time) []types.MarketData {
	data := make([]types.MarketData, count)
	for i := 0; i < count; i++ {
		data[i] = types.MarketData{
			Symbol: symbol,
			Time:   startTime.Add(time.Duration(i) * time.Minute),
			Open:   100.0 + float64(i),
			High:   101.0 + float64(i),
			Low:    99.0 + float64(i),
			Close:  100.5 + float64(i),
			Volume: 1000.0,
		}
	}
	return data
}

func TestInMemoryIndexedDataSource_Preload(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	assert.False(t, indexedDS.IsPreloaded())

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)
	assert.True(t, indexedDS.IsPreloaded())

	// Verify data count
	assert.Equal(t, 100, indexedDS.GetTotalBars("TEST"))
}

func TestInMemoryIndexedDataSource_GetPreviousNBars(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Set current bar index to 50
	indexedDS.SetCurrentBarIndex(50)

	// Get previous 10 bars
	bars, err := indexedDS.GetPreviousNBars("TEST", 10)
	assert.NoError(t, err)
	assert.Len(t, bars, 10)

	// Verify bars are in chronological order (oldest to newest)
	for i := 1; i < len(bars); i++ {
		assert.True(t, bars[i].Time.After(bars[i-1].Time))
	}

	// The last bar should be at index 50
	assert.Equal(t, testData[50].Time, bars[len(bars)-1].Time)
}

func TestInMemoryIndexedDataSource_GetPreviousNBars_InsufficientData(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Set current bar index to 5
	indexedDS.SetCurrentBarIndex(5)

	// Request more bars than available
	_, err = indexedDS.GetPreviousNBars("TEST", 20)
	assert.Error(t, err)
}

func TestInMemoryIndexedDataSource_GetBarAtIndex(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Get bar at index 25
	bar, err := indexedDS.GetBarAtIndex("TEST", 25)
	assert.NoError(t, err)
	assert.Equal(t, testData[25].Time, bar.Time)
	assert.Equal(t, testData[25].Close, bar.Close)
}

func TestInMemoryIndexedDataSource_GetBarAtIndex_OutOfRange(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Try to get bar at invalid index
	_, err = indexedDS.GetBarAtIndex("TEST", 150)
	assert.Error(t, err)
}

func TestInMemoryIndexedDataSource_GetPreviousNumberOfDataPoints(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Get previous 10 data points ending at bar 50's time
	endTime := testData[50].Time
	bars, err := indexedDS.GetPreviousNumberOfDataPoints(endTime, "TEST", 10)
	assert.NoError(t, err)
	assert.Len(t, bars, 10)

	// The last bar should match the end time
	assert.Equal(t, endTime, bars[len(bars)-1].Time)
}

func TestInMemoryIndexedDataSource_ReadAll_WithPreload(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	// Iterate through all data
	count := 0
	for md, err := range indexedDS.ReadAll(optional.None[time.Time](), optional.None[time.Time]()) {
		assert.NoError(t, err)
		assert.Equal(t, testData[count].Time, md.Time)
		count++
	}
	assert.Equal(t, 100, count)
}

func TestInMemoryIndexedDataSource_GetAllSymbols(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 50, startTime)
	testData = append(testData, generateTestData("AAPL", 50, startTime)...)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	symbols, err := indexedDS.GetAllSymbols()
	assert.NoError(t, err)
	assert.Len(t, symbols, 2)
}

func TestInMemoryIndexedDataSource_NotPreloaded(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	testData := generateTestData("TEST", 100, startTime)

	mockDS := &MockDataSource{data: testData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	// Without preloading, GetPreviousNBars should fail
	_, err := indexedDS.GetPreviousNBars("TEST", 10)
	assert.Error(t, err)
}

func TestInMemoryIndexedDataSource_MultipleSymbols(t *testing.T) {
	startTime := time.Date(2024, 1, 1, 9, 30, 0, 0, time.UTC)
	aaplData := generateTestData("AAPL", 50, startTime)
	googData := generateTestData("GOOG", 50, startTime)
	allData := append(aaplData, googData...)

	mockDS := &MockDataSource{data: allData}
	indexedDS := NewInMemoryIndexedDataSource(mockDS)

	err := indexedDS.Preload(optional.None[time.Time](), optional.None[time.Time]())
	assert.NoError(t, err)

	assert.Equal(t, 50, indexedDS.GetTotalBars("AAPL"))
	assert.Equal(t, 50, indexedDS.GetTotalBars("GOOG"))
	assert.Equal(t, 0, indexedDS.GetTotalBars("MSFT")) // Non-existent symbol
}

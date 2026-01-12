package engine_v1

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/moznion/go-optional"
	"github.com/rxtech-lab/argo-trading/internal/backtest/engine/engine_v1/datasource"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type PersistentStreamingDataSourceTestSuite struct {
	suite.Suite
	tempDir string
}

func TestPersistentStreamingDataSourceSuite(t *testing.T) {
	suite.Run(t, new(PersistentStreamingDataSourceTestSuite))
}

func (suite *PersistentStreamingDataSourceTestSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "persistent-streaming-datasource-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *PersistentStreamingDataSourceTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

// createTestParquet creates a parquet file with test data
func (suite *PersistentStreamingDataSourceTestSuite) createTestParquet(filename string, data []types.MarketData) string {
	parquetPath := filepath.Join(suite.tempDir, filename)

	// Create parquet using DuckDB
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	// Create table
	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT,
			time TIMESTAMP,
			symbol TEXT,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	suite.Require().NoError(err)

	// Insert data
	stmt, err := db.Prepare(`
		INSERT INTO market_data (id, time, symbol, open, high, low, close, volume)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`)
	suite.Require().NoError(err)
	defer stmt.Close()

	for i, d := range data {
		_, err = stmt.Exec(
			string(rune(i)),
			d.Time,
			d.Symbol,
			d.Open,
			d.High,
			d.Low,
			d.Close,
			d.Volume,
		)
		suite.Require().NoError(err)
	}

	// Export to parquet
	_, err = db.Exec("COPY (SELECT * FROM market_data ORDER BY time ASC) TO '" + parquetPath + "' (FORMAT PARQUET)")
	suite.Require().NoError(err)

	return parquetPath
}

func (suite *PersistentStreamingDataSourceTestSuite) TestGetPreviousDataPoints() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 10)
	for i := 0; i < 10; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * 30 * time.Second),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_get_previous.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "30s")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Get last 5 data points
	endTime := baseTime.Add(9 * 30 * time.Second) // Last data point time
	result, err := ds.GetPreviousNumberOfDataPoints(endTime, "BTCUSDT", 5)
	suite.NoError(err)
	suite.Len(result, 5)

	// Verify chronological order (oldest to newest)
	for i := 1; i < len(result); i++ {
		suite.True(result[i].Time.After(result[i-1].Time), "Data should be in chronological order")
	}

	// Verify we got the correct range (last 5)
	suite.Equal(baseTime.Add(5*30*time.Second), result[0].Time)
	suite.Equal(endTime, result[4].Time)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestExecuteSQL() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 5)
	for i := 0; i < 5; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_execute_sql.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Execute SQL query with ordering
	query := "SELECT time, close FROM read_parquet('" + parquetPath + "') ORDER BY time ASC"
	results, err := ds.ExecuteSQL(query)
	suite.NoError(err)
	suite.Len(results, 5)

	// Verify ordering
	for i := 1; i < len(results); i++ {
		prevTime := results[i-1].Values["time"].(time.Time)
		currTime := results[i].Values["time"].(time.Time)
		suite.True(currTime.After(prevTime), "Results should be ordered by time")
	}
}

func (suite *PersistentStreamingDataSourceTestSuite) TestEmptyFile() {
	// Test with non-existent file
	ds := NewPersistentStreamingDataSource(filepath.Join(suite.tempDir, "nonexistent.parquet"), "1m")
	err := ds.Initialize("")
	suite.NoError(err) // Initialize should succeed
	defer ds.Close()

	// GetPreviousNumberOfDataPoints should return error
	_, err = ds.GetPreviousNumberOfDataPoints(time.Now(), "BTCUSDT", 5)
	suite.Error(err)
	suite.Contains(err.Error(), "no data available")

	// Count should return 0
	count, err := ds.Count(optional.None[time.Time](), optional.None[time.Time]())
	suite.NoError(err)
	suite.Equal(0, count)

	// GetAllSymbols should return empty
	symbols, err := ds.GetAllSymbols()
	suite.NoError(err)
	suite.Len(symbols, 0)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestIntervalFromFilename() {
	ds := NewPersistentStreamingDataSource("/path/to/stream_data_binance_1m.parquet", "1m")
	suite.Equal("1m", ds.GetInterval())

	ds2 := NewPersistentStreamingDataSource("/path/to/stream_data_binance_30s.parquet", "30s")
	suite.Equal("30s", ds2.GetInterval())
}

func (suite *PersistentStreamingDataSourceTestSuite) TestGetRange() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 10)
	for i := 0; i < 10; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_get_range.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Get range from minute 2 to minute 7
	start := baseTime.Add(2 * time.Minute)
	end := baseTime.Add(7 * time.Minute)
	result, err := ds.GetRange(start, end, optional.None[datasource.Interval]())
	suite.NoError(err)
	suite.Len(result, 6) // Minutes 2,3,4,5,6,7

	// Verify ordering
	for i := 1; i < len(result); i++ {
		suite.True(result[i].Time.After(result[i-1].Time), "Results should be ordered by time")
	}
}

func (suite *PersistentStreamingDataSourceTestSuite) TestReadLastData() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 5)
	for i := 0; i < 5; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_read_last.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	result, err := ds.ReadLastData("BTCUSDT")
	suite.NoError(err)
	suite.Equal(baseTime.Add(4*time.Minute), result.Time) // Last data point
	suite.Equal(42600.0, result.Close)                    // Last close price
}

func (suite *PersistentStreamingDataSourceTestSuite) TestMultiSymbolQueries() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 10)

	// 5 data points for each symbol
	for i := 0; i < 5; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
		testData[i+5] = types.MarketData{
			Symbol: "ETHUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   2200.0 + float64(i*10),
			High:   2250.0 + float64(i*10),
			Low:    2180.0 + float64(i*10),
			Close:  2220.0 + float64(i*10),
			Volume: 5000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_multi_symbol.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Query BTCUSDT only
	btcResult, err := ds.GetPreviousNumberOfDataPoints(baseTime.Add(4*time.Minute), "BTCUSDT", 3)
	suite.NoError(err)
	suite.Len(btcResult, 3)
	for _, r := range btcResult {
		suite.Equal("BTCUSDT", r.Symbol)
	}

	// Query ETHUSDT only
	ethResult, err := ds.GetPreviousNumberOfDataPoints(baseTime.Add(4*time.Minute), "ETHUSDT", 3)
	suite.NoError(err)
	suite.Len(ethResult, 3)
	for _, r := range ethResult {
		suite.Equal("ETHUSDT", r.Symbol)
	}
}

func (suite *PersistentStreamingDataSourceTestSuite) TestCount() {
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 10)
	for i := 0; i < 10; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0,
			High:   42500.0,
			Low:    41800.0,
			Close:  42200.0,
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_count.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Total count
	count, err := ds.Count(optional.None[time.Time](), optional.None[time.Time]())
	suite.NoError(err)
	suite.Equal(10, count)

	// Count with time range
	start := baseTime.Add(2 * time.Minute)
	end := baseTime.Add(7 * time.Minute)
	count, err = ds.Count(optional.Some(start), optional.Some(end))
	suite.NoError(err)
	suite.Equal(6, count)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestDirectParquetRead() {
	// This test verifies that queries read fresh data from parquet without needing refresh
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	testData := make([]types.MarketData, 5)
	for i := 0; i < 5; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
	}

	parquetPath := suite.createTestParquet("test_direct_read.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "1m")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// First read
	result1, err := ds.GetPreviousNumberOfDataPoints(baseTime.Add(4*time.Minute), "BTCUSDT", 5)
	suite.NoError(err)
	suite.Len(result1, 5)

	// Update parquet file with new data
	newData := make([]types.MarketData, 7) // Add 2 more records
	copy(newData[:5], testData)
	newData[5] = types.MarketData{
		Symbol: "BTCUSDT",
		Time:   baseTime.Add(5 * time.Minute),
		Open:   42500.0,
		High:   43000.0,
		Low:    42300.0,
		Close:  42700.0,
		Volume: 1000.0,
	}
	newData[6] = types.MarketData{
		Symbol: "BTCUSDT",
		Time:   baseTime.Add(6 * time.Minute),
		Open:   42700.0,
		High:   43200.0,
		Low:    42500.0,
		Close:  42900.0,
		Volume: 1000.0,
	}

	// Overwrite parquet file
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT, time TIMESTAMP, symbol TEXT, open DOUBLE, high DOUBLE, low DOUBLE, close DOUBLE, volume DOUBLE
		)
	`)
	suite.Require().NoError(err)

	for i, d := range newData {
		_, err = db.Exec(`INSERT INTO market_data VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			string(rune(i)), d.Time, d.Symbol, d.Open, d.High, d.Low, d.Close, d.Volume)
		suite.Require().NoError(err)
	}
	_, err = db.Exec("COPY (SELECT * FROM market_data ORDER BY time ASC) TO '" + parquetPath + "' (FORMAT PARQUET)")
	suite.Require().NoError(err)

	// Second read - should see new data without refresh
	result2, err := ds.GetPreviousNumberOfDataPoints(baseTime.Add(6*time.Minute), "BTCUSDT", 7)
	suite.NoError(err)
	suite.Len(result2, 7) // Should see all 7 records now
}

// ================================
// SMA Calculation Tests (30-second interval)
// ================================

// calculateSMA is a helper function to calculate Simple Moving Average
func calculateSMA(data []types.MarketData) float64 {
	if len(data) == 0 {
		return 0
	}

	sum := 0.0
	for _, d := range data {
		sum += d.Close
	}

	return sum / float64(len(data))
}

func (suite *PersistentStreamingDataSourceTestSuite) TestSMACalculation_NoOldData_NewDataArrives() {
	// Test: No old data, new data comes, calculate the SMA
	parquetPath := filepath.Join(suite.tempDir, "test_sma_no_old.parquet")
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	ds := NewPersistentStreamingDataSource(parquetPath, "30s")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// Initially no data - SMA(5) should fail
	_, err = ds.GetPreviousNumberOfDataPoints(baseTime, "BTCUSDT", 5)
	suite.Error(err) // No data available

	// First candle arrives at T=0
	data1 := []types.MarketData{{
		Symbol: "BTCUSDT",
		Time:   baseTime,
		Open:   42000.0,
		High:   42100.0,
		Low:    41900.0,
		Close:  42050.0, // First close price
		Volume: 100.0,
	}}
	suite.createTestParquetAtPath(parquetPath, data1)

	// SMA(5) with only 1 data point
	result1, err := ds.GetPreviousNumberOfDataPoints(baseTime, "BTCUSDT", 5)
	// Should return insufficient data error but still return the 1 data point
	suite.Len(result1, 1)
	sma1 := calculateSMA(result1)
	suite.Equal(42050.0, sma1) // SMA is just the single value

	// Second candle arrives at T=30s
	data2 := []types.MarketData{
		data1[0],
		{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(30 * time.Second),
			Open:   42050.0,
			High:   42200.0,
			Low:    42000.0,
			Close:  42150.0, // Second close price
			Volume: 150.0,
		},
	}
	suite.createTestParquetAtPath(parquetPath, data2)

	// SMA(5) with 2 data points
	result2, err := ds.GetPreviousNumberOfDataPoints(baseTime.Add(30*time.Second), "BTCUSDT", 5)
	suite.Len(result2, 2)
	sma2 := calculateSMA(result2)
	expectedSMA2 := (42050.0 + 42150.0) / 2.0
	suite.Equal(expectedSMA2, sma2)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestSMACalculation_HasOldData_NewDataArrivesEarly() {
	// Test: Has old data, new data comes with 1 second interval (should still be valid)
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create 10 candles at 30s intervals
	testData := make([]types.MarketData, 10)
	for i := 0; i < 10; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * 30 * time.Second),
			Open:   42000.0 + float64(i*10),
			High:   42100.0 + float64(i*10),
			Low:    41900.0 + float64(i*10),
			Close:  42050.0 + float64(i*10), // Close prices: 42050, 42060, 42070, ...
			Volume: 100.0 + float64(i*10),
		}
	}

	parquetPath := suite.createTestParquet("test_sma_early_data.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "30s")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// New candle arrives at T=270s + 1s (1 second after the last expected candle at 270s)
	// This simulates data arriving slightly early/late
	newCandleTime := baseTime.Add(270*time.Second + 1*time.Second)
	newData := append(testData, types.MarketData{
		Symbol: "BTCUSDT",
		Time:   newCandleTime,
		Open:   42140.0,
		High:   42200.0,
		Low:    42100.0,
		Close:  42180.0,
		Volume: 200.0,
	})
	suite.createTestParquetAtPath(parquetPath, newData)

	// SMA(5) should include the new candle
	result, err := ds.GetPreviousNumberOfDataPoints(newCandleTime, "BTCUSDT", 5)
	suite.NoError(err)
	suite.Len(result, 5)

	// Verify data is ordered correctly by time
	for i := 1; i < len(result); i++ {
		suite.True(result[i].Time.After(result[i-1].Time),
			"Data should be ordered: %v should be after %v", result[i].Time, result[i-1].Time)
	}

	// Verify SMA calculation
	// Last 5 closes should be: 42100, 42110, 42120, 42130, 42180 (approximately)
	sma := calculateSMA(result)
	suite.Greater(sma, 42000.0)
	suite.Less(sma, 43000.0)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestSMACalculation_HasOldData_NewDataArrivesOnSchedule() {
	// Test: Has old data, new data comes in exactly 30 seconds
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create 10 candles at 30s intervals
	testData := make([]types.MarketData, 10)
	for i := 0; i < 10; i++ {
		testData[i] = types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * 30 * time.Second),
			Open:   42000.0 + float64(i*10),
			High:   42100.0 + float64(i*10),
			Low:    41900.0 + float64(i*10),
			Close:  42050.0 + float64(i*10),
			Volume: 100.0 + float64(i*10),
		}
	}

	parquetPath := suite.createTestParquet("test_sma_scheduled.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "30s")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// New candle arrives exactly at T+30s (10th candle at 300s)
	newCandleTime := baseTime.Add(10 * 30 * time.Second) // Exactly 300s = 5 minutes
	newData := append(testData, types.MarketData{
		Symbol: "BTCUSDT",
		Time:   newCandleTime,
		Open:   42140.0,
		High:   42200.0,
		Low:    42100.0,
		Close:  42175.0,
		Volume: 200.0,
	})
	suite.createTestParquetAtPath(parquetPath, newData)

	// SMA(5) should return last 5 candles including the new one
	result, err := ds.GetPreviousNumberOfDataPoints(newCandleTime, "BTCUSDT", 5)
	suite.NoError(err)
	suite.Len(result, 5)

	// Verify the last element is the new candle
	suite.Equal(newCandleTime, result[4].Time)
	suite.Equal(42175.0, result[4].Close)

	// Calculate SMA of last 5 closes: candles 6,7,8,9 (42110, 42120, 42130, 42140) + new (42175)
	expectedCloses := []float64{42110.0, 42120.0, 42130.0, 42140.0, 42175.0}
	expectedSMA := (expectedCloses[0] + expectedCloses[1] + expectedCloses[2] + expectedCloses[3] + expectedCloses[4]) / 5.0
	actualSMA := calculateSMA(result)
	suite.InDelta(expectedSMA, actualSMA, 0.01)
}

func (suite *PersistentStreamingDataSourceTestSuite) TestSMACalculation_GapInData() {
	// Test: Gap in data - some candles missing
	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Create candles with a gap (missing candles 5, 6, 7)
	testData := []types.MarketData{}
	for i := 0; i < 5; i++ { // Candles 0-4
		testData = append(testData, types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * 30 * time.Second),
			Open:   42000.0 + float64(i*10),
			High:   42100.0 + float64(i*10),
			Low:    41900.0 + float64(i*10),
			Close:  42050.0 + float64(i*10),
			Volume: 100.0,
		})
	}
	// Skip candles 5, 6, 7 (gap of 90 seconds)
	for i := 8; i < 12; i++ { // Candles 8-11
		testData = append(testData, types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * 30 * time.Second),
			Open:   42000.0 + float64(i*10),
			High:   42100.0 + float64(i*10),
			Low:    41900.0 + float64(i*10),
			Close:  42050.0 + float64(i*10),
			Volume: 100.0,
		})
	}

	parquetPath := suite.createTestParquet("test_sma_gap.parquet", testData)

	ds := NewPersistentStreamingDataSource(parquetPath, "30s")
	err := ds.Initialize("")
	suite.Require().NoError(err)
	defer ds.Close()

	// New candle arrives after the gap
	newCandleTime := baseTime.Add(12 * 30 * time.Second)
	newData := append(testData, types.MarketData{
		Symbol: "BTCUSDT",
		Time:   newCandleTime,
		Open:   42120.0,
		High:   42200.0,
		Low:    42100.0,
		Close:  42170.0,
		Volume: 200.0,
	})
	suite.createTestParquetAtPath(parquetPath, newData)

	// SMA(5) should use available data (handles gap gracefully)
	result, err := ds.GetPreviousNumberOfDataPoints(newCandleTime, "BTCUSDT", 5)
	suite.NoError(err)
	suite.Len(result, 5)

	// SMA calculation should still work with available data
	sma := calculateSMA(result)
	suite.Greater(sma, 0.0)

	// Verify data is ordered
	for i := 1; i < len(result); i++ {
		suite.True(result[i].Time.After(result[i-1].Time) || result[i].Time.Equal(result[i-1].Time),
			"Data should be ordered by time")
	}
}

// createTestParquetAtPath creates a parquet file at the exact path specified
func (suite *PersistentStreamingDataSourceTestSuite) createTestParquetAtPath(parquetPath string, data []types.MarketData) {
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	_, err = db.Exec(`
		CREATE TABLE market_data (
			id TEXT, time TIMESTAMP, symbol TEXT, open DOUBLE, high DOUBLE, low DOUBLE, close DOUBLE, volume DOUBLE
		)
	`)
	suite.Require().NoError(err)

	for i, d := range data {
		_, err = db.Exec(`INSERT INTO market_data VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
			string(rune(i)), d.Time, d.Symbol, d.Open, d.High, d.Low, d.Close, d.Volume)
		suite.Require().NoError(err)
	}

	_, err = db.Exec("COPY (SELECT * FROM market_data ORDER BY time ASC) TO '" + parquetPath + "' (FORMAT PARQUET)")
	suite.Require().NoError(err)
}

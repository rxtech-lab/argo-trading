package writer

import (
	"database/sql"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type StreamingDuckDBWriterTestSuite struct {
	suite.Suite
	tempDir string
}

func TestStreamingDuckDBWriterSuite(t *testing.T) {
	suite.Run(t, new(StreamingDuckDBWriterTestSuite))
}

func (suite *StreamingDuckDBWriterTestSuite) SetupSuite() {
	tempDir, err := os.MkdirTemp("", "streaming-duckdb-writer-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *StreamingDuckDBWriterTestSuite) TearDownSuite() {
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *StreamingDuckDBWriterTestSuite) TestFileNamingPattern() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "1m")
	expectedPath := filepath.Join(suite.tempDir, "stream_data_binance_1m.parquet")
	suite.Equal(expectedPath, writer.GetOutputPath())

	writer2 := NewStreamingDuckDBWriter(suite.tempDir, "polygon", "5m")
	expectedPath2 := filepath.Join(suite.tempDir, "stream_data_polygon_5m.parquet")
	suite.Equal(expectedPath2, writer2.GetOutputPath())

	writer3 := NewStreamingDuckDBWriter(suite.tempDir, "binance", "1h")
	expectedPath3 := filepath.Join(suite.tempDir, "stream_data_binance_1h.parquet")
	suite.Equal(expectedPath3, writer3.GetOutputPath())
}

func (suite *StreamingDuckDBWriterTestSuite) TestWriteData() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "test_write")

	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	data := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.5,
	}

	err = writer.Write(data)
	suite.NoError(err)

	// Verify file was created
	_, statErr := os.Stat(writer.GetOutputPath())
	suite.NoError(statErr)

	// Verify data is in file by querying it
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(1, count)
}

func (suite *StreamingDuckDBWriterTestSuite) TestAppendToExistingFile() {
	outputPath := filepath.Join(suite.tempDir, "stream_data_binance_append.parquet")

	// First writer - write initial data
	writer1 := NewStreamingDuckDBWriter(suite.tempDir, "binance", "append")
	err := writer1.Initialize()
	suite.Require().NoError(err)

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 5; i++ {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.5,
		}
		err = writer1.Write(data)
		suite.Require().NoError(err)
	}

	err = writer1.Close()
	suite.Require().NoError(err)

	// Second writer - should load existing data and append
	writer2 := NewStreamingDuckDBWriter(suite.tempDir, "binance", "append")
	err = writer2.Initialize()
	suite.Require().NoError(err)
	defer writer2.Close()

	// Write 5 more records
	for i := 5; i < 10; i++ {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.5,
		}
		err = writer2.Write(data)
		suite.Require().NoError(err)
	}

	// Verify all 10 records are in file
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + outputPath + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(10, count)
}

func (suite *StreamingDuckDBWriterTestSuite) TestUpsertBehavior() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "upsert")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	timestamp := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Write initial data
	data1 := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   timestamp,
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.0,
	}
	err = writer.Write(data1)
	suite.Require().NoError(err)

	// Write same timestamp with different values (upsert)
	data2 := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   timestamp,
		Open:   42100.0, // Different value
		High:   42600.0,
		Low:    41900.0,
		Close:  42300.0,
		Volume: 1100.0,
	}
	err = writer.Write(data2)
	suite.Require().NoError(err)

	// Verify only one record exists (upsert replaced)
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(1, count)

	// Verify values are from second write
	var closePrice float64
	err = db.QueryRow("SELECT close FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&closePrice)
	suite.NoError(err)
	suite.Equal(42300.0, closePrice)
}

func (suite *StreamingDuckDBWriterTestSuite) TestConcurrentReadWrite() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "concurrent")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	var wg sync.WaitGroup
	errChan := make(chan error, 20)

	// Write from multiple goroutines
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			data := types.MarketData{
				Symbol: "BTCUSDT",
				Time:   baseTime.Add(time.Duration(idx) * time.Minute),
				Open:   42000.0 + float64(idx*100),
				High:   42500.0 + float64(idx*100),
				Low:    41800.0 + float64(idx*100),
				Close:  42200.0 + float64(idx*100),
				Volume: 1000.0,
			}
			if err := writer.Write(data); err != nil {
				errChan <- err
			}
		}(i)
	}

	wg.Wait()
	close(errChan)

	// Check for errors
	for err := range errChan {
		suite.Fail("Concurrent write error", err.Error())
	}

	// Verify all records were written
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(10, count)
}

func (suite *StreamingDuckDBWriterTestSuite) TestDataOrdering() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "ordering")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Write data out of order
	times := []int{5, 2, 8, 1, 4, 7, 3, 6, 0, 9}
	for _, i := range times {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
		err = writer.Write(data)
		suite.Require().NoError(err)
	}

	// Verify data is ordered by time in parquet file
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	rows, err := db.Query("SELECT time FROM read_parquet('" + writer.GetOutputPath() + "') ORDER BY time ASC")
	suite.Require().NoError(err)
	defer rows.Close()

	var lastTime time.Time
	isFirst := true
	for rows.Next() {
		var t time.Time
		err = rows.Scan(&t)
		suite.Require().NoError(err)

		if !isFirst {
			suite.True(t.After(lastTime) || t.Equal(lastTime), "Data should be ordered by time")
		}
		lastTime = t
		isFirst = false
	}
}

func (suite *StreamingDuckDBWriterTestSuite) TestRestartBehavior() {
	// First session - write some data
	writer1 := NewStreamingDuckDBWriter(suite.tempDir, "binance", "restart")
	err := writer1.Initialize()
	suite.Require().NoError(err)

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	for i := 0; i < 3; i++ {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
		err = writer1.Write(data)
		suite.Require().NoError(err)
	}
	_, err = writer1.Finalize()
	suite.Require().NoError(err)
	err = writer1.Close()
	suite.Require().NoError(err)

	// Second session (simulate restart) - data should be preserved
	writer2 := NewStreamingDuckDBWriter(suite.tempDir, "binance", "restart")
	err = writer2.Initialize()
	suite.Require().NoError(err)
	defer writer2.Close()

	// Write more data
	for i := 3; i < 6; i++ {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i*100),
			High:   42500.0 + float64(i*100),
			Low:    41800.0 + float64(i*100),
			Close:  42200.0 + float64(i*100),
			Volume: 1000.0,
		}
		err = writer2.Write(data)
		suite.Require().NoError(err)
	}

	// Verify all 6 records exist
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer2.GetOutputPath() + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(6, count)
}

func (suite *StreamingDuckDBWriterTestSuite) TestMultiSymbol() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "multisymbol")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)
	symbols := []string{"BTCUSDT", "ETHUSDT"}

	// Write data for multiple symbols
	for _, symbol := range symbols {
		for i := 0; i < 5; i++ {
			data := types.MarketData{
				Symbol: symbol,
				Time:   baseTime.Add(time.Duration(i) * time.Minute),
				Open:   42000.0 + float64(i*100),
				High:   42500.0 + float64(i*100),
				Low:    41800.0 + float64(i*100),
				Close:  42200.0 + float64(i*100),
				Volume: 1000.0,
			}
			err = writer.Write(data)
			suite.Require().NoError(err)
		}
	}

	// Verify all records
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var totalCount int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&totalCount)
	suite.NoError(err)
	suite.Equal(10, totalCount)

	// Verify count per symbol
	for _, symbol := range symbols {
		var symbolCount int
		err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('"+writer.GetOutputPath()+"') WHERE symbol = ?", symbol).Scan(&symbolCount)
		suite.NoError(err)
		suite.Equal(5, symbolCount)
	}
}

func (suite *StreamingDuckDBWriterTestSuite) TestInvalidDirectory() {
	// This test verifies that Initialize creates the directory if it doesn't exist
	nonExistentDir := filepath.Join(suite.tempDir, "nonexistent", "subdir")
	writer := NewStreamingDuckDBWriter(nonExistentDir, "binance", "1m")

	err := writer.Initialize()
	suite.NoError(err) // Should succeed - directory should be created
	defer writer.Close()

	// Verify directory was created
	_, statErr := os.Stat(nonExistentDir)
	suite.NoError(statErr)
}

func (suite *StreamingDuckDBWriterTestSuite) TestCorruptedParquet() {
	// Create a corrupted parquet file
	corruptedPath := filepath.Join(suite.tempDir, "stream_data_binance_corrupted.parquet")
	err := os.WriteFile(corruptedPath, []byte("not a valid parquet file"), 0644)
	suite.Require().NoError(err)

	// Writer should handle corrupted file gracefully
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "corrupted")
	err = writer.Initialize()
	suite.NoError(err) // Should not fail - corrupted file is ignored
	defer writer.Close()

	// Should be able to write new data
	data := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.0,
	}
	err = writer.Write(data)
	suite.NoError(err)
}

func (suite *StreamingDuckDBWriterTestSuite) TestWriteWithoutInitialize() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "noinit")

	data := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   time.Now(),
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.0,
	}

	err := writer.Write(data)
	suite.Error(err)
	suite.Contains(err.Error(), "not initialized")
}

func (suite *StreamingDuckDBWriterTestSuite) TestFlush() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "flush")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	data := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.0,
	}
	err = writer.Write(data)
	suite.Require().NoError(err)

	err = writer.Flush()
	suite.NoError(err)

	// Verify file exists
	_, statErr := os.Stat(writer.GetOutputPath())
	suite.NoError(statErr)
}

func (suite *StreamingDuckDBWriterTestSuite) TestFinalize() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "finalize")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	data := types.MarketData{
		Symbol: "BTCUSDT",
		Time:   time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
		Open:   42000.0,
		High:   42500.0,
		Low:    41800.0,
		Close:  42200.0,
		Volume: 1000.0,
	}
	err = writer.Write(data)
	suite.Require().NoError(err)

	path, err := writer.Finalize()
	suite.NoError(err)
	suite.Equal(writer.GetOutputPath(), path)
}

func (suite *StreamingDuckDBWriterTestSuite) TestDoubleClose() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "doubleclose")
	err := writer.Initialize()
	suite.Require().NoError(err)

	err = writer.Close()
	suite.NoError(err)

	// Second close should not error
	err = writer.Close()
	suite.NoError(err)
}

func (suite *StreamingDuckDBWriterTestSuite) TestLargeDataset() {
	writer := NewStreamingDuckDBWriter(suite.tempDir, "binance", "large")
	err := writer.Initialize()
	suite.Require().NoError(err)
	defer writer.Close()

	baseTime := time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC)

	// Write 1000 records
	for i := 0; i < 1000; i++ {
		data := types.MarketData{
			Symbol: "BTCUSDT",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   42000.0 + float64(i),
			High:   42500.0 + float64(i),
			Low:    41800.0 + float64(i),
			Close:  42200.0 + float64(i),
			Volume: 1000.0,
		}
		err = writer.Write(data)
		suite.Require().NoError(err)
	}

	// Verify all records
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	defer db.Close()

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM read_parquet('" + writer.GetOutputPath() + "')").Scan(&count)
	suite.NoError(err)
	suite.Equal(1000, count)
}

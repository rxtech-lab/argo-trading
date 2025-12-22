package writer

import (
	"os"
	"testing"
	"time"

	"github.com/rxtech-lab/argo-trading/internal/types"
	"github.com/stretchr/testify/suite"
)

type DuckDBWriterTestSuite struct {
	suite.Suite
	tempDir string
}

func TestDuckDBWriterSuite(t *testing.T) {
	suite.Run(t, new(DuckDBWriterTestSuite))
}

func (suite *DuckDBWriterTestSuite) SetupSuite() {
	// Create a temporary directory for test output
	tempDir, err := os.MkdirTemp("", "duckdb-writer-test")
	suite.Require().NoError(err)
	suite.tempDir = tempDir
}

func (suite *DuckDBWriterTestSuite) TearDownSuite() {
	// Cleanup temp directory
	if suite.tempDir != "" {
		os.RemoveAll(suite.tempDir)
	}
}

func (suite *DuckDBWriterTestSuite) TestNewDuckDBWriter() {
	outputPath := suite.tempDir + "/test.parquet"
	writer := NewDuckDBWriter(outputPath)

	suite.NotNil(writer)

	// Cast to check internal state
	duckWriter, ok := writer.(*DuckDBWriter)
	suite.True(ok)
	suite.Equal(outputPath, duckWriter.outputPath)
	suite.Nil(duckWriter.db)
	suite.Nil(duckWriter.tx)
	suite.Nil(duckWriter.stmt)
}

func (suite *DuckDBWriterTestSuite) TestInitialize() {
	outputPath := suite.tempDir + "/test_init.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.NoError(err)

	// Check internal state after initialization
	duckWriter := writer.(*DuckDBWriter)
	suite.NotNil(duckWriter.db)
	suite.NotNil(duckWriter.tx)
	suite.NotNil(duckWriter.stmt)

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestWriteWithoutInitialize() {
	outputPath := suite.tempDir + "/test_no_init.parquet"
	writer := NewDuckDBWriter(outputPath)

	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Now(),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err := writer.Write(data)
	suite.Error(err)
	suite.Contains(err.Error(), "not initialized")
}

func (suite *DuckDBWriterTestSuite) TestWriteAfterInitialize() {
	outputPath := suite.tempDir + "/test_write.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.Require().NoError(err)

	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.NoError(err)

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestWriteMultipleRecords() {
	outputPath := suite.tempDir + "/test_multi.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.Require().NoError(err)

	baseTime := time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC)
	for i := 0; i < 10; i++ {
		data := types.MarketData{
			Symbol: "AAPL",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   150.0 + float64(i),
			High:   155.0 + float64(i),
			Low:    148.0 + float64(i),
			Close:  152.0 + float64(i),
			Volume: 1000000.0 + float64(i*100),
		}

		err = writer.Write(data)
		suite.NoError(err)
	}

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestFinalizeWithoutInitialize() {
	outputPath := suite.tempDir + "/test_finalize_no_init.parquet"
	writer := NewDuckDBWriter(outputPath)

	_, err := writer.Finalize()
	suite.Error(err)
	suite.Contains(err.Error(), "not initialized")
}

func (suite *DuckDBWriterTestSuite) TestFinalizeAfterWrite() {
	outputPath := suite.tempDir + "/test_finalize.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.Require().NoError(err)

	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	path, err := writer.Finalize()
	suite.NoError(err)
	suite.Equal(outputPath, path)

	// Verify file was created
	_, statErr := os.Stat(outputPath)
	suite.NoError(statErr)

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestCloseWithoutInitialize() {
	outputPath := suite.tempDir + "/test_close_no_init.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Close()
	suite.NoError(err)
}

func (suite *DuckDBWriterTestSuite) TestCloseAfterInitialize() {
	outputPath := suite.tempDir + "/test_close_init.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.Require().NoError(err)

	err = writer.Close()
	suite.NoError(err)

	// Verify internal state is cleared
	duckWriter := writer.(*DuckDBWriter)
	suite.Nil(duckWriter.db)
	suite.Nil(duckWriter.tx)
	suite.Nil(duckWriter.stmt)
}

func (suite *DuckDBWriterTestSuite) TestDoubleClose() {
	outputPath := suite.tempDir + "/test_double_close.parquet"
	writer := NewDuckDBWriter(outputPath)

	err := writer.Initialize()
	suite.Require().NoError(err)

	// First close
	err = writer.Close()
	suite.NoError(err)

	// Second close should not error
	err = writer.Close()
	suite.NoError(err)
}

func (suite *DuckDBWriterTestSuite) TestFullWorkflow() {
	outputPath := suite.tempDir + "/test_workflow.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data
	for i := 0; i < 5; i++ {
		data := types.MarketData{
			Symbol: "SPY",
			Time:   time.Date(2023, 6, 15, 9, 30+i, 0, 0, time.UTC),
			Open:   450.0 + float64(i),
			High:   455.0 + float64(i),
			Low:    448.0 + float64(i),
			Close:  452.0 + float64(i),
			Volume: 5000000.0,
		}

		err = writer.Write(data)
		suite.Require().NoError(err)
	}

	// Finalize
	path, err := writer.Finalize()
	suite.NoError(err)
	suite.Equal(outputPath, path)

	// Verify file exists
	fileInfo, err := os.Stat(path)
	suite.NoError(err)
	suite.Greater(fileInfo.Size(), int64(0))

	// Close
	err = writer.Close()
	suite.NoError(err)
}

func (suite *DuckDBWriterTestSuite) TestWriteAfterFinalize() {
	outputPath := suite.tempDir + "/test_write_after_finalize.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write initial data
	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	// Finalize
	_, err = writer.Finalize()
	suite.Require().NoError(err)

	// Close - this will set stmt to nil
	err = writer.Close()
	suite.Require().NoError(err)

	// Try to write after close - should error because stmt is nil
	err = writer.Write(data)
	suite.Error(err)
	suite.Contains(err.Error(), "not initialized")
}

func (suite *DuckDBWriterTestSuite) TestDoubleFinalize() {
	outputPath := suite.tempDir + "/test_double_finalize.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data
	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	// First finalize
	_, err = writer.Finalize()
	suite.NoError(err)

	// Second finalize - should error because tx is nil
	_, err = writer.Finalize()
	suite.Error(err)
	suite.Contains(err.Error(), "not initialized")

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestCloseAfterFinalize() {
	outputPath := suite.tempDir + "/test_close_after_finalize.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data
	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	// Finalize
	_, err = writer.Finalize()
	suite.Require().NoError(err)

	// Close after finalize should work
	err = writer.Close()
	suite.NoError(err)

	// Verify internal state is cleared
	duckWriter := writer.(*DuckDBWriter)
	suite.Nil(duckWriter.db)
	suite.Nil(duckWriter.tx)
	suite.Nil(duckWriter.stmt)
}

func (suite *DuckDBWriterTestSuite) TestFinalizeExportError() {
	// Use an invalid path that cannot be written to
	outputPath := "/nonexistent/directory/test.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data
	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	// Finalize should fail because the output path doesn't exist
	_, err = writer.Finalize()
	suite.Error(err)
	suite.Contains(err.Error(), "failed to export to Parquet")

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestMultipleWritesWithDifferentSymbols() {
	outputPath := suite.tempDir + "/test_multi_symbols.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data for different symbols
	symbols := []string{"AAPL", "GOOGL", "MSFT", "AMZN", "META"}
	baseTime := time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC)

	for i, symbol := range symbols {
		data := types.MarketData{
			Symbol: symbol,
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   100.0 + float64(i)*10,
			High:   110.0 + float64(i)*10,
			Low:    90.0 + float64(i)*10,
			Close:  105.0 + float64(i)*10,
			Volume: float64(1000000 * (i + 1)),
		}

		err = writer.Write(data)
		suite.NoError(err)
	}

	// Finalize
	path, err := writer.Finalize()
	suite.NoError(err)
	suite.Equal(outputPath, path)

	// Cleanup
	writer.Close()
}

func (suite *DuckDBWriterTestSuite) TestCloseWithActiveTransaction() {
	outputPath := suite.tempDir + "/test_close_active_tx.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write data but don't finalize - transaction is still active
	data := types.MarketData{
		Symbol: "AAPL",
		Time:   time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC),
		Open:   150.0,
		High:   155.0,
		Low:    148.0,
		Close:  152.0,
		Volume: 1000000.0,
	}

	err = writer.Write(data)
	suite.Require().NoError(err)

	// Close without finalizing - should rollback the transaction
	err = writer.Close()
	suite.NoError(err)

	// Verify internal state is cleared
	duckWriter := writer.(*DuckDBWriter)
	suite.Nil(duckWriter.db)
	suite.Nil(duckWriter.tx)
	suite.Nil(duckWriter.stmt)
}

func (suite *DuckDBWriterTestSuite) TestWriteLargeDataset() {
	outputPath := suite.tempDir + "/test_large_dataset.parquet"
	writer := NewDuckDBWriter(outputPath)

	// Initialize
	err := writer.Initialize()
	suite.Require().NoError(err)

	// Write 1000 records
	baseTime := time.Date(2023, 6, 15, 9, 30, 0, 0, time.UTC)
	for i := 0; i < 1000; i++ {
		data := types.MarketData{
			Symbol: "SPY",
			Time:   baseTime.Add(time.Duration(i) * time.Minute),
			Open:   400.0 + float64(i%100)*0.1,
			High:   400.5 + float64(i%100)*0.1,
			Low:    399.5 + float64(i%100)*0.1,
			Close:  400.2 + float64(i%100)*0.1,
			Volume: float64(1000000 + i*100),
		}

		err = writer.Write(data)
		suite.NoError(err)
	}

	// Finalize
	path, err := writer.Finalize()
	suite.NoError(err)
	suite.Equal(outputPath, path)

	// Verify file exists and has content
	fileInfo, err := os.Stat(path)
	suite.NoError(err)
	suite.Greater(fileInfo.Size(), int64(0))

	// Cleanup
	writer.Close()
}

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

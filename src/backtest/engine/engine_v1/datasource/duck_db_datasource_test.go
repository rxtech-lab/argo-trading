package datasource

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/suite"
)

// DuckDBTestSuite is a test suite for DuckDBDataSource
type DuckDBTestSuite struct {
	suite.Suite
	ds     *DuckDBDataSource
	logger *logger.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *DuckDBTestSuite) SetupSuite() {
	db, err := sql.Open("duckdb", "")
	suite.Require().NoError(err)
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
	suite.ds = &DuckDBDataSource{db: db, logger: suite.logger}
}

// TearDownSuite runs once after all tests in the suite
func (suite *DuckDBTestSuite) TearDownSuite() {
	if suite.ds != nil && suite.ds.db != nil {
		suite.ds.db.Close()
	}
}

// cleanupMarketData drops both the view and table if they exist
func (suite *DuckDBTestSuite) cleanupMarketData() {
	// Drop view first if it exists (ignore errors)
	suite.ds.db.Exec("DROP VIEW IF EXISTS market_data")
	// Drop table if it exists (ignore errors)
	suite.ds.db.Exec("DROP TABLE IF EXISTS market_data_source")
}

// SetupTest runs before each test
func (suite *DuckDBTestSuite) SetupTest() {
	suite.cleanupMarketData()
}

// TearDownTest runs after each test
func (suite *DuckDBTestSuite) TearDownTest() {
	suite.cleanupMarketData()
}

// TestDuckDBDataSourceSuite runs the test suite
func TestDuckDBDataSourceSuite(t *testing.T) {
	suite.Run(t, new(DuckDBTestSuite))
}

func (suite *DuckDBTestSuite) TestInitialize() {
	// Create a temporary directory for test files
	tmpDir := suite.T().TempDir()

	// Create a test parquet file
	testData := []types.MarketData{
		{
			Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			Open:   100.0,
			High:   101.0,
			Low:    99.0,
			Close:  100.5,
			Volume: 1000.0,
			Symbol: "AAPL",
		},
		{
			Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
			Open:   100.5,
			High:   102.0,
			Low:    100.0,
			Close:  101.5,
			Volume: 1500.0,
			Symbol: "AAPL",
		},
	}

	// Write test data to parquet file
	testFilePath := filepath.Join(tmpDir, "test.parquet")
	err := writeTestDataToParquet(testData, testFilePath)
	suite.Require().NoError(err)

	// Define test cases for Initialize
	tests := []struct {
		name        string
		parquetPath string
		expectError bool
	}{
		{
			name:        "Valid parquet file",
			parquetPath: testFilePath,
			expectError: false,
		},
		{
			name:        "Invalid parquet path",
			parquetPath: "nonexistent.parquet",
			expectError: true,
		},
		{
			name:        "Empty path",
			parquetPath: "",
			expectError: true,
		},
	}

	// Run Initialize tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData()

			// Test Initialize
			err := suite.ds.Initialize(tc.parquetPath)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)

			// Verify data was loaded correctly
			if tc.parquetPath == testFilePath {
				count, err := suite.ds.Count()
				suite.Assert().NoError(err)
				suite.Assert().Equal(len(testData), count, "Data count mismatch")

				// Verify data content
				results, err := suite.ds.ExecuteSQL("SELECT * FROM market_data ORDER BY time ASC")
				suite.Assert().NoError(err)
				suite.Assert().Equal(len(testData), len(results), "Number of rows mismatch")

				for i, expected := range testData {
					suite.Assert().Equal(expected.Time.UTC(), results[i].Values["time"].(time.Time).UTC(), "Time mismatch")
					suite.Assert().Equal(expected.Open, results[i].Values["open"].(float64), "Open price mismatch")
					suite.Assert().Equal(expected.High, results[i].Values["high"].(float64), "High price mismatch")
					suite.Assert().Equal(expected.Low, results[i].Values["low"].(float64), "Low price mismatch")
					suite.Assert().Equal(expected.Close, results[i].Values["close"].(float64), "Close price mismatch")
					suite.Assert().Equal(expected.Volume, results[i].Values["volume"].(float64), "Volume mismatch")
					suite.Assert().Equal(expected.Symbol, results[i].Values["symbol"].(string), "Symbol mismatch")
				}
			}
		})
	}
}

func (suite *DuckDBTestSuite) TestReadAll() {
	// Define test cases for ReadAll
	tests := []struct {
		name         string
		setupData    string
		expectedData []types.MarketData
		expectError  bool
	}{
		{
			name: "Read valid market data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 'AAPL', 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 'AAPL', 100.5, 102.0, 100.0, 101.5, 1500.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			expectedData: []types.MarketData{
				{
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					Open:   100.0,
					High:   101.0,
					Low:    99.0,
					Close:  100.5,
					Volume: 1000.0,
					Symbol: "AAPL",
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					Open:   100.5,
					High:   102.0,
					Low:    100.0,
					Close:  101.5,
					Volume: 1500.0,
					Symbol: "AAPL",
				},
			},
			expectError: false,
		},
		{
			name: "Read empty data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			expectedData: []types.MarketData{},
			expectError:  false,
		},
	}

	// Run ReadAll tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData() // Ensure clean state before each subtest

			// Setup test data
			_, err := suite.ds.db.Exec(tc.setupData)
			suite.Require().NoError(err)

			// Collect results from ReadAll
			var results []types.MarketData
			iterator := suite.ds.ReadAll()
			iterator(func(data types.MarketData, err error) bool {
				if err != nil {
					suite.Assert().Fail("Unexpected error in ReadAll: %v", err)
					return false
				}
				results = append(results, data)
				return true
			})

			// Verify results
			suite.Assert().Equal(len(tc.expectedData), len(results), "Number of records mismatch")
			if len(tc.expectedData) > 0 {
				for i, expected := range tc.expectedData {
					suite.Assert().Equal(expected.Time.UTC(), results[i].Time.UTC(), "Time mismatch")
					suite.Assert().Equal(expected.Open, results[i].Open, "Open price mismatch")
					suite.Assert().Equal(expected.High, results[i].High, "High price mismatch")
					suite.Assert().Equal(expected.Low, results[i].Low, "Low price mismatch")
					suite.Assert().Equal(expected.Close, results[i].Close, "Close price mismatch")
					suite.Assert().Equal(expected.Volume, results[i].Volume, "Volume mismatch")
				}
			}
		})
	}
}

func (suite *DuckDBTestSuite) TestReadRange() {
	// Define test cases for ReadRange
	tests := []struct {
		name         string
		setupData    string
		start        time.Time
		end          time.Time
		interval     Interval
		expectedData []types.MarketData
		expectError  bool
	}{
		{
			name: "Read 1-minute data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 'AAPL', 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 'AAPL', 100.5, 102.0, 100.0, 101.5, 1500.0),
			('2024-01-01 10:02:00'::TIMESTAMP, 'AAPL', 101.5, 103.0, 101.0, 102.5, 2000.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			start:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 10, 2, 0, 0, time.UTC),
			interval: Interval1m,
			expectedData: []types.MarketData{
				{
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					Open:   100.0,
					High:   101.0,
					Low:    99.0,
					Close:  100.5,
					Volume: 1000.0,
					Symbol: "AAPL",
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					Open:   100.5,
					High:   102.0,
					Low:    100.0,
					Close:  101.5,
					Volume: 1500.0,
					Symbol: "AAPL",
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 2, 0, 0, time.UTC),
					Open:   101.5,
					High:   103.0,
					Low:    101.0,
					Close:  102.5,
					Volume: 2000.0,
					Symbol: "AAPL",
				},
			},
			expectError: false,
		},
		{
			name: "Read 5-minute aggregated data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 'AAPL', 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 'AAPL', 100.5, 102.0, 100.0, 101.5, 1500.0),
			('2024-01-01 10:02:00'::TIMESTAMP, 'AAPL', 101.5, 103.0, 101.0, 102.5, 2000.0),
			('2024-01-01 10:03:00'::TIMESTAMP, 'AAPL', 102.5, 104.0, 102.0, 103.5, 2500.0),
			('2024-01-01 10:04:00'::TIMESTAMP, 'AAPL', 103.5, 105.0, 103.0, 104.5, 3000.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			start:    time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			end:      time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
			interval: Interval5m,
			expectedData: []types.MarketData{
				{
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					Open:   100.0,
					High:   105.0,
					Low:    99.0,
					Close:  104.5,
					Volume: 10000.0,
					Symbol: "AAPL",
				},
			},
			expectError: false,
		},
		{
			name: "Read empty data range",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			start:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
			interval:     Interval5m,
			expectedData: []types.MarketData{},
			expectError:  false,
		},
		{
			name: "Invalid interval",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			start:        time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
			end:          time.Date(2024, 1, 1, 10, 5, 0, 0, time.UTC),
			interval:     Interval("invalid"),
			expectedData: nil,
			expectError:  true,
		},
	}

	// Run ReadRange tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData() // Ensure clean state before each subtest

			// Setup test data
			_, err := suite.ds.db.Exec(tc.setupData)
			suite.Require().NoError(err)

			// Test ReadRange
			results, err := suite.ds.ReadRange(tc.start, tc.end, tc.interval)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(len(tc.expectedData), len(results), "Number of records mismatch")

			if len(tc.expectedData) > 0 {
				for i, expected := range tc.expectedData {
					suite.Assert().Equal(expected.Time.UTC(), results[i].Time.UTC(), "Time mismatch")
					suite.Assert().Equal(expected.Open, results[i].Open, "Open price mismatch")
					suite.Assert().Equal(expected.High, results[i].High, "High price mismatch")
					suite.Assert().Equal(expected.Low, results[i].Low, "Low price mismatch")
					suite.Assert().Equal(expected.Close, results[i].Close, "Close price mismatch")
					suite.Assert().Equal(expected.Volume, results[i].Volume, "Volume mismatch")
				}
			}
		})
	}
}

func (suite *DuckDBTestSuite) TestNewDataSource() {
	tests := []struct {
		name        string
		path        string
		expectError bool
	}{
		{
			name:        "Valid in-memory database",
			path:        ":memory:",
			expectError: false,
		},
		{
			name:        "Valid file path",
			path:        "test.db",
			expectError: false,
		},
		{
			name:        "Invalid path",
			path:        "/invalid/path/test.db",
			expectError: true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			logger, err := logger.NewLogger()
			suite.Require().NoError(err)

			ds, err := NewDataSource(tc.path, logger)
			if tc.expectError {
				suite.Assert().Error(err)
				suite.Assert().Nil(ds)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().NotNil(ds)
			suite.Assert().NotNil(ds.(*DuckDBDataSource).db)
			suite.Assert().NotNil(ds.(*DuckDBDataSource).logger)

			// Clean up if file was created
			if tc.path != ":memory:" {
				ds.(*DuckDBDataSource).db.Close()
				os.Remove(tc.path)
			}
		})
	}
}

func (suite *DuckDBTestSuite) TestCount() {
	tests := []struct {
		name          string
		setupData     string
		expectedCount int
		expectError   bool
	}{
		{
			name: "Count with data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 'AAPL', 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 'AAPL', 100.5, 102.0, 100.0, 101.5, 1500.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			expectedCount: 2,
			expectError:   false,
		},
		{
			name: "Count with empty data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			expectedCount: 0,
			expectError:   false,
		},
		{
			name:          "Count with invalid view",
			setupData:     `DROP VIEW IF EXISTS market_data`,
			expectedCount: 0,
			expectError:   true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData()

			// Setup test data
			_, err := suite.ds.db.Exec(tc.setupData)
			suite.Require().NoError(err)

			// Test Count
			count, err := suite.ds.Count()
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(tc.expectedCount, count, "Count mismatch")
		})
	}
}

func (suite *DuckDBTestSuite) TestExecuteSQL() {
	tests := []struct {
		name         string
		setupData    string
		query        string
		params       []interface{}
		expectedRows []map[string]interface{}
		expectError  bool
	}{
		{
			name: "Execute SELECT query",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 'AAPL', 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 'AAPL', 100.5, 102.0, 100.0, 101.5, 1500.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			query:  "SELECT * FROM market_data WHERE symbol = ? ORDER BY time ASC",
			params: []interface{}{"AAPL"},
			expectedRows: []map[string]interface{}{
				{
					"time":   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					"symbol": "AAPL",
					"open":   100.0,
					"high":   101.0,
					"low":    99.0,
					"close":  100.5,
					"volume": 1000.0,
				},
				{
					"time":   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					"symbol": "AAPL",
					"open":   100.5,
					"high":   102.0,
					"low":    100.0,
					"close":  101.5,
					"volume": 1500.0,
				},
			},
			expectError: false,
		},
		{
			name: "Execute invalid query",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			query:        "SELECT * FROM nonexistent_table",
			params:       []interface{}{},
			expectedRows: nil,
			expectError:  true,
		},
		{
			name: "Execute query with wrong number of parameters",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				symbol TEXT,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			query:        "SELECT * FROM market_data WHERE symbol = ? AND time > ?",
			params:       []interface{}{"AAPL"}, // Missing second parameter
			expectedRows: nil,
			expectError:  true,
		},
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData()

			// Setup test data
			_, err := suite.ds.db.Exec(tc.setupData)
			suite.Require().NoError(err)

			// Test ExecuteSQL
			results, err := suite.ds.ExecuteSQL(tc.query, tc.params...)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}

			suite.Assert().NoError(err)
			suite.Assert().Equal(len(tc.expectedRows), len(results), "Number of rows mismatch")

			if len(tc.expectedRows) > 0 {
				for i, expected := range tc.expectedRows {
					for key, value := range expected {
						suite.Assert().Equal(value, results[i].Values[key], "Value mismatch for key: %s", key)
					}
				}
			}
		})
	}
}

// Helper function to write test data to parquet file
func writeTestDataToParquet(data []types.MarketData, filepath string) error {
	// Create a temporary DuckDB database
	db, err := sql.Open("duckdb", ":memory:")
	if err != nil {
		return err
	}
	defer db.Close()

	// Create table and insert data
	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol TEXT,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	if err != nil {
		return err
	}

	// Insert data
	for _, d := range data {
		_, err = db.Exec(`
			INSERT INTO market_data VALUES (?, ?, ?, ?, ?, ?, ?)
		`, d.Time, d.Symbol, d.Open, d.High, d.Low, d.Close, d.Volume)
		if err != nil {
			return err
		}
	}

	// Export to parquet
	_, err = db.Exec(fmt.Sprintf(`
		COPY market_data TO '%s' (FORMAT PARQUET)
	`, filepath))
	return err
}

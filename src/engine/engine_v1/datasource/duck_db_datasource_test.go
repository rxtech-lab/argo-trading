package datasource

import (
	"database/sql"
	"testing"
	"time"

	_ "github.com/marcboeker/go-duckdb"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/suite"
)

// DuckDBTestSuite is a test suite for DuckDBDataSource
type DuckDBTestSuite struct {
	suite.Suite
	ds *DuckDBDataSource
}

// SetupSuite runs once before all tests in the suite
func (suite *DuckDBTestSuite) SetupSuite() {
	db, err := sql.Open("duckdb", "")
	suite.Require().NoError(err)
	suite.ds = &DuckDBDataSource{db: db}
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
	// Define test cases for Initialize
	tests := []struct {
		name        string
		setupData   string // SQL to create test data
		parquetPath string // Path to parquet file (or invalid path for error case)
		expectError bool
	}{
		{
			name:        "Invalid parquet path",
			parquetPath: "nonexistent.parquet",
			expectError: true,
		},
	}

	// Run Initialize tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData() // Ensure clean state before each subtest

			// Test Initialize
			err := suite.ds.Initialize(tc.parquetPath)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}
			suite.Assert().NoError(err)
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
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 100.5, 102.0, 100.0, 101.5, 1500.0);
			CREATE VIEW market_data AS SELECT * FROM market_data_source`,
			expectedData: []types.MarketData{
				{
					Time:   time.Date(2024, 1, 1, 10, 0, 0, 0, time.UTC),
					Open:   100.0,
					High:   101.0,
					Low:    99.0,
					Close:  100.5,
					Volume: 1000.0,
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					Open:   100.5,
					High:   102.0,
					Low:    100.0,
					Close:  101.5,
					Volume: 1500.0,
				},
			},
			expectError: false,
		},
		{
			name: "Read empty data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
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
			suite.ds.ReadAll(func(data types.MarketData, err error) bool {
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
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 100.5, 102.0, 100.0, 101.5, 1500.0),
			('2024-01-01 10:02:00'::TIMESTAMP, 101.5, 103.0, 101.0, 102.5, 2000.0);
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
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 1, 0, 0, time.UTC),
					Open:   100.5,
					High:   102.0,
					Low:    100.0,
					Close:  101.5,
					Volume: 1500.0,
				},
				{
					Time:   time.Date(2024, 1, 1, 10, 2, 0, 0, time.UTC),
					Open:   101.5,
					High:   103.0,
					Low:    101.0,
					Close:  102.5,
					Volume: 2000.0,
				},
			},
			expectError: false,
		},
		{
			name: "Read 5-minute aggregated data",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
				open DOUBLE,
				high DOUBLE,
				low DOUBLE,
				close DOUBLE,
				volume DOUBLE
			);
			INSERT INTO market_data_source VALUES
			('2024-01-01 10:00:00'::TIMESTAMP, 100.0, 101.0, 99.0, 100.5, 1000.0),
			('2024-01-01 10:01:00'::TIMESTAMP, 100.5, 102.0, 100.0, 101.5, 1500.0),
			('2024-01-01 10:02:00'::TIMESTAMP, 101.5, 103.0, 101.0, 102.5, 2000.0),
			('2024-01-01 10:03:00'::TIMESTAMP, 102.5, 104.0, 102.0, 103.5, 2500.0),
			('2024-01-01 10:04:00'::TIMESTAMP, 103.5, 105.0, 103.0, 104.5, 3000.0);
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
				},
			},
			expectError: false,
		},
		{
			name: "Read empty data range",
			setupData: `CREATE TABLE market_data_source (
				time TIMESTAMP,
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

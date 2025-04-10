package datasource

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/alifiroozi80/duckdb"
	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/src/logger"
	"github.com/sirily11/argo-trading-go/src/types"
	"github.com/stretchr/testify/suite"
	"gorm.io/gorm"
)

// DuckDBTestSuite is a test suite for DuckDBDataSource
type DuckDBTestSuite struct {
	suite.Suite
	ds     *DuckDBDataSource
	logger *logger.Logger
}

// SetupSuite runs once before all tests in the suite
func (suite *DuckDBTestSuite) SetupSuite() {
	logger, err := logger.NewLogger()
	suite.Require().NoError(err)
	suite.logger = logger
}

// TearDownSuite runs once after all tests in the suite
func (suite *DuckDBTestSuite) TearDownSuite() {
	if suite.ds != nil {
		suite.ds.Close()
	}
}

// cleanupMarketData cleans up any market data files created during testing
func (suite *DuckDBTestSuite) cleanupMarketData() {
	// Remove any temporary test data files
	files, err := filepath.Glob("testdata_*.parquet")
	if err == nil {
		for _, file := range files {
			os.Remove(file)
		}
	}

	// Drop views and tables if they exist
	if suite.ds != nil && suite.ds.db != nil {
		sqlDB, err := suite.ds.db.DB()
		if err == nil {
			sqlDB.Exec("DROP VIEW IF EXISTS market_data")
			sqlDB.Exec("DROP TABLE IF EXISTS market_data_source")
		}
	}
}

// SetupTest runs before each test
func (suite *DuckDBTestSuite) SetupTest() {
	// Create a new in-memory database for each test
	db, err := gorm.Open(duckdb.Open(":memory:"), &gorm.Config{})
	suite.Require().NoError(err)

	// Create the datasource
	suite.ds = &DuckDBDataSource{
		db:     db,
		logger: suite.logger,
	}
}

// TearDownTest runs after each test
func (suite *DuckDBTestSuite) TearDownTest() {
	// Clean up any test files
	suite.cleanupMarketData()

	// Close the datasource
	if suite.ds != nil {
		suite.ds.Close()
		suite.ds = nil
	}
}

// TestDuckDBDataSourceSuite runs the test suite
func TestDuckDBDataSourceSuite(t *testing.T) {
	suite.Run(t, new(DuckDBTestSuite))
}

// Helper function to execute SQL statements for test setup
func (suite *DuckDBTestSuite) execSQL(sql string) {
	sqlDB, err := suite.ds.db.DB()
	suite.Require().NoError(err)

	_, err = sqlDB.Exec(sql)
	suite.Require().NoError(err)
}

func (suite *DuckDBTestSuite) TestInitialize() {
	// Test cases for Initialize
	tests := []struct {
		name        string
		filePath    string
		expectError bool
		setup       func() string
		validate    func(path string)
		cleanup     func(path string)
	}{
		// Test cases implementation
	}

	for _, tc := range tests {
		suite.Run(tc.name, func() {
			filePath := tc.setup()
			defer tc.cleanup(filePath)

			err := suite.ds.Initialize(filePath)
			if tc.expectError {
				suite.Assert().Error(err)
				return
			}
			suite.Assert().NoError(err)

			tc.validate(filePath)
		})
	}
}

func (suite *DuckDBTestSuite) TestReadAll() {
	// Define test cases for ReadAll
	tests := []struct {
		name         string
		setupData    string
		start        optional.Option[time.Time]
		end          optional.Option[time.Time]
		expectedData []types.MarketData
		expectError  bool
	}{
		// Test cases implementation
	}

	// Run ReadAll tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData() // Ensure clean state before each subtest

			// Setup test data
			suite.execSQL(tc.setupData)

			// Collect results from ReadAll
			var results []types.MarketData
			iterator := suite.ds.ReadAll(tc.start, tc.end)
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
					suite.Assert().Equal(expected.Symbol, results[i].Symbol, "Symbol mismatch")
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
		// Test cases implementation
	}

	// Run ReadRange tests
	for _, tc := range tests {
		suite.Run(tc.name, func() {
			suite.cleanupMarketData() // Ensure clean state before each subtest

			// Setup test data
			suite.execSQL(tc.setupData)

			// Test ReadRange
			results, err := suite.ds.GetRange(tc.start, tc.end, tc.interval)
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
					suite.Assert().Equal(expected.Symbol, results[i].Symbol, "Symbol mismatch")
				}
			}
		})
	}
}

// Replace all other test methods similarly, using the execSQL helper function
// instead of direct db.Exec calls

package indicator

import (
	"database/sql"
	"testing"
	"time"

	"github.com/moznion/go-optional"
	"github.com/sirily11/argo-trading-go/internal/backtest/engine/engine_v1/datasource"
	"github.com/sirily11/argo-trading-go/internal/types"
	"github.com/stretchr/testify/suite"
)

// EMATestSuite is a test suite for the EMA indicator
type EMATestSuite struct {
	suite.Suite
	db         *sql.DB
	dataSource datasource.DataSource
	ema        *EMA
}

// SetupSuite sets up the test suite
func (suite *EMATestSuite) SetupSuite() {
	// Create an in-memory DuckDB database for testing
	db, err := sql.Open("duckdb", ":memory:")
	suite.Require().NoError(err)
	suite.db = db

	// Create a mock data source that uses the real database
	dataSource := &databaseMockDataSource{
		db: db,
	}
	suite.dataSource = dataSource

	// Initialize EMA with a default period
	ema := NewEMA()
	ema.Config(20)
	suite.ema = ema.(*EMA)
	suite.Require().NotNil(suite.ema)

	// Create market_data table
	_, err = db.Exec(`
		CREATE TABLE market_data (
			time TIMESTAMP,
			symbol VARCHAR,
			open DOUBLE,
			high DOUBLE,
			low DOUBLE,
			close DOUBLE,
			volume DOUBLE
		)
	`)
	suite.Require().NoError(err)
}

// TearDownSuite tears down the test suite
func (suite *EMATestSuite) TearDownSuite() {
	if suite.db != nil {
		suite.db.Close()
	}
}

// SetupTest runs before each test
func (suite *EMATestSuite) SetupTest() {
	// Clear the market_data table before each test
	_, err := suite.db.Exec("DELETE FROM market_data")
	suite.Require().NoError(err)
}

// TestEMACalcSuite runs the test suite
func TestEMACalcSuite(t *testing.T) {
	suite.Run(t, new(EMATestSuite))
}

// databaseMockDataSource implements a mock data source that uses a real database
type databaseMockDataSource struct {
	db *sql.DB
}

func (m *databaseMockDataSource) Initialize(path string) error { return nil }
func (m *databaseMockDataSource) ReadAll(start, end optional.Option[time.Time]) func(yield func(types.MarketData, error) bool) {
	return nil
}
func (m *databaseMockDataSource) GetRange(start time.Time, end time.Time, interval optional.Option[datasource.Interval]) ([]types.MarketData, error) {
	return nil, nil
}
func (m *databaseMockDataSource) ReadLastData(symbol string) (types.MarketData, error) {
	return types.MarketData{}, nil
}

// ExecuteSQL executes SQL directly on the test database
func (m *databaseMockDataSource) ExecuteSQL(query string, params ...interface{}) ([]datasource.SQLResult, error) {
	rows, err := m.db.Query(query, params...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	results := []datasource.SQLResult{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		rowMap := make(map[string]interface{})
		for i, col := range columns {
			rowMap[col] = values[i]
		}

		results = append(results, datasource.SQLResult{Values: rowMap})
	}

	return results, nil
}

func (m *databaseMockDataSource) Count(start optional.Option[time.Time], end optional.Option[time.Time]) (int, error) {
	return 0, nil
}
func (m *databaseMockDataSource) Close() error { return nil }

// TestCalculateEMAFromSQL tests the calculateEMAFromSQL function
func (suite *EMATestSuite) TestCalculateEMAFromSQL() {
	// Define test cases
	testCases := []struct {
		name            string
		setupData       func() // Function to set up test data
		symbol          string
		period          int
		expectedEMA     float64
		expectedErrorIs string // Empty means no error expected
	}{
		{
			name: "Simple EMA calculation with constant price",
			setupData: func() {
				// Generate 60 data points with constant price 100.0
				baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				// Insert test data
				stmt, err := suite.db.Prepare(`
					INSERT INTO market_data (time, symbol, open, high, low, close, volume)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`)
				suite.Require().NoError(err)
				defer stmt.Close()

				for i := 0; i < 60; i++ {
					timestamp := baseTime.Add(time.Duration(i) * time.Hour)
					_, err = stmt.Exec(timestamp, "AAPL", 100.0, 100.0, 100.0, 100.0, 1000.0)
					suite.Require().NoError(err)
				}
			},
			symbol:      "AAPL",
			period:      20,
			expectedEMA: 100.0, // EMA of constant values is the constant
		},
		{
			name: "Linear increasing price",
			setupData: func() {
				// Generate 60 data points with linearly increasing price
				baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				stmt, err := suite.db.Prepare(`
					INSERT INTO market_data (time, symbol, open, high, low, close, volume)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`)
				suite.Require().NoError(err)
				defer stmt.Close()

				for i := 0; i < 60; i++ {
					timestamp := baseTime.Add(time.Duration(i) * time.Hour)
					price := 100.0 + float64(i)
					_, err = stmt.Exec(timestamp, "GOOGL", price, price, price, price, 1000.0)
					suite.Require().NoError(err)
				}
			},
			symbol: "GOOGL",
			period: 20,
			// For linearly increasing data, EMA will lag behind the latest price
			// The exact value depends on how EMA is calculated, but we can compute it manually
			expectedEMA: calculateExpectedEMA(20, func(i int) float64 { return 100.0 + float64(i) }, 60),
		},
		{
			name: "Fewer data points than period",
			setupData: func() {
				// Generate 10 data points (less than period 20)
				baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				stmt, err := suite.db.Prepare(`
					INSERT INTO market_data (time, symbol, open, high, low, close, volume)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`)
				suite.Require().NoError(err)
				defer stmt.Close()

				for i := 0; i < 10; i++ {
					timestamp := baseTime.Add(time.Duration(i) * time.Hour)
					_, err = stmt.Exec(timestamp, "MSFT", 100.0, 100.0, 100.0, 100.0, 1000.0)
					suite.Require().NoError(err)
				}
			},
			symbol:      "MSFT",
			period:      20,
			expectedEMA: 100.0, // Simple average of all points
		},
		{
			name: "No data for symbol",
			setupData: func() {
				// Generate data for a different symbol
				baseTime := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)

				stmt, err := suite.db.Prepare(`
					INSERT INTO market_data (time, symbol, open, high, low, close, volume)
					VALUES (?, ?, ?, ?, ?, ?, ?)
				`)
				suite.Require().NoError(err)
				defer stmt.Close()

				for i := 0; i < 10; i++ {
					timestamp := baseTime.Add(time.Duration(i) * time.Hour)
					_, err = stmt.Exec(timestamp, "AMZN", 100.0, 100.0, 100.0, 100.0, 1000.0)
					suite.Require().NoError(err)
				}
			},
			symbol:          "TSLA", // Query for a symbol with no data
			period:          20,
			expectedErrorIs: "no data found",
		},
	}

	for _, tc := range testCases {
		suite.Run(tc.name, func() {
			// Clear previous test data
			_, err := suite.db.Exec("DELETE FROM market_data")
			suite.Require().NoError(err)

			// Set up test data
			tc.setupData()

			// Create indicator context
			ctx := IndicatorContext{
				DataSource: suite.dataSource,
			}

			// Execute the test
			currentTime := time.Date(2023, 1, 31, 0, 0, 0, 0, time.UTC) // Time after all data points
			result, err := suite.ema.calculateEMAFromSQL(tc.symbol, currentTime, &ctx, optional.Some(tc.period))

			if tc.expectedErrorIs != "" {
				suite.Require().Error(err)
				suite.Require().Contains(err.Error(), tc.expectedErrorIs)
			} else {
				suite.Require().NoError(err)
				suite.Assert().InDelta(tc.expectedEMA, result, 0.2, "EMA calculation mismatch")
			}
		})
	}
}

// Helper function to calculate expected EMA for test verification
func calculateExpectedEMA(period int, priceFunc func(int) float64, numPoints int) float64 {
	// Calculate SMA for the first 'period' points as seed
	sum := 0.0
	for i := 0; i < period; i++ {
		sum += priceFunc(i)
	}
	ema := sum / float64(period)

	// Apply EMA formula for subsequent points
	multiplier := 2.0 / (float64(period) + 1.0)
	for i := period; i < numPoints; i++ {
		ema = (priceFunc(i)-ema)*multiplier + ema
	}

	return ema
}

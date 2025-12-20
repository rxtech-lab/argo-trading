package datasource

import (
	"testing"

	"github.com/stretchr/testify/suite"
)

type DatasourceUtilsTestSuite struct {
	suite.Suite
}

func TestDatasourceUtilsSuite(t *testing.T) {
	suite.Run(t, new(DatasourceUtilsTestSuite))
}

func (suite *DatasourceUtilsTestSuite) TestGetIntervalMinutes() {
	tests := []struct {
		interval        Interval
		expectedMinutes int
		expectError     bool
	}{
		{Interval1m, 1, false},
		{Interval5m, 5, false},
		{Interval15m, 15, false},
		{Interval30m, 30, false},
		{Interval1h, 60, false},
		{Interval4h, 240, false},
		{Interval6h, 360, false},
		{Interval8h, 480, false},
		{Interval12h, 720, false},
		{Interval1d, 1440, false},
		{Interval1w, 10080, false},
	}

	for _, tc := range tests {
		suite.Run(string(tc.interval), func() {
			minutes, err := getIntervalMinutes(tc.interval)

			if tc.expectError {
				suite.Error(err)
			} else {
				suite.NoError(err)
				suite.Equal(tc.expectedMinutes, minutes)
			}
		})
	}
}

func (suite *DatasourceUtilsTestSuite) TestGetIntervalMinutesUnsupportedInterval() {
	unsupportedInterval := Interval("invalid")
	minutes, err := getIntervalMinutes(unsupportedInterval)

	suite.Error(err)
	suite.Equal(0, minutes)
	suite.Contains(err.Error(), "unsupported interval")
	suite.Contains(err.Error(), "invalid")
}

func (suite *DatasourceUtilsTestSuite) TestGetIntervalMinutesEmptyInterval() {
	emptyInterval := Interval("")
	minutes, err := getIntervalMinutes(emptyInterval)

	suite.Error(err)
	suite.Equal(0, minutes)
}

func (suite *DatasourceUtilsTestSuite) TestGetIntervalMinutesMonthlyNotSupported() {
	// Interval1M is defined as a constant but not handled in the switch
	monthlyInterval := Interval1M
	minutes, err := getIntervalMinutes(monthlyInterval)

	suite.Error(err)
	suite.Equal(0, minutes)
	suite.Contains(err.Error(), "unsupported interval")
}

func (suite *DatasourceUtilsTestSuite) TestIntervalConstants() {
	suite.Equal(Interval("1m"), Interval1m)
	suite.Equal(Interval("5m"), Interval5m)
	suite.Equal(Interval("15m"), Interval15m)
	suite.Equal(Interval("30m"), Interval30m)
	suite.Equal(Interval("1h"), Interval1h)
	suite.Equal(Interval("4h"), Interval4h)
	suite.Equal(Interval("6h"), Interval6h)
	suite.Equal(Interval("8h"), Interval8h)
	suite.Equal(Interval("12h"), Interval12h)
	suite.Equal(Interval("1d"), Interval1d)
	suite.Equal(Interval("1w"), Interval1w)
	suite.Equal(Interval("1M"), Interval1M)
}

func (suite *DatasourceUtilsTestSuite) TestSQLResultStruct() {
	result := SQLResult{
		Values: map[string]interface{}{
			"column1": "value1",
			"column2": 123,
			"column3": 45.67,
			"column4": true,
		},
	}

	suite.Equal("value1", result.Values["column1"])
	suite.Equal(123, result.Values["column2"])
	suite.Equal(45.67, result.Values["column3"])
	suite.Equal(true, result.Values["column4"])
}

func (suite *DatasourceUtilsTestSuite) TestSQLResultEmptyValues() {
	result := SQLResult{
		Values: map[string]interface{}{},
	}

	suite.Empty(result.Values)
	suite.Nil(result.Values["nonexistent"])
}

func (suite *DatasourceUtilsTestSuite) TestSQLResultNilValues() {
	result := SQLResult{}

	suite.Nil(result.Values)
}

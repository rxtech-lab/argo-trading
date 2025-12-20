package marketdata

import (
	"testing"

	"github.com/polygon-io/client-go/rest/models"
	"github.com/stretchr/testify/suite"
)

type TimespanTestSuite struct {
	suite.Suite
}

func TestTimespanSuite(t *testing.T) {
	suite.Run(t, new(TimespanTestSuite))
}

func (suite *TimespanTestSuite) TestMultiplier() {
	tests := []struct {
		timespan Timespan
		expected int
	}{
		{TimespanOneSecond, 1},
		{TimespanOneMinute, 1},
		{TimespanThreeMinutes, 3},
		{TimespanFiveMinutes, 5},
		{TimespanFifteenMinutes, 15},
		{TimespanThirtyMinutes, 30},
		{TimespanOneHour, 1},
		{TimespanTwoHours, 2},
		{TimespanFourHours, 4},
		{TimespanSixHours, 6},
		{TimespanEightHours, 8},
		{TimespanTwelveHours, 12},
		{TimespanOneDay, 1},
		{TimespanThreeDays, 3},
		{TimespanOneWeek, 1},
		{TimespanOneMonth, 1},
	}

	for _, tc := range tests {
		suite.Run(string(tc.timespan), func() {
			result := tc.timespan.Multiplier()
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *TimespanTestSuite) TestMultiplierDefault() {
	// Test with an unknown timespan
	unknownTimespan := Timespan("unknown")
	result := unknownTimespan.Multiplier()
	suite.Equal(1, result)
}

func (suite *TimespanTestSuite) TestTimespan() {
	tests := []struct {
		timespan Timespan
		expected models.Timespan
	}{
		{TimespanOneSecond, models.Second},
		{TimespanOneMinute, models.Minute},
		{TimespanThreeMinutes, models.Minute},
		{TimespanFiveMinutes, models.Minute},
		{TimespanFifteenMinutes, models.Minute},
		{TimespanThirtyMinutes, models.Minute},
		{TimespanOneHour, models.Hour},
		{TimespanTwoHours, models.Hour},
		{TimespanFourHours, models.Hour},
		{TimespanSixHours, models.Hour},
		{TimespanEightHours, models.Hour},
		{TimespanTwelveHours, models.Hour},
		{TimespanOneDay, models.Day},
		{TimespanThreeDays, models.Day},
		{TimespanOneWeek, models.Week},
		{TimespanOneMonth, models.Month},
	}

	for _, tc := range tests {
		suite.Run(string(tc.timespan), func() {
			result := tc.timespan.Timespan()
			suite.Equal(tc.expected, result)
		})
	}
}

func (suite *TimespanTestSuite) TestTimespanDefault() {
	// Test with an unknown timespan
	unknownTimespan := Timespan("unknown")
	result := unknownTimespan.Timespan()
	suite.Equal(models.Day, result)
}

func (suite *TimespanTestSuite) TestTimespanConstants() {
	// Verify the constant values are as expected
	suite.Equal(Timespan("1s"), TimespanOneSecond)
	suite.Equal(Timespan("1m"), TimespanOneMinute)
	suite.Equal(Timespan("3m"), TimespanThreeMinutes)
	suite.Equal(Timespan("5m"), TimespanFiveMinutes)
	suite.Equal(Timespan("15m"), TimespanFifteenMinutes)
	suite.Equal(Timespan("30m"), TimespanThirtyMinutes)
	suite.Equal(Timespan("1h"), TimespanOneHour)
	suite.Equal(Timespan("2h"), TimespanTwoHours)
	suite.Equal(Timespan("4h"), TimespanFourHours)
	suite.Equal(Timespan("6h"), TimespanSixHours)
	suite.Equal(Timespan("8h"), TimespanEightHours)
	suite.Equal(Timespan("12h"), TimespanTwelveHours)
	suite.Equal(Timespan("1d"), TimespanOneDay)
	suite.Equal(Timespan("3d"), TimespanThreeDays)
	suite.Equal(Timespan("1w"), TimespanOneWeek)
	suite.Equal(Timespan("1M"), TimespanOneMonth)
}

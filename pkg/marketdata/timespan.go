package marketdata

import "github.com/polygon-io/client-go/rest/models"

type Timespan string

const (
	TimespanOneSecond      Timespan = "1s"
	TimespanOneMinute      Timespan = "1m"
	TimespanThreeMinutes   Timespan = "3m"
	TimespanFiveMinutes    Timespan = "5m"
	TimespanFifteenMinutes Timespan = "15m"
	TimespanThirtyMinutes  Timespan = "30m"
	TimespanOneHour        Timespan = "1h"
	TimespanTwoHours       Timespan = "2h"
	TimespanFourHours      Timespan = "4h"
	TimespanSixHours       Timespan = "6h"
	TimespanEightHours     Timespan = "8h"
	TimespanTwelveHours    Timespan = "12h"
	TimespanOneDay         Timespan = "1d"
	TimespanThreeDays      Timespan = "3d"
	TimespanOneWeek        Timespan = "1w"
	TimespanOneMonth       Timespan = "1M"
)

func (t Timespan) Multiplier() int {
	switch t {
	case TimespanOneSecond:
		return 1
	case TimespanOneMinute:
		return 1
	case TimespanThreeMinutes:
		return 3
	case TimespanFiveMinutes:
		return 5
	case TimespanFifteenMinutes:
		return 15
	case TimespanThirtyMinutes:
		return 30
	case TimespanOneHour:
		return 1
	case TimespanTwoHours:
		return 2
	case TimespanFourHours:
		return 4
	case TimespanSixHours:
		return 6
	case TimespanEightHours:
		return 8
	case TimespanTwelveHours:
		return 12
	case TimespanOneDay:
		return 1
	case TimespanThreeDays:
		return 3
	case TimespanOneWeek:
		return 1
	case TimespanOneMonth:
		return 1
	default:
		return 1
	}
}

func (t Timespan) Timespan() models.Timespan {
	switch t {
	case TimespanOneSecond:
		return models.Second
	case TimespanOneMinute, TimespanThreeMinutes, TimespanFiveMinutes, TimespanFifteenMinutes, TimespanThirtyMinutes:
		return models.Minute
	case TimespanOneHour, TimespanTwoHours, TimespanFourHours, TimespanSixHours, TimespanEightHours, TimespanTwelveHours:
		return models.Hour
	case TimespanOneDay, TimespanThreeDays:
		return models.Day
	case TimespanOneWeek:
		return models.Week
	case TimespanOneMonth:
		return models.Month
	default:
		return models.Day
	}
}

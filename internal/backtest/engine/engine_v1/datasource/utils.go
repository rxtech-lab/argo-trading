package datasource

import "fmt"

func getIntervalMinutes(interval Interval) (int, error) {
	var intervalMinutes int

	switch interval {
	case Interval1m:
		intervalMinutes = 1
	case Interval5m:
		intervalMinutes = 5
	case Interval15m:
		intervalMinutes = 15
	case Interval30m:
		intervalMinutes = 30
	case Interval1h:
		intervalMinutes = 60
	case Interval4h:
		intervalMinutes = 240
	case Interval6h:
		intervalMinutes = 360
	case Interval8h:
		intervalMinutes = 480
	case Interval12h:
		intervalMinutes = 720
	case Interval1d:
		intervalMinutes = 1440
	case Interval1w:
		intervalMinutes = 10080
	default:
		return 0, fmt.Errorf("unsupported interval: %s", interval)
	}

	return intervalMinutes, nil
}

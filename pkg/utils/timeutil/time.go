package timeutil

import (
	"time"
)

// TimeRange represents a time range with start and end time.
type TimeRange struct {
	Start time.Time
	End   time.Time
}

// NewTimeRange creates a TimeRange from start/end timestamps or duration.
func NewTimeRange(start, end int64, rangeDuration string) TimeRange {
	if start > 0 && end > 0 {
		return TimeRange{
			Start: parseUnixTimestamp(start),
			// The sequence is inclusive, so we add 1 to the end time to make it exclusive.
			End: parseUnixTimestamp(end + 1),
		}
	}

	endTime := time.Now()
	if rangeDuration != "" {
		if duration, err := time.ParseDuration(rangeDuration); err == nil {
			return TimeRange{
				Start: endTime.Add(-duration),
				End:   endTime,
			}
		}
	}

	// Default to 1 hour if no valid range is provided
	return TimeRange{
		Start: endTime.Add(-time.Hour),
		End:   endTime,
	}
}

// Infer the time unit of timestamp.
func parseUnixTimestamp(ts int64) time.Time {
	switch {
	case ts > 1e18:
		// ns
		return time.Unix(0, ts)
	case ts > 1e15:
		// µs
		return time.Unix(0, ts*1e3)
	case ts > 1e12:
		// ms
		return time.Unix(0, ts*1e6)
	default:
		// s
		return time.Unix(ts, 0)
	}
}

// Duration returns the duration of the time range.
func (tr TimeRange) Duration() time.Duration {
	return tr.End.Sub(tr.Start)
}

// CalculateOptimalStep calculates the optimal step size based on time range.
func (tr TimeRange) calculateOptimalStep() time.Duration {
	duration := tr.Duration()

	switch {
	case duration <= time.Hour:
		// 1 hour: 30 seconds per step (120 points).
		return 30 * time.Second
	case duration <= 3*time.Hour:
		// 3 hours: 1 minute per step (180 points).
		return 1 * time.Minute
	case duration <= 6*time.Hour:
		// 6 hours: 2 minutes per step (180 points).
		return 2 * time.Minute
	case duration <= 12*time.Hour:
		// 12 hours: 5 minutes per step (144 points).
		return 5 * time.Minute
	case duration <= 24*time.Hour:
		// 24 hours: 10 minutes per step (144 points).
		return 10 * time.Minute
	case duration <= 7*24*time.Hour:
		// 1 week: 2 hour per step (168 points).
		return 2 * time.Hour
	case duration <= 30*24*time.Hour:
		// 1 month: 4 hours per step (180 points).
		return 8 * time.Hour
	default:
		// Longer periods: 1 day per step.
		return 24 * time.Hour
	}
}

func (tr TimeRange) GetStepAndAlignRange(step time.Duration) time.Duration {
	if step == 0 {
		step = tr.calculateOptimalStep()
	}
	// Align the range to the step to get stable points.
	tr.Start = tr.Start.Truncate(step)
	tr.End = tr.End.Truncate(step).Add(step)
	return step
}

func Format(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

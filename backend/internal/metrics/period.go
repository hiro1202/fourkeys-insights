package metrics

import "time"

// Period represents a time range for metrics calculation.
type Period struct {
	Start time.Time
	End   time.Time
	Days  int
}

// CalcPeriod returns the most recent confirmed period for the given aggregation unit.
// Weekly: previous Monday 00:00 UTC to Sunday 23:59:59 UTC.
// Monthly: previous 1st 00:00 UTC to last day 23:59:59 UTC.
func CalcPeriod(now time.Time, unit string) Period {
	now = now.UTC()
	switch unit {
	case "monthly":
		// Previous month
		firstOfThisMonth := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := firstOfThisMonth.Add(-time.Second) // last second of prev month
		start := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
		days := int(end.Sub(start).Hours()/24) + 1
		return Period{Start: start, End: end, Days: days}
	default: // "weekly"
		// Previous week (Mon-Sun)
		// Find the most recent Monday (start of this week)
		weekday := now.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		daysFromMonday := int(weekday) - int(time.Monday)
		thisMonday := time.Date(now.Year(), now.Month(), now.Day()-daysFromMonday, 0, 0, 0, 0, time.UTC)
		// Previous week
		start := thisMonday.AddDate(0, 0, -7)
		end := thisMonday.Add(-time.Second) // Sunday 23:59:59
		return Period{Start: start, End: end, Days: 7}
	}
}

// CalcPreviousPeriod returns the period before the given period.
func CalcPreviousPeriod(p Period, unit string) Period {
	switch unit {
	case "monthly":
		firstOfMonth := time.Date(p.Start.Year(), p.Start.Month(), 1, 0, 0, 0, 0, time.UTC)
		end := firstOfMonth.Add(-time.Second)
		start := time.Date(end.Year(), end.Month(), 1, 0, 0, 0, 0, time.UTC)
		days := int(end.Sub(start).Hours()/24) + 1
		return Period{Start: start, End: end, Days: days}
	default: // "weekly"
		start := p.Start.AddDate(0, 0, -7)
		end := p.Start.Add(-time.Second)
		return Period{Start: start, End: end, Days: 7}
	}
}

// CalcTrendPeriods generates a list of periods between since and until for trend charts.
func CalcTrendPeriods(since, until time.Time, unit string) []Period {
	since = since.UTC()
	until = until.UTC()
	var periods []Period

	switch unit {
	case "monthly":
		// Start from the 1st of since's month
		cur := time.Date(since.Year(), since.Month(), 1, 0, 0, 0, 0, time.UTC)
		for cur.Before(until) {
			nextMonth := time.Date(cur.Year(), cur.Month()+1, 1, 0, 0, 0, 0, time.UTC)
			end := nextMonth.Add(-time.Second)
			if end.After(until) {
				break // don't include incomplete month
			}
			days := int(end.Sub(cur).Hours()/24) + 1
			periods = append(periods, Period{Start: cur, End: end, Days: days})
			cur = nextMonth
		}
	default: // "weekly"
		// Start from the Monday of since's week
		weekday := since.Weekday()
		if weekday == time.Sunday {
			weekday = 7
		}
		daysFromMonday := int(weekday) - int(time.Monday)
		cur := time.Date(since.Year(), since.Month(), since.Day()-daysFromMonday, 0, 0, 0, 0, time.UTC)
		for cur.Before(until) {
			end := cur.AddDate(0, 0, 7).Add(-time.Second)
			if end.After(until) {
				break // don't include incomplete week
			}
			periods = append(periods, Period{Start: cur, End: end, Days: 7})
			cur = cur.AddDate(0, 0, 7)
		}
	}

	return periods
}

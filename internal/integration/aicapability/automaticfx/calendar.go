package automaticfx

import (
	"errors"
	"net/url"
	"time"
)

var ErrInvalidCalendar = errors.New("invalid publication calendar")

// YearCalendar is trusted, pre-verified configuration for the onshore RMB
// market, not a calendar inferred from the ordinary office work week.
type YearCalendar struct {
	Source             string
	CoverageFrom       string
	CoverageUntil      string // inclusive
	ExceptionsComplete bool   // trusted operator attests all exceptions in coverage
	Closed             []string
	Open               []string // official exceptional openings override normal weekends
}

type Calendar struct{ Years map[int]YearCalendar }

func (c Calendar) NextExpiry(announcementDate string) (time.Time, error) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	day, err := time.ParseInLocation("2006-01-02", announcementDate, loc)
	if err != nil || day.Format("2006-01-02") != announcementDate {
		return time.Time{}, ErrInvalidCalendar
	}
	for i := 1; i <= 370; i++ {
		next := day.AddDate(0, 0, i)
		fallback := time.Date(next.Year(), next.Month(), next.Day(), 9, 15, 0, 0, loc)
		known, open, err := c.classify(next)
		if err != nil {
			return time.Time{}, err
		}
		if !known || open {
			return fallback, nil
		}
	}
	return time.Time{}, ErrInvalidCalendar
}

// Eligible rejects dates for which the publication day or 09:15 release is
// not established. Caller must apply it before preparing a publication.
func (c Calendar) Eligible(date string, now time.Time) bool {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	day, err := time.ParseInLocation("2006-01-02", date, loc)
	if err != nil || day.Format("2006-01-02") != date || now.IsZero() {
		return false
	}
	known, open, err := c.classify(day)
	release := time.Date(day.Year(), day.Month(), day.Day(), 9, 15, 0, 0, loc)
	return err == nil && known && open && !now.Before(release)
}

func (c Calendar) classify(day time.Time) (bool, bool, error) {
	calendar, ok := c.Years[day.Year()]
	if !ok {
		return false, false, nil
	}
	u, err := url.Parse(calendar.Source)
	if err != nil || u.Scheme != "https" || (u.Hostname() != "www.chinamoney.com.cn" && u.Hostname() != "www.chinamoney.org.cn") || u.User != nil || u.Fragment != "" || u.Port() != "" {
		return false, false, ErrInvalidCalendar
	}
	if !calendar.ExceptionsComplete {
		return false, false, nil
	}
	from, e1 := time.Parse("2006-01-02", calendar.CoverageFrom)
	until, e2 := time.Parse("2006-01-02", calendar.CoverageUntil)
	if e1 != nil || e2 != nil || from.Year() != day.Year() || until.Year() != day.Year() || from.After(until) {
		return false, false, ErrInvalidCalendar
	}
	seen := map[string]bool{}
	for _, row := range append(append([]string(nil), calendar.Closed...), calendar.Open...) {
		d, e := time.Parse("2006-01-02", row)
		if e != nil || d.Year() != day.Year() || d.Format("2006-01-02") != row || row < calendar.CoverageFrom || row > calendar.CoverageUntil || seen[row] {
			return false, false, ErrInvalidCalendar
		}
		seen[row] = true
	}
	text := day.Format("2006-01-02")
	if text < calendar.CoverageFrom || text > calendar.CoverageUntil {
		return false, false, nil
	}
	for _, row := range calendar.Closed {
		if row == text {
			return true, false, nil
		}
	}
	for _, row := range calendar.Open {
		if row == text {
			return true, true, nil
		}
	}
	return true, day.Weekday() != time.Saturday && day.Weekday() != time.Sunday, nil
}

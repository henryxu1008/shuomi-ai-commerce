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
	Source string
	Closed []string
	Open   []string // official exceptional openings override normal weekends
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
		calendar, ok := c.Years[next.Year()]
		if !ok {
			return fallback, nil
		}
		u, err := url.Parse(calendar.Source)
		if err != nil || u.Scheme != "https" || (u.Hostname() != "www.chinamoney.com.cn" && u.Hostname() != "www.chinamoney.org.cn") || u.User != nil || u.Fragment != "" || u.Port() != "" {
			return time.Time{}, ErrInvalidCalendar
		}
		closed, open := false, false
		seen := map[string]bool{}
		for _, row := range append(append([]string(nil), calendar.Closed...), calendar.Open...) {
			d, e := time.Parse("2006-01-02", row)
			if e != nil || d.Year() != next.Year() || d.Format("2006-01-02") != row || seen[row] {
				return time.Time{}, ErrInvalidCalendar
			}
			seen[row] = true
		}
		for _, row := range calendar.Closed {
			closed = closed || row == next.Format("2006-01-02")
		}
		for _, row := range calendar.Open {
			open = open || row == next.Format("2006-01-02")
		}
		if open || (!closed && next.Weekday() != time.Saturday && next.Weekday() != time.Sunday) {
			return fallback, nil
		}
	}
	return time.Time{}, ErrInvalidCalendar
}

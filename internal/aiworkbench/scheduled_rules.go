package aiworkbench

import (
	"errors"
	"math"
	"strings"
	"time"
)

// ScheduleFrequency is the cadence of a user-approved bounded plan.
// An occurrence always has a single UTC slot independent of worker restarts.
type ScheduleFrequency string

const (
	ScheduleOnce   ScheduleFrequency = "once"
	ScheduleDaily  ScheduleFrequency = "daily"
	ScheduleWeekly ScheduleFrequency = "weekly"
)

var (
	ErrInvalidSchedule        = errors.New("invalid workbench schedule")
	ErrScheduleEnded          = errors.New("workbench schedule has no next occurrence")
	ErrScheduleCatchupPending = errors.New("workbench schedule missed-slot recording is incomplete")
)

// ScheduleRule describes wall-clock recurrence without using a process-local
// timezone. For recurring rules, Weekday is consulted only for weekly cadence.
// Daylight-saving gaps are skipped; ambiguous fall-back hours run once.
type ScheduleRule struct {
	Frequency ScheduleFrequency `json:"frequency"`
	TimeZone  string            `json:"timeZone"`
	AtUTC     time.Time         `json:"atUtc,omitzero"`
	Hour      int               `json:"hour"`
	Minute    int               `json:"minute"`
	Weekday   time.Weekday      `json:"weekday"`
	UntilUTC  time.Time         `json:"untilUtc,omitzero"`
}

// Valid intentionally separates calendar shape from budget/permission admission.
// Those constraints belong to the plan service and the original resource owner.
func (r ScheduleRule) Valid() bool {
	if r.TimeZone == "" || strings.TrimSpace(r.TimeZone) != r.TimeZone {
		return false
	}
	// "Local" and short timezone abbreviations depend on the worker's machine
	// and must never become durable scheduling semantics.
	if r.TimeZone != "UTC" && !strings.Contains(r.TimeZone, "/") {
		return false
	}
	if _, err := time.LoadLocation(r.TimeZone); err != nil {
		return false
	}
	if !r.UntilUTC.IsZero() && (r.UntilUTC.Year() < 1970 || r.UntilUTC.Year() > 9999) {
		return false
	}
	switch r.Frequency {
	case ScheduleOnce:
		return !r.AtUTC.IsZero() && r.AtUTC.Year() >= 1970 &&
			r.AtUTC.Year() <= 9999 && (r.UntilUTC.IsZero() || !r.AtUTC.After(r.UntilUTC))
	case ScheduleDaily, ScheduleWeekly:
		if !r.AtUTC.IsZero() || r.Hour < 0 || r.Hour > 23 || r.Minute < 0 || r.Minute > 59 {
			return false
		}
		return r.Frequency != ScheduleWeekly || (r.Weekday >= time.Sunday && r.Weekday <= time.Saturday)
	default:
		return false
	}
}

// NextAfter returns the first scheduled UTC slot strictly after the supplied
// instant. The caller must persist this exact slot as the occurrence identity.
// A skipped spring-forward local time is not silently shifted to another hour.
func (r ScheduleRule) NextAfter(after time.Time) (time.Time, error) {
	if !r.Valid() || after.IsZero() {
		return time.Time{}, ErrInvalidSchedule
	}
	if r.Frequency == ScheduleOnce {
		t := r.AtUTC.UTC()
		if t.After(after) {
			return t, nil
		}
		return time.Time{}, ErrScheduleEnded
	}
	loc, _ := time.LoadLocation(r.TimeZone)
	local := after.In(loc)
	year, month, day := local.Date()
	// A weekly slot can be missing at a spring transition. Include the next
	// eligible week (fourteen days from a just-consumed weekly occurrence).
	for offset := 0; offset <= 14; offset++ {
		date := time.Date(year, month, day+offset, 12, 0, 0, 0, loc)
		if r.Frequency == ScheduleWeekly && date.Weekday() != r.Weekday {
			continue
		}
		y, m, d := date.Date()
		slot := time.Date(y, m, d, r.Hour, r.Minute, 0, 0, loc)
		ly, lm, ld := slot.In(loc).Date()
		if ly != y || lm != m || ld != d ||
			slot.In(loc).Hour() != r.Hour || slot.In(loc).Minute() != r.Minute {
			continue // nonexistent local clock time at DST transition
		}
		slot = slot.UTC()
		if !slot.After(after) {
			continue
		}
		if !r.UntilUTC.IsZero() && slot.After(r.UntilUTC) {
			return time.Time{}, ErrScheduleEnded
		}
		return slot, nil
	}
	return time.Time{}, ErrScheduleEnded
}

// ScheduleBudget is an authorization ceiling, not a shadow charge ledger.
// Per-run cost must already be a conservative bound supplied and accepted by
// the existing AI/resource owner; estimated-only prices do not qualify.
type ScheduleBudget struct {
	Unit                   string
	MaxOccurrences         uint64
	PerRunUpperBoundMicros uint64
}

func (b ScheduleBudget) TotalUpperBoundMicros() (uint64, error) {
	if b.MaxOccurrences == 0 || b.PerRunUpperBoundMicros == 0 ||
		b.MaxOccurrences > math.MaxUint64/b.PerRunUpperBoundMicros {
		return 0, ErrInvalidSchedule
	}
	return b.MaxOccurrences * b.PerRunUpperBoundMicros, nil
}

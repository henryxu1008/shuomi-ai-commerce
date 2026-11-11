package aiworkbench

import (
	"errors"
	"math"
	"testing"
	"time"
)

func TestScheduledRuleOnceNoReplay(t *testing.T) {
	start := time.Date(2026, 11, 12, 12, 30, 0, 0, time.UTC)
	rule := ScheduleRule{Frequency: ScheduleOnce, TimeZone: "Asia/Shanghai", AtUTC: start}
	got, err := rule.NextAfter(start.Add(-time.Second))
	if err != nil || !got.Equal(start) {
		t.Fatalf("first slot = %s, %v", got, err)
	}
	if _, err := rule.NextAfter(start); !errors.Is(err, ErrScheduleEnded) {
		t.Fatalf("same slot replay must not create a second execution: %v", err)
	}
}

func TestScheduledDailyPreservesLocalClock(t *testing.T) {
	rule := ScheduleRule{Frequency: ScheduleDaily, TimeZone: "Asia/Shanghai", Hour: 22, Minute: 15}
	start := time.Date(2026, 11, 11, 14, 15, 0, 0, time.UTC)
	got, err := rule.NextAfter(start)
	want := time.Date(2026, 11, 12, 14, 15, 0, 0, time.UTC)
	if err != nil || !got.Equal(want) {
		t.Fatalf("next = %s, %v, want %s", got, err, want)
	}
}

func TestScheduledWeeklyRespectsWeekday(t *testing.T) {
	rule := ScheduleRule{Frequency: ScheduleWeekly, TimeZone: "UTC", Weekday: time.Monday, Hour: 9}
	start := time.Date(2026, 11, 9, 9, 0, 0, 0, time.UTC) // Monday
	got, err := rule.NextAfter(start)
	want := start.AddDate(0, 0, 7)
	if err != nil || !got.Equal(want) {
		t.Fatalf("next = %s, %v, want %s", got, err, want)
	}
}

func TestScheduleSkipsDSTNonexistentLocalTime(t *testing.T) {
	// New York advances from 01:59 to 03:00 on 2027-03-14.
	rule := ScheduleRule{Frequency: ScheduleDaily, TimeZone: "America/New_York", Hour: 2, Minute: 30}
	after := time.Date(2027, 3, 14, 5, 0, 0, 0, time.UTC)
	got, err := rule.NextAfter(after)
	want := time.Date(2027, 3, 15, 6, 30, 0, 0, time.UTC)
	if err != nil || !got.Equal(want) {
		t.Fatalf("next after DST gap = %s, %v, want %s", got, err, want)
	}
}

func TestScheduleFallBackCreatesOneSlotPerDay(t *testing.T) {
	rule := ScheduleRule{Frequency: ScheduleDaily, TimeZone: "America/New_York", Hour: 1, Minute: 30}
	after := time.Date(2027, 11, 7, 4, 0, 0, 0, time.UTC)
	first, err := rule.NextAfter(after)
	if err != nil {
		t.Fatal(err)
	}
	// Both 05:30 and 06:30 UTC represent 01:30 local on fall-back day.
	// The same local date must not produce two scheduled occurrences.
	next, err := rule.NextAfter(first)
	if err != nil {
		t.Fatal(err)
	}
	loc, _ := time.LoadLocation(rule.TimeZone)
	fy, fm, fd := first.In(loc).Date()
	ny, nm, nd := next.In(loc).Date()
	if fy == ny && fm == nm && fd == nd {
		t.Fatalf("repeated wall-clock hour scheduled twice: %s and %s", first, next)
	}
}

func TestScheduledUntilAndInvalidZones(t *testing.T) {
	rule := ScheduleRule{Frequency: ScheduleDaily, TimeZone: "UTC", Hour: 10,
		UntilUTC: time.Date(2026, 11, 12, 10, 0, 0, 0, time.UTC)}
	if _, err := rule.NextAfter(rule.UntilUTC); !errors.Is(err, ErrScheduleEnded) {
		t.Fatalf("after end should be stopped: %v", err)
	}
	for _, zone := range []string{"Local", "PST", "", "Made/Up", " UTC"} {
		rule.TimeZone = zone
		if rule.Valid() {
			t.Errorf("invalid or worker-local timezone accepted: %q", zone)
		}
	}
}

func TestScheduleBudgetBoundedAndOverflowSafe(t *testing.T) {
	b := ScheduleBudget{MaxOccurrences: 3, PerRunUpperBoundMicros: 2500}
	got, err := b.TotalUpperBoundMicros()
	if err != nil || got != 7500 {
		t.Fatalf("budget = %d, %v", got, err)
	}
	for _, bad := range []ScheduleBudget{
		{MaxOccurrences: 0, PerRunUpperBoundMicros: 100},
		{MaxOccurrences: 2, PerRunUpperBoundMicros: 0},
		{MaxOccurrences: 2, PerRunUpperBoundMicros: math.MaxUint64},
	} {
		if _, err := bad.TotalUpperBoundMicros(); !errors.Is(err, ErrInvalidSchedule) {
			t.Fatalf("invalid bound %+v accepted: %v", bad, err)
		}
	}
}

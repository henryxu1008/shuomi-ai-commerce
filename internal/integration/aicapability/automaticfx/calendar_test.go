package automaticfx

import (
	"testing"
	"time"
)

func TestNextExpiryRequiresCoverageAndUsesMarketDays(t *testing.T) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	known := Calendar{Years: map[int]YearCalendar{2026: {Source: "https://www.chinamoney.com.cn/official", CoverageFrom: "2026-01-01", CoverageUntil: "2026-12-31", ExceptionsComplete: true, Closed: []string{"2026-10-01", "2026-10-02", "2026-10-05", "2026-10-06", "2026-10-07"}}}}
	tests := []struct {
		name, day, want string
		calendar        Calendar
	}{
		{"weekend", "2026-11-13", "2026-11-16T09:15:00+08:00", known},
		{"public holiday", "2026-09-30", "2026-10-08T09:15:00+08:00", known},
		{"no calendar", "2026-11-13", "2026-11-14T09:15:00+08:00", Calendar{}},
		{"next year missing", "2026-12-31", "2027-01-01T09:15:00+08:00", known},
		{"partial coverage", "2026-11-13", "2026-11-15T09:15:00+08:00", Calendar{Years: map[int]YearCalendar{2026: {Source: "https://www.chinamoney.com.cn/official", CoverageFrom: "2026-11-13", CoverageUntil: "2026-11-14", ExceptionsComplete: true}}}},
		{"exceptions not verified", "2026-11-13", "2026-11-14T09:15:00+08:00", Calendar{Years: map[int]YearCalendar{2026: {Source: "https://www.chinamoney.com.cn/official", CoverageFrom: "2026-01-01", CoverageUntil: "2026-12-31"}}}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.calendar.NextExpiry(tt.day)
			if err != nil {
				t.Fatal(err)
			}
			want, err := time.Parse(time.RFC3339, tt.want)
			if err != nil {
				t.Fatal(err)
			}
			if !got.In(loc).Equal(want) {
				t.Fatalf("got %s want %s", got.In(loc), want)
			}
		})
	}
}

func TestAnnouncementEligibility(t *testing.T) {
	calendar := Calendar{Years: map[int]YearCalendar{2026: {Source: "https://www.chinamoney.com.cn/official", CoverageFrom: "2026-11-01", CoverageUntil: "2026-11-30", ExceptionsComplete: true}}}
	for _, tc := range []struct {
		day, now string
		valid    bool
	}{
		{"2026-11-12", "2026-11-12T09:14:59+08:00", false},
		{"2026-11-12", "2026-11-12T09:15:00+08:00", true},
		{"2026-11-14", "2026-11-14T12:00:00+08:00", false},
		{"2026-12-01", "2026-12-01T10:00:00+08:00", false},
	} {
		now, err := time.Parse(time.RFC3339, tc.now)
		if err != nil {
			t.Fatal(err)
		}
		if got := calendar.Eligible(tc.day, now); got != tc.valid {
			t.Fatalf("%s %s: got %v", tc.day, tc.now, got)
		}
	}
}

func TestCalendarRejectsUntrustedOrConflictingConfiguration(t *testing.T) {
	for _, tc := range []YearCalendar{
		{Source: "https://example.invalid/official", CoverageFrom: "2026-01-01", CoverageUntil: "2026-12-31", ExceptionsComplete: true},
		{Source: "https://www.chinamoney.com.cn/official", CoverageFrom: "2026-01-01", CoverageUntil: "2026-12-31", ExceptionsComplete: true, Closed: []string{"2026-11-13"}, Open: []string{"2026-11-13"}},
	} {
		if _, err := (Calendar{Years: map[int]YearCalendar{2026: tc}}).NextExpiry("2026-11-12"); err == nil {
			t.Fatal("expected invalid calendar")
		}
	}
}

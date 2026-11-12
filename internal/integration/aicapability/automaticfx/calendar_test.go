package automaticfx

import (
	"testing"
	"time"
)

func TestNextExpiryRequiresCoverageAndUsesMarketDays(t *testing.T) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	known := Calendar{Years: map[int]YearCalendar{2026: {Source: "https://www.chinamoney.com.cn/official", Closed: []string{"2026-10-01", "2026-10-02", "2026-10-05", "2026-10-06", "2026-10-07"}}}}
	tests := []struct {
		name, day, want string
		calendar        Calendar
	}{
		{"weekend", "2026-11-13", "2026-11-16T09:15:00+08:00", known},
		{"public holiday", "2026-09-30", "2026-10-08T09:15:00+08:00", known},
		{"no calendar", "2026-11-13", "2026-11-14T09:15:00+08:00", Calendar{}},
		{"next year missing", "2026-12-31", "2027-01-01T09:15:00+08:00", known},
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

func TestCalendarRejectsUntrustedOrConflictingConfiguration(t *testing.T) {
	for _, tc := range []YearCalendar{
		{Source: "https://example.invalid/official"},
		{Source: "https://www.chinamoney.com.cn/official", Closed: []string{"2026-11-13"}, Open: []string{"2026-11-13"}},
	} {
		if _, err := (Calendar{Years: map[int]YearCalendar{2026: tc}}).NextExpiry("2026-11-12"); err == nil {
			t.Fatal("expected invalid calendar")
		}
	}
}

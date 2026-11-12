package automaticfx

import (
	"strings"
	"testing"
	"time"
)

func TestParseCFETSObservation(t *testing.T) {
	now := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
	tests := []struct {
		name, body string
		wantMicros int64
		wantErr    bool
	}{
		{"observation", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":6.7174}`, 6717400, false},
		{"exact small", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":0.000001}`, 1, false},
		{"too precise", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":6.7174001}`, 0, true},
		{"excess decimal places", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":6.7174000}`, 0, true},
		{"quoted numeric", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":"6.7174"}`, 0, true},
		{"wrong pair", `{"date":"2026-11-12","base":"EUR","quote":"CNY","rate":6.7174}`, 0, true},
		{"future", `{"date":"2026-11-13","base":"USD","quote":"CNY","rate":6.7174}`, 0, true},
		{"zero", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":0}`, 0, true},
		{"duplicate", `{"date":"2026-11-12","base":"USD","base":"EUR","quote":"CNY","rate":6.7174}`, 0, true},
		{"trailing JSON", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":6.7174} {}`, 0, true},
		{"huge", `{"date":"2026-11-12","base":"USD","quote":"CNY","rate":999999999999999999999}`, 0, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ParseCFETSObservation(strings.NewReader(tt.body), now)
			if (err != nil) != tt.wantErr {
				t.Fatalf("error = %v", err)
			}
			if err == nil && (got.CNYMicrosPerUSD != tt.wantMicros || got.Date != "2026-11-12") {
				t.Fatalf("got %+v", got)
			}
		})
	}
}

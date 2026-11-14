package reportcenterpersistence

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestReportWireTimesRemainCanonicalUTCAfterDatabaseRead(t *testing.T) {
	local := time.Date(2026, 11, 14, 22, 30, 0, 0, time.FixedZone("UTC+8", 8*3600))
	raw, e := json.Marshal(rowSummary(reportRow{CapturedAt: local, SourceAt: &local}, false))
	if e != nil || !strings.Contains(string(raw), `"capturedAt":"2026-11-14T14:30:00Z"`) || !strings.Contains(string(raw), `"sourceAt":"2026-11-14T14:30:00Z"`) {
		t.Fatalf("database timezone leaked into UTC wire contract: %s %v", raw, e)
	}
	if local.Hour() != 22 {
		t.Fatal("normalization mutated source time")
	}
}

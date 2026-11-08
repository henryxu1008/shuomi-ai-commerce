package workbenchmetrics

import (
	"net/url"
	"testing"
	"time"
)

func TestParseCompleteDatesAndStores(t *testing.T) {
	now := time.Date(2026, 11, 7, 16, 10, 0, 0, time.UTC)
	first := "8ceec001-3e70-4811-9e05-000000000001"
	second := "8ceec001-3e70-4811-9e05-000000000002"
	query := "startDate=2026-11-01&endDate=2026-11-07&storeId=" + first
	filter, err := Parse(query+"&storeId="+second, now)
	if err != nil || !filter.Start.Equal(time.Date(2026, 10, 31, 16, 0, 0, 0, time.UTC)) ||
		!filter.End.Equal(time.Date(2026, 11, 7, 16, 0, 0, 0, time.UTC)) || len(filter.Stores) != 2 {
		t.Fatalf("expected the two stores and the UTC+8 half-open date range: %+v, %v", filter, err)
	}
	for _, raw := range []string{query + "&storeId=" + first, query + "&extra=1", query + "&endDate=2026-11-07",
		"startDate=2026-10-08&endDate=2026-11-07&storeId=" + first,
		"startDate=2026-11-08&endDate=2026-11-08&storeId=" + first,
		"startDate=2026-11-01&endDate=2026-11-07",
		"startDate=2026-11-01&endDate=2026-11-07&storeId=" + url.QueryEscape("not a store")} {
		if _, err := Parse(raw, now); err == nil {
			t.Errorf("accepted %s", raw)
		}
	}
}

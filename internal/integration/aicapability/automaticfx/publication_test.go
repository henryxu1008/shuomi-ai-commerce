package automaticfx

import (
	"testing"
	"time"

	d "task-processor/internal/aicapability/routeconfig"
)

func TestPrepareRetainsOtherFactsAndCutsOverUSD(t *testing.T) {
	now := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
	s := d.PlatformPricesSnapshot{FX: []d.PublishedFXRate{
		{Version: "old-usd", Currency: "USD", RateDate: "2026-11-11", CNYMicrosPerCurrencyUnit: 6719100, EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(time.Hour)},
		{Version: "keep-eur", Currency: "EUR", RateDate: "2026-11-11", CNYMicrosPerCurrencyUnit: 7800000, EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(time.Hour)},
	}}
	got, changed, err := PreparePublication("old-revision", s, Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}, now, now.Add(12*time.Hour))
	if err != nil || !changed {
		t.Fatalf("changed=%v err=%v", changed, err)
	}
	if got.ExpectedRevision != "old-revision" || len(got.FX) != 3 || !got.FX[0].EffectiveUntil.Equal(now) || got.FX[1].Currency != "EUR" || got.FX[1].Version != "" || got.FX[2].CNYMicrosPerCurrencyUnit != 6717400 || got.FX[2].SourceURL != officialCFETSSource {
		t.Fatalf("incorrect publication: %+v", got)
	}
	if s.FX[0].Version != "old-usd" || !s.FX[0].EffectiveUntil.Equal(now.Add(time.Hour)) {
		t.Fatal("mutated prior snapshot")
	}
}

func TestPrepareRejectsRollbackCorrectionAndFutureConflict(t *testing.T) {
	now := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
	base := d.PlatformPricesSnapshot{FX: []d.PublishedFXRate{{Currency: "USD", RateDate: "2026-11-12", CNYMicrosPerCurrencyUnit: 6717400, EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(time.Hour)}}}
	if _, changed, err := PreparePublication("rev", base, Observation{Date: "2026-11-12", CNYMicrosPerUSD: 6717400}, now, now.Add(time.Hour)); err != nil || changed {
		t.Fatalf("same observation: %v %v", changed, err)
	}
	for _, obs := range []Observation{{Date: "2026-11-11", CNYMicrosPerUSD: 6717400}, {Date: "2026-11-12", CNYMicrosPerUSD: 6719000}} {
		if _, _, err := PreparePublication("rev", base, obs, now, now.Add(time.Hour)); err == nil {
			t.Fatal("expected rejection")
		}
	}
	base.FX = append(base.FX, d.PublishedFXRate{Currency: "USD", RateDate: "2026-11-13", CNYMicrosPerCurrencyUnit: 6717400, EffectiveFrom: now.Add(2 * time.Hour), EffectiveUntil: now.Add(3 * time.Hour)})
	if _, _, err := PreparePublication("rev", base, Observation{Date: "2026-11-13", CNYMicrosPerUSD: 6717400}, now, now.Add(4*time.Hour)); err == nil {
		t.Fatal("expected future conflict")
	}
}

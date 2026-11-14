package automaticfx

import (
	"github.com/stretchr/testify/require"
	d "task-processor/internal/aicapability/routeconfig"
	"testing"
	"time"
)

func TestWeeklyPublicationExpiryAndSourceDate(t *testing.T) {
	loc := time.FixedZone("Asia/Shanghai", 8*3600)
	now := time.Date(2026, 11, 12, 9, 15, 0, 0, loc)
	for _, tc := range []struct {
		day   string
		at    time.Time
		valid bool
	}{
		{"2026-11-12", now.Add(-time.Second), false}, {"2026-11-12", now, true},
		{"2026-11-13", now, false}, {"2026-11-05", now, true}, {"2026-11-04", now, false},
	} {
		input, changed, err := PrepareWeeklyPublication("rev", d.PlatformPricesSnapshot{}, Observation{Date: tc.day, CNYMicrosPerUSD: 6717400}, tc.at)
		if tc.valid {
			require.NoError(t, err)
			require.True(t, changed)
			require.Equal(t, tc.at.Add(7*24*time.Hour), input.FX[0].EffectiveUntil)
		} else {
			require.Error(t, err)
		}
	}
}

func TestWeeklyRepeatedAnnouncementDoesNotExtendExpiry(t *testing.T) {
	now := time.Date(2026, 11, 12, 2, 0, 0, 0, time.UTC)
	prior := d.PublishedFXRate{Currency: "USD", RateDate: "2026-11-11", CNYMicrosPerCurrencyUnit: 6717400, EffectiveFrom: now.Add(-time.Hour), EffectiveUntil: now.Add(6 * 24 * time.Hour)}
	_, changed, err := PrepareWeeklyPublication("rev", d.PlatformPricesSnapshot{FX: []d.PublishedFXRate{prior}}, Observation{Date: prior.RateDate, CNYMicrosPerUSD: prior.CNYMicrosPerCurrencyUnit}, now)
	require.NoError(t, err)
	require.False(t, changed)
}

package routeconfig

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestPublishedTextWindowGroupRequiresCompletePeakIdleAndCoverage(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	peak := PublishedModelPrice{ID: "synthetic-deepseek", Provider: "deepseek", Model: "deepseek-v4.1-flash", Channel: "bailian-beijing", Region: "cn-beijing", ServiceType: "standard-sync", Currency: "CNY", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 2_000_000, CachedInputText: 40_000, OutputText: 8_000_000}, SourceURL: "https://api-docs.deepseek.com/zh-cn/news/news260910/", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(time.Hour)}
	idle := peak
	idle.EffectiveFrom, idle.EffectiveUntil = peak.EffectiveUntil, now.Add(2*time.Hour)
	idle.Rates = aicapability.OfficialModelRates{InputText: 1_000_000, CachedInputText: 20_000, OutputText: 4_000_000}
	in := PlatformPricesInput{Prices: []PublishedModelPrice{idle, peak}}
	s, err := FreezePlatformPrices(in, "synthetic-revision", nil, now)
	require.NoError(t, err)
	r := TextPriceWindowRequest{ID: peak.ID, Revision: "synthetic-revision", Model: peak.Model, Channel: peak.Channel, Region: peak.Region, ServiceType: peak.ServiceType, ClockPolicyVersion: "synthetic-clock-not-admitted", BoundEvidenceVersion: "synthetic-complete-evidence", Now: now.Add(30 * time.Minute), Until: now.Add(90 * time.Minute)}
	p, err := s.ResolveTextWindows(r)
	require.NoError(t, err)
	require.Equal(t, peak.Rates, p.Rates)
	_, err = s.Resolve(peak.ID, peak.Model, peak.Channel, r.Now)
	require.Error(t, err, "a single current price cannot bypass the complete time proof")
	for _, kind := range []string{"idle-only", "peak-only", "other-region", "other-model", "channel-cache-price", "gap", "deadline", "evidence", "revision"} {
		t.Run(kind, func(t *testing.T) {
			candidate := PlatformPricesInput{Prices: append([]PublishedModelPrice(nil), in.Prices...)}
			request := r
			switch kind {
			case "idle-only":
				candidate.Prices = candidate.Prices[:1]
				request.Now = now.Add(70 * time.Minute)
			case "peak-only":
				candidate.Prices = candidate.Prices[1:]
				request.Until = now.Add(50 * time.Minute)
			case "other-region":
				candidate.Prices[0].Region = "other"
			case "other-model":
				candidate.Prices[0].Model = "deepseek-v4.1-flash-1017"
			case "channel-cache-price":
				candidate.Prices[0].Rates.CachedInputText = 100_000
			case "gap":
				candidate.Prices[0].EffectiveFrom = now.Add(61 * time.Minute)
			case "deadline":
				request.Until = now.Add(3 * time.Hour)
			case "evidence":
				request.BoundEvidenceVersion = ""
			case "revision":
				request.Revision = ""
			}
			snapshot, err := FreezePlatformPrices(candidate, "synthetic-revision", nil, now)
			require.NoError(t, err)
			_, err = snapshot.ResolveTextWindows(request)
			require.Error(t, err)
		})
	}
}

package routeconfig

import (
	"github.com/stretchr/testify/require"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestOfficialPriceAndFXSnapshotsStaySeparateFromEnterpriseChoices(t *testing.T) {
	now := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	input := PlatformPricesInput{ExpectedRevision: NoRevision, Prices: []PublishedModelPrice{{ID: "mini-standard", Region: "global", ServiceType: "standard-sync", Provider: "openai", Channel: "openai-direct", Model: "synthetic-mini", Currency: "USD", UsageVersion: "openai-text-v1", Rates: aicapability.OfficialModelRates{InputText: 750000, CachedInputText: 75000, OutputText: 4500000}, SourceURL: "https://developers.openai.com/api/docs/pricing", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(24 * time.Hour)}}, FX: []PublishedFXRate{{Currency: "USD", CNYMicrosPerCurrencyUnit: 7000000, SourceURL: "https://www.chinamoney.com.cn/", RateDate: "2026-11-03", VerifiedAt: now, EffectiveFrom: now, EffectiveUntil: now.Add(24 * time.Hour)}}}
	snapshot, err := FreezePlatformPrices(input, "revision-1", nil, now)
	require.NoError(t, err)
	price, err := snapshot.Resolve("mini-standard", "synthetic-mini", "openai-direct", now)
	require.NoError(t, err)
	require.Equal(t, "USD", price.Currency)
	require.EqualValues(t, 7000000, price.CNYMicrosPerCurrencyUnit)
	require.NotEmpty(t, price.PriceVersion)
	require.NotEmpty(t, price.FXVersion)
	_, err = snapshot.Resolve("mini-standard", "synthetic-mini", "another-channel", now)
	require.Error(t, err)
	_, err = snapshot.Resolve("mini-standard", "synthetic-mini", "openai-direct", now.Add(24*time.Hour))
	require.Error(t, err)
	next, err := FreezePlatformPrices(input, "revision-2", &snapshot, now)
	require.NoError(t, err)
	require.Equal(t, snapshot.Prices[0].Version, next.Prices[0].Version, "unmodified price facts are reused without invalidating an unrelated category")
	input.Prices[0].Rates.OutputText++
	changed, err := FreezePlatformPrices(input, "revision-3", &next, now)
	require.NoError(t, err)
	require.NotEqual(t, next.Prices[0].Version, changed.Prices[0].Version)
	input.Prices = append(input.Prices, input.Prices[0])
	_, err = FreezePlatformPrices(input, "bad", nil, now)
	require.Error(t, err, "overlapping applicable price ranges cannot be ambiguous")
}

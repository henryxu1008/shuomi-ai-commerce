package aicapability

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func syntheticWindowTariff(t *testing.T) OfficialModelTariff {
	t.Helper()
	windows, err := NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", "synthetic-clock-not-provider-admission", "synthetic-complete-bound", []OfficialTextPriceWindow{
		{EntryVersion: "peak", FromUTC: "2026-11-03T00:00:00Z", UntilUTC: "2026-11-03T01:00:00Z", Rates: OfficialModelRates{InputText: 2_000_000, CachedInputText: 40_000, OutputText: 8_000_000}},
		{EntryVersion: "idle", FromUTC: "2026-11-03T01:00:00Z", UntilUTC: "2026-11-03T02:00:00Z", Rates: OfficialModelRates{InputText: 1_000_000, CachedInputText: 20_000, OutputText: 4_000_000}},
	})
	require.NoError(t, err)
	return OfficialModelTariff{UnitVersion: AIPointMicrounitV1, PriceVersion: windows.Version(), FXVersion: "cny-identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1_000_000, Model: "synthetic-window-model", Channel: "synthetic-channel", UsageVersion: "openai-text-v1", Rates: windows.MaximumRates(), TextPriceWindows: windows}
}

func TestFrozenTextWindowsReserveUpperBoundAndSettleUniqueTrustedTime(t *testing.T) {
	p := syntheticWindowTariff(t)
	r, err := p.ReserveTextMicropoints(32000, 4000)
	require.NoError(t, err)
	require.EqualValues(t, 9_600_000, r, "one of the two allowed calls; complete peak/idle upper bound")
	u := BillableModelUsage{MappingVersion: p.UsageVersion, Complete: true, InputText: 32000, OutputText: 4000, BillingTimePolicyVersion: p.TextPriceWindows.ClockPolicyVersion()}
	for _, tc := range []struct {
		at       string
		expected int64
	}{{"2026-11-03T00:59:59.999999999Z", 9_600_000}, {"2026-11-03T01:00:00Z", 4_800_000}} {
		u.BillingTimeUTC = tc.at
		a, err := p.SettlementMicropoints(u)
		require.NoError(t, err)
		require.Equal(t, tc.expected, a)
		require.LessOrEqual(t, a, r)
	}
	for _, at := range []string{"", "2026-11-02T23:59:59Z", "2026-11-03T02:00:00Z", "2026-11-03T09:00:00+08:00"} {
		u.BillingTimeUTC = at
		_, err = p.SettlementMicropoints(u)
		require.Error(t, err, "absent, out-of-group or noncanonical time cannot charge: %s", at)
	}
	u.BillingTimeUTC = "2026-11-03T01:00:00Z"
	u.BillingTimePolicyVersion = "other-clock"
	_, err = p.SettlementMicropoints(u)
	require.Error(t, err)
	p.Rates.InputText = 1_000_000
	require.Error(t, p.Validate(), "outer rates cannot lower a frozen peak bound")
}

func TestFrozenTextWindowsPreserveComparableValueAndStrictJSON(t *testing.T) {
	p := syntheticWindowTariff(t)
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	var roundtrip OfficialModelTariff
	require.NoError(t, json.Unmarshal(raw, &roundtrip))
	require.True(t, p == roundtrip)
	a, b := ModelProfile{OfficialTariff: p}, ModelProfile{OfficialTariff: roundtrip}
	require.True(t, a == b, "current/frozen ModelProfile keeps direct value equality")
	require.NotContains(t, string(raw), `"fromUTC":""`)
	for _, invalid := range []string{`null`, `{}`, `{"schemaVersion":1,"schemaVersion":1}`, strings.Replace(string(mustWindowJSON(t, p.TextPriceWindows)), `"windows":[`, `"unknown":1,"windows":[`, 1), strings.Replace(string(mustWindowJSON(t, p.TextPriceWindows)), `"entryVersion":"peak"`, `"entryVersion":"changed"`, 1)} {
		var g OfficialTextPriceWindows
		require.Error(t, json.Unmarshal([]byte(invalid), &g), invalid)
	}
	entries := p.TextPriceWindows.Entries()
	entries[0].EntryVersion = "changed"
	require.True(t, p == roundtrip, "caller cannot mutate frozen array through an accessor")
	require.True(t, p.TextPriceWindows.Covers(time.Date(2026, 11, 3, 0, 30, 0, 0, time.UTC), time.Date(2026, 11, 3, 1, 30, 0, 0, time.UTC)))
	require.False(t, p.TextPriceWindows.Covers(time.Date(2026, 11, 3, 1, 30, 0, 0, time.UTC), time.Date(2026, 11, 3, 2, 0, 0, 1, time.UTC)))
}

func mustWindowJSON(t *testing.T, g OfficialTextPriceWindows) []byte {
	t.Helper()
	raw, err := json.Marshal(g)
	require.NoError(t, err)
	return raw
}

func TestNonWindowOfficialJSONRemainsByteIdentical(t *testing.T) {
	p := officialMiniPrice()
	raw, err := json.Marshal(p)
	require.NoError(t, err)
	require.Equal(t, `{"unitVersion":"ai-micropoint-v1","priceVersion":"official-mini-standard-v1","fxVersion":"synthetic-fx-7","currency":"USD","cnyMicrosPerCurrencyUnit":7000000,"model":"gpt-5.4-mini-2026-03-17","channel":"openai-direct","usageVersion":"openai-text-v1","rates":{"inputText":750000,"cachedInputText":75000,"inputImage":0,"cachedInputImage":0,"outputText":4500000,"outputImage":0}}`, string(raw))
	u := BillableModelUsage{MappingVersion: "openai-text-v1", Complete: true, InputText: 10, OutputText: 2}
	raw, err = json.Marshal(u)
	require.NoError(t, err)
	require.Equal(t, `{"MappingVersion":"openai-text-v1","Complete":true,"InputText":10,"CachedInputText":0,"InputImage":0,"CachedInputImage":0,"OutputText":2,"OutputImage":0}`, string(raw))
}

func TestTextWindowCanonicalTimesTailsAndWholeProfileBound(t *testing.T) {
	p := syntheticWindowTariff(t)
	entries := p.TextPriceWindows.Entries()
	entries[0].FromUTC = "2026-11-03T08:00:00+08:00"
	normalized, err := NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", p.TextPriceWindows.ClockPolicyVersion(), "synthetic-complete-bound", []OfficialTextPriceWindow{entries[1], entries[0]})
	require.NoError(t, err)
	require.True(t, normalized == p.TextPriceWindows, "time aliases/order canonicalize before value equality")
	poisoned := normalized
	poisoned.entries[31] = entries[0]
	require.Error(t, poisoned.Validate(), "inactive tails must remain zero")
	entries[0].Rates.CachedInputText = 0
	entries[1].Rates.CachedInputText = 0
	zeroCache, err := NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", normalized.ClockPolicyVersion(), "synthetic-complete-bound", entries)
	require.NoError(t, err)
	raw := mustWindowJSON(t, zeroCache)
	raw = []byte(strings.Replace(string(raw), `"cachedInputText":0`, `"cachedInputText":null`, 1))
	var decoded OfficialTextPriceWindows
	require.Error(t, json.Unmarshal(raw, &decoded), "null is not an explicitly proven zero rate")
	base := time.Date(2026, 11, 3, 0, 0, 0, 0, time.UTC)
	all := []OfficialTextPriceWindow{}
	checkedWholeBound := false
	for n := 1; n <= 33; n++ {
		all = append(all, OfficialTextPriceWindow{EntryVersion: fmt.Sprintf("%02d-%s", n, strings.Repeat("v", 120)), FromUTC: base.Add(time.Duration(n-1) * time.Minute).Format(time.RFC3339Nano), UntilUTC: base.Add(time.Duration(n) * time.Minute).Format(time.RFC3339Nano), Rates: p.Rates})
		g, err := NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", normalized.ClockPolicyVersion(), "synthetic-complete-bound", all)
		if err != nil {
			continue
		}
		profile := testTextProfile()
		profile.OfficialTariff = p
		profile.ModelID = p.Model
		profile.OfficialTariff.TextPriceWindows = g
		profile.OfficialTariff.PriceVersion = g.Version()
		raw, err := json.Marshal(profile)
		require.NoError(t, err)
		if len(raw) > 8192 {
			require.Error(t, profile.Validate(), "nested profile must fit the original bound even when group alone fits")
			checkedWholeBound = true
		}
	}
	require.True(t, checkedWholeBound)
	_, err = NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", normalized.ClockPolicyVersion(), "synthetic-complete-bound", all)
	require.Error(t, err, "original max32 and byte bounds are not enlarged")
}

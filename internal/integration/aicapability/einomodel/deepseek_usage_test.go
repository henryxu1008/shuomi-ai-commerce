package einomodel

import (
	"context"
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	"testing"
	"time"
)

func TestFrozenTextWindowCoverageIsCheckedBeforeClaimAndAtFreshAdmission(t *testing.T) {
	now := time.Now().UTC()
	for _, tc := range []struct {
		name    string
		end     time.Time
		allowed bool
	}{{"expired", now.Add(-time.Second), false}, {"before-deadline", now.Add(time.Second), false}, {"covered", now.Add(time.Minute), true}} {
		t.Run(tc.name, func(t *testing.T) {
			input := deepSeekTestInput("http://127.0.0.1:1/v1")
			input.Profile.ProviderID, input.Profile.ModelID, input.Profile.AdapterPolicyVersion = "synthetic", "synthetic-window-model", "synthetic-adapter"
			g, err := aicapability.NewOfficialTextPriceWindows("synthetic-price", "synthetic-revision", "synthetic-clock", "synthetic-bound", []aicapability.OfficialTextPriceWindow{{EntryVersion: "synthetic-entry", FromUTC: now.Add(-time.Minute).Format(time.RFC3339Nano), UntilUTC: tc.end.Format(time.RFC3339Nano), Rates: aicapability.OfficialModelRates{InputText: 10000, OutputText: 20000}}})
			require.NoError(t, err)
			input.Profile.OfficialTariff = aicapability.OfficialModelTariff{UnitVersion: aicapability.AIPointMicrounitV1, PriceVersion: g.Version(), FXVersion: "cny-identity-v1", Currency: "CNY", CNYMicrosPerCurrencyUnit: 1_000_000, Model: input.Profile.ModelID, Channel: "synthetic-channel", UsageVersion: "openai-text-v1", Rates: g.MaximumRates(), TextPriceWindows: g}
			ledger := &recordingLedger{}
			executor := deepSeekTestExecutor("http://127.0.0.1:1/v1", input.Profile, ledger)
			ctx, cancel := context.WithDeadline(context.Background(), now.Add(30*time.Second))
			defer cancel()
			_, err = executor.admitted(ctx, input)
			if tc.allowed {
				require.NoError(t, err)
			} else {
				require.ErrorIs(t, err, ErrNotDispatched)
			}
			require.False(t, ledger.claimed)
			require.False(t, ledger.reserved)
		})
	}
}

func TestDeepSeekPartitionsKeepPresenceAndNeverInventClockEvidence(t *testing.T) {
	const raw = `{"model":"deepseek-v4.1-flash","created":1793664000,"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}}}`
	u, err := observeDeepSeekPartitions([]byte(raw))
	require.NoError(t, err)
	require.EqualValues(t, 10, u.InputText)
	require.EqualValues(t, 3, u.CachedInputText)
	require.EqualValues(t, 5, u.OutputText)
	require.Empty(t, u.BillingTimeUTC)
	require.Empty(t, u.BillingTimePolicyVersion)
	require.Nil(t, observeDeepSeekUsage([]byte(raw)).BillableUsage)
	for _, bad := range []string{
		strings.Replace(raw, `"cached_tokens":3`, `"cached_tokens":null`, 1),
		strings.Replace(raw, `"cached_tokens":3`, `"cached_tokens":11`, 1),
		strings.Replace(raw, `"cached_tokens":3`, `"cached_tokens":-1`, 1),
		strings.Replace(raw, `"cached_tokens":3`, `"cached_tokens":3,"cached_tokens":0`, 1),
		strings.Replace(raw, `"cached_tokens":3`, `"extra_billed_input":3`, 1),
		strings.Replace(raw, `"reasoning_tokens":2`, `"reasoning_tokens":6`, 1),
		strings.Replace(raw, `"prompt_tokens":10`, `"audio_tokens":1,"prompt_tokens":10`, 1),
		strings.Replace(raw, `"model":"deepseek-v4.1-flash"`, `"model":"deepseek-v4.1-flash-1017"`, 1),
		strings.Replace(raw, `"total_tokens":15`, `"total_tokens":16`, 1),
		strings.Replace(raw, `"model":"deepseek-v4.1-flash"`, `"model":"deepseek-v4.1-flash","model":"deepseek-v4.1-flash"`, 1),
	} {
		_, err := observeDeepSeekPartitions([]byte(bad))
		require.ErrorIs(t, err, ErrUsageUnknown, bad)
		require.False(t, observeDeepSeekUsage([]byte(bad)).Known)
	}
}

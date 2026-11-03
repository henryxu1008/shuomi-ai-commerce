package einomodel

import (
	"github.com/stretchr/testify/require"
	"strings"
	"task-processor/internal/aicapability"
	"testing"
)

func TestDeepSeekReferenceTotalsDoesNotRequireCacheOrBillingClock(t *testing.T) {
	for _, detail := range []string{"", `,"prompt_tokens_details":{"cached_tokens":3}`, `,"new_provider_metadata":{"future":true}`} {
		u := observeDeepSeekUsage([]byte(`{"model":"deepseek-v4.1-flash","usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15` + detail + `}}`))
		require.True(t, u.Known)
		require.NotNil(t, u.BillableUsage)
		require.Equal(t, aicapability.ReferenceTextTotalsV1, u.BillableUsage.MappingVersion)
		require.True(t, u.BillableUsage.Complete, "only this policy's explicit basic counters are complete")
		require.EqualValues(t, 10, u.BillableUsage.InputText)
		require.EqualValues(t, 5, u.BillableUsage.OutputText)
	}
}

func TestDeepSeekReferenceTotalsRejectsMissingContradictoryOrAmbiguousBasics(t *testing.T) {
	const raw = `{"model":"deepseek-v4.1-flash","created":1793664000,"usage":{"prompt_tokens":10,"completion_tokens":5,"total_tokens":15,"prompt_tokens_details":{"cached_tokens":3},"completion_tokens_details":{"reasoning_tokens":2}}}`
	for _, bad := range []string{
		strings.Replace(raw, `"prompt_tokens":10`, `"prompt_tokens":null`, 1),
		strings.Replace(raw, `"prompt_tokens":10,`, ``, 1),
		strings.Replace(raw, `"prompt_tokens":10`, `"prompt_tokens":-1`, 1),
		strings.Replace(raw, `"prompt_tokens":10`, `"prompt_tokens":1073741825`, 1),
		strings.Replace(raw, `"prompt_tokens":10`, `"prompt_tokens":10,"prompt_tokens":10`, 1),
		strings.Replace(raw, `"model":"deepseek-v4.1-flash"`, `"model":"deepseek-v4.1-flash-1017"`, 1),
		strings.Replace(raw, `"total_tokens":15`, `"total_tokens":16`, 1),
		strings.Replace(raw, `"model":"deepseek-v4.1-flash"`, `"model":"deepseek-v4.1-flash","model":"deepseek-v4.1-flash"`, 1),
	} {
		require.False(t, observeDeepSeekUsage([]byte(bad)).Known, bad)
		require.Nil(t, observeDeepSeekUsage([]byte(bad)).BillableUsage)
	}
}

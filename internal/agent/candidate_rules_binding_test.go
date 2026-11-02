package agent

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

func candidateRulesBindingFixture(t *testing.T, category string) Binding {
	t.Helper()
	raw := `{"ContextKind":"listing-candidate-rules-v1","ContextID":"a4b35ad2-8f83-4f65-bdf1-67bb9e4c71cb","ProductKey":"product","CatalogVersion":"1","PublicationID":"publication","TargetPlatform":"shein","CandidateRules":{"BaseRevision":"1","BaseDigest":"` + strings.Repeat("a", 64) + `","StoreID":"54eef217-d263-4677-9619-6ea5e5120a13","Site":"us","MerchantDigest":"` + strings.Repeat("b", 64) + `","CandidateCategoryID":"` + category + `","SelectorDigest":"` + strings.Repeat("c", 64) + `"}}`
	var binding Binding
	require.NoError(t, json.Unmarshal([]byte(raw), &binding))
	return binding
}

func TestCandidateRulesBindingRequiresCompleteServerSelector(t *testing.T) {
	b := Binding{ContextKind: "listing-candidate-rules-v1", ContextID: "record", ProductKey: "product", CatalogVersion: "1", PublicationID: "publication", TargetPlatform: "shein"}
	require.False(t, b.Valid(), "query context without its exact selector must not pass generic Product binding admission")
	require.True(t, candidateRulesBindingFixture(t, "22").Valid())
	for _, category := range []string{"", "0", "-1", "022", "1.0", "9223372036854775808"} {
		require.False(t, candidateRulesBindingFixture(t, category).Valid(), category)
	}
}

func TestCandidateRulesSelectorParticipatesInRequestIdentity(t *testing.T) {
	a, b := candidateRulesBindingFixture(t, "22"), candidateRulesBindingFixture(t, "23")
	require.NotEqual(t, a, b, "a different selected category cannot disappear from the comparable frozen binding")
	a.ContextKind = "listing-target-record"
	require.False(t, a.Valid(), "ordinary material binding must reject a query selector")
	b.TargetPlatform = "walmart"
	require.False(t, b.Valid(), "candidate query is SHEIN only")
}

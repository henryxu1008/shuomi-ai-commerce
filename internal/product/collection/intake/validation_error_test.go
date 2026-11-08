package intake

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestValidationIssueRoundTripExcludesArbitraryContent(t *testing.T) {
	issue := ValidationIssue{"rows[1].stock", "stock_evidence"}
	require.Equal(t, &issue, IssueFromToken(IssueToken(issue)))
	for _, token := range []string{
		`own-intake-validation:{"field":"secret model text","code":"stock_evidence"}`,
		`own-intake-validation:{"field":"rows[1].stock","code":"secret provider error"}`,
		`own-intake-validation:{"field":"rows[1].stock","code":"stock_evidence","raw":"secret"}`,
		`own-intake-validation:{"field":"rows[256].stock","code":"stock_evidence"}`,
		`own-intake-validation:{"field":"rows[1].stock","field":"rows[2].stock","code":"stock_evidence"}`,
	} {
		require.Nil(t, IssueFromToken(token))
	}
	require.NotContains(t, IssueToken(errors.New("secret response")), "secret")
	for _, raw := range []string{`{"title":`, `{"title":"ok","secret":"never return this"}`} {
		_, err := Decode([]byte(raw), Input{Notes: "测试"})
		require.ErrorIs(t, err, ErrInvalid)
		require.NotContains(t, IssueToken(err), "secret")
		require.NotNil(t, IssueFromToken(IssueToken(err)))
	}
}

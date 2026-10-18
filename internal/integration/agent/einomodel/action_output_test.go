package einomodel

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func TestModelActionTypeRejectionIdentifiesTheContractWithoutRetainingContent(t *testing.T) {
	for _, tc := range []struct{ raw, field string }{
		{`{"Kind":"propose","Candidate":{"Changes":[{"Field":"selling_points","Value":["private product text"],"EvidenceIDs":[]}]}}`, "Candidate.Changes.Value"},
		{`{"Kind":"propose","Confidence":[{"Field":"description","Value":"private product text","Known":true}]}`, "Confidence.Value"},
	} {
		_, err := decodeModelAction(tc.raw)
		require.ErrorIs(t, err, governed.ErrInvalid)
		var rejection *actionOutputError
		require.True(t, errors.As(err, &rejection))
		require.Equal(t, "json_type", rejection.Code)
		require.Equal(t, tc.field, rejection.Field)
		require.NotContains(t, err.Error(), "private product text")
	}
	for _, raw := range []string{`{"Kind":"propose","private_unknown_key":"secret"}`, `{"Kind":"tool","Kind":"propose"}`, "```json\n{}\n```"} {
		_, err := decodeModelAction(raw)
		require.ErrorIs(t, err, governed.ErrInvalid)
		require.NotContains(t, err.Error(), "private_unknown_key")
		require.NotContains(t, err.Error(), "secret")
	}
}

func TestModelActionAcceptsOnlyTheCanonicalStringEncodedArray(t *testing.T) {
	for _, line := range strings.Split(contentTextSystem, "\n") {
		if strings.HasPrefix(line, `{"Kind":"propose"`) {
			action, err := decodeModelAction(line)
			require.NoError(t, err)
			require.Equal(t, "propose", action.Kind)
			require.Len(t, action.Candidate.Changes, 4)
		}
	}
}

package einomodel

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	logtest "github.com/sirupsen/logrus/hooks/test"
	"github.com/stretchr/testify/require"
	"task-processor/internal/agent"
	governed "task-processor/internal/integration/aicapability/einomodel"
)

func TestModelActionStrictRejectionDistinguishesProtocolFailures(t *testing.T) {
	for _, tc := range []struct{ name, raw, code, field string }{
		{"unknown root", `{"Kind":"propose","private_product_note":"secret product copy"}`, "unknown_field", "Action"},
		{"unknown nested", `{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"secret product copy","private_key":"private_value"}]}}`, "unknown_field", "Candidate.Changes"},
		{"wrong case", `{"kind":"interrupt"}`, "unknown_field", "Kind"},
		{"duplicate root", `{"Kind":"interrupt","Kind":"propose"}`, "duplicate_field", "Kind"},
		{"duplicate nested", `{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"secret first","Value":"secret second"}]}}`, "duplicate_field", "Candidate.Changes.Value"},
		{"mixed", `{"Kind":"interrupt","Kind":"propose","private_key":"secret product copy"}`, "unknown_and_duplicate_fields", "Kind"},
		{"duplicate unknown key", `{"Kind":"interrupt","private_key":"secret first","private_key":"secret second"}`, "unknown_and_duplicate_fields", "Action"},
		{"protocol-shaped unknown key", `{"Kind":"interrupt","Candidate.Changes[0].Value":"secret product copy"}`, "unknown_field", "Candidate.Changes.Value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			action, err := decodeModelAction(tc.raw)
			require.ErrorIs(t, err, governed.ErrInvalid)
			require.Equal(t, agent.Action{}, action, "a diagnostic decode must never admit rejected output")
			var rejection *actionOutputError
			require.ErrorAs(t, err, &rejection)
			require.Equal(t, tc.code, rejection.Code)
			require.Equal(t, tc.field, rejection.Field)
			require.NotContains(t, err.Error(), "private_")
			require.NotContains(t, err.Error(), "secret")
		})
	}
}

func TestModelActionRejectionLogContainsOnlyProtocolMetadata(t *testing.T) {
	logger := logrus.StandardLogger()
	previousHooks := logger.ReplaceHooks(make(logrus.LevelHooks))
	defer logger.ReplaceHooks(previousHooks)
	hook := logtest.NewGlobal()
	raw := `{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"private value","do not leak this key":"private details"}]}}`
	_, err := decodeModelAction(raw)
	require.Error(t, err)
	reportActionOutputFailure(context.Background(), "invocation-private", raw, err)
	entry := hook.LastEntry()
	require.NotNil(t, entry)
	require.Len(t, entry.Data, 6)
	require.Equal(t, "unknown_field", entry.Data["category"])
	require.Equal(t, "Candidate.Changes", entry.Data["field"])
	invocationHash, outputHash := sha256.Sum256([]byte("invocation-private")), sha256.Sum256([]byte(raw))
	require.Equal(t, hex.EncodeToString(invocationHash[:]), entry.Data["invocation_id_hash"])
	require.Equal(t, hex.EncodeToString(outputHash[:]), entry.Data["output_hash"])
	encoded, err := json.Marshal(entry.Data)
	require.NoError(t, err)
	for _, private := range []string{"private value", "private details", "do not leak this key", "invocation-private"} {
		require.NotContains(t, string(encoded), private)
	}
}

func TestModelActionStrictDiagnosticsPreserveValidEnglishCopy(t *testing.T) {
	raw := `{"Kind":"propose","Candidate":{"Changes":[{"Field":"title","Value":"DRÖNA \"box\" — it's dark grey","EvidenceIDs":["source-1"]},{"Field":"selling_points","Value":"[\"Fabric storage\",\"Fits the product's purpose\"]","EvidenceIDs":["source-1"]}]}}`
	action, err := decodeModelAction(raw)
	require.NoError(t, err)
	require.Equal(t, `DRÖNA "box" — it's dark grey`, action.Candidate.Changes[0].Value)
	require.JSONEq(t, `["Fabric storage","Fits the product's purpose"]`, action.Candidate.Changes[1].Value)
}

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

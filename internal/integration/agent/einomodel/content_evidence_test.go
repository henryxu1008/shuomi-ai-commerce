package einomodel

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	reviewproposal "task-processor/internal/product/review/proposal"
)

func TestContentPromptNeverSilentlyOmitsSelectedWholeValues(t *testing.T) {
	selection, err := reviewproposal.NewContentSelection("en-US", []string{"description"})
	require.NoError(t, err)
	for _, description := range []string{"whole selected description", strings.Repeat("selected", 1500)} {
		raw, err := json.Marshal(map[string]any{"product_key": "product", "catalog_version": "1", "catalog_publication_id": "publication", "snapshot": map[string]any{"sources": []any{map[string]string{"detail": "source"}}, "title": "source title", "description": description, "images": []any{map[string]string{"url": strings.Repeat("unselected", 3000)}}}})
		require.NoError(t, err)
		projected, err := ContentEvidenceForPrompt(raw, selection)
		if len(description) > promptEvidenceBytes {
			require.Error(t, err)
			require.Nil(t, projected)
			continue
		}
		require.NoError(t, err)
		var view titleEvidenceView
		require.NoError(t, json.Unmarshal(projected, &view))
		require.Equal(t, "content-evidence-v1", view.Projection)
		require.Equal(t, agentTextHash(raw), view.OriginalSHA256)
		var snapshot map[string]json.RawMessage
		require.NoError(t, json.Unmarshal(view.Evidence["snapshot"], &snapshot))
		var got string
		require.NoError(t, json.Unmarshal(snapshot["description"], &got))
		require.Equal(t, description, got)
		require.Contains(t, view.OmittedFields, "snapshot.images")
		require.False(t, view.Complete)
	}
}
